package storage

import (
	"testing"

	"proxygate/config"
)

func TestCalculateQualityGradeForConfigUsesRuntimeThresholds(t *testing.T) {
	cfg := &config.Config{
		MaxLatencyHealthy: 2000,
		MaxLatencyMs:      2500,
	}

	cases := []struct {
		latency int
		want    string
	}{
		{latency: 400, want: "S"},
		{latency: 900, want: "A"},
		{latency: 2300, want: "B"},
		{latency: 2600, want: "C"},
	}

	for _, tc := range cases {
		if got := CalculateQualityGradeForConfig(tc.latency, cfg); got != tc.want {
			t.Fatalf("CalculateQualityGradeForConfig(%d) = %q, want %q", tc.latency, got, tc.want)
		}
	}
}

func TestCalculateQualityGradeForConfigNormalizesMisorderedThresholds(t *testing.T) {
	cfg := &config.Config{
		MaxLatencyHealthy: 2000,
		MaxLatencyMs:      1200,
	}

	if got := CalculateQualityGradeForConfig(1500, cfg); got != "B" {
		t.Fatalf("CalculateQualityGradeForConfig(1500) = %q, want %q", got, "B")
	}
	if got := CalculateQualityGradeForConfig(2100, cfg); got != "C" {
		t.Fatalf("CalculateQualityGradeForConfig(2100) = %q, want %q", got, "C")
	}
}
