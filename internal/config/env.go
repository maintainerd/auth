package config

import (
	"fmt"
	"os"
)

// GetEnv returns the value of the environment variable identified by key.
// It returns an error if the variable is not set or empty, allowing callers
// to decide how to handle the failure instead of calling os.Exit.
func GetEnv(key string) (string, error) {
	val := os.Getenv(key)
	if val == "" {
		return "", fmt.Errorf("required environment variable %q is not set", key)
	}
	return val, nil
}

func GetEnvOrDefault(key, defaultVal string) string {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	return val
}
