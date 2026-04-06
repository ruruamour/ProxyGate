package webui

import (
	"net/http/httptest"
	"testing"
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
