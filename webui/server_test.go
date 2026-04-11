package webui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"proxygate/config"
	"proxygate/pool"
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

func newTestStorage(t *testing.T) *storage.Storage {
	t.Helper()

	store, err := storage.New(filepath.Join(t.TempDir(), "proxy.db"))
	if err != nil {
		t.Fatalf("storage.New() error = %v", err)
	}
	t.Cleanup(func() {
		_ = store.Close()
	})

	return store
}

func newTestWebUIServer(t *testing.T) *Server {
	t.Helper()

	store := newTestStorage(t)
	return New(store, config.DefaultConfig(), nil, nil, nil, nil)
}

func newTestWebUIServerWithPool(t *testing.T, cfg *config.Config) *Server {
	t.Helper()

	store := newTestStorage(t)
	return New(store, cfg, pool.NewManager(store, cfg), nil, nil, nil)
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

func TestAPIProxiesMasksSensitiveFieldsForGuests(t *testing.T) {
	srv := newTestWebUIServer(t)

	if err := srv.storage.AddProxyWithSource("203.0.113.10:8080", "http", "free"); err != nil {
		t.Fatalf("AddProxyWithSource(free) error = %v", err)
	}
	if err := srv.storage.UpdateExitInfo("203.0.113.10:8080", "198.51.100.10", "US Seattle", 120); err != nil {
		t.Fatalf("UpdateExitInfo(free) error = %v", err)
	}

	subID, err := srv.storage.AddSubscription("Private Sub", "", "https://example.com/sub?token=secret", "", "auto", 60)
	if err != nil {
		t.Fatalf("AddSubscription() error = %v", err)
	}
	if err := srv.storage.AddProxyWithSource("127.0.0.1:20001", "socks5", "custom", subID); err != nil {
		t.Fatalf("AddProxyWithSource(custom) error = %v", err)
	}
	if err := srv.storage.EnableProxy("127.0.0.1:20001"); err != nil {
		t.Fatalf("EnableProxy(custom) error = %v", err)
	}
	if err := srv.storage.UpdateExitInfo("127.0.0.1:20001", "203.0.113.77", "JP Tokyo", 90); err != nil {
		t.Fatalf("UpdateExitInfo(custom) error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "https://proxy.example.com/api/proxies", nil)
	w := httptest.NewRecorder()
	srv.apiProxies(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("apiProxies() status = %d, want %d", w.Code, http.StatusOK)
	}

	var proxies []storage.Proxy
	if err := json.Unmarshal(w.Body.Bytes(), &proxies); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(proxies) != 2 {
		t.Fatalf("len(proxies) = %d, want 2", len(proxies))
	}
	for _, p := range proxies {
		if p.ExitIP != "" {
			t.Fatalf("guest proxy exit_ip = %q, want masked empty", p.ExitIP)
		}
		if p.Address == "203.0.113.10:8080" || p.Address == "127.0.0.1:20001" {
			t.Fatalf("guest proxy address leaked: %q", p.Address)
		}
		if p.Address == "" {
			t.Fatal("guest proxy address = empty, want masked identifier")
		}
	}
}

func TestAPISubscriptionsHidesURLAndFilePathForGuests(t *testing.T) {
	srv := newTestWebUIServer(t)

	subID, err := srv.storage.AddSubscription("Secret URL", "", "https://example.com/sub?token=topsecret", "", "auto", 60)
	if err != nil {
		t.Fatalf("AddSubscription(url) error = %v", err)
	}
	if _, err := srv.storage.AddSubscription("Secret File", "", "", "/tmp/private-sub.yaml", "auto", 60); err != nil {
		t.Fatalf("AddSubscription(file) error = %v", err)
	}
	if err := srv.storage.AddProxyWithSource("127.0.0.1:20001", "socks5", "custom", subID); err != nil {
		t.Fatalf("AddProxyWithSource(custom) error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "https://proxy.example.com/api/subscriptions", nil)
	w := httptest.NewRecorder()
	srv.apiSubscriptions(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("apiSubscriptions() status = %d, want %d", w.Code, http.StatusOK)
	}

	var payload []map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(payload) != 2 {
		t.Fatalf("len(payload) = %d, want 2", len(payload))
	}
	for _, item := range payload {
		if got, _ := item["url"].(string); got != "" {
			t.Fatalf("guest subscription url = %q, want empty", got)
		}
		if got, _ := item["file_path"].(string); got != "" {
			t.Fatalf("guest subscription file_path = %q, want empty", got)
		}
	}
}

func TestAPIStatsAndPoolStatusUseFreePoolScopeInMixedMode(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.CustomProxyMode = "mixed"
	cfg.PoolMaxSize = 10
	cfg.PoolHTTPRatio = 0.3

	srv := newTestWebUIServerWithPool(t, cfg)

	if err := srv.storage.AddProxyWithSource("198.51.100.10:8080", "http", "free"); err != nil {
		t.Fatalf("AddProxyWithSource(free http) error = %v", err)
	}
	if err := srv.storage.UpdateExitInfo("198.51.100.10:8080", "203.0.113.10", "US Seattle", 100); err != nil {
		t.Fatalf("UpdateExitInfo(free http) error = %v", err)
	}
	if err := srv.storage.AddProxyWithSource("198.51.100.11:1080", "socks5", "free"); err != nil {
		t.Fatalf("AddProxyWithSource(free socks5) error = %v", err)
	}
	if err := srv.storage.UpdateExitInfo("198.51.100.11:1080", "203.0.113.11", "JP Tokyo", 200); err != nil {
		t.Fatalf("UpdateExitInfo(free socks5) error = %v", err)
	}

	subID, err := srv.storage.AddSubscription("mixed-sub", "", "https://example.com/sub", "", "auto", 60)
	if err != nil {
		t.Fatalf("AddSubscription() error = %v", err)
	}
	if err := srv.storage.AddProxyWithSource("198.51.100.20:8080", "http", "custom", subID); err != nil {
		t.Fatalf("AddProxyWithSource(custom http) error = %v", err)
	}
	if err := srv.storage.EnableProxy("198.51.100.20:8080"); err != nil {
		t.Fatalf("EnableProxy(custom http) error = %v", err)
	}
	if err := srv.storage.UpdateExitInfo("198.51.100.20:8080", "203.0.113.20", "US San Jose", 50); err != nil {
		t.Fatalf("UpdateExitInfo(custom http) error = %v", err)
	}
	if err := srv.storage.AddProxyWithSource("198.51.100.21:1080", "socks5", "custom", subID); err != nil {
		t.Fatalf("AddProxyWithSource(custom socks5) error = %v", err)
	}
	if err := srv.storage.EnableProxy("198.51.100.21:1080"); err != nil {
		t.Fatalf("EnableProxy(custom socks5) error = %v", err)
	}
	if err := srv.storage.UpdateExitInfo("198.51.100.21:1080", "203.0.113.21", "GB London", 75); err != nil {
		t.Fatalf("UpdateExitInfo(custom socks5) error = %v", err)
	}

	statsReq := httptest.NewRequest(http.MethodGet, "https://proxy.example.com/api/stats", nil)
	statsRec := httptest.NewRecorder()
	srv.apiStats(statsRec, statsReq)
	if statsRec.Code != http.StatusOK {
		t.Fatalf("apiStats() status = %d, want %d", statsRec.Code, http.StatusOK)
	}

	var statsResp struct {
		Total       int    `json:"total"`
		HTTP        int    `json:"http"`
		SOCKS5      int    `json:"socks5"`
		CustomCount int    `json:"custom_count"`
		Port        string `json:"port"`
	}
	if err := json.Unmarshal(statsRec.Body.Bytes(), &statsResp); err != nil {
		t.Fatalf("json.Unmarshal(apiStats) error = %v", err)
	}
	if statsResp.Total != 2 || statsResp.HTTP != 1 || statsResp.SOCKS5 != 1 || statsResp.CustomCount != 2 {
		t.Fatalf("apiStats() = %+v, want free total/http/socks5 2/1/1 and custom_count 2", statsResp)
	}

	statusReq := httptest.NewRequest(http.MethodGet, "https://proxy.example.com/api/pool/status", nil)
	statusRec := httptest.NewRecorder()
	srv.apiPoolStatus(statusRec, statusReq)
	if statusRec.Code != http.StatusOK {
		t.Fatalf("apiPoolStatus() status = %d, want %d", statusRec.Code, http.StatusOK)
	}

	var statusResp pool.PoolStatus
	if err := json.Unmarshal(statusRec.Body.Bytes(), &statusResp); err != nil {
		t.Fatalf("json.Unmarshal(apiPoolStatus) error = %v", err)
	}
	if statusResp.Total != 2 || statusResp.HTTP != 1 || statusResp.SOCKS5 != 1 || statusResp.CustomCount != 2 {
		t.Fatalf("apiPoolStatus() = %+v, want free total/http/socks5 2/1/1 and custom_count 2", statusResp)
	}
	if statusResp.AvgLatencyHTTP != 100 || statusResp.AvgLatencySocks5 != 200 {
		t.Fatalf("apiPoolStatus() avg latency = %d/%d, want 100/200 from free pool only", statusResp.AvgLatencyHTTP, statusResp.AvgLatencySocks5)
	}
}

func TestDashboardPoolStatusBindingUsesExplicitFreePoolValues(t *testing.T) {
	if strings.Contains(dashboardHTML, "status.Total - (status.CustomCount || 0)") {
		t.Fatal("dashboardHTML still subtracts CustomCount from Total")
	}
	if !strings.Contains(dashboardHTML, "document.getElementById('stat-total').textContent = status.Total;") {
		t.Fatal("dashboardHTML does not bind stat-total directly to status.Total")
	}
	if !strings.Contains(dashboardHTML, "data-i18n=\"health.free_proxies\"") {
		t.Fatal("dashboardHTML does not label the total card as free proxies")
	}
	if !strings.Contains(dashboardHTML, "'proxy.th_usage': 'Historical Usage'") {
		t.Fatal("dashboardHTML does not clarify the usage column as historical usage")
	}
}

func TestDashboardSubscriptionStatusDoesNotTreatStoppedSingBoxAsReady(t *testing.T) {
	if strings.Contains(dashboardHTML, "subCount > 0 ? t('health.ready') : t('health.not_added')") {
		t.Fatal("dashboardHTML still treats a stopped sing-box with subscriptions as ready")
	}
	if !strings.Contains(dashboardHTML, "subCount > 0 ? t('health.singbox_stopped') : t('health.not_added')") {
		t.Fatal("dashboardHTML does not surface stopped sing-box state when subscriptions exist")
	}
}
