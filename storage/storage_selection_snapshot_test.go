package storage

import (
	"path/filepath"
	"testing"
)

func newTestStorage(t *testing.T) *Storage {
	t.Helper()

	store, err := New(filepath.Join(t.TempDir(), "proxy.db"))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(func() {
		_ = store.Close()
	})
	return store
}

func TestSelectProxyUsesSnapshotFilters(t *testing.T) {
	store := newTestStorage(t)

	if err := store.AddProxyWithSource("127.0.0.1:20001", "socks5", "custom", 1); err != nil {
		t.Fatalf("AddProxyWithSource(custom1) error = %v", err)
	}
	if err := store.UpdateExitInfo("127.0.0.1:20001", "1.1.1.1", "JP Tokyo", 80); err != nil {
		t.Fatalf("UpdateExitInfo(custom1) error = %v", err)
	}
	if err := store.EnableProxy("127.0.0.1:20001"); err != nil {
		t.Fatalf("EnableProxy(custom1) error = %v", err)
	}

	if err := store.AddProxyWithSource("127.0.0.1:20002", "socks5", "custom", 1); err != nil {
		t.Fatalf("AddProxyWithSource(custom2) error = %v", err)
	}
	if err := store.UpdateExitInfo("127.0.0.1:20002", "2.2.2.2", "JP Osaka", 140); err != nil {
		t.Fatalf("UpdateExitInfo(custom2) error = %v", err)
	}
	if err := store.EnableProxy("127.0.0.1:20002"); err != nil {
		t.Fatalf("EnableProxy(custom2) error = %v", err)
	}

	if err := store.AddProxy("3.3.3.3:80", "http"); err != nil {
		t.Fatalf("AddProxy(free) error = %v", err)
	}
	if err := store.UpdateExitInfo("3.3.3.3:80", "3.3.3.3", "US Seattle", 60); err != nil {
		t.Fatalf("UpdateExitInfo(free) error = %v", err)
	}

	selected, err := store.SelectProxy("custom", "socks5", "JP", "TOKYO", nil, true)
	if err != nil {
		t.Fatalf("SelectProxy(custom, socks5, JP, TOKYO) error = %v", err)
	}
	if selected.Address != "127.0.0.1:20001" {
		t.Fatalf("SelectProxy(custom, socks5, JP, TOKYO) = %s, want %s", selected.Address, "127.0.0.1:20001")
	}

	selected, err = store.SelectProxy("custom", "socks5", "JP", "", []string{"127.0.0.1:20001"}, true)
	if err != nil {
		t.Fatalf("SelectProxy(custom, socks5, JP, exclude) error = %v", err)
	}
	if selected.Address != "127.0.0.1:20002" {
		t.Fatalf("SelectProxy(custom, socks5, JP, exclude) = %s, want %s", selected.Address, "127.0.0.1:20002")
	}

	selected, err = store.SelectProxy("free", "http", "US", "", nil, true)
	if err != nil {
		t.Fatalf("SelectProxy(free, http, US) error = %v", err)
	}
	if selected.Address != "3.3.3.3:80" {
		t.Fatalf("SelectProxy(free, http, US) = %s, want %s", selected.Address, "3.3.3.3:80")
	}
}

func TestGetByAddressSnapshotInvalidatesOnDisable(t *testing.T) {
	store := newTestStorage(t)

	if err := store.AddProxy("4.4.4.4:80", "http"); err != nil {
		t.Fatalf("AddProxy() error = %v", err)
	}
	if err := store.UpdateExitInfo("4.4.4.4:80", "4.4.4.4", "US", 90); err != nil {
		t.Fatalf("UpdateExitInfo() error = %v", err)
	}

	if _, err := store.GetByAddress("4.4.4.4:80"); err != nil {
		t.Fatalf("GetByAddress(active) error = %v", err)
	}

	if err := store.DisableProxy("4.4.4.4:80"); err != nil {
		t.Fatalf("DisableProxy() error = %v", err)
	}
	if _, err := store.GetByAddress("4.4.4.4:80"); err == nil {
		t.Fatalf("GetByAddress(disabled) error = nil, want not found")
	}

	if err := store.EnableProxy("4.4.4.4:80"); err != nil {
		t.Fatalf("EnableProxy() error = %v", err)
	}
	if _, err := store.GetByAddress("4.4.4.4:80"); err != nil {
		t.Fatalf("GetByAddress(re-enabled) error = %v", err)
	}
}

func TestSelectProxySnapshotRefreshesAfterLatencyUpdate(t *testing.T) {
	store := newTestStorage(t)

	if err := store.AddProxy("5.5.5.5:80", "http"); err != nil {
		t.Fatalf("AddProxy(first) error = %v", err)
	}
	if err := store.UpdateExitInfo("5.5.5.5:80", "5.5.5.5", "SG", 200); err != nil {
		t.Fatalf("UpdateExitInfo(first) error = %v", err)
	}

	if err := store.AddProxy("6.6.6.6:80", "http"); err != nil {
		t.Fatalf("AddProxy(second) error = %v", err)
	}
	if err := store.UpdateExitInfo("6.6.6.6:80", "6.6.6.6", "SG", 100); err != nil {
		t.Fatalf("UpdateExitInfo(second) error = %v", err)
	}

	selected, err := store.SelectProxy("free", "http", "SG", "", nil, true)
	if err != nil {
		t.Fatalf("SelectProxy(before UpdateLatency) error = %v", err)
	}
	if selected.Address != "6.6.6.6:80" {
		t.Fatalf("SelectProxy(before UpdateLatency) = %s, want %s", selected.Address, "6.6.6.6:80")
	}

	if err := store.UpdateLatency("5.5.5.5:80", 50); err != nil {
		t.Fatalf("UpdateLatency() error = %v", err)
	}

	selected, err = store.SelectProxy("free", "http", "SG", "", nil, true)
	if err != nil {
		t.Fatalf("SelectProxy(after UpdateLatency) error = %v", err)
	}
	if selected.Address != "5.5.5.5:80" {
		t.Fatalf("SelectProxy(after UpdateLatency) = %s, want %s", selected.Address, "5.5.5.5:80")
	}
}

func TestRecordProxyUseSuccessFlushesBatched(t *testing.T) {
	store := newTestStorage(t)

	if err := store.AddProxy("7.7.7.7:80", "http"); err != nil {
		t.Fatalf("AddProxy() error = %v", err)
	}
	if err := store.UpdateExitInfo("7.7.7.7:80", "7.7.7.7", "US", 70); err != nil {
		t.Fatalf("UpdateExitInfo() error = %v", err)
	}

	for i := 0; i < 3; i++ {
		if err := store.RecordProxyUse("7.7.7.7:80", true); err != nil {
			t.Fatalf("RecordProxyUse(success) error = %v", err)
		}
	}
	if err := store.flushPendingSuccessUse(); err != nil {
		t.Fatalf("flushPendingSuccessUse() error = %v", err)
	}

	var useCount, successCount, failCount int
	if err := store.db.QueryRow(
		`SELECT use_count, success_count, fail_count FROM proxies WHERE address = ?`,
		"7.7.7.7:80",
	).Scan(&useCount, &successCount, &failCount); err != nil {
		t.Fatalf("QueryRow() error = %v", err)
	}

	if useCount != 3 || successCount != 3 || failCount != 0 {
		t.Fatalf("counts = (%d, %d, %d), want (3, 3, 0)", useCount, successCount, failCount)
	}
}

func TestRecordProxyUseFailureStillWritesSynchronously(t *testing.T) {
	store := newTestStorage(t)

	if err := store.AddProxy("8.8.8.8:80", "http"); err != nil {
		t.Fatalf("AddProxy() error = %v", err)
	}
	if err := store.UpdateExitInfo("8.8.8.8:80", "8.8.8.8", "US", 90); err != nil {
		t.Fatalf("UpdateExitInfo() error = %v", err)
	}

	if err := store.RecordProxyUse("8.8.8.8:80", false); err != nil {
		t.Fatalf("RecordProxyUse(failure) error = %v", err)
	}

	var useCount, successCount, failCount int
	if err := store.db.QueryRow(
		`SELECT use_count, success_count, fail_count FROM proxies WHERE address = ?`,
		"8.8.8.8:80",
	).Scan(&useCount, &successCount, &failCount); err != nil {
		t.Fatalf("QueryRow() error = %v", err)
	}

	if useCount != 1 || successCount != 0 || failCount != 1 {
		t.Fatalf("counts = (%d, %d, %d), want (1, 0, 1)", useCount, successCount, failCount)
	}
}
