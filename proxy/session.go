package proxy

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"proxygate/storage"
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

type sessionKeyLock struct {
	mu   sync.Mutex
	refs int
}

const sessionCleanupInterval = time.Minute

type SessionManager struct {
	mu          sync.RWMutex
	sessions    map[string]sessionEntry
	lastCleanup time.Time
	lockMu      sync.Mutex
	locks       map[string]*sessionKeyLock
}

func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[string]sessionEntry),
		locks:    make(map[string]*sessionKeyLock),
	}
}

func (m *SessionManager) cleanupExpiredSessionsLocked(now time.Time) {
	if !m.lastCleanup.IsZero() && now.Sub(m.lastCleanup) < sessionCleanupInterval {
		return
	}
	for key, entry := range m.sessions {
		if now.After(entry.ExpiresAt) {
			delete(m.sessions, key)
		}
	}
	m.lastCleanup = now
}

func (m *SessionManager) Get(key string) (string, bool) {
	if key == "" {
		return "", false
	}

	now := time.Now()
	m.mu.RLock()
	entry, ok := m.sessions[key]
	m.mu.RUnlock()
	if !ok {
		return "", false
	}
	if now.After(entry.ExpiresAt) {
		m.mu.Lock()
		if current, ok := m.sessions[key]; ok && current == entry && now.After(current.ExpiresAt) {
			delete(m.sessions, key)
		}
		m.mu.Unlock()
		return "", false
	}
	return entry.Address, true
}

func (m *SessionManager) Put(key, address string, ttl time.Duration) {
	if key == "" || address == "" || ttl <= 0 {
		return
	}

	now := time.Now()
	m.mu.Lock()
	m.cleanupExpiredSessionsLocked(now)
	defer m.mu.Unlock()
	m.sessions[key] = sessionEntry{
		Address:   address,
		ExpiresAt: now.Add(ttl),
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

	m.lockMu.Lock()
	lock := m.locks[key]
	if lock == nil {
		lock = &sessionKeyLock{}
		m.locks[key] = lock
	}
	lock.refs++
	m.lockMu.Unlock()

	lock.mu.Lock()
	return func() {
		lock.mu.Unlock()

		m.lockMu.Lock()
		defer m.lockMu.Unlock()
		lock.refs--
		if lock.refs == 0 {
			if current := m.locks[key]; current == lock {
				delete(m.locks, key)
			}
		}
	}
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

	if sessions != nil && opts.SessionKey != "" {
		if stickyAddress, ok := sessions.Get(opts.SessionKey); ok && stickyAddress != "" {
			stickyTried := false
			for _, addr := range tried {
				if addr == stickyAddress {
					stickyTried = true
					break
				}
			}
			if !stickyTried {
				proxy, err := store.GetByAddress(stickyAddress)
				if err == nil && matchesProxyFilters(*proxy, protocol, opts) {
					return proxy, nil
				}
			}
			sessions.Delete(opts.SessionKey)
		}
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
