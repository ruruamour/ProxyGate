package custom

import (
	"context"
	"net"
	"testing"
)

func TestValidatePublicSubscriptionURLRejectsLocalTargets(t *testing.T) {
	cases := []string{
		"http://127.0.0.1/sub",
		"http://localhost/sub",
		"http://10.0.0.1/sub",
		"http://[::1]/sub",
	}

	for _, raw := range cases {
		if err := validatePublicSubscriptionURL(raw); err == nil {
			t.Fatalf("validatePublicSubscriptionURL(%q) error = nil, want reject", raw)
		}
	}
}

func TestEnsurePublicSubscriptionHostRejectsPrivateResolvedIP(t *testing.T) {
	if err := ensurePublicSubscriptionHost(context.Background(), "127.0.0.1"); err == nil {
		t.Fatal("ensurePublicSubscriptionHost(loopback) error = nil, want reject")
	}

	if err := ensurePublicSubscriptionHost(context.Background(), "10.0.0.8"); err == nil {
		t.Fatal("ensurePublicSubscriptionHost(private) error = nil, want reject")
	}

	if err := ensurePublicSubscriptionHost(context.Background(), net.IPv6loopback.String()); err == nil {
		t.Fatal("ensurePublicSubscriptionHost(ipv6 loopback) error = nil, want reject")
	}
}
