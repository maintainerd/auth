package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/crypto"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
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
		_, err := svc.Get(context.Background(), IdentityProviderServiceGetFilter{TenantID: tenantID, Page: 1, Limit: 10})
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
		result, err := svc.Get(context.Background(), IdentityProviderServiceGetFilter{TenantID: tenantID, Page: 1, Limit: 10})
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
		_, err := svc.GetByUUID(context.Background(), idpUUID, tenantID)
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
		_, err := svc.GetByUUID(context.Background(), idpUUID, tenantID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "access denied")
	})

	t.Run("found, same tenant → success", func(t *testing.T) {
		idp := newIDP(tenantID, "local")
		idpRepo := &mockIdentityProviderRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.IdentityProvider, error) { return idp, nil },
		}
		svc := NewIdentityProviderService(nil, idpRepo, &mockTenantRepo{}, &mockUserRepo{})
		result, err := svc.GetByUUID(context.Background(), idpUUID, tenantID)
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
		_, err := svc.DeleteByUUID(context.Background(), idpUUID, tenantID, actorUUID)
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
		_, err := svc.DeleteByUUID(context.Background(), idpUUID, tenantID, actorUUID)
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
		_, err := svc.DeleteByUUID(context.Background(), idpUUID, tenantID, actorUUID)
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
		_, err := svc.DeleteByUUID(context.Background(), idpUUID, tenantID, actorUUID)
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
		result, err := svc.DeleteByUUID(context.Background(), idpUUID, tenantID, actorUUID)
		require.NoError(t, err)
		assert.Equal(t, "local", result.Name)
	})

	t.Run("default idp → cannot delete", func(t *testing.T) {
		idp := newIDP(tenantID, "default-idp")
		idp.IsDefault = true
		idpRepo := &mockIdentityProviderRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.IdentityProvider, error) { return idp, nil },
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return actorUserWithDefaultTenant(tenantID), nil
			},
		}
		svc := NewIdentityProviderService(nil, idpRepo, &mockTenantRepo{}, userRepo)
		_, err := svc.DeleteByUUID(context.Background(), idpUUID, tenantID, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "default idp")
	})

	t.Run("delete repo error", func(t *testing.T) {
		idp := newIDP(tenantID, "local")
		idpRepo := &mockIdentityProviderRepo{
			findByUUIDFn:   func(_ any, _ ...string) (*model.IdentityProvider, error) { return idp, nil },
			deleteByUUIDFn: func(_ any) error { return errors.New("del err") },
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return actorUserWithDefaultTenant(tenantID), nil
			},
		}
		svc := NewIdentityProviderService(nil, idpRepo, &mockTenantRepo{}, userRepo)
		_, err := svc.DeleteByUUID(context.Background(), idpUUID, tenantID, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "del err")
	})

	t.Run("ValidateTenantAccess error", func(t *testing.T) {
		idp := newIDP(tenantID, "local")
		idpRepo := &mockIdentityProviderRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.IdentityProvider, error) { return idp, nil },
		}
		// Non-default tenant user trying to access a different tenant
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return &model.User{
					UserID: 1,
					UserIdentities: []model.UserIdentity{
						{TenantID: 999, Tenant: &model.Tenant{TenantID: 999, IsDefault: false}},
					},
				}, nil
			},
		}
		svc := NewIdentityProviderService(nil, idpRepo, &mockTenantRepo{}, userRepo)
		_, err := svc.DeleteByUUID(context.Background(), idpUUID, tenantID, actorUUID)
		require.Error(t, err)
	})
}

// ---------------------------------------------------------------------------
// IdentityProviderService.Create
// ---------------------------------------------------------------------------

func TestIdentityProviderService_Create(t *testing.T) {
	tenantID := int64(1)
	actorUUID := uuid.New()
	tenantUUID := uuid.New()
	cfg := datatypes.JSON([]byte(`{}`))

	t.Run("invalid tenant UUID", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewIdentityProviderService(gormDB, &mockIdentityProviderRepo{}, &mockTenantRepo{}, &mockUserRepo{})
		_, err := svc.Create(context.Background(), "idp", "IDP", "local", "password", cfg, "active", "invalid-uuid", tenantID, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid tenant UUID")
	})

	t.Run("tenant not found", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		tenantRepo := &mockTenantRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Tenant, error) { return nil, nil },
		}
		svc := NewIdentityProviderService(gormDB, &mockIdentityProviderRepo{}, tenantRepo, &mockUserRepo{})
		_, err := svc.Create(context.Background(), "idp", "IDP", "local", "password", cfg, "active", tenantUUID.String(), tenantID, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "tenant not found")
	})

	t.Run("tenant find error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		tenantRepo := &mockTenantRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Tenant, error) { return nil, errors.New("db err") },
		}
		svc := NewIdentityProviderService(gormDB, &mockIdentityProviderRepo{}, tenantRepo, &mockUserRepo{})
		_, err := svc.Create(context.Background(), "idp", "IDP", "local", "password", cfg, "active", tenantUUID.String(), tenantID, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "tenant not found")
	})

	t.Run("tenant ownership mismatch", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		tenantRepo := &mockTenantRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Tenant, error) {
				return &model.Tenant{TenantID: 999}, nil // different from tenantID=1
			},
		}
		svc := NewIdentityProviderService(gormDB, &mockIdentityProviderRepo{}, tenantRepo, &mockUserRepo{})
		_, err := svc.Create(context.Background(), "idp", "IDP", "local", "password", cfg, "active", tenantUUID.String(), tenantID, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "access denied")
	})

	t.Run("actor user not found", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		tenantRepo := &mockTenantRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Tenant, error) {
				return &model.Tenant{TenantID: tenantID, IsDefault: true}, nil
			},
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) { return nil, nil },
		}
		svc := NewIdentityProviderService(gormDB, &mockIdentityProviderRepo{}, tenantRepo, userRepo)
		_, err := svc.Create(context.Background(), "idp", "IDP", "local", "password", cfg, "active", tenantUUID.String(), tenantID, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "actor user not found")
	})

	t.Run("ValidateTenantAccess error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		tenantRepo := &mockTenantRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Tenant, error) {
				return &model.Tenant{TenantID: tenantID, IsDefault: false}, nil
			},
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return &model.User{
					UserID: 1,
					UserIdentities: []model.UserIdentity{
						{TenantID: 999, Tenant: &model.Tenant{TenantID: 999, IsDefault: false}},
					},
				}, nil
			},
		}
		svc := NewIdentityProviderService(gormDB, &mockIdentityProviderRepo{}, tenantRepo, userRepo)
		_, err := svc.Create(context.Background(), "idp", "IDP", "local", "password", cfg, "active", tenantUUID.String(), tenantID, actorUUID)
		require.Error(t, err)
	})

	t.Run("findByName error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		tenantRepo := &mockTenantRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Tenant, error) {
				return &model.Tenant{TenantID: tenantID, IsDefault: true}, nil
			},
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return actorUserWithDefaultTenant(tenantID), nil
			},
		}
		idpRepo := &mockIdentityProviderRepo{
			findByNameFn: func(_ string, _ int64) (*model.IdentityProvider, error) {
				return nil, errors.New("db err")
			},
		}
		svc := NewIdentityProviderService(gormDB, idpRepo, tenantRepo, userRepo)
		_, err := svc.Create(context.Background(), "idp", "IDP", "local", "password", cfg, "active", tenantUUID.String(), tenantID, actorUUID)
		require.Error(t, err)
	})

	t.Run("idp already exists", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		tenantRepo := &mockTenantRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Tenant, error) {
				return &model.Tenant{TenantID: tenantID, IsDefault: true}, nil
			},
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return actorUserWithDefaultTenant(tenantID), nil
			},
		}
		idpRepo := &mockIdentityProviderRepo{
			findByNameFn: func(_ string, _ int64) (*model.IdentityProvider, error) {
				return &model.IdentityProvider{Name: "idp"}, nil
			},
		}
		svc := NewIdentityProviderService(gormDB, idpRepo, tenantRepo, userRepo)
		_, err := svc.Create(context.Background(), "idp", "IDP", "local", "password", cfg, "active", tenantUUID.String(), tenantID, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
	})

	t.Run("GenerateIdentifier failure", func(t *testing.T) {
		orig := crypto.GenerateIdentifier
		defer func() { crypto.GenerateIdentifier = orig }()
		crypto.GenerateIdentifier = func(int) (string, error) { return "", errors.New("rand failure") }

		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		tenantRepo := &mockTenantRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Tenant, error) {
				return &model.Tenant{TenantID: tenantID, IsDefault: true}, nil
			},
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return actorUserWithDefaultTenant(tenantID), nil
			},
		}
		idpRepo := &mockIdentityProviderRepo{}
		svc := NewIdentityProviderService(gormDB, idpRepo, tenantRepo, userRepo)
		_, err := svc.Create(context.Background(), "idp", "IDP", "local", "password", cfg, "active", tenantUUID.String(), tenantID, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "rand failure")
	})

	t.Run("createOrUpdate error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		tenantRepo := &mockTenantRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Tenant, error) {
				return &model.Tenant{TenantID: tenantID, IsDefault: true}, nil
			},
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return actorUserWithDefaultTenant(tenantID), nil
			},
		}
		idpRepo := &mockIdentityProviderRepo{
			createOrUpdateFn: func(_ *model.IdentityProvider) (*model.IdentityProvider, error) {
				return nil, errors.New("create err")
			},
		}
		svc := NewIdentityProviderService(gormDB, idpRepo, tenantRepo, userRepo)
		_, err := svc.Create(context.Background(), "idp", "IDP", "local", "password", cfg, "active", tenantUUID.String(), tenantID, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "create err")
	})

	t.Run("findByUUID after create error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		tenantRepo := &mockTenantRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Tenant, error) {
				return &model.Tenant{TenantID: tenantID, IsDefault: true}, nil
			},
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return actorUserWithDefaultTenant(tenantID), nil
			},
		}
		idpRepo := &mockIdentityProviderRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.IdentityProvider, error) {
				return nil, errors.New("fetch err")
			},
		}
		svc := NewIdentityProviderService(gormDB, idpRepo, tenantRepo, userRepo)
		_, err := svc.Create(context.Background(), "idp", "IDP", "local", "password", cfg, "active", tenantUUID.String(), tenantID, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "fetch err")
	})

	t.Run("success", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		tenant := &model.Tenant{TenantID: tenantID, TenantUUID: tenantUUID, IsDefault: true}
		tenantRepo := &mockTenantRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Tenant, error) { return tenant, nil },
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return actorUserWithDefaultTenant(tenantID), nil
			},
		}
		idpRepo := &mockIdentityProviderRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.IdentityProvider, error) {
				return &model.IdentityProvider{
					Name: "idp", DisplayName: "IDP", TenantID: tenantID,
					Tenant: tenant,
				}, nil
			},
		}
		svc := NewIdentityProviderService(gormDB, idpRepo, tenantRepo, userRepo)
		res, err := svc.Create(context.Background(), "idp", "IDP", "local", "password", cfg, "active", tenantUUID.String(), tenantID, actorUUID)
		require.NoError(t, err)
		assert.Equal(t, "idp", res.Name)
		assert.NotNil(t, res.Tenant)
	})
}

// ---------------------------------------------------------------------------
// IdentityProviderService.Update
// ---------------------------------------------------------------------------

func TestIdentityProviderService_Update(t *testing.T) {
	tenantID := int64(1)
	idpUUID := uuid.New()
	actorUUID := uuid.New()
	cfg := datatypes.JSON([]byte(`{}`))

	t.Run("idp not found", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		idpRepo := &mockIdentityProviderRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.IdentityProvider, error) { return nil, nil },
		}
		svc := NewIdentityProviderService(gormDB, idpRepo, &mockTenantRepo{}, &mockUserRepo{})
		_, err := svc.Update(context.Background(), idpUUID, "n", "d", "local", "password", cfg, "active", tenantID, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("wrong tenant", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		idpRepo := &mockIdentityProviderRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.IdentityProvider, error) {
				return newIDP(999, "other"), nil
			},
		}
		svc := NewIdentityProviderService(gormDB, idpRepo, &mockTenantRepo{}, &mockUserRepo{})
		_, err := svc.Update(context.Background(), idpUUID, "n", "d", "local", "password", cfg, "active", tenantID, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "access denied")
	})

	t.Run("actor user not found", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		idpRepo := &mockIdentityProviderRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.IdentityProvider, error) {
				return newIDP(tenantID, "local"), nil
			},
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) { return nil, nil },
		}
		svc := NewIdentityProviderService(gormDB, idpRepo, &mockTenantRepo{}, userRepo)
		_, err := svc.Update(context.Background(), idpUUID, "n", "d", "local", "password", cfg, "active", tenantID, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "actor user not found")
	})

	t.Run("ValidateTenantAccess error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		idpRepo := &mockIdentityProviderRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.IdentityProvider, error) {
				return newIDP(tenantID, "local"), nil
			},
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return &model.User{
					UserID: 1,
					UserIdentities: []model.UserIdentity{
						{TenantID: 999, Tenant: &model.Tenant{TenantID: 999, IsDefault: false}},
					},
				}, nil
			},
		}
		svc := NewIdentityProviderService(gormDB, idpRepo, &mockTenantRepo{}, userRepo)
		_, err := svc.Update(context.Background(), idpUUID, "n", "d", "local", "password", cfg, "active", tenantID, actorUUID)
		require.Error(t, err)
	})

	t.Run("system idp blocked", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
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
		svc := NewIdentityProviderService(gormDB, idpRepo, &mockTenantRepo{}, userRepo)
		_, err := svc.Update(context.Background(), idpUUID, "n", "d", "local", "password", cfg, "active", tenantID, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "system idp")
	})

	t.Run("default idp blocked", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		idp := newIDP(tenantID, "def")
		idp.IsDefault = true
		idpRepo := &mockIdentityProviderRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.IdentityProvider, error) { return idp, nil },
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return actorUserWithDefaultTenant(tenantID), nil
			},
		}
		svc := NewIdentityProviderService(gormDB, idpRepo, &mockTenantRepo{}, userRepo)
		_, err := svc.Update(context.Background(), idpUUID, "n", "d", "local", "password", cfg, "active", tenantID, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "default idp")
	})

	t.Run("duplicate name error from findByName", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		idp := newIDP(tenantID, "old-name")
		idpRepo := &mockIdentityProviderRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.IdentityProvider, error) { return idp, nil },
			findByNameFn: func(_ string, _ int64) (*model.IdentityProvider, error) {
				return nil, errors.New("db err")
			},
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return actorUserWithDefaultTenant(tenantID), nil
			},
		}
		svc := NewIdentityProviderService(gormDB, idpRepo, &mockTenantRepo{}, userRepo)
		_, err := svc.Update(context.Background(), idpUUID, "new-name", "d", "local", "password", cfg, "active", tenantID, actorUUID)
		require.Error(t, err)
	})

	t.Run("duplicate name exists", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		idp := newIDP(tenantID, "old-name")
		otherUUID := uuid.New()
		idpRepo := &mockIdentityProviderRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.IdentityProvider, error) { return idp, nil },
			findByNameFn: func(_ string, _ int64) (*model.IdentityProvider, error) {
				return &model.IdentityProvider{IdentityProviderUUID: otherUUID}, nil
			},
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return actorUserWithDefaultTenant(tenantID), nil
			},
		}
		svc := NewIdentityProviderService(gormDB, idpRepo, &mockTenantRepo{}, userRepo)
		_, err := svc.Update(context.Background(), idpUUID, "new-name", "d", "local", "password", cfg, "active", tenantID, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
	})

	t.Run("createOrUpdate error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		idp := newIDP(tenantID, "local")
		idpRepo := &mockIdentityProviderRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.IdentityProvider, error) { return idp, nil },
			createOrUpdateFn: func(_ *model.IdentityProvider) (*model.IdentityProvider, error) {
				return nil, errors.New("save err")
			},
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return actorUserWithDefaultTenant(tenantID), nil
			},
		}
		svc := NewIdentityProviderService(gormDB, idpRepo, &mockTenantRepo{}, userRepo)
		_, err := svc.Update(context.Background(), idpUUID, "local", "d", "local", "password", cfg, "active", tenantID, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "save err")
	})

	t.Run("success same name", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		idp := newIDP(tenantID, "local")
		idpRepo := &mockIdentityProviderRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.IdentityProvider, error) { return idp, nil },
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return actorUserWithDefaultTenant(tenantID), nil
			},
		}
		svc := NewIdentityProviderService(gormDB, idpRepo, &mockTenantRepo{}, userRepo)
		res, err := svc.Update(context.Background(), idpUUID, "local", "New Display", "local", "password", cfg, "active", tenantID, actorUUID)
		require.NoError(t, err)
		assert.Equal(t, "New Display", res.DisplayName)
	})

	t.Run("success different name no conflict", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		idp := newIDP(tenantID, "old-name")
		idpRepo := &mockIdentityProviderRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.IdentityProvider, error) { return idp, nil },
			findByNameFn: func(_ string, _ int64) (*model.IdentityProvider, error) { return nil, nil },
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return actorUserWithDefaultTenant(tenantID), nil
			},
		}
		svc := NewIdentityProviderService(gormDB, idpRepo, &mockTenantRepo{}, userRepo)
		res, err := svc.Update(context.Background(), idpUUID, "new-name", "d", "local", "password", cfg, "active", tenantID, actorUUID)
		require.NoError(t, err)
		assert.Equal(t, "new-name", res.Name)
	})
}

// ---------------------------------------------------------------------------
// IdentityProviderService.SetStatusByUUID
// ---------------------------------------------------------------------------

func TestIdentityProviderService_SetStatusByUUID(t *testing.T) {
	tenantID := int64(1)
	idpUUID := uuid.New()
	actorUUID := uuid.New()

	t.Run("idp not found", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		idpRepo := &mockIdentityProviderRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.IdentityProvider, error) { return nil, nil },
		}
		svc := NewIdentityProviderService(gormDB, idpRepo, &mockTenantRepo{}, &mockUserRepo{})
		_, err := svc.SetStatusByUUID(context.Background(), idpUUID, "active", tenantID, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("wrong tenant", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		idpRepo := &mockIdentityProviderRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.IdentityProvider, error) {
				return newIDP(999, "other"), nil
			},
		}
		svc := NewIdentityProviderService(gormDB, idpRepo, &mockTenantRepo{}, &mockUserRepo{})
		_, err := svc.SetStatusByUUID(context.Background(), idpUUID, "active", tenantID, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "access denied")
	})

	t.Run("actor user not found", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		idpRepo := &mockIdentityProviderRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.IdentityProvider, error) {
				return newIDP(tenantID, "local"), nil
			},
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) { return nil, nil },
		}
		svc := NewIdentityProviderService(gormDB, idpRepo, &mockTenantRepo{}, userRepo)
		_, err := svc.SetStatusByUUID(context.Background(), idpUUID, "active", tenantID, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "actor user not found")
	})

	t.Run("ValidateTenantAccess error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		idpRepo := &mockIdentityProviderRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.IdentityProvider, error) {
				return newIDP(tenantID, "local"), nil
			},
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return &model.User{
					UserID: 1,
					UserIdentities: []model.UserIdentity{
						{TenantID: 999, Tenant: &model.Tenant{TenantID: 999, IsDefault: false}},
					},
				}, nil
			},
		}
		svc := NewIdentityProviderService(gormDB, idpRepo, &mockTenantRepo{}, userRepo)
		_, err := svc.SetStatusByUUID(context.Background(), idpUUID, "active", tenantID, actorUUID)
		require.Error(t, err)
	})

	t.Run("system idp blocked", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
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
		svc := NewIdentityProviderService(gormDB, idpRepo, &mockTenantRepo{}, userRepo)
		_, err := svc.SetStatusByUUID(context.Background(), idpUUID, "active", tenantID, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "system idp")
	})

	t.Run("default idp blocked", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		idp := newIDP(tenantID, "def")
		idp.IsDefault = true
		idpRepo := &mockIdentityProviderRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.IdentityProvider, error) { return idp, nil },
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return actorUserWithDefaultTenant(tenantID), nil
			},
		}
		svc := NewIdentityProviderService(gormDB, idpRepo, &mockTenantRepo{}, userRepo)
		_, err := svc.SetStatusByUUID(context.Background(), idpUUID, "active", tenantID, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "default idp")
	})

	t.Run("createOrUpdate error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		idp := newIDP(tenantID, "local")
		idpRepo := &mockIdentityProviderRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.IdentityProvider, error) { return idp, nil },
			createOrUpdateFn: func(_ *model.IdentityProvider) (*model.IdentityProvider, error) {
				return nil, errors.New("save err")
			},
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return actorUserWithDefaultTenant(tenantID), nil
			},
		}
		svc := NewIdentityProviderService(gormDB, idpRepo, &mockTenantRepo{}, userRepo)
		_, err := svc.SetStatusByUUID(context.Background(), idpUUID, "active", tenantID, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "save err")
	})

	t.Run("success", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		idp := newIDP(tenantID, "local")
		idpRepo := &mockIdentityProviderRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.IdentityProvider, error) { return idp, nil },
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return actorUserWithDefaultTenant(tenantID), nil
			},
		}
		svc := NewIdentityProviderService(gormDB, idpRepo, &mockTenantRepo{}, userRepo)
		res, err := svc.SetStatusByUUID(context.Background(), idpUUID, "inactive", tenantID, actorUUID)
		require.NoError(t, err)
		assert.Equal(t, "local", res.Name)
	})
}

// ---------------------------------------------------------------------------
// toIdpServiceDataResult
// ---------------------------------------------------------------------------

func TestToIdpServiceDataResult(t *testing.T) {
	t.Run("nil input", func(t *testing.T) {
		assert.Nil(t, toIdpServiceDataResult(nil))
	})

	t.Run("with tenant", func(t *testing.T) {
		idp := &model.IdentityProvider{
			Name:   "idp",
			Tenant: &model.Tenant{TenantID: 1, Name: "t"},
		}
		res := toIdpServiceDataResult(idp)
		require.NotNil(t, res.Tenant)
		assert.Equal(t, "t", res.Tenant.Name)
	})

	t.Run("without tenant", func(t *testing.T) {
		idp := &model.IdentityProvider{Name: "idp"}
		res := toIdpServiceDataResult(idp)
		assert.Nil(t, res.Tenant)
	})
}
