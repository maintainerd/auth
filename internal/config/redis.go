package config

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

// NewRedisClient creates and verifies a Redis client. It returns the client and
// any connection error, so main() can decide how to handle failures instead of
// the library calling panic().
func NewRedisClient() (*redis.Client, error) {
	addr := GetEnvOrDefault("REDIS_ADDR", "redis-db:6379")
	password := GetEnvOrDefault("REDIS_PASSWORD", "")

	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       0,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis at %s: %w", addr, err)
	}

	slog.Info("Redis connected", "addr", addr)
	return rdb, nil
}
