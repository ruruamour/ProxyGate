package storage

import (
	"path/filepath"
	"testing"
)

func TestGetBatchForHealthCheckFiltersBySource(t *testing.T) {
	store, err := New(filepath.Join(t.TempDir(), "proxy.db"))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer store.Close()

	if err := store.AddProxyWithSource("1.1.1.1:80", "http", "free"); err != nil {
		t.Fatalf("AddProxyWithSource(free) error = %v", err)
	}
	if err := store.AddProxyWithSource("127.0.0.1:20000", "socks5", "custom", 1); err != nil {
		t.Fatalf("AddProxyWithSource(custom) error = %v", err)
	}

	proxies, err := store.GetBatchForHealthCheck(10, false, "free")
	if err != nil {
		t.Fatalf("GetBatchForHealthCheck() error = %v", err)
	}
	if len(proxies) != 1 {
		t.Fatalf("len(GetBatchForHealthCheck()) = %d, want 1", len(proxies))
	}
	if proxies[0].Source != "free" {
		t.Fatalf("GetBatchForHealthCheck() source = %q, want %q", proxies[0].Source, "free")
	}
}
