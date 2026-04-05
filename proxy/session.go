package proxy

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"goproxy/storage"
)

type RequestOptions struct {
	Region     string
	State      string
	SessionID  string
	SessionTTL time.Duration
	SessionKey string
}

type sessionEntry struct {
	Address   string
	ExpiresAt time.Time
}

type SessionManager struct {
	mu       sync.Mutex
	sessions map[string]sessionEntry
	locks    sync.Map
}

func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[string]sessionEntry),
	}
}

func (m *SessionManager) Get(key string) (string, bool) {
	if key == "" {
		return "", false
	}

	now := time.Now()
	m.mu.Lock()
	defer m.mu.Unlock()

	entry, ok := m.sessions[key]
	if !ok {
		return "", false
	}
	if now.After(entry.ExpiresAt) {
		delete(m.sessions, key)
		return "", false
	}
	return entry.Address, true
}

func (m *SessionManager) Put(key, address string, ttl time.Duration) {
	if key == "" || address == "" || ttl <= 0 {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[key] = sessionEntry{
		Address:   address,
		ExpiresAt: time.Now().Add(ttl),
	}
}

func (m *SessionManager) Delete(key string) {
	if key == "" {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, key)
}

func (m *SessionManager) LockKey(key string) func() {
	if key == "" {
		return func() {}
	}

	lockValue, _ := m.locks.LoadOrStore(key, &sync.Mutex{})
	lock := lockValue.(*sync.Mutex)
	lock.Lock()
	return lock.Unlock
}

func parseUsernameOptions(expectedBase, username string, namespace string) (RequestOptions, error) {
	opts := RequestOptions{}

	if username == expectedBase {
		return opts, nil
	}
	if !strings.HasPrefix(username, expectedBase+"-") {
		return opts, fmt.Errorf("username mismatch")
	}

	tokens := strings.Split(strings.TrimPrefix(username, expectedBase+"-"), "-")
	for i := 0; i+1 < len(tokens); i += 2 {
		key := strings.ToLower(strings.TrimSpace(tokens[i]))
		value := strings.TrimSpace(tokens[i+1])
		if value == "" {
			continue
		}
		switch key {
		case "region":
			opts.Region = normalizeSessionToken(value)
		case "st":
			opts.State = normalizeSessionToken(value)
		case "sid":
			opts.SessionID = value
		case "t":
			minutes := parseSessionMinutes(value)
			if minutes > 0 {
				opts.SessionTTL = time.Duration(minutes) * time.Minute
			}
		}
	}

	if opts.Region == "RANDOM" || opts.Region == "GLOBAL" || opts.Region == "ALL" {
		opts.Region = ""
	}
	if opts.State == "RANDOM" || opts.State == "GLOBAL" || opts.State == "ALL" {
		opts.State = ""
	}

	if opts.SessionID != "" {
		if opts.SessionTTL <= 0 {
			opts.SessionTTL = 10 * time.Minute
		}
		opts.SessionKey = buildSessionKey(namespace, opts)
	}

	return opts, nil
}

func parseSessionMinutes(raw string) int {
	var minutes int
	if _, err := fmt.Sscanf(raw, "%d", &minutes); err != nil {
		return 0
	}
	if minutes < 1 {
		return 1
	}
	if minutes > 120 {
		return 120
	}
	return minutes
}

func buildSessionKey(namespace string, opts RequestOptions) string {
	return strings.Join([]string{
		namespace,
		strings.ToUpper(opts.SessionID),
		opts.Region,
		opts.State,
	}, "|")
}

func normalizeSessionToken(raw string) string {
	return strings.ToUpper(strings.TrimSpace(raw))
}

func matchesProxyFilters(p storage.Proxy, protocol string, opts RequestOptions) bool {
	if protocol != "" && p.Protocol != protocol {
		return false
	}
	if opts.Region != "" {
		fields := strings.Fields(strings.ToUpper(p.ExitLocation))
		if len(fields) == 0 || fields[0] != opts.Region {
			return false
		}
	}
	if opts.State != "" {
		location := strings.ToUpper(strings.TrimSpace(p.ExitLocation))
		if location == "" || !strings.Contains(location, opts.State) {
			return false
		}
	}
	return true
}

func selectExistingStickyProxy(
	store *storage.Storage,
	sessions *SessionManager,
	protocol string,
	tried []string,
	opts RequestOptions,
) *storage.Proxy {
	if store == nil || sessions == nil || opts.SessionKey == "" {
		return nil
	}

	stickyAddress, ok := sessions.Get(opts.SessionKey)
	if !ok || stickyAddress == "" {
		return nil
	}

	for _, addr := range tried {
		if addr == stickyAddress {
			return nil
		}
	}

	proxy, err := store.GetByAddress(stickyAddress)
	if err != nil || !matchesProxyFilters(*proxy, protocol, opts) {
		sessions.Delete(opts.SessionKey)
		return nil
	}

	return proxy
}

func selectFromPool(
	store *storage.Storage,
	sessions *SessionManager,
	sourceFilter string,
	namespace string,
	protocol string,
	tried []string,
	lowestLatency bool,
	opts RequestOptions,
) (*storage.Proxy, error) {
	if sessions != nil && opts.SessionKey != "" {
		unlock := sessions.LockKey(opts.SessionKey)
		defer unlock()
	}

	stickyAddress := ""
	if sessions != nil && opts.SessionKey != "" {
		stickyAddress, _ = sessions.Get(opts.SessionKey)
	}

	if stickyAddress != "" && sessions != nil {
		sessions.Delete(opts.SessionKey)
	}

	picked, err := store.SelectProxy(sourceFilter, protocol, opts.Region, opts.State, tried, lowestLatency)
	if err != nil {
		scope := "proxy"
		if protocol != "" {
			scope = protocol + " proxy"
		}
		if sourceFilter != "" {
			scope = sourceFilter + " " + scope
		}
		return nil, fmt.Errorf("no available %s", scope)
	}

	if sessions != nil && opts.SessionKey != "" {
		sessions.Put(opts.SessionKey, picked.Address, opts.SessionTTL)
	}

	return picked, nil
}
