package config

import (
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRedisClient_Success(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	t.Setenv("REDIS_ADDR", mr.Addr())
	t.Setenv("REDIS_PASSWORD", "")

	rdb, err := NewRedisClient()
	require.NoError(t, err)
	require.NotNil(t, rdb)
	defer rdb.Close()

	// Verify the client is functional
	assert.NoError(t, rdb.Ping(t.Context()).Err())
}

func TestNewRedisClient_WithPassword(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	mr.RequireAuth("s3cret")

	t.Setenv("REDIS_ADDR", mr.Addr())
	t.Setenv("REDIS_PASSWORD", "s3cret")

	rdb, err := NewRedisClient()
	require.NoError(t, err)
	require.NotNil(t, rdb)
	defer rdb.Close()

	assert.NoError(t, rdb.Ping(t.Context()).Err())
}

func TestNewRedisClient_WrongPassword(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	mr.RequireAuth("correct-password")

	t.Setenv("REDIS_ADDR", mr.Addr())
	t.Setenv("REDIS_PASSWORD", "wrong-password")

	rdb, err := NewRedisClient()
	assert.Nil(t, rdb)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to connect to Redis")
}

func TestNewRedisClient_Unreachable(t *testing.T) {
	t.Setenv("REDIS_ADDR", "127.0.0.1:1")
	t.Setenv("REDIS_PASSWORD", "")

	rdb, err := NewRedisClient()
	assert.Nil(t, rdb)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to connect to Redis")
}

func TestNewRedisClient_DefaultAddr(t *testing.T) {
	// When no REDIS_ADDR is set, falls back to "redis-db:6379" which is unreachable in tests.
	t.Setenv("REDIS_ADDR", "")

	rdb, err := NewRedisClient()
	assert.Nil(t, rdb)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to connect to Redis")
}

func TestNewRedisClient_OTelTracingRegistered(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	t.Setenv("REDIS_ADDR", mr.Addr())
	t.Setenv("REDIS_PASSWORD", "")

	rdb, err := NewRedisClient()
	require.NoError(t, err)
	require.NotNil(t, rdb)
	defer rdb.Close()

	// After NewRedisClient, redisotel.InstrumentTracing was called.
	// Verify the client still works (tracing hook is transparent).
	require.NoError(t, rdb.Set(t.Context(), "test-key", "test-value", 0).Err())
	val, err := rdb.Get(t.Context(), "test-key").Result()
	require.NoError(t, err)
	assert.Equal(t, "test-value", val)
}
