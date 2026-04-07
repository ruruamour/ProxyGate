package custom

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"proxygate/config"
	"proxygate/storage"
)

func TestDeleteSubscriptionRemovesUploadedFile(t *testing.T) {
	t.Setenv("DATA_DIR", t.TempDir())

	store, err := storage.New(":memory:")
	if err != nil {
		t.Fatalf("storage.New: %v", err)
	}
	defer store.Close()

	filePath := filepath.Join(t.TempDir(), "sub.yaml")
	if err := os.WriteFile(filePath, []byte("proxies: []\n"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	cfg := config.DefaultConfig()
	manager := NewManager(store, nil, cfg)

	subID, err := store.AddSubscription("upload", "", filePath, "auto", 60)
	if err != nil {
		t.Fatalf("AddSubscription: %v", err)
	}
	if err := store.AddProxyWithSource("127.0.0.1:21001", "socks5", "custom", subID); err != nil {
		t.Fatalf("AddProxyWithSource: %v", err)
	}

	if err := manager.DeleteSubscription(subID); err != nil {
		t.Fatalf("DeleteSubscription: %v", err)
	}

	if _, err := store.GetSubscription(subID); err == nil {
		t.Fatal("subscription still exists after delete")
	}

	if _, err := os.Stat(filePath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("uploaded file still exists, stat err = %v", err)
	}

	if proxies, err := store.GetAllFiltered("custom"); err != nil {
		t.Fatalf("GetAllFiltered(custom): %v", err)
	} else if len(proxies) != 0 {
		t.Fatalf("custom proxies still exist after subscription delete: %d", len(proxies))
	}
}
