package client

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func int64Ptr(v int64) *int64         { return &v }
func timePtr(tp time.Time) *time.Time { return &tp }

func TestNewAPIKeyAuthenticator(t *testing.T) {
	a := NewAPIKeyAuthenticator(&mockAPIKeyRepo{}, &mockAPIKeyAPIRepo{})
	require.NotNil(t, a)
	assert.NotNil(t, a.apiKeyRepo)
	assert.NotNil(t, a.apiKeyAPIRepo)
}

func TestAuthenticateAPIKey(t *testing.T) {
	apiKeyUUID := uuid.New()
	testKey := "test-api-key-raw-value"

	activeKey := &APIKey{
		APIKeyUUID: apiKeyUUID,
		TenantID:   1,
		Status:     shared.StatusActive,
		CreatedBy:  int64Ptr(42),
	}

	expiredKey := &APIKey{
		APIKeyUUID: apiKeyUUID,
		TenantID:   1,
		Status:     shared.StatusActive,
		ExpiresAt:  timePtr(time.Now().Add(-1 * time.Hour)),
	}

	t.Run("empty key", func(t *testing.T) {
		a := NewAPIKeyAuthenticator(&mockAPIKeyRepo{}, &mockAPIKeyAPIRepo{})
		_, err := a.AuthenticateAPIKey(context.Background(), "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "API key is required")
	})

	t.Run("whitespace key", func(t *testing.T) {
		a := NewAPIKeyAuthenticator(&mockAPIKeyRepo{}, &mockAPIKeyAPIRepo{})
		_, err := a.AuthenticateAPIKey(context.Background(), "   ")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "API key is required")
	})

	t.Run("FindByKeyHash error", func(t *testing.T) {
		repo := &mockAPIKeyRepo{
			findByKeyHashFn: func(string) (*APIKey, error) { return nil, errors.New("db error") },
		}
		a := NewAPIKeyAuthenticator(repo, &mockAPIKeyAPIRepo{})
		_, err := a.AuthenticateAPIKey(context.Background(), testKey)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "db error")
	})

	t.Run("apiKey nil", func(t *testing.T) {
		repo := &mockAPIKeyRepo{
			findByKeyHashFn: func(string) (*APIKey, error) { return nil, nil },
		}
		a := NewAPIKeyAuthenticator(repo, &mockAPIKeyAPIRepo{})
		_, err := a.AuthenticateAPIKey(context.Background(), testKey)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid API key")
	})

	t.Run("apiKey inactive", func(t *testing.T) {
		inactiveKey := &APIKey{
			APIKeyUUID: apiKeyUUID,
			TenantID:   1,
			Status:     shared.StatusInactive,
		}
		repo := &mockAPIKeyRepo{
			findByKeyHashFn: func(string) (*APIKey, error) { return inactiveKey, nil },
		}
		a := NewAPIKeyAuthenticator(repo, &mockAPIKeyAPIRepo{})
		_, err := a.AuthenticateAPIKey(context.Background(), testKey)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid API key")
	})

	t.Run("apiKey expired", func(t *testing.T) {
		repo := &mockAPIKeyRepo{
			findByKeyHashFn: func(string) (*APIKey, error) { return expiredKey, nil },
		}
		a := NewAPIKeyAuthenticator(repo, &mockAPIKeyAPIRepo{})
		_, err := a.AuthenticateAPIKey(context.Background(), testKey)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "API key has expired")
	})

	t.Run("FindByAPIKeyUUID error", func(t *testing.T) {
		repo := &mockAPIKeyRepo{
			findByKeyHashFn: func(string) (*APIKey, error) { return activeKey, nil },
		}
		apiAPIRepo := &mockAPIKeyAPIRepo{
			findByAPIKeyUUIDFn: func(uuid.UUID) ([]APIKeyAPI, error) {
				return nil, errors.New("api repo error")
			},
		}
		a := NewAPIKeyAuthenticator(repo, apiAPIRepo)
		_, err := a.AuthenticateAPIKey(context.Background(), testKey)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "api repo error")
	})

	t.Run("success with permissions", func(t *testing.T) {
		repo := &mockAPIKeyRepo{
			findByKeyHashFn: func(string) (*APIKey, error) { return activeKey, nil },
		}
		perm := &Permission{PermissionID: 1, PermissionUUID: uuid.New(), Name: "read"}
		apiAPIRepo := &mockAPIKeyAPIRepo{
			findByAPIKeyUUIDFn: func(uuid.UUID) ([]APIKeyAPI, error) {
				return []APIKeyAPI{{
					APIKeyAPIUUID: uuid.New(),
					Permissions: []APIKeyPermission{{
						Permission: perm,
					}},
				}}, nil
			},
		}
		a := NewAPIKeyAuthenticator(repo, apiAPIRepo)
		ctx, err := a.AuthenticateAPIKey(context.Background(), testKey)
		require.NoError(t, err)
		assert.NotNil(t, ctx)
		assert.NotNil(t, ctx.User)
		assert.Equal(t, int64(42), ctx.User.UserID)
		assert.NotNil(t, ctx.Tenant)
		assert.Equal(t, int64(1), ctx.Tenant.TenantID)
		require.Len(t, ctx.User.Roles, 1)
		assert.Equal(t, "api-key", ctx.User.Roles[0].Name)
		require.Len(t, ctx.User.Roles[0].Permissions, 1)
		assert.Equal(t, "read", ctx.User.Roles[0].Permissions[0].Name)
	})

	t.Run("success with nil CreatedBy", func(t *testing.T) {
		key := &APIKey{
			APIKeyUUID: apiKeyUUID,
			TenantID:   1,
			Status:     shared.StatusActive,
			CreatedBy:  nil,
		}
		repo := &mockAPIKeyRepo{
			findByKeyHashFn: func(string) (*APIKey, error) { return key, nil },
		}
		apiAPIRepo := &mockAPIKeyAPIRepo{
			findByAPIKeyUUIDFn: func(uuid.UUID) ([]APIKeyAPI, error) { return nil, nil },
		}
		a := NewAPIKeyAuthenticator(repo, apiAPIRepo)
		ctx, err := a.AuthenticateAPIKey(context.Background(), testKey)
		require.NoError(t, err)
		assert.Equal(t, int64(0), ctx.User.UserID)
	})

	t.Run("success with duplicated permission", func(t *testing.T) {
		repo := &mockAPIKeyRepo{
			findByKeyHashFn: func(string) (*APIKey, error) { return activeKey, nil },
		}
		perm := &Permission{PermissionID: 1, PermissionUUID: uuid.New(), Name: "read"}
		apiAPIRepo := &mockAPIKeyAPIRepo{
			findByAPIKeyUUIDFn: func(uuid.UUID) ([]APIKeyAPI, error) {
				return []APIKeyAPI{
					{
						APIKeyAPIUUID: uuid.New(),
						Permissions: []APIKeyPermission{{
							Permission: perm,
						}},
					},
					{
						APIKeyAPIUUID: uuid.New(),
						Permissions: []APIKeyPermission{{
							Permission: perm,
						}},
					},
				}, nil
			},
		}
		a := NewAPIKeyAuthenticator(repo, apiAPIRepo)
		ctx, err := a.AuthenticateAPIKey(context.Background(), testKey)
		require.NoError(t, err)
		require.Len(t, ctx.User.Roles[0].Permissions, 1)
	})

	t.Run("success with nil permission pointer", func(t *testing.T) {
		repo := &mockAPIKeyRepo{
			findByKeyHashFn: func(string) (*APIKey, error) { return activeKey, nil },
		}
		apiAPIRepo := &mockAPIKeyAPIRepo{
			findByAPIKeyUUIDFn: func(uuid.UUID) ([]APIKeyAPI, error) {
				return []APIKeyAPI{{
					APIKeyAPIUUID: uuid.New(),
					Permissions: []APIKeyPermission{{
						Permission: nil,
					}},
				}}, nil
			},
		}
		a := NewAPIKeyAuthenticator(repo, apiAPIRepo)
		ctx, err := a.AuthenticateAPIKey(context.Background(), testKey)
		require.NoError(t, err)
		assert.Len(t, ctx.User.Roles[0].Permissions, 0)
	})
}
