//go:build integration

package cache_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/platform/cache"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegration_JTIDenyList_DenyAndCheck(t *testing.T) {
	c, mr := newTestCache(t)
	ctx := context.Background()

	err := c.DenyJTI(ctx, "jti-abc-123", time.Hour)
	require.NoError(t, err)

	assert.True(t, mr.Exists(cache.FormatKey("jti:deny", "jti-abc-123")))

	denied, err := c.IsJTIDenied(ctx, "jti-abc-123")
	require.NoError(t, err)
	assert.True(t, denied)
}

func TestIntegration_JTIDenyList_NotDenied(t *testing.T) {
	c, _ := newTestCache(t)
	ctx := context.Background()

	denied, err := c.IsJTIDenied(ctx, "jti-never-seen")
	require.NoError(t, err)
	assert.False(t, denied)
}

func TestIntegration_JTIDenyList_TTL(t *testing.T) {
	c, mr := newTestCache(t)
	ctx := context.Background()

	ttl := 5 * time.Minute
	err := c.DenyJTI(ctx, "jti-ttl-test", ttl)
	require.NoError(t, err)

	actualTTL := mr.TTL(cache.FormatKey("jti:deny", "jti-ttl-test"))
	assert.Greater(t, actualTTL, 4*time.Minute)
	assert.Less(t, actualTTL, time.Hour)
}

func TestIntegration_JTIDenyList_MultipleJTIs(t *testing.T) {
	c, _ := newTestCache(t)
	ctx := context.Background()

	jtis := []string{"jti-1", "jti-2", "jti-3"}
	for _, jti := range jtis {
		require.NoError(t, c.DenyJTI(ctx, jti, time.Hour))
	}

	for _, jti := range jtis {
		denied, err := c.IsJTIDenied(ctx, jti)
		require.NoError(t, err)
		assert.True(t, denied, "JTI %s should be denied", jti)
	}

	denied, err := c.IsJTIDenied(ctx, "jti-not-in-list")
	require.NoError(t, err)
	assert.False(t, denied)
}

func TestIntegration_Session_SetGetDelete(t *testing.T) {
	c, mr := newTestCache(t)
	ctx := context.Background()

	type sessionData struct {
		Challenge string `json:"challenge"`
		UserID    int64  `json:"user_id"`
	}

	data := sessionData{Challenge: "abc123", UserID: 42}
	err := c.SetSession(ctx, "session:webauthn:test", data, 10*time.Minute)
	require.NoError(t, err)

	assert.True(t, mr.Exists("session:webauthn:test"))

	var retrieved sessionData
	err = c.GetSession(ctx, "session:webauthn:test", &retrieved)
	require.NoError(t, err)
	assert.Equal(t, "abc123", retrieved.Challenge)
	assert.Equal(t, int64(42), retrieved.UserID)

	err = c.DeleteSession(ctx, "session:webauthn:test")
	require.NoError(t, err)
	assert.False(t, mr.Exists("session:webauthn:test"))

	err = c.GetSession(ctx, "session:webauthn:test", &retrieved)
	assert.ErrorIs(t, err, redis.Nil)
}

func TestIntegration_DeleteByPattern(t *testing.T) {
	c, mr := newTestCache(t)
	ctx := context.Background()

	uc := &cache.UserContext{
		User: &cache.AuthUser{UserUUID: uuid.New(), Email: "cleanup@example.com"},
	}

	for i := 0; i < 5; i++ {
		sub := fmt.Sprintf("sub-%d", i)
		c.SetUserContext(ctx, sub, "client1", uc)
	}

	assert.True(t, mr.Exists("user:sub-0:client1"))

	c.InvalidateAllUsers(ctx)

	for i := 0; i < 5; i++ {
		sub := fmt.Sprintf("sub-%d", i)
		assert.Nil(t, c.GetUserContext(ctx, sub, "client1"))
	}
}
