package custom

import (
	"errors"
	"net/http"
	"net/http/httptest"
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

	subID, err := store.AddSubscription("upload", "", "", filePath, "auto", 60)
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

func TestFetchURLWithClientPrefersResponseWithMoreNodes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Header.Get("User-Agent") {
		case "v2rayN":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("trojan://single@example.com:443#single"))
		default:
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("trojan://one@example.com:443#one\nvless://11111111-1111-1111-1111-111111111111@example.com:443?encryption=none#two"))
		}
	}))
	defer server.Close()

	manager := &Manager{}
	client := &http.Client{}
	body, err := manager.fetchURLWithClient(server.URL, client)
	if err != nil {
		t.Fatalf("fetchURLWithClient() error = %v", err)
	}

	nodes, err := Parse(body, "auto")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(nodes) != 2 {
		t.Fatalf("node count = %d, want 2", len(nodes))
	}
}
