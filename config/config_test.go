package config

import "testing"

func TestCloneConfigCopiesSlices(t *testing.T) {
	original := &Config{
		BlockedCountries:     []string{"CN"},
		AllowedCountries:     []string{"US"},
		ValidateFallbackURLs: []string{"https://example.com"},
	}

	cloned := cloneConfig(original)
	cloned.BlockedCountries[0] = "JP"
	cloned.AllowedCountries[0] = "DE"
	cloned.ValidateFallbackURLs[0] = "https://changed.example.com"

	if original.BlockedCountries[0] != "CN" {
		t.Fatalf("original blocked countries mutated: %v", original.BlockedCountries)
	}
	if original.AllowedCountries[0] != "US" {
		t.Fatalf("original allowed countries mutated: %v", original.AllowedCountries)
	}
	if original.ValidateFallbackURLs[0] != "https://example.com" {
		t.Fatalf("original fallback URLs mutated: %v", original.ValidateFallbackURLs)
	}
}

func TestGetReturnsIndependentSnapshot(t *testing.T) {
	cfgMu.Lock()
	previous := globalCfg
	globalCfg = &Config{
		PoolMaxSize:           100,
		BlockedCountries:      []string{"CN"},
		ValidateFallbackURLs:  []string{"https://example.com"},
		CustomRefreshInterval: 60,
	}
	cfgMu.Unlock()
	t.Cleanup(func() {
		cfgMu.Lock()
		globalCfg = previous
		cfgMu.Unlock()
	})

	snapshot := Get()
	snapshot.PoolMaxSize = 999
	snapshot.BlockedCountries[0] = "JP"
	snapshot.ValidateFallbackURLs[0] = "https://changed.example.com"

	fresh := Get()
	if fresh.PoolMaxSize != 100 {
		t.Fatalf("fresh PoolMaxSize = %d, want 100", fresh.PoolMaxSize)
	}
	if fresh.BlockedCountries[0] != "CN" {
		t.Fatalf("fresh blocked countries = %v, want [CN]", fresh.BlockedCountries)
	}
	if fresh.ValidateFallbackURLs[0] != "https://example.com" {
		t.Fatalf("fresh fallback URLs = %v, want original value", fresh.ValidateFallbackURLs)
	}
}
