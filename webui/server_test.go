package webui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"proxygate/config"
	"proxygate/storage"
)

func TestSameOriginRequestAcceptsMatchingOrigin(t *testing.T) {
	req := httptest.NewRequest("POST", "https://proxy.example.com/api/fetch", nil)
	req.Header.Set("Origin", "https://proxy.example.com")

	if !sameOriginRequest(req) {
		t.Fatal("sameOriginRequest() = false, want true")
	}
}

func TestSameOriginRequestRejectsCrossOrigin(t *testing.T) {
	req := httptest.NewRequest("POST", "https://proxy.example.com/api/fetch", nil)
	req.Header.Set("Origin", "https://evil.example.com")

	if sameOriginRequest(req) {
		t.Fatal("sameOriginRequest() = true, want false")
	}
}

func TestSameOriginRequestRejectsMissingBrowserOrigin(t *testing.T) {
	req := httptest.NewRequest("POST", "https://proxy.example.com/api/fetch", nil)

	if sameOriginRequest(req) {
		t.Fatal("sameOriginRequest() = true, want false")
	}
}

func resetContributionState() {
	contributionWindowsMu.Lock()
	contributionWindows = make(map[string]contributionRateWindow)
	contributionWindowsMu.Unlock()
}

func newTestWebUIServer(t *testing.T) *Server {
	t.Helper()

	store, err := storage.New(filepath.Join(t.TempDir(), "proxy.db"))
	if err != nil {
		t.Fatalf("storage.New() error = %v", err)
	}
	t.Cleanup(func() {
		_ = store.Close()
	})

	return New(store, config.DefaultConfig(), nil, nil, nil, nil)
}

func TestAPIContributionRejectsPrivateURL(t *testing.T) {
	resetContributionState()
	srv := newTestWebUIServer(t)

	body, err := json.Marshal(map[string]string{
		"name": "private",
		"url":  "http://127.0.0.1/sub",
	})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "https://proxy.example.com/api/subscription/contribute", bytes.NewReader(body))
	req.Header.Set("Origin", "https://proxy.example.com")
	w := httptest.NewRecorder()

	srv.apiSubscriptionContribute(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("apiSubscriptionContribute() status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestAPIContributionRateLimitPerIP(t *testing.T) {
	resetContributionState()
	srv := newTestWebUIServer(t)

	makeRequest := func(name string) *httptest.ResponseRecorder {
		body, err := json.Marshal(map[string]string{
			"name": name,
			"url":  "https://example.com/" + name,
		})
		if err != nil {
			t.Fatalf("json.Marshal() error = %v", err)
		}
		req := httptest.NewRequest(http.MethodPost, "https://proxy.example.com/api/subscription/contribute", bytes.NewReader(body))
		req.Header.Set("Origin", "https://proxy.example.com")
		req.RemoteAddr = "198.51.100.10:12345"
		w := httptest.NewRecorder()
		srv.apiSubscriptionContribute(w, req)
		return w
	}

	for i := 0; i < contributionRateLimit; i++ {
		w := makeRequest(fmt.Sprintf("sub-%d", i))
		if w.Code != http.StatusOK {
			t.Fatalf("request %d status = %d, want %d", i+1, w.Code, http.StatusOK)
		}
	}

	w := makeRequest("sub-over-limit")
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("rate-limited request status = %d, want %d", w.Code, http.StatusTooManyRequests)
	}
	if got := w.Header().Get("Retry-After"); got == "" {
		t.Fatal("Retry-After header = empty, want value")
	}
}
