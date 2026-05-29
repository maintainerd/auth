package cache

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestCache(t *testing.T) (*Cache, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	return New(rdb), mr
}

// ---------------------------------------------------------------------------
// GetUserContext / SetUserContext
// ---------------------------------------------------------------------------

func TestSetAndGetUserContext(t *testing.T) {
	c, _ := newTestCache(t)
	ctx := context.Background()

	uc := &UserContext{
		User: &AuthUser{
			UserUUID: uuid.New(),
			Email:    "alice@example.com",
			Fullname: "Alice Example",
		},
		Tenant: &AuthTenant{
			TenantID:   42,
			TenantUUID: uuid.New(),
		},
	}

	c.SetUserContext(ctx, "sub1", "client1", uc)

	got := c.GetUserContext(ctx, "sub1", "client1")
	require.NotNil(t, got)
	assert.Equal(t, "alice@example.com", got.User.Email)
	assert.Equal(t, "Alice Example", got.User.Fullname)
	assert.Equal(t, int64(42), got.Tenant.TenantID)
}

func TestGetUserContext_Miss(t *testing.T) {
	c, _ := newTestCache(t)
	ctx := context.Background()

	got := c.GetUserContext(ctx, "nonexistent", "client1")
	assert.Nil(t, got)
}

func TestGetUserContext_CorruptData(t *testing.T) {
	c, mr := newTestCache(t)
	ctx := context.Background()

	// Write garbage data directly
	err := mr.Set(userContextKey("sub1", "client1"), "not-json")
	require.NoError(t, err)

	got := c.GetUserContext(ctx, "sub1", "client1")
	assert.Nil(t, got)
}

func TestSetUserContext_TTL(t *testing.T) {
	c, mr := newTestCache(t)
	ctx := context.Background()

	uc := &UserContext{User: &AuthUser{UserUUID: uuid.New(), Email: "bob@example.com"}}
	c.SetUserContext(ctx, "sub1", "client1", uc)

	ttl := mr.TTL(userContextKey("sub1", "client1"))
	assert.Equal(t, UserContextTTL, ttl)
}

// ---------------------------------------------------------------------------
// InvalidateUser
// ---------------------------------------------------------------------------

func TestInvalidateUser(t *testing.T) {
	c, _ := newTestCache(t)
	ctx := context.Background()

	uc := &UserContext{User: &AuthUser{UserUUID: uuid.New(), Email: "alice@example.com"}}
	c.SetUserContext(ctx, "sub1", "client1", uc)

	// Also set another key for same sub but different client
	c.SetUserContext(ctx, "sub1", "client2", uc)

	c.InvalidateUser(ctx, "sub1", "client1")

	assert.Nil(t, c.GetUserContext(ctx, "sub1", "client1"), "invalidated key should be nil")
	assert.NotNil(t, c.GetUserContext(ctx, "sub1", "client2"), "other client key should remain")
}

// ---------------------------------------------------------------------------
// InvalidateUserAll
// ---------------------------------------------------------------------------

func TestInvalidateUserAll(t *testing.T) {
	c, _ := newTestCache(t)
	ctx := context.Background()

	uc := &UserContext{User: &AuthUser{UserUUID: uuid.New(), Email: "alice@example.com"}}
	c.SetUserContext(ctx, "sub1", "client1", uc)
	c.SetUserContext(ctx, "sub1", "client2", uc)
	c.SetUserContext(ctx, "sub2", "client1", uc)

	c.InvalidateUserAll(ctx, "sub1")

	assert.Nil(t, c.GetUserContext(ctx, "sub1", "client1"))
	assert.Nil(t, c.GetUserContext(ctx, "sub1", "client2"))
	assert.NotNil(t, c.GetUserContext(ctx, "sub2", "client1"), "other sub should remain")
}

// ---------------------------------------------------------------------------
// InvalidateAllUsers
// ---------------------------------------------------------------------------

func TestInvalidateAllUsers(t *testing.T) {
	c, _ := newTestCache(t)
	ctx := context.Background()

	uc := &UserContext{User: &AuthUser{UserUUID: uuid.New(), Email: "alice@example.com"}}
	c.SetUserContext(ctx, "sub1", "client1", uc)
	c.SetUserContext(ctx, "sub2", "client2", uc)

	c.InvalidateAllUsers(ctx)

	assert.Nil(t, c.GetUserContext(ctx, "sub1", "client1"))
	assert.Nil(t, c.GetUserContext(ctx, "sub2", "client2"))
}

// ---------------------------------------------------------------------------
// Key helpers
// ---------------------------------------------------------------------------

func TestUserContextKeyFor(t *testing.T) {
	assert.Equal(t, "user:sub1:client1", UserContextKeyFor("sub1", "client1"))
}

func TestFormatKey(t *testing.T) {
	assert.Equal(t, "prefix:a:b", FormatKey("prefix", "a", "b"))
	assert.Equal(t, "prefix", FormatKey("prefix"))
}

// ---------------------------------------------------------------------------
// NopInvalidator
// ---------------------------------------------------------------------------

func TestNopInvalidator(t *testing.T) {
	var nop NopInvalidator
	ctx := context.Background()

	// Just ensure no panics
	nop.InvalidateUser(ctx, "sub", "client")
	nop.InvalidateUserAll(ctx, "sub")
	nop.InvalidateAllUsers(ctx)
}
