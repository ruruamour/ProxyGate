package validator

import (
	"net/http"
	"testing"
	"time"
)

func TestNewProbeTransportUsesShortLivedConnections(t *testing.T) {
	timeout := 7 * time.Second
	transport := newProbeTransport(timeout)

	if !transport.DisableKeepAlives {
		t.Fatal("DisableKeepAlives = false, want true")
	}
	if transport.ForceAttemptHTTP2 {
		t.Fatal("ForceAttemptHTTP2 = true, want false")
	}
	if transport.MaxConnsPerHost != 1 {
		t.Fatalf("MaxConnsPerHost = %d, want 1", transport.MaxConnsPerHost)
	}
	if transport.TLSHandshakeTimeout != timeout {
		t.Fatalf("TLSHandshakeTimeout = %v, want %v", transport.TLSHandshakeTimeout, timeout)
	}
	if transport.ResponseHeaderTimeout != timeout {
		t.Fatalf("ResponseHeaderTimeout = %v, want %v", transport.ResponseHeaderTimeout, timeout)
	}
}

func TestNewHTTPClientReturnsTransportCleanup(t *testing.T) {
	client, cleanup, err := newHTTPClient("127.0.0.1:8080", 5*time.Second)
	if err != nil {
		t.Fatalf("newHTTPClient() error = %v", err)
	}
	if cleanup == nil {
		t.Fatal("cleanup = nil")
	}

	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("client.Transport type = %T, want *http.Transport", client.Transport)
	}
	if transport.Proxy == nil {
		t.Fatal("transport.Proxy = nil")
	}
}
