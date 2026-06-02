package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/maintainerd/auth/internal/authctx"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

const (
	// userContextPrefix is the key prefix for cached user context entries.
	userContextPrefix = "user:"

	// jtiDenylistPrefix is the key prefix for revoked access token JTIs.
	jtiDenylistPrefix = "jti:deny:"

	// UserContextTTL is how long a user context entry stays in cache.
	UserContextTTL = 10 * time.Minute

	// scanBatchSize is the COUNT hint for SCAN commands.
	scanBatchSize = 100
)

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
func (c *Cache) GetUserContext(ctx context.Context, sub, clientID string) *authctx.UserContext {
	_, span := otel.Tracer("cache").Start(ctx, "cache.get_user_context")
	defer span.End()
	span.SetAttributes(
		attribute.String("sub", sub),
		attribute.String("client_id", clientID),
	)

	raw, err := c.rdb.Get(ctx, userContextKey(sub, clientID)).Result()
	if err != nil {
		span.SetStatus(codes.Error, "cache miss")
		return nil
	}
	var uc authctx.UserContext
	if err := json.Unmarshal([]byte(raw), &uc); err != nil {
		span.SetStatus(codes.Error, "deserialize failed")
		return nil
	}
	span.SetStatus(codes.Ok, "")
	return &uc
}

// SetUserContext caches a user context entry with the default TTL.
func (c *Cache) SetUserContext(ctx context.Context, sub, clientID string, uc *authctx.UserContext) {
	_, span := otel.Tracer("cache").Start(ctx, "cache.set_user_context")
	defer span.End()
	span.SetAttributes(
		attribute.String("sub", sub),
		attribute.String("client_id", clientID),
	)

	data, err := json.Marshal(uc)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "serialize failed")
		return
	}
	_ = c.rdb.Set(ctx, userContextKey(sub, clientID), data, UserContextTTL).Err()
	span.SetStatus(codes.Ok, "")
}

// ---------------------------------------------------------------------------
// JTI denylist — access token revocation
// ---------------------------------------------------------------------------

// jtiDenylistKey builds the Redis key for a denied JTI.
func jtiDenylistKey(jti string) string {
	return jtiDenylistPrefix + jti
}

// DenyJTI adds a JTI to the denylist with the given TTL.
// Call this when an access token is explicitly revoked.
func (c *Cache) DenyJTI(ctx context.Context, jti string, ttl time.Duration) error {
	_, span := otel.Tracer("cache").Start(ctx, "cache.deny_jti")
	defer span.End()
	span.SetAttributes(attribute.String("jti", jti))

	if err := c.rdb.Set(ctx, jtiDenylistKey(jti), "1", ttl).Err(); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "jti deny failed")
		return err
	}
	span.SetStatus(codes.Ok, "")
	return nil
}

// IsJTIDenied reports whether a JTI has been revoked. Returns false on Redis
// errors so that a cache outage does not break token validation.
func (c *Cache) IsJTIDenied(ctx context.Context, jti string) (bool, error) {
	_, span := otel.Tracer("cache").Start(ctx, "cache.is_jti_denied")
	defer span.End()
	span.SetAttributes(attribute.String("jti", jti))

	result, err := c.rdb.Exists(ctx, jtiDenylistKey(jti)).Result()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "jti check failed")
		return false, err
	}
	denied := result > 0
	span.SetStatus(codes.Ok, "")
	return denied, nil
}

// ---------------------------------------------------------------------------
// Invalidation
// ---------------------------------------------------------------------------

// InvalidateUser removes the cached context for a specific user + client pair.
func (c *Cache) InvalidateUser(ctx context.Context, sub, clientID string) {
	_, span := otel.Tracer("cache").Start(ctx, "cache.invalidate_user")
	defer span.End()
	span.SetAttributes(
		attribute.String("sub", sub),
		attribute.String("client_id", clientID),
	)

	_ = c.rdb.Del(ctx, userContextKey(sub, clientID)).Err()
	span.SetStatus(codes.Ok, "")
}

// InvalidateUserAll removes every cached context entry for the given sub
// (across all client IDs) using an iterative SCAN to avoid blocking Redis.
func (c *Cache) InvalidateUserAll(ctx context.Context, sub string) {
	_, span := otel.Tracer("cache").Start(ctx, "cache.invalidate_user_all")
	defer span.End()
	span.SetAttributes(attribute.String("sub", sub))

	c.deleteByPattern(ctx, userContextPrefix+sub+":*")
	span.SetStatus(codes.Ok, "")
}

// InvalidateAllUsers removes every user-context cache entry. Use this when a
// change potentially affects many users (e.g. role permission updates).
func (c *Cache) InvalidateAllUsers(ctx context.Context) {
	_, span := otel.Tracer("cache").Start(ctx, "cache.invalidate_all_users")
	defer span.End()

	c.deleteByPattern(ctx, userContextPrefix+"*")
	span.SetStatus(codes.Ok, "")
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

// JTIDenylister is the interface consumed by JWT validation and token
// revocation to manage the access-token denylist.
type JTIDenylister interface {
	// DenyJTI adds the JTI to the denylist for the given TTL.
	DenyJTI(ctx context.Context, jti string, ttl time.Duration) error
	// IsJTIDenied reports whether the JTI has been revoked.
	IsJTIDenied(ctx context.Context, jti string) (bool, error)
}

// Compile-time check that *Cache satisfies JTIDenylister.
var _ JTIDenylister = (*Cache)(nil)

// NopJTIDenylister is a no-op JTIDenylister for use in tests.
type NopJTIDenylister struct{}

func (NopJTIDenylister) DenyJTI(context.Context, string, time.Duration) error { return nil }
func (NopJTIDenylister) IsJTIDenied(context.Context, string) (bool, error)    { return false, nil }

// Compile-time check.
var _ JTIDenylister = NopJTIDenylister{}

// ---------------------------------------------------------------------------
// Generic JSON session storage (used by WebAuthn ceremony sessions)
// ---------------------------------------------------------------------------

// SetSession serializes value to JSON and stores it under key with TTL.
func (c *Cache) SetSession(ctx context.Context, key string, value any, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return c.rdb.Set(ctx, key, data, ttl).Err()
}

// GetSession retrieves the key from Redis and deserializes it into dest.
// Returns redis.Nil when the key does not exist.
func (c *Cache) GetSession(ctx context.Context, key string, dest any) error {
	raw, err := c.rdb.Get(ctx, key).Result()
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(raw), dest)
}

// DeleteSession removes a session key from Redis.
func (c *Cache) DeleteSession(ctx context.Context, key string) error {
	return c.rdb.Del(ctx, key).Err()
}

// WebAuthnSessionStore is the interface used by WebAuthnService to persist
// ceremony session data (challenge, user ID) between Begin and Finish calls.
type WebAuthnSessionStore interface {
	SetSession(ctx context.Context, key string, value any, ttl time.Duration) error
	GetSession(ctx context.Context, key string, dest any) error
	DeleteSession(ctx context.Context, key string) error
}

// Compile-time check that *Cache satisfies WebAuthnSessionStore.
var _ WebAuthnSessionStore = (*Cache)(nil)

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
