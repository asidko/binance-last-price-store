package config

import (
	"log/slog"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	DBPath   string
	HTTPPort int
	LogLevel slog.Level
}

func Load() Config {
	return Config{
		DBPath:   getEnv("DB_PATH", "./.data/ticks.db"),
		HTTPPort: getEnvInt("HTTP_PORT", 8080),
		LogLevel: getLogLevel("LOG_LEVEL", slog.LevelInfo),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}

func getLogLevel(key string, fallback slog.Level) slog.Level {
	v := strings.ToUpper(os.Getenv(key))
	switch v {
	case "DEBUG":
		return slog.LevelDebug
	case "INFO":
		return slog.LevelInfo
	case "WARN", "WARNING":
		return slog.LevelWarn
	case "ERROR":
		return slog.LevelError
	default:
		return fallback
	}
}
