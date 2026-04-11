package pool

import (
	"path/filepath"
	"testing"

	"proxygate/config"
	"proxygate/storage"
)

func TestNeedsFetchSkipsFreeFetchInCustomOnlyMode(t *testing.T) {
	mgr := &Manager{
		cfg: &config.Config{CustomProxyMode: "custom_only"},
	}

	need, mode, protocol := mgr.NeedsFetch(&PoolStatus{
		HTTP:        0,
		SOCKS5:      0,
		HTTPSlots:   30,
		SOCKS5Slots: 70,
		State:       "emergency",
	})
	if need {
		t.Fatalf("NeedsFetch() = true, want false in custom_only mode (mode=%q protocol=%q)", mode, protocol)
	}
}

func newTestManager(t *testing.T, cfg *config.Config) (*Manager, *storage.Storage) {
	t.Helper()

	store, err := storage.New(filepath.Join(t.TempDir(), "proxy.db"))
	if err != nil {
		t.Fatalf("storage.New() error = %v", err)
	}
	t.Cleanup(func() {
		_ = store.Close()
	})

	return NewManager(store, cfg), store
}

func TestGetStatusUsesFreePoolScopeInMixedMode(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.CustomProxyMode = "mixed"
	cfg.PoolMaxSize = 10
	cfg.PoolHTTPRatio = 0.3

	mgr, store := newTestManager(t, cfg)

	if err := store.AddProxyWithSource("198.51.100.10:8080", "http", "free"); err != nil {
		t.Fatalf("AddProxyWithSource(free http) error = %v", err)
	}
	if err := store.UpdateExitInfo("198.51.100.10:8080", "203.0.113.10", "US Seattle", 100); err != nil {
		t.Fatalf("UpdateExitInfo(free http) error = %v", err)
	}
	if err := store.AddProxyWithSource("198.51.100.11:1080", "socks5", "free"); err != nil {
		t.Fatalf("AddProxyWithSource(free socks5) error = %v", err)
	}
	if err := store.UpdateExitInfo("198.51.100.11:1080", "203.0.113.11", "JP Tokyo", 200); err != nil {
		t.Fatalf("UpdateExitInfo(free socks5) error = %v", err)
	}

	subID, err := store.AddSubscription("mixed-sub", "", "https://example.com/sub", "", "auto", 60)
	if err != nil {
		t.Fatalf("AddSubscription() error = %v", err)
	}
	if err := store.AddProxyWithSource("198.51.100.20:8080", "http", "custom", subID); err != nil {
		t.Fatalf("AddProxyWithSource(custom http) error = %v", err)
	}
	if err := store.EnableProxy("198.51.100.20:8080"); err != nil {
		t.Fatalf("EnableProxy(custom http) error = %v", err)
	}
	if err := store.UpdateExitInfo("198.51.100.20:8080", "203.0.113.20", "US San Jose", 50); err != nil {
		t.Fatalf("UpdateExitInfo(custom http) error = %v", err)
	}
	if err := store.AddProxyWithSource("198.51.100.21:1080", "socks5", "custom", subID); err != nil {
		t.Fatalf("AddProxyWithSource(custom socks5) error = %v", err)
	}
	if err := store.EnableProxy("198.51.100.21:1080"); err != nil {
		t.Fatalf("EnableProxy(custom socks5) error = %v", err)
	}
	if err := store.UpdateExitInfo("198.51.100.21:1080", "203.0.113.21", "GB London", 75); err != nil {
		t.Fatalf("UpdateExitInfo(custom socks5) error = %v", err)
	}

	status, err := mgr.GetStatus()
	if err != nil {
		t.Fatalf("GetStatus() error = %v", err)
	}

	if status.Total != 2 {
		t.Fatalf("status.Total = %d, want 2 free proxies", status.Total)
	}
	if status.HTTP != 1 {
		t.Fatalf("status.HTTP = %d, want 1 free HTTP proxy", status.HTTP)
	}
	if status.SOCKS5 != 1 {
		t.Fatalf("status.SOCKS5 = %d, want 1 free SOCKS5 proxy", status.SOCKS5)
	}
	if status.CustomCount != 2 {
		t.Fatalf("status.CustomCount = %d, want 2 custom proxies", status.CustomCount)
	}
	if status.AvgLatencyHTTP != 100 {
		t.Fatalf("status.AvgLatencyHTTP = %d, want 100 from free pool only", status.AvgLatencyHTTP)
	}
	if status.AvgLatencySocks5 != 200 {
		t.Fatalf("status.AvgLatencySocks5 = %d, want 200 from free pool only", status.AvgLatencySocks5)
	}
	if status.State != "critical" {
		t.Fatalf("status.State = %q, want %q when free pool is short on both protocols", status.State, "critical")
	}
}
