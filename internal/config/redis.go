package config

import (
	"context"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

func NewRedisClient() *redis.Client {
	addr := GetEnvOrDefault("REDIS_ADDR", "redis-db:6379")
	password := GetEnvOrDefault("REDIS_PASSWORD", "")

	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       0,
	})

	// Optional: ping to test connection
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		slog.Error("Failed to connect to Redis", "error", err, "addr", addr)
		panic("redis connection failed")
	}

	slog.Info("Redis connected", "addr", addr)
	return rdb
}
