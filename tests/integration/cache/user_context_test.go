//go:build integration

package cache_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/platform/cache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegration_UserContext_SetAndGet(t *testing.T) {
	c, _ := newTestCache(t)
	ctx := context.Background()
	sub := uuid.New().String()
	clientID := "test-client"

	uc := &cache.UserContext{
		User: &cache.AuthUser{
			UserID:   42,
			UserUUID: uuid.New(),
			Email:    "alice@example.com",
			Fullname: "Alice Example",
			Roles: []cache.AuthRole{
				{Name: "admin", Permissions: []cache.AuthPermission{{Name: "tenant:read"}}},
			},
		},
		Tenant: &cache.AuthTenant{
			TenantID:   1,
			TenantUUID: uuid.New(),
		},
		Provider: &cache.AuthProvider{
			IdentityProviderID:   1,
			IdentityProviderUUID: uuid.New(),
		},
		Client: &cache.AuthClient{
			ClientID:   1,
			ClientUUID: uuid.New(),
		},
	}

	c.SetUserContext(ctx, sub, clientID, uc)
	got := c.GetUserContext(ctx, sub, clientID)

	require.NotNil(t, got)
	assert.Equal(t, "alice@example.com", got.User.Email)
	assert.Equal(t, "Alice Example", got.User.Fullname)
	assert.Len(t, got.User.Roles, 1)
	assert.Equal(t, "admin", got.User.Roles[0].Name)
}

func TestIntegration_UserContext_CacheMiss(t *testing.T) {
	c, _ := newTestCache(t)
	ctx := context.Background()

	got := c.GetUserContext(ctx, "nonexistent", "client1")
	assert.Nil(t, got)
}

func TestIntegration_UserContext_Invalidation(t *testing.T) {
	c, _ := newTestCache(t)
	ctx := context.Background()

	uc := &cache.UserContext{
		User: &cache.AuthUser{UserID: 1, UserUUID: uuid.New(), Email: "bob@example.com"},
	}

	c.SetUserContext(ctx, "sub1", "client1", uc)
	c.SetUserContext(ctx, "sub1", "client2", uc)
	c.SetUserContext(ctx, "sub2", "client1", uc)

	c.InvalidateUser(ctx, "sub1", "client1")

	assert.Nil(t, c.GetUserContext(ctx, "sub1", "client1"))
	assert.NotNil(t, c.GetUserContext(ctx, "sub1", "client2"))
	assert.NotNil(t, c.GetUserContext(ctx, "sub2", "client1"))

	c.InvalidateUserAll(ctx, "sub1")
	assert.Nil(t, c.GetUserContext(ctx, "sub1", "client2"))
	assert.NotNil(t, c.GetUserContext(ctx, "sub2", "client1"))
}

func TestIntegration_UserContext_TTL(t *testing.T) {
	c, mr := newTestCache(t)
	ctx := context.Background()

	uc := &cache.UserContext{
		User: &cache.AuthUser{UserID: 1, UserUUID: uuid.New(), Email: "ttl@example.com"},
	}

	c.SetUserContext(ctx, "sub1", "client1", uc)

	ttl := mr.TTL(cache.FormatKey("user", "sub1", "client1"))
	assert.Greater(t, ttl, 9*time.Minute)
}

func TestIntegration_UserContext_SerializationRoundTrip(t *testing.T) {
	c, _ := newTestCache(t)
	ctx := context.Background()

	profileURL := "https://cdn.example.com/avatar.jpg"
	dn := "Display Name"
	ln := "Doe"
	uc := &cache.UserContext{
		User: &cache.AuthUser{
			UserID:          1,
			UserUUID:        uuid.New(),
			Email:           "pic@example.com",
			IsEmailVerified: true,
			Phone:           "+1234567890",
			IsPhoneVerified: false,
			Fullname:        "Jane Doe",
			Profile: &cache.AuthProfile{
				DisplayName: &dn,
				FirstName:   "Jane",
				LastName:    &ln,
				ProfileURL:  &profileURL,
			},
		},
	}

	c.SetUserContext(ctx, "sub1", "client1", uc)
	got := c.GetUserContext(ctx, "sub1", "client1")

	require.NotNil(t, got)
	assert.Equal(t, "pic@example.com", got.User.Email)
	assert.True(t, got.User.IsEmailVerified)
	assert.Equal(t, "+1234567890", got.User.Phone)
	assert.False(t, got.User.IsPhoneVerified)
	assert.Equal(t, "Display Name", *got.User.Profile.DisplayName)
	assert.Equal(t, "Jane", got.User.Profile.FirstName)
	assert.Equal(t, "Doe", *got.User.Profile.LastName)
	assert.Equal(t, profileURL, *got.User.Profile.ProfileURL)
}

func TestIntegration_UserContext_JSONSerialization(t *testing.T) {
	uc := &cache.UserContext{
		User: &cache.AuthUser{
			UserUUID: uuid.New(),
			Email:    "json@example.com",
		},
	}

	data, err := json.Marshal(uc)
	require.NoError(t, err)

	var decoded cache.UserContext
	require.NoError(t, json.Unmarshal(data, &decoded))
	assert.Equal(t, "json@example.com", decoded.User.Email)
}
