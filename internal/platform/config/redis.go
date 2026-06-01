package config

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"
)

// NewRedisClient creates and verifies a Redis client. It returns the client and
// any connection error, so main() can decide how to handle failures instead of
// the library calling panic().
func NewRedisClient() (*redis.Client, error) {
	addr := GetEnvOrDefault("REDIS_ADDR", "redis-db:6379")
	password := GetEnvOrDefault("REDIS_PASSWORD", "")

	opts := &redis.Options{
		Addr:     addr,
		Password: password,
		DB:       0,
	}

	useTLS := GetEnvOrDefault("REDIS_TLS", "") == "true" || strings.HasPrefix(addr, "rediss://")
	if useTLS {
		opts.TLSConfig = &tls.Config{MinVersion: tls.VersionTLS12}
		slog.Info("Redis TLS enabled")
	}

	rdb := redis.NewClient(opts)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis at %s: %w", addr, err)
	}

	slog.Info("Redis connected", "addr", addr)

	if err := redisotel.InstrumentTracing(rdb); err != nil {
		return nil, fmt.Errorf("failed to register redisotel tracing: %w", err)
	}

	return rdb, nil
}
