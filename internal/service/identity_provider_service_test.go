package service

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func newIDP(tenantID int64, name string) *model.IdentityProvider {
	return &model.IdentityProvider{
		IdentityProviderID:   1,
		IdentityProviderUUID: uuid.New(),
		TenantID:             tenantID,
		Name:                 name,
		DisplayName:          name,
		Provider:             "local",
		ProviderType:         "password",
		Status:               model.StatusActive,
		Tenant:               &model.Tenant{TenantID: tenantID},
	}
}

func actorUserWithDefaultTenant(tenantID int64) *model.User {
	return &model.User{
		UserID: 1,
		UserIdentities: []model.UserIdentity{
			{TenantID: tenantID, Tenant: &model.Tenant{TenantID: tenantID, IsDefault: true}},
		},
	}
}

// ---------------------------------------------------------------------------
// IdentityProviderService.Get
// ---------------------------------------------------------------------------

func TestIdentityProviderService_Get(t *testing.T) {
	tenantID := int64(1)

	t.Run("repo error → propagated", func(t *testing.T) {
		idpRepo := &mockIdentityProviderRepo{
			findPaginatedFn: func(_ repository.IdentityProviderRepositoryGetFilter) (*repository.PaginationResult[model.IdentityProvider], error) {
				return nil, errors.New("db error")
			},
		}
		svc := NewIdentityProviderService(nil, idpRepo, &mockTenantRepo{}, &mockUserRepo{})
		_, err := svc.Get(IdentityProviderServiceGetFilter{TenantID: tenantID, Page: 1, Limit: 10})
		require.Error(t, err)
	})

	t.Run("success → returns mapped results", func(t *testing.T) {
		idp := newIDP(tenantID, "local")
		idpRepo := &mockIdentityProviderRepo{
			findPaginatedFn: func(_ repository.IdentityProviderRepositoryGetFilter) (*repository.PaginationResult[model.IdentityProvider], error) {
				return &repository.PaginationResult[model.IdentityProvider]{
					Data: []model.IdentityProvider{*idp}, Total: 1, Page: 1, Limit: 10, TotalPages: 1,
				}, nil
			},
		}
		svc := NewIdentityProviderService(nil, idpRepo, &mockTenantRepo{}, &mockUserRepo{})
		result, err := svc.Get(IdentityProviderServiceGetFilter{TenantID: tenantID, Page: 1, Limit: 10})
		require.NoError(t, err)
		assert.Len(t, result.Data, 1)
		assert.Equal(t, "local", result.Data[0].Name)
	})
}

// ---------------------------------------------------------------------------
// IdentityProviderService.GetByUUID
// ---------------------------------------------------------------------------

func TestIdentityProviderService_GetByUUID(t *testing.T) {
	tenantID := int64(1)
	idpUUID := uuid.New()

	t.Run("idp not found → error", func(t *testing.T) {
		idpRepo := &mockIdentityProviderRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.IdentityProvider, error) { return nil, nil },
		}
		svc := NewIdentityProviderService(nil, idpRepo, &mockTenantRepo{}, &mockUserRepo{})
		_, err := svc.GetByUUID(idpUUID, tenantID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("wrong tenant → access denied", func(t *testing.T) {
		idpRepo := &mockIdentityProviderRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.IdentityProvider, error) {
				return newIDP(999, "other"), nil // different tenant
			},
		}
		svc := NewIdentityProviderService(nil, idpRepo, &mockTenantRepo{}, &mockUserRepo{})
		_, err := svc.GetByUUID(idpUUID, tenantID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "access denied")
	})

	t.Run("found, same tenant → success", func(t *testing.T) {
		idp := newIDP(tenantID, "local")
		idpRepo := &mockIdentityProviderRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.IdentityProvider, error) { return idp, nil },
		}
		svc := NewIdentityProviderService(nil, idpRepo, &mockTenantRepo{}, &mockUserRepo{})
		result, err := svc.GetByUUID(idpUUID, tenantID)
		require.NoError(t, err)
		assert.Equal(t, "local", result.Name)
	})
}

// ---------------------------------------------------------------------------
// IdentityProviderService.DeleteByUUID
// ---------------------------------------------------------------------------

func TestIdentityProviderService_DeleteByUUID(t *testing.T) {
	tenantID := int64(1)
	idpUUID := uuid.New()
	actorUUID := uuid.New()

	t.Run("idp not found → error", func(t *testing.T) {
		idpRepo := &mockIdentityProviderRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.IdentityProvider, error) { return nil, nil },
		}
		svc := NewIdentityProviderService(nil, idpRepo, &mockTenantRepo{}, &mockUserRepo{})
		_, err := svc.DeleteByUUID(idpUUID, tenantID, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("wrong tenant → access denied", func(t *testing.T) {
		idpRepo := &mockIdentityProviderRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.IdentityProvider, error) {
				return newIDP(999, "other"), nil
			},
		}
		svc := NewIdentityProviderService(nil, idpRepo, &mockTenantRepo{}, &mockUserRepo{})
		_, err := svc.DeleteByUUID(idpUUID, tenantID, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "access denied")
	})

	t.Run("actor user not found → error", func(t *testing.T) {
		idpRepo := &mockIdentityProviderRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.IdentityProvider, error) {
				return newIDP(tenantID, "local"), nil
			},
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) { return nil, nil },
		}
		svc := NewIdentityProviderService(nil, idpRepo, &mockTenantRepo{}, userRepo)
		_, err := svc.DeleteByUUID(idpUUID, tenantID, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "actor user not found")
	})

	t.Run("system idp → cannot delete", func(t *testing.T) {
		idp := newIDP(tenantID, "sys")
		idp.IsSystem = true
		idpRepo := &mockIdentityProviderRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.IdentityProvider, error) { return idp, nil },
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return actorUserWithDefaultTenant(tenantID), nil
			},
		}
		svc := NewIdentityProviderService(nil, idpRepo, &mockTenantRepo{}, userRepo)
		_, err := svc.DeleteByUUID(idpUUID, tenantID, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "system idp")
	})

	t.Run("success → deleted", func(t *testing.T) {
		idp := newIDP(tenantID, "local")
		idpRepo := &mockIdentityProviderRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.IdentityProvider, error) { return idp, nil },
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return actorUserWithDefaultTenant(tenantID), nil
			},
		}
		svc := NewIdentityProviderService(nil, idpRepo, &mockTenantRepo{}, userRepo)
		result, err := svc.DeleteByUUID(idpUUID, tenantID, actorUUID)
		require.NoError(t, err)
		assert.Equal(t, "local", result.Name)
	})
}
