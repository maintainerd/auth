package config

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/maintainerd/maintainerd-auth/internal/platform/retry"
	"github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"
)

// NewRedisClient creates and verifies a Redis client, retrying the initial ping
// with exponential backoff until ctx is cancelled or all attempts are exhausted.
func NewRedisClient(ctx context.Context) (*redis.Client, error) {
	addr := GetEnvOrDefault("REDIS_ADDR", "redis-db:6379")
	password := GetEnvOrDefault("REDIS_PASSWORD", "")

	opts := &redis.Options{
		Addr:     addr,
		Password: password,
		DB:       0,
	}

	useTLS, _ := strconv.ParseBool(GetEnvOrDefault("REDIS_TLS", "false"))
	useTLS = useTLS || strings.HasPrefix(addr, "rediss://")
	if useTLS {
		opts.TLSConfig = &tls.Config{MinVersion: tls.VersionTLS12}
		slog.Info("Redis TLS enabled")
	}

	rdb := redis.NewClient(opts)

	if err := retry.WithBackoff(ctx, "redis", 10, 2*time.Second, func() error {
		pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		return rdb.Ping(pingCtx).Err()
	}); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis at %s: %w", addr, err)
	}

	slog.Info("Redis connected", "addr", addr)

	if err := redisotel.InstrumentTracing(rdb); err != nil {
		return nil, fmt.Errorf("failed to register redisotel tracing: %w", err)
	}

	return rdb, nil
}
