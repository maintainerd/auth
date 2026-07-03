//go:build integration

package cache_test

import (
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/maintainerd/maintainerd-auth/internal/platform/cache"
	"github.com/redis/go-redis/v9"
)

func newTestCache(t *testing.T) (*cache.Cache, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	return cache.New(rdb), mr
}
