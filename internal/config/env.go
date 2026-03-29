package config

import (
	"log/slog"
	"os"
)

func GetEnv(key string) string {
	val := os.Getenv(key)
	if val == "" {
		slog.Error("missing required environment variable", "key", key)
		os.Exit(1)
	}
	return val
}

func GetEnvOrDefault(key, defaultVal string) string {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	return val
}
