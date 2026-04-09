package proxy

import (
	"testing"
	"time"

	"proxygate/config"
	"proxygate/storage"
)

func TestParseUsernameOptions(t *testing.T) {
	opts, err := parseUsernameOptions("proxy", "proxy-region-us-st-losangeles-sid-AbC123xy-t-5", "http-random")
	if err != nil {
		t.Fatalf("parseUsernameOptions: %v", err)
	}
	if opts.Region != "US" {
		t.Fatalf("Region = %q", opts.Region)
	}
	if opts.State != "LOSANGELES" {
		t.Fatalf("State = %q", opts.State)
	}
	if opts.SessionID != "AbC123xy" {
		t.Fatalf("SessionID = %q", opts.SessionID)
	}
	if opts.SessionTTL != 5*time.Minute {
		t.Fatalf("SessionTTL = %v", opts.SessionTTL)
	}
	if opts.SessionKey == "" {
		t.Fatalf("SessionKey should not be empty")
	}
}

func TestSelectFromPoolStickySession(t *testing.T) {
	store, err := storage.New(":memory:")
	if err != nil {
		t.Fatalf("storage.New: %v", err)
	}
	defer store.Close()

	proxies := []storage.Proxy{
		{Address: "127.0.0.1:20001", Protocol: "socks5", ExitLocation: "US Los Angeles", Latency: 120, Source: "custom"},
		{Address: "127.0.0.1:20002", Protocol: "socks5", ExitLocation: "US Seattle", Latency: 140, Source: "custom"},
	}
	for _, p := range proxies {
		if err := store.AddProxyWithSource(p.Address, p.Protocol, p.Source); err != nil {
			t.Fatalf("AddProxyWithSource: %v", err)
		}
		if err := store.EnableProxy(p.Address); err != nil {
			t.Fatalf("EnableProxy: %v", err)
		}
		if err := store.UpdateExitInfo(p.Address, p.Address, p.ExitLocation, p.Latency); err != nil {
			t.Fatalf("UpdateExitInfo: %v", err)
		}
	}

	sessions := NewSessionManager()
	opts := RequestOptions{
		Region:     "US",
		SessionID:  "ABC12345",
		SessionTTL: 5 * time.Minute,
		SessionKey: buildSessionKey("socks5-random", RequestOptions{Region: "US", SessionID: "ABC12345"}),
	}

	first, err := selectFromPool(store, sessions, "custom", "socks5-random", "socks5", nil, false, opts)
	if err != nil {
		t.Fatalf("first select: %v", err)
	}
	second, err := selectFromPool(store, sessions, "custom", "socks5-random", "socks5", nil, false, opts)
	if err != nil {
		t.Fatalf("second select: %v", err)
	}
	if first.Address != second.Address {
		t.Fatalf("sticky session mismatch: %s != %s", first.Address, second.Address)
	}
}

func TestSelectProxyKeepsStickyAcrossSourcePriority(t *testing.T) {
	store, err := storage.New(":memory:")
	if err != nil {
		t.Fatalf("storage.New: %v", err)
	}
	defer store.Close()

	proxies := []storage.Proxy{
		{Address: "198.51.100.10:8080", Protocol: "http", ExitLocation: "US Chicago", Latency: 200, Source: "free"},
		{Address: "127.0.0.1:20001", Protocol: "socks5", ExitLocation: "US Los Angeles", Latency: 50, Source: "custom"},
	}
	for _, p := range proxies {
		if err := store.AddProxyWithSource(p.Address, p.Protocol, p.Source); err != nil {
			t.Fatalf("AddProxyWithSource: %v", err)
		}
		if err := store.EnableProxy(p.Address); err != nil {
			t.Fatalf("EnableProxy: %v", err)
		}
		if err := store.UpdateExitInfo(p.Address, p.Address, p.ExitLocation, p.Latency); err != nil {
			t.Fatalf("UpdateExitInfo: %v", err)
		}
	}

	cfg := config.DefaultConfig()
	cfg.CustomProxyMode = "mixed"
	cfg.CustomPriority = true
	cfg.CustomFreePriority = false

	sessions := NewSessionManager()
	opts := RequestOptions{
		Region:     "US",
		SessionID:  "ABCD1234",
		SessionTTL: 5 * time.Minute,
		SessionKey: buildSessionKey("http-random", RequestOptions{Region: "US", SessionID: "ABCD1234"}),
	}
	sessions.Put(opts.SessionKey, proxies[0].Address, opts.SessionTTL)

	server := &Server{
		storage:          store,
		cfg:              cfg,
		sessions:         sessions,
		sessionNamespace: "http-random",
		mode:             "random",
	}

	selected, err := server.selectProxy(nil, false, opts)
	if err != nil {
		t.Fatalf("selectProxy: %v", err)
	}
	if selected.Address != proxies[0].Address {
		t.Fatalf("expected sticky proxy %s, got %s", proxies[0].Address, selected.Address)
	}
}

func TestSelectProxyUsesUnifiedPoolInMixedMode(t *testing.T) {
	store, err := storage.New(":memory:")
	if err != nil {
		t.Fatalf("storage.New: %v", err)
	}
	defer store.Close()

	proxies := []storage.Proxy{
		{Address: "198.51.100.10:8080", Protocol: "http", ExitLocation: "US Chicago", Latency: 20, Source: "free"},
		{Address: "127.0.0.1:20001", Protocol: "socks5", ExitLocation: "US Los Angeles", Latency: 80, Source: "custom"},
	}
	for _, p := range proxies {
		if err := store.AddProxyWithSource(p.Address, p.Protocol, p.Source); err != nil {
			t.Fatalf("AddProxyWithSource: %v", err)
		}
		if err := store.EnableProxy(p.Address); err != nil {
			t.Fatalf("EnableProxy: %v", err)
		}
		if err := store.UpdateExitInfo(p.Address, p.Address, p.ExitLocation, p.Latency); err != nil {
			t.Fatalf("UpdateExitInfo: %v", err)
		}
	}

	cfg := config.DefaultConfig()
	cfg.CustomProxyMode = "mixed"
	cfg.CustomPriority = true
	cfg.CustomFreePriority = false

	server := &Server{
		storage:          store,
		cfg:              cfg,
		sessions:         NewSessionManager(),
		sessionNamespace: "http-random",
		mode:             "random",
	}

	selected, err := server.selectProxy(nil, true, RequestOptions{Region: "US"})
	if err != nil {
		t.Fatalf("selectProxy: %v", err)
	}
	if selected.Address != proxies[0].Address {
		t.Fatalf("expected lowest-latency proxy %s from unified pool, got %s", proxies[0].Address, selected.Address)
	}
}

func TestSOCKS5SelectProxyUsesUnifiedPoolInMixedMode(t *testing.T) {
	store, err := storage.New(":memory:")
	if err != nil {
		t.Fatalf("storage.New: %v", err)
	}
	defer store.Close()

	proxies := []storage.Proxy{
		{Address: "198.51.100.10:8080", Protocol: "http", ExitLocation: "US Chicago", Latency: 20, Source: "free"},
		{Address: "127.0.0.1:20001", Protocol: "socks5", ExitLocation: "US Los Angeles", Latency: 80, Source: "custom"},
	}
	for _, p := range proxies {
		if err := store.AddProxyWithSource(p.Address, p.Protocol, p.Source); err != nil {
			t.Fatalf("AddProxyWithSource: %v", err)
		}
		if err := store.EnableProxy(p.Address); err != nil {
			t.Fatalf("EnableProxy: %v", err)
		}
		if err := store.UpdateExitInfo(p.Address, p.Address, p.ExitLocation, p.Latency); err != nil {
			t.Fatalf("UpdateExitInfo: %v", err)
		}
	}

	cfg := config.DefaultConfig()
	cfg.CustomProxyMode = "mixed"
	cfg.CustomPriority = true
	cfg.CustomFreePriority = false

	server := &SOCKS5Server{
		storage:          store,
		cfg:              cfg,
		sessions:         NewSessionManager(),
		sessionNamespace: "socks5-random",
		mode:             "lowest-latency",
	}

	selected, err := server.selectSOCKS5Proxy(nil, RequestOptions{Region: "US"})
	if err != nil {
		t.Fatalf("selectSOCKS5Proxy: %v", err)
	}
	if selected.Address != proxies[0].Address {
		t.Fatalf("expected lowest-latency proxy %s from unified pool, got %s", proxies[0].Address, selected.Address)
	}
}

func TestSessionManagerReleasesPerKeyLocks(t *testing.T) {
	sessions := NewSessionManager()

	unlock := sessions.LockKey("sticky-key")

	sessions.lockMu.Lock()
	if len(sessions.locks) != 1 {
		sessions.lockMu.Unlock()
		t.Fatalf("len(locks) = %d, want 1", len(sessions.locks))
	}
	sessions.lockMu.Unlock()

	unlock()

	sessions.lockMu.Lock()
	defer sessions.lockMu.Unlock()
	if len(sessions.locks) != 0 {
		t.Fatalf("len(locks) = %d, want 0", len(sessions.locks))
	}
}

func TestSessionManagerPutCleansExpiredEntries(t *testing.T) {
	sessions := NewSessionManager()
	now := time.Now()
	sessions.sessions["expired"] = sessionEntry{
		Address:   "127.0.0.1:10001",
		ExpiresAt: now.Add(-time.Minute),
	}
	sessions.lastCleanup = now.Add(-2 * sessionCleanupInterval)

	sessions.Put("fresh", "127.0.0.1:10002", 5*time.Minute)

	sessions.mu.RLock()
	defer sessions.mu.RUnlock()
	if _, ok := sessions.sessions["expired"]; ok {
		t.Fatal("expired session still present after Put cleanup")
	}
	if _, ok := sessions.sessions["fresh"]; !ok {
		t.Fatal("fresh session missing after Put")
	}
}
