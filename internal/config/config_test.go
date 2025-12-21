package config

import (
	"os"
	"testing"
)

func TestLoad_Defaults(t *testing.T) {
	os.Unsetenv("DB_PATH")
	os.Unsetenv("HTTP_PORT")

	cfg := Load()

	if cfg.DBPath != "./.data/ticks.db" {
		t.Errorf("expected default DBPath, got %s", cfg.DBPath)
	}
	if cfg.HTTPPort != 8080 {
		t.Errorf("expected default HTTPPort 8080, got %d", cfg.HTTPPort)
	}
}

func TestLoad_FromEnv(t *testing.T) {
	os.Setenv("DB_PATH", "/custom/path.db")
	os.Setenv("HTTP_PORT", "9090")
	defer os.Unsetenv("DB_PATH")
	defer os.Unsetenv("HTTP_PORT")

	cfg := Load()

	if cfg.DBPath != "/custom/path.db" {
		t.Errorf("expected /custom/path.db, got %s", cfg.DBPath)
	}
	if cfg.HTTPPort != 9090 {
		t.Errorf("expected 9090, got %d", cfg.HTTPPort)
	}
}

func TestLoad_InvalidPort(t *testing.T) {
	os.Setenv("HTTP_PORT", "invalid")
	defer os.Unsetenv("HTTP_PORT")

	cfg := Load()

	if cfg.HTTPPort != 8080 {
		t.Errorf("expected fallback to 8080, got %d", cfg.HTTPPort)
	}
}
