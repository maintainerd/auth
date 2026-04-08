package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetEnv(t *testing.T) {
	t.Run("returns value when set", func(t *testing.T) {
		t.Setenv("TEST_GET_ENV_KEY", "hello")
		val, err := GetEnv("TEST_GET_ENV_KEY")
		require.NoError(t, err)
		assert.Equal(t, "hello", val)
	})

	t.Run("returns error when not set", func(t *testing.T) {
		val, err := GetEnv("TEST_GET_ENV_MISSING")
		assert.Empty(t, val)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "TEST_GET_ENV_MISSING")
	})

	t.Run("returns error when empty", func(t *testing.T) {
		t.Setenv("TEST_GET_ENV_EMPTY", "")
		val, err := GetEnv("TEST_GET_ENV_EMPTY")
		assert.Empty(t, val)
		require.Error(t, err)
	})
}

func TestGetEnvOrDefault(t *testing.T) {
	t.Run("returns value when set", func(t *testing.T) {
		t.Setenv("TEST_ENV_OR_DEFAULT", "custom")
		assert.Equal(t, "custom", GetEnvOrDefault("TEST_ENV_OR_DEFAULT", "fallback"))
	})

	t.Run("returns default when not set", func(t *testing.T) {
		assert.Equal(t, "fallback", GetEnvOrDefault("TEST_ENV_OR_DEFAULT_MISSING", "fallback"))
	})

	t.Run("returns default when empty", func(t *testing.T) {
		t.Setenv("TEST_ENV_OR_DEFAULT_EMPTY", "")
		assert.Equal(t, "fallback", GetEnvOrDefault("TEST_ENV_OR_DEFAULT_EMPTY", "fallback"))
	})
}
