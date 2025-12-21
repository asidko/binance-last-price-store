package config

import (
	"log/slog"
	"os"
	"testing"
)

func TestLoad_Defaults(t *testing.T) {
	os.Unsetenv("DB_PATH")
	os.Unsetenv("HTTP_PORT")
	os.Unsetenv("LOG_LEVEL")

	cfg := Load()

	if cfg.DBPath != "./.data/ticks.db" {
		t.Errorf("expected default DBPath, got %s", cfg.DBPath)
	}
	if cfg.HTTPPort != 8080 {
		t.Errorf("expected default HTTPPort 8080, got %d", cfg.HTTPPort)
	}
	if cfg.LogLevel != slog.LevelInfo {
		t.Errorf("expected default LogLevel INFO, got %s", cfg.LogLevel)
	}
}

func TestLoad_FromEnv(t *testing.T) {
	os.Setenv("DB_PATH", "/custom/path.db")
	os.Setenv("HTTP_PORT", "9090")
	os.Setenv("LOG_LEVEL", "DEBUG")
	defer os.Unsetenv("DB_PATH")
	defer os.Unsetenv("HTTP_PORT")
	defer os.Unsetenv("LOG_LEVEL")

	cfg := Load()

	if cfg.DBPath != "/custom/path.db" {
		t.Errorf("expected /custom/path.db, got %s", cfg.DBPath)
	}
	if cfg.HTTPPort != 9090 {
		t.Errorf("expected 9090, got %d", cfg.HTTPPort)
	}
	if cfg.LogLevel != slog.LevelDebug {
		t.Errorf("expected DEBUG, got %s", cfg.LogLevel)
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

func TestLoad_LogLevels(t *testing.T) {
	tests := []struct {
		env   string
		level slog.Level
	}{
		{"DEBUG", slog.LevelDebug},
		{"debug", slog.LevelDebug},
		{"INFO", slog.LevelInfo},
		{"WARN", slog.LevelWarn},
		{"WARNING", slog.LevelWarn},
		{"ERROR", slog.LevelError},
		{"invalid", slog.LevelInfo},
	}

	for _, tt := range tests {
		os.Setenv("LOG_LEVEL", tt.env)
		cfg := Load()
		if cfg.LogLevel != tt.level {
			t.Errorf("LOG_LEVEL=%q: expected %s, got %s", tt.env, tt.level, cfg.LogLevel)
		}
	}
	os.Unsetenv("LOG_LEVEL")
}
