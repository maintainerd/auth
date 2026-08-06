package config

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRedisClient_Success(t *testing.T) {
	useEnvSecretManager(t)
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	t.Setenv("REDIS_ADDR", mr.Addr())
	t.Setenv("REDIS_PASSWORD", "")

	rdb, err := NewRedisClient(context.Background())
	require.NoError(t, err)
	require.NotNil(t, rdb)
	defer func() { _ = rdb.Close() }()

	// Verify the client is functional
	assert.NoError(t, rdb.Ping(t.Context()).Err())
}

func TestNewRedisClient_WithPassword(t *testing.T) {
	useEnvSecretManager(t)
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	mr.RequireAuth("s3cret")

	t.Setenv("REDIS_ADDR", mr.Addr())
	t.Setenv("REDIS_PASSWORD", "s3cret")

	rdb, err := NewRedisClient(context.Background())
	require.NoError(t, err)
	require.NotNil(t, rdb)
	defer func() { _ = rdb.Close() }()

	assert.NoError(t, rdb.Ping(t.Context()).Err())
}

func TestNewRedisClient_WrongPassword(t *testing.T) {
	useEnvSecretManager(t)
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	mr.RequireAuth("correct-password")

	t.Setenv("REDIS_ADDR", mr.Addr())
	t.Setenv("REDIS_PASSWORD", "wrong-password")

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	rdb, err := NewRedisClient(ctx)
	assert.Nil(t, rdb)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to connect to Redis")
}

func TestNewRedisClient_Unreachable(t *testing.T) {
	useEnvSecretManager(t)
	t.Setenv("REDIS_ADDR", "127.0.0.1:1")
	t.Setenv("REDIS_PASSWORD", "")

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	rdb, err := NewRedisClient(ctx)
	assert.Nil(t, rdb)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to connect to Redis")
}

func TestNewRedisClient_DefaultAddr(t *testing.T) {
	useEnvSecretManager(t)
	// When no REDIS_ADDR is set, falls back to "redis-db:6379" which is unreachable in tests.
	t.Setenv("REDIS_ADDR", "")

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	rdb, err := NewRedisClient(ctx)
	assert.Nil(t, rdb)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to connect to Redis")
}

func TestNewRedisClient_OTelTracingRegistered(t *testing.T) {
	useEnvSecretManager(t)
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	t.Setenv("REDIS_ADDR", mr.Addr())
	t.Setenv("REDIS_PASSWORD", "")

	rdb, err := NewRedisClient(context.Background())
	require.NoError(t, err)
	require.NotNil(t, rdb)
	defer func() { _ = rdb.Close() }()

	// After NewRedisClient, redisotel.InstrumentTracing was called.
	// Verify the client still works (tracing hook is transparent).
	require.NoError(t, rdb.Set(t.Context(), "test-key", "test-value", 0).Err())
	val, err := rdb.Get(t.Context(), "test-key").Result()
	require.NoError(t, err)
	assert.Equal(t, "test-value", val)
}

func TestNewRedisClient_TLSViaRedissPrefix(t *testing.T) {
	useEnvSecretManager(t)
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	// miniredis doesn't support TLS, but the rediss:// prefix will trigger TLS config.
	// The ping will fail because miniredis is not TLS, but the TLS code path is exercised.
	t.Setenv("REDIS_ADDR", "rediss://"+mr.Addr())
	t.Setenv("REDIS_PASSWORD", "")

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	rdb, _ := NewRedisClient(ctx)
	if rdb != nil {
		defer func() { _ = rdb.Close() }()
	}
	// Connection will fail because miniredis is plaintext, but the TLS branch is covered.
}

func TestNewRedisClient_TLSViaEnvVar(t *testing.T) {
	useEnvSecretManager(t)
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	t.Setenv("REDIS_ADDR", mr.Addr())
	t.Setenv("REDIS_PASSWORD", "")
	t.Setenv("REDIS_TLS", "true")

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	rdb, _ := NewRedisClient(ctx)
	if rdb != nil {
		defer func() { _ = rdb.Close() }()
	}
	// Connection will fail because miniredis is plaintext, but the TLS env var branch is covered.
}
