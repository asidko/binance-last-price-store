package config

import (
	"os"
	"strconv"
)

type Config struct {
	DBPath   string
	HTTPPort int
}

func Load() Config {
	return Config{
		DBPath:   getEnv("DB_PATH", "./.data/ticks.db"),
		HTTPPort: getEnvInt("HTTP_PORT", 8080),
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
