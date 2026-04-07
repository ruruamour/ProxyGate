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

func TestGetReturnsStoredSnapshot(t *testing.T) {
	previous := globalCfg.Load()
	globalCfg.Store(&Config{
		PoolMaxSize:           100,
		BlockedCountries:      []string{"CN"},
		ValidateFallbackURLs:  []string{"https://example.com"},
		CustomRefreshInterval: 60,
	})
	t.Cleanup(func() {
		globalCfg.Store(previous)
	})

	snapshot := Get()
	if snapshot == nil {
		t.Fatal("Get() returned nil")
	}
	if snapshot.PoolMaxSize != 100 {
		t.Fatalf("snapshot PoolMaxSize = %d, want 100", snapshot.PoolMaxSize)
	}
	if snapshot.BlockedCountries[0] != "CN" {
		t.Fatalf("snapshot blocked countries = %v, want [CN]", snapshot.BlockedCountries)
	}
	if snapshot.ValidateFallbackURLs[0] != "https://example.com" {
		t.Fatalf("snapshot fallback URLs = %v, want original value", snapshot.ValidateFallbackURLs)
	}
}

func TestSaveClonesCallerInput(t *testing.T) {
	previous := globalCfg.Load()
	t.Cleanup(func() {
		globalCfg.Store(previous)
	})
	t.Setenv("DATA_DIR", t.TempDir())

	cfg := &Config{
		PoolMaxSize:           100,
		BlockedCountries:      []string{"CN"},
		AllowedCountries:      []string{"US"},
		ValidateFallbackURLs:  []string{"https://example.com"},
		CustomRefreshInterval: 60,
	}

	if err := Save(cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	cfg.PoolMaxSize = 999
	cfg.BlockedCountries[0] = "JP"
	cfg.AllowedCountries[0] = "DE"
	cfg.ValidateFallbackURLs[0] = "https://changed.example.com"

	fresh := Get()
	if fresh.PoolMaxSize != 100 {
		t.Fatalf("fresh PoolMaxSize = %d, want 100", fresh.PoolMaxSize)
	}
	if fresh.BlockedCountries[0] != "CN" {
		t.Fatalf("fresh blocked countries = %v, want [CN]", fresh.BlockedCountries)
	}
	if fresh.AllowedCountries[0] != "US" {
		t.Fatalf("fresh allowed countries = %v, want [US]", fresh.AllowedCountries)
	}
	if fresh.ValidateFallbackURLs[0] != "https://example.com" {
		t.Fatalf("fresh fallback URLs = %v, want original value", fresh.ValidateFallbackURLs)
	}
}
