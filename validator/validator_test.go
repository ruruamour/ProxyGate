package validator

import (
	"net/http"
	"testing"
	"time"

	"proxygate/config"
)

func TestValidationTargetsDedupesPrimaryAndFallbacks(t *testing.T) {
	cfg := &config.Config{
		ValidateFallbackURLs: []string{
			"https://www.cloudflare.com/",
			"https://www.cloudflare.com/",
			"https://httpbin.org/ip",
		},
	}

	targets := validationTargets("https://www.cloudflare.com/", cfg)
	if len(targets) != 2 {
		t.Fatalf("len(validationTargets()) = %d, want 2", len(targets))
	}
	if targets[0] != "https://www.cloudflare.com/" {
		t.Fatalf("targets[0] = %q, want primary target", targets[0])
	}
	if targets[1] != "https://httpbin.org/ip" {
		t.Fatalf("targets[1] = %q, want fallback target", targets[1])
	}
}

func TestSplitValidationTargetsPromotesFirstFallbackWhenPrimaryEmpty(t *testing.T) {
	cfg := &config.Config{
		ValidateFallbackURLs: []string{
			"https://cp.cloudflare.com/generate_204",
			"https://cp.cloudflare.com/generate_204",
			"https://httpbin.org/ip",
		},
	}

	primary, fallbacks := splitValidationTargets("", cfg)
	if primary != "https://cp.cloudflare.com/generate_204" {
		t.Fatalf("primary = %q, want first fallback", primary)
	}
	if len(fallbacks) != 1 {
		t.Fatalf("len(fallbacks) = %d, want 1", len(fallbacks))
	}
	if fallbacks[0] != "https://httpbin.org/ip" {
		t.Fatalf("fallbacks[0] = %q, want remaining fallback", fallbacks[0])
	}
}

func TestValidationAttemptsCapsProbeFanout(t *testing.T) {
	if got := validationAttempts(nil); got != 0 {
		t.Fatalf("validationAttempts(nil) = %d, want 0", got)
	}
	if got := validationAttempts([]string{"a"}); got != 1 {
		t.Fatalf("validationAttempts(1) = %d, want 1", got)
	}
	if got := validationAttempts([]string{"a", "b", "c", "d"}); got != 2 {
		t.Fatalf("validationAttempts(4) = %d, want 2", got)
	}
	if got := validationAttempts([]string{"a", "b", "c", "d", "e"}); got != 3 {
		t.Fatalf("validationAttempts(5) = %d, want 3", got)
	}
}

func TestCloneClientWithTimeoutOnlyTightensTimeout(t *testing.T) {
	client := &http.Client{Timeout: 10 * time.Second}

	cloned := cloneClientWithTimeout(client, 4*time.Second)
	if cloned.Timeout != 4*time.Second {
		t.Fatalf("cloneClientWithTimeout() timeout = %v, want 4s", cloned.Timeout)
	}
	if client.Timeout != 10*time.Second {
		t.Fatalf("original client timeout = %v, want unchanged 10s", client.Timeout)
	}

	cloned = cloneClientWithTimeout(client, 20*time.Second)
	if cloned.Timeout != 10*time.Second {
		t.Fatalf("cloneClientWithTimeout() timeout = %v, want unchanged 10s", cloned.Timeout)
	}
}
