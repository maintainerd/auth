package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/maintainerd/auth/internal/model"
	"github.com/redis/go-redis/v9"
)

const (
	// userContextPrefix is the key prefix for cached user context entries.
	userContextPrefix = "user:"

	// UserContextTTL is how long a user context entry stays in cache.
	UserContextTTL = 10 * time.Minute

	// scanBatchSize is the COUNT hint for SCAN commands.
	scanBatchSize = 100
)

// UserContext is the data stored in the user-context cache.
type UserContext struct {
	User     *model.User             `json:"user"`
	Tenant   *model.Tenant           `json:"tenant"`
	Provider *model.IdentityProvider `json:"provider"`
	Client   *model.Client           `json:"client"`
}

// Cache provides typed helpers around a Redis client for user-context
// caching and invalidation.
type Cache struct {
	rdb *redis.Client
}

// New creates a Cache backed by the given Redis client.
func New(rdb *redis.Client) *Cache {
	return &Cache{rdb: rdb}
}

// ---------------------------------------------------------------------------
// User context — read / write
// ---------------------------------------------------------------------------

// userContextKey builds the Redis key for a user context entry.
func userContextKey(sub, clientID string) string {
	return userContextPrefix + sub + ":" + clientID
}

// GetUserContext retrieves a cached user context. Returns nil when the key
// does not exist or cannot be deserialized (cache miss).
func (c *Cache) GetUserContext(ctx context.Context, sub, clientID string) *UserContext {
	raw, err := c.rdb.Get(ctx, userContextKey(sub, clientID)).Result()
	if err != nil {
		return nil
	}
	var uc UserContext
	if err := json.Unmarshal([]byte(raw), &uc); err != nil {
		return nil
	}
	return &uc
}

// SetUserContext caches a user context entry with the default TTL.
func (c *Cache) SetUserContext(ctx context.Context, sub, clientID string, uc *UserContext) {
	data, err := json.Marshal(uc)
	if err != nil {
		return
	}
	_ = c.rdb.Set(ctx, userContextKey(sub, clientID), data, UserContextTTL).Err()
}

// ---------------------------------------------------------------------------
// Invalidation
// ---------------------------------------------------------------------------

// InvalidateUser removes the cached context for a specific user + client pair.
func (c *Cache) InvalidateUser(ctx context.Context, sub, clientID string) {
	_ = c.rdb.Del(ctx, userContextKey(sub, clientID)).Err()
}

// InvalidateUserAll removes every cached context entry for the given sub
// (across all client IDs) using an iterative SCAN to avoid blocking Redis.
func (c *Cache) InvalidateUserAll(ctx context.Context, sub string) {
	c.deleteByPattern(ctx, userContextPrefix+sub+":*")
}

// InvalidateAllUsers removes every user-context cache entry. Use this when a
// change potentially affects many users (e.g. role permission updates).
func (c *Cache) InvalidateAllUsers(ctx context.Context) {
	c.deleteByPattern(ctx, userContextPrefix+"*")
}

// deleteByPattern iterates with SCAN and deletes matching keys in batches.
func (c *Cache) deleteByPattern(ctx context.Context, pattern string) {
	var cursor uint64
	for {
		keys, nextCursor, err := c.rdb.Scan(ctx, cursor, pattern, scanBatchSize).Result()
		if err != nil {
			return
		}
		if len(keys) > 0 {
			_ = c.rdb.Del(ctx, keys...).Err()
		}
		cursor = nextCursor
		if cursor == 0 {
			return
		}
	}
}

// ---------------------------------------------------------------------------
// Invalidator interface — consumed by services
// ---------------------------------------------------------------------------

// Invalidator is the subset of Cache that services use to invalidate cached
// data after mutations. Keeping this as an interface allows services to be
// tested without a real Redis connection.
type Invalidator interface {
	// InvalidateUser removes the cache entry for a specific sub + clientID.
	InvalidateUser(ctx context.Context, sub, clientID string)
	// InvalidateUserAll removes all cache entries for the given sub.
	InvalidateUserAll(ctx context.Context, sub string)
	// InvalidateAllUsers removes every user-context cache entry.
	InvalidateAllUsers(ctx context.Context)
}

// Compile-time check that *Cache satisfies Invalidator.
var _ Invalidator = (*Cache)(nil)

// NopInvalidator is a no-op Invalidator for use in tests or when caching is
// disabled.
type NopInvalidator struct{}

func (NopInvalidator) InvalidateUser(context.Context, string, string) {}
func (NopInvalidator) InvalidateUserAll(context.Context, string)      {}
func (NopInvalidator) InvalidateAllUsers(context.Context)             {}

// Compile-time check.
var _ Invalidator = NopInvalidator{}

// ---------------------------------------------------------------------------
// Key helpers (exported for middleware)
// ---------------------------------------------------------------------------

// UserContextKeyFor returns the Redis key for a given sub and clientID.
// Exported so the middleware can set/get using the same key scheme.
func UserContextKeyFor(sub, clientID string) string {
	return userContextKey(sub, clientID)
}

// FormatKey builds a namespaced cache key. General-purpose helper for future
// cache entries beyond user context.
func FormatKey(prefix string, parts ...string) string {
	key := prefix
	for _, p := range parts {
		key += fmt.Sprintf(":%s", p)
	}
	return key
}
