package cache

import (
	"context"
	"testing"
	"time"

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

// ---------------------------------------------------------------------------
// DenyJTI / IsJTIDenied
// ---------------------------------------------------------------------------

func TestDenyJTI_Success(t *testing.T) {
	c, mr := newTestCache(t)
	ctx := context.Background()

	err := c.DenyJTI(ctx, "jti-123", time.Hour)
	require.NoError(t, err)

	// Verify the key exists
	assert.True(t, mr.Exists(jtiDenylistKey("jti-123")))
}

func TestDenyJTI_Error(t *testing.T) {
	c, mr := newTestCache(t)
	ctx := context.Background()
	mr.Close()

	err := c.DenyJTI(ctx, "jti-123", time.Hour)
	require.Error(t, err)
}

func TestIsJTIDenied_Denied(t *testing.T) {
	c, _ := newTestCache(t)
	ctx := context.Background()

	err := c.DenyJTI(ctx, "jti-abc", time.Hour)
	require.NoError(t, err)

	denied, err := c.IsJTIDenied(ctx, "jti-abc")
	require.NoError(t, err)
	assert.True(t, denied)
}

func TestIsJTIDenied_NotDenied(t *testing.T) {
	c, _ := newTestCache(t)
	ctx := context.Background()

	denied, err := c.IsJTIDenied(ctx, "jti-never-set")
	require.NoError(t, err)
	assert.False(t, denied)
}

func TestIsJTIDenied_Error(t *testing.T) {
	c, mr := newTestCache(t)
	ctx := context.Background()
	mr.Close()

	denied, err := c.IsJTIDenied(ctx, "jti-err")
	require.Error(t, err)
	assert.False(t, denied)
}

// ---------------------------------------------------------------------------
// SetSession / GetSession / DeleteSession
// ---------------------------------------------------------------------------

func TestSetSession_Success(t *testing.T) {
	c, mr := newTestCache(t)
	ctx := context.Background()

	err := c.SetSession(ctx, "session-key", map[string]string{"challenge": "abc123"}, time.Hour)
	require.NoError(t, err)
	assert.True(t, mr.Exists("session-key"))
}

func TestSetSession_MarshalError(t *testing.T) {
	c, _ := newTestCache(t)
	ctx := context.Background()

	// Passing an un-marshalable value (channel) causes json.Marshal to fail.
	err := c.SetSession(ctx, "bad-key", make(chan int), time.Hour)
	require.Error(t, err)
}

func TestGetSession_Success(t *testing.T) {
	c, _ := newTestCache(t)
	ctx := context.Background()

	type testData struct {
		Challenge string `json:"challenge"`
	}
	err := c.SetSession(ctx, "get-key", testData{Challenge: "xyz"}, time.Hour)
	require.NoError(t, err)

	var result testData
	err = c.GetSession(ctx, "get-key", &result)
	require.NoError(t, err)
	assert.Equal(t, "xyz", result.Challenge)
}

func TestGetSession_Miss(t *testing.T) {
	c, _ := newTestCache(t)
	ctx := context.Background()

	var result map[string]string
	err := c.GetSession(ctx, "nonexistent-key", &result)
	assert.ErrorIs(t, err, redis.Nil)
}

func TestDeleteSession_Success(t *testing.T) {
	c, mr := newTestCache(t)
	ctx := context.Background()

	err := c.SetSession(ctx, "del-key", "data", time.Hour)
	require.NoError(t, err)
	assert.True(t, mr.Exists("del-key"))

	err = c.DeleteSession(ctx, "del-key")
	require.NoError(t, err)
	assert.False(t, mr.Exists("del-key"))
}

// ---------------------------------------------------------------------------
// NopJTIDenylister
// ---------------------------------------------------------------------------

func TestNopJTIDenylister(t *testing.T) {
	var nop NopJTIDenylister
	ctx := context.Background()

	err := nop.DenyJTI(ctx, "any-jti", time.Hour)
	require.NoError(t, err)

	denied, err := nop.IsJTIDenied(ctx, "any-jti")
	require.NoError(t, err)
	assert.False(t, denied)
}

// ---------------------------------------------------------------------------
// deleteByPattern — SCAN error path
// ---------------------------------------------------------------------------

func TestInvalidateAllUsers_RedisClosed(t *testing.T) {
	c, mr := newTestCache(t)
	ctx := context.Background()

	// Set some data, then close Redis so SCAN fails.
	uc := &UserContext{User: &AuthUser{UserUUID: uuid.New(), Email: "alice@example.com"}}
	c.SetUserContext(ctx, "sub1", "client1", uc)

	mr.Close()
	// Should not panic when Redis is unreachable.
	c.InvalidateAllUsers(ctx)
}
