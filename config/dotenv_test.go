package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDotEnv(t *testing.T) {
	tmpDir := t.TempDir()
	envFile := filepath.Join(tmpDir, ".env")
	content := `# comment
WEBUI_PASSWORD=from-file
BLOCKED_COUNTRIES="US,JP"
export DATA_DIR=./data
SINGBOX_PATH='custom-sing-box'
`
	if err := os.WriteFile(envFile, []byte(content), 0644); err != nil {
		t.Fatalf("write .env: %v", err)
	}

	t.Setenv("WEBUI_PASSWORD", "from-env")

	if err := LoadDotEnv(envFile); err != nil {
		t.Fatalf("LoadDotEnv: %v", err)
	}

	if got := os.Getenv("WEBUI_PASSWORD"); got != "from-env" {
		t.Fatalf("WEBUI_PASSWORD = %q, want process env to win", got)
	}
	if got := os.Getenv("BLOCKED_COUNTRIES"); got != "US,JP" {
		t.Fatalf("BLOCKED_COUNTRIES = %q", got)
	}
	if got := os.Getenv("DATA_DIR"); got != "./data" {
		t.Fatalf("DATA_DIR = %q", got)
	}
	if got := os.Getenv("SINGBOX_PATH"); got != "custom-sing-box" {
		t.Fatalf("SINGBOX_PATH = %q", got)
	}
}

func TestLoadDotEnvRejectsMalformedLine(t *testing.T) {
	tmpDir := t.TempDir()
	envFile := filepath.Join(tmpDir, ".env")
	if err := os.WriteFile(envFile, []byte("BROKEN_LINE\n"), 0644); err != nil {
		t.Fatalf("write .env: %v", err)
	}

	if err := LoadDotEnv(envFile); err == nil {
		t.Fatal("LoadDotEnv succeeded for malformed line, want error")
	}
}

func TestDefaultConfigReadsPortEnv(t *testing.T) {
	t.Setenv("WEBUI_PORT", "18080")
	t.Setenv("RANDOM_PORT", "18081")
	t.Setenv("STABLE_PORT", ":18082")
	t.Setenv("SOCKS5_RANDOM_PORT", "18083")
	t.Setenv("SOCKS5_STABLE_PORT", "18084")

	cfg := DefaultConfig()

	if cfg.WebUIPort != ":18080" {
		t.Fatalf("WebUIPort = %q", cfg.WebUIPort)
	}
	if cfg.ProxyPort != ":18081" {
		t.Fatalf("ProxyPort = %q", cfg.ProxyPort)
	}
	if cfg.StableProxyPort != ":18082" {
		t.Fatalf("StableProxyPort = %q", cfg.StableProxyPort)
	}
	if cfg.SOCKS5Port != ":18083" {
		t.Fatalf("SOCKS5Port = %q", cfg.SOCKS5Port)
	}
	if cfg.StableSOCKS5Port != ":18084" {
		t.Fatalf("StableSOCKS5Port = %q", cfg.StableSOCKS5Port)
	}
}
