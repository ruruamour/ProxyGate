package proxy

import "testing"

func TestShouldPenalizeProxyForTarget(t *testing.T) {
	tests := []struct {
		name   string
		target string
		want   bool
	}{
		{name: "public https url", target: "https://www.cloudflare.com/", want: true},
		{name: "public host port", target: "api.github.com:443", want: true},
		{name: "synthetic hostname", target: "http://automationcontrolled/", want: false},
		{name: "localhost callback", target: "http://localhost:1455/auth/callback", want: false},
		{name: "loopback ipv4", target: "127.0.0.1:8080", want: false},
		{name: "private ipv4", target: "192.168.1.10:8080", want: false},
		{name: "carrier nat ipv4", target: "100.64.0.10:8080", want: false},
		{name: "loopback ipv6", target: "[::1]:8080", want: false},
		{name: "local suffix", target: "printer.local:80", want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := shouldPenalizeProxyForTarget(tc.target); got != tc.want {
				t.Fatalf("shouldPenalizeProxyForTarget(%q) = %v, want %v", tc.target, got, tc.want)
			}
		})
	}
}
