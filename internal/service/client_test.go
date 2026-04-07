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

func buildClientService(t *testing.T, clientRepo *mockClientRepo, idpRepo *mockIdentityProviderRepo, userRepo *mockUserRepo) ClientService {
	t.Helper()
	db, _ := newMockGormDB(t)
	return NewClientService(db, clientRepo, &mockClientURIRepo{}, idpRepo,
		&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{},
		&mockAPIRepo{}, userRepo, &mockTenantRepo{})
}

// helper: builds a full ClientService with all mock repos exposed
func buildFullClientService(
	t *testing.T,
	clientRepo *mockClientRepo,
	clientURIRepo *mockClientURIRepo,
	idpRepo *mockIdentityProviderRepo,
	permRepo *mockPermissionRepo,
	cpRepo *mockClientPermissionRepo,
	caRepo *mockClientAPIRepo,
	apiRepo *mockAPIRepo,
	userRepo *mockUserRepo,
	tenantRepo *mockTenantRepo,
) ClientService {
	t.Helper()
	db, _ := newMockGormDB(t)
	return NewClientService(db, clientRepo, clientURIRepo, idpRepo, permRepo, cpRepo, caRepo, apiRepo, userRepo, tenantRepo)
}

func clientWithIDP(tenantID int64) *model.Client {
	return &model.Client{
		ClientID:   1,
		ClientUUID: uuid.New(),
		Name:       "test",
		TenantID:   tenantID,
		Status:     model.StatusActive,
		IdentityProvider: &model.IdentityProvider{
			TenantID: tenantID,
			Tenant:   &model.Tenant{TenantID: tenantID, IsDefault: true},
		},
	}
}

func actorUser(tenantID int64) *model.User {
	return &model.User{
		UserID: 1,
		UserIdentities: []model.UserIdentity{
			{TenantID: tenantID, Tenant: &model.Tenant{TenantID: tenantID, IsDefault: true}},
		},
	}
}

// ===========================================================================
// Get
// ===========================================================================

func TestClientService_Get(t *testing.T) {
	idpUUID := uuid.New().String()

	t.Run("idp not found - returns empty result", func(t *testing.T) {
		idpRepo := &mockIdentityProviderRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.IdentityProvider, error) { return nil, nil },
		}
		svc := buildClientService(t, &mockClientRepo{}, idpRepo, &mockUserRepo{})
		res, err := svc.Get(ClientServiceGetFilter{IdentityProviderUUID: &idpUUID, TenantID: 1})
		require.NoError(t, err)
		assert.NotNil(t, res)
		assert.Empty(t, res.Data)
	})

	t.Run("paginate error", func(t *testing.T) {
		clientRepo := &mockClientRepo{
			findPaginatedFn: func(_ repository.ClientRepositoryGetFilter) (*repository.PaginationResult[model.Client], error) {
				return nil, errors.New("db error")
			},
		}
		svc := buildClientService(t, clientRepo, &mockIdentityProviderRepo{}, &mockUserRepo{})
		_, err := svc.Get(ClientServiceGetFilter{TenantID: 1})
		require.Error(t, err)
	})

	t.Run("success with no filter - returns empty list", func(t *testing.T) {
		svc := buildClientService(t, &mockClientRepo{}, &mockIdentityProviderRepo{}, &mockUserRepo{})
		res, err := svc.Get(ClientServiceGetFilter{TenantID: 1})
		require.NoError(t, err)
		assert.NotNil(t, res)
	})

	t.Run("success with data and IDP preloaded", func(t *testing.T) {
		c := clientWithIDP(1)
		uris := []model.ClientURI{{ClientURIUUID: uuid.New(), URI: "https://cb.example.com", Type: "redirect"}}
		c.ClientURIs = &uris
		clientRepo := &mockClientRepo{
			findPaginatedFn: func(_ repository.ClientRepositoryGetFilter) (*repository.PaginationResult[model.Client], error) {
				return &repository.PaginationResult[model.Client]{
					Data: []model.Client{*c}, Total: 1, Page: 1, Limit: 10, TotalPages: 1,
				}, nil
			},
		}
		idpRepo := &mockIdentityProviderRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.IdentityProvider, error) {
				return &model.IdentityProvider{IdentityProviderID: 1}, nil
			},
		}
		svc := buildClientService(t, clientRepo, idpRepo, &mockUserRepo{})
		res, err := svc.Get(ClientServiceGetFilter{TenantID: 1, IdentityProviderUUID: &idpUUID})
		require.NoError(t, err)
		assert.Len(t, res.Data, 1)
		assert.NotNil(t, res.Data[0].IdentityProvider)
		assert.NotNil(t, res.Data[0].ClientURIs)
	})
}

// ===========================================================================
// GetByUUID
// ===========================================================================

func TestClientService_GetByUUID(t *testing.T) {
	cUUID := uuid.New()

	t.Run("client not found returns error", func(t *testing.T) {
		svc := buildClientService(t, &mockClientRepo{}, &mockIdentityProviderRepo{}, &mockUserRepo{})
		_, err := svc.GetByUUID(cUUID, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("repo error returns error", func(t *testing.T) {
		clientRepo := &mockClientRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.Client, error) {
				return nil, errors.New("db error")
			},
		}
		svc := buildClientService(t, clientRepo, &mockIdentityProviderRepo{}, &mockUserRepo{})
		_, err := svc.GetByUUID(cUUID, 1)
		require.Error(t, err)
	})

	t.Run("success", func(t *testing.T) {
		clientRepo := &mockClientRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.Client, error) {
				return &model.Client{ClientUUID: cUUID}, nil
			},
		}
		svc := buildClientService(t, clientRepo, &mockIdentityProviderRepo{}, &mockUserRepo{})
		res, err := svc.GetByUUID(cUUID, 1)
		require.NoError(t, err)
		assert.NotNil(t, res)
	})
}

// ===========================================================================
// GetSecretByUUID
// ===========================================================================

func TestClientService_GetSecretByUUID(t *testing.T) {
	cUUID := uuid.New()

	t.Run("not found → error", func(t *testing.T) {
		svc := buildClientService(t, &mockClientRepo{}, &mockIdentityProviderRepo{}, &mockUserRepo{})
		_, err := svc.GetSecretByUUID(cUUID, 1)
		require.Error(t, err)
	})

	t.Run("repo error → propagated", func(t *testing.T) {
		clientRepo := &mockClientRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.Client, error) {
				return nil, errors.New("db err")
			},
		}
		svc := buildClientService(t, clientRepo, &mockIdentityProviderRepo{}, &mockUserRepo{})
		_, err := svc.GetSecretByUUID(cUUID, 1)
		require.Error(t, err)
	})

	t.Run("success", func(t *testing.T) {
		id := "client-id"
		secret := "client-secret"
		clientRepo := &mockClientRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.Client, error) {
				return &model.Client{Identifier: &id, Secret: &secret}, nil
			},
		}
		svc := buildClientService(t, clientRepo, &mockIdentityProviderRepo{}, &mockUserRepo{})
		res, err := svc.GetSecretByUUID(cUUID, 1)
		require.NoError(t, err)
		assert.Equal(t, id, res.ClientID)
		assert.Equal(t, &secret, res.ClientSecret)
	})
}

// ===========================================================================
// GetConfigByUUID
// ===========================================================================

func TestClientService_GetConfigByUUID(t *testing.T) {
	cUUID := uuid.New()

	t.Run("not found → error", func(t *testing.T) {
		svc := buildClientService(t, &mockClientRepo{}, &mockIdentityProviderRepo{}, &mockUserRepo{})
		_, err := svc.GetConfigByUUID(cUUID, 1)
		require.Error(t, err)
	})

	t.Run("repo error → propagated", func(t *testing.T) {
		clientRepo := &mockClientRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.Client, error) {
				return nil, errors.New("db err")
			},
		}
		svc := buildClientService(t, clientRepo, &mockIdentityProviderRepo{}, &mockUserRepo{})
		_, err := svc.GetConfigByUUID(cUUID, 1)
		require.Error(t, err)
	})

	t.Run("success", func(t *testing.T) {
		clientRepo := &mockClientRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.Client, error) {
				return &model.Client{Config: []byte(`{"key":"value"}`)}, nil
			},
		}
		svc := buildClientService(t, clientRepo, &mockIdentityProviderRepo{}, &mockUserRepo{})
		cfg, err := svc.GetConfigByUUID(cUUID, 1)
		require.NoError(t, err)
		assert.NotNil(t, cfg)
	})
}

// ===========================================================================
// Create
// ===========================================================================

func TestClientService_Create(t *testing.T) {
	actorUUID := uuid.New()
	tenantID := int64(1)

	t.Run("invalid identity provider UUID", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewClientService(gormDB, &mockClientRepo{}, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{},
			&mockAPIRepo{}, &mockUserRepo{}, &mockTenantRepo{})
		_, err := svc.Create(tenantID, "test", "Test", "public", "example.com", nil, "active", false, "not-a-valid-uuid", actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid identity provider UUID")
	})

	t.Run("identity provider not found", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		idpRepo := &mockIdentityProviderRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.IdentityProvider, error) { return nil, nil },
		}
		svc := NewClientService(gormDB, &mockClientRepo{}, &mockClientURIRepo{}, idpRepo,
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{},
			&mockAPIRepo{}, &mockUserRepo{}, &mockTenantRepo{})
		_, err := svc.Create(tenantID, "test", "Test", "public", "example.com", nil, "active", false, uuid.New().String(), actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "identity provider not found")
	})

	t.Run("actor user not found", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		idpRepo := &mockIdentityProviderRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.IdentityProvider, error) {
				return &model.IdentityProvider{IdentityProviderID: 1, Tenant: &model.Tenant{TenantID: tenantID, IsDefault: true}}, nil
			},
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) { return nil, nil },
		}
		svc := NewClientService(gormDB, &mockClientRepo{}, &mockClientURIRepo{}, idpRepo,
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{},
			&mockAPIRepo{}, userRepo, &mockTenantRepo{})
		_, err := svc.Create(tenantID, "test", "Test", "public", "example.com", nil, "active", false, uuid.New().String(), actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "actor user not found")
	})

	t.Run("tenant access denied", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		idpRepo := &mockIdentityProviderRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.IdentityProvider, error) {
				return &model.IdentityProvider{IdentityProviderID: 1, Tenant: &model.Tenant{TenantID: tenantID, IsDefault: false}}, nil
			},
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return &model.User{UserID: 1, UserIdentities: []model.UserIdentity{
					{TenantID: 999, Tenant: &model.Tenant{TenantID: 999, IsDefault: false}},
				}}, nil
			},
		}
		svc := NewClientService(gormDB, &mockClientRepo{}, &mockClientURIRepo{}, idpRepo,
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{},
			&mockAPIRepo{}, userRepo, &mockTenantRepo{})
		_, err := svc.Create(tenantID, "test", "Test", "public", "example.com", nil, "active", false, uuid.New().String(), actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "access denied")
	})

	t.Run("client name already exists", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		idpRepo := &mockIdentityProviderRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.IdentityProvider, error) {
				return &model.IdentityProvider{IdentityProviderID: 1, Tenant: &model.Tenant{TenantID: tenantID, IsDefault: true}}, nil
			},
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) { return actorUser(tenantID), nil },
		}
		clientRepo := &mockClientRepo{
			findByNameAndIdentityProviderFn: func(_ string, _ int64, _ int64) (*model.Client, error) {
				return &model.Client{}, nil
			},
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, idpRepo,
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{},
			&mockAPIRepo{}, userRepo, &mockTenantRepo{})
		_, err := svc.Create(tenantID, "test", "Test", "public", "example.com", nil, "active", false, uuid.New().String(), actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
	})

	t.Run("findByNameAndIdentityProvider error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		idpRepo := &mockIdentityProviderRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.IdentityProvider, error) {
				return &model.IdentityProvider{IdentityProviderID: 1, Tenant: &model.Tenant{TenantID: tenantID, IsDefault: true}}, nil
			},
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) { return actorUser(tenantID), nil },
		}
		clientRepo := &mockClientRepo{
			findByNameAndIdentityProviderFn: func(_ string, _ int64, _ int64) (*model.Client, error) {
				return nil, errors.New("db err")
			},
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, idpRepo,
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{},
			&mockAPIRepo{}, userRepo, &mockTenantRepo{})
		_, err := svc.Create(tenantID, "test", "Test", "public", "example.com", nil, "active", false, uuid.New().String(), actorUUID)
		require.Error(t, err)
	})

	t.Run("createOrUpdate error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		idpRepo := &mockIdentityProviderRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.IdentityProvider, error) {
				return &model.IdentityProvider{IdentityProviderID: 1, Tenant: &model.Tenant{TenantID: tenantID, IsDefault: true}}, nil
			},
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) { return actorUser(tenantID), nil },
		}
		clientRepo := &mockClientRepo{
			createOrUpdateFn: func(_ *model.Client) (*model.Client, error) { return nil, errors.New("save err") },
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, idpRepo,
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{},
			&mockAPIRepo{}, userRepo, &mockTenantRepo{})
		_, err := svc.Create(tenantID, "test", "Test", "public", "example.com", nil, "active", false, uuid.New().String(), actorUUID)
		require.Error(t, err)
	})

	t.Run("fetch after save error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		idpRepo := &mockIdentityProviderRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.IdentityProvider, error) {
				return &model.IdentityProvider{IdentityProviderID: 1, Tenant: &model.Tenant{TenantID: tenantID, IsDefault: true}}, nil
			},
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) { return actorUser(tenantID), nil },
		}
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Client, error) { return nil, errors.New("fetch err") },
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, idpRepo,
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{},
			&mockAPIRepo{}, userRepo, &mockTenantRepo{})
		_, err := svc.Create(tenantID, "test", "Test", "public", "example.com", nil, "active", false, uuid.New().String(), actorUUID)
		require.Error(t, err)
	})

	t.Run("success", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		idpRepo := &mockIdentityProviderRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.IdentityProvider, error) {
				return &model.IdentityProvider{IdentityProviderID: 1, Tenant: &model.Tenant{TenantID: tenantID, IsDefault: true}}, nil
			},
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) { return actorUser(tenantID), nil },
		}
		created := clientWithIDP(tenantID)
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Client, error) { return created, nil },
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, idpRepo,
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{},
			&mockAPIRepo{}, userRepo, &mockTenantRepo{})
		res, err := svc.Create(tenantID, "test", "Test", "public", "example.com", nil, "active", false, uuid.New().String(), actorUUID)
		require.NoError(t, err)
		assert.NotNil(t, res)
	})
}

// ===========================================================================
// Update
// ===========================================================================

func TestClientService_Update(t *testing.T) {
	cUUID := uuid.New()
	actorUUID := uuid.New()
	tenantID := int64(1)

	t.Run("client not found", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Client, error) { return nil, nil },
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{},
			&mockAPIRepo{}, &mockUserRepo{}, &mockTenantRepo{})
		_, err := svc.Update(cUUID, tenantID, "n", "d", "pub", "ex.com", nil, "active", false, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("actor user not found", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Client, error) { return clientWithIDP(tenantID), nil },
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) { return nil, nil },
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{},
			&mockAPIRepo{}, userRepo, &mockTenantRepo{})
		_, err := svc.Update(cUUID, tenantID, "n", "d", "pub", "ex.com", nil, "active", false, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "actor user not found")
	})

	t.Run("default client cannot be updated", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		c := clientWithIDP(tenantID)
		c.IsDefault = true
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Client, error) { return c, nil },
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) { return actorUser(tenantID), nil },
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{},
			&mockAPIRepo{}, userRepo, &mockTenantRepo{})
		_, err := svc.Update(cUUID, tenantID, "n", "d", "pub", "ex.com", nil, "active", false, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "default")
	})

	t.Run("name conflict", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		c := clientWithIDP(tenantID)
		c.Name = "old-name"
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Client, error) { return c, nil },
			findByNameAndIdentityProviderFn: func(_ string, _ int64, _ int64) (*model.Client, error) {
				other := clientWithIDP(tenantID)
				other.ClientUUID = uuid.New()
				return other, nil
			},
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) { return actorUser(tenantID), nil },
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{},
			&mockAPIRepo{}, userRepo, &mockTenantRepo{})
		_, err := svc.Update(cUUID, tenantID, "new-name", "d", "pub", "ex.com", nil, "active", false, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
	})

	t.Run("save error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		c := clientWithIDP(tenantID)
		clientRepo := &mockClientRepo{
			findByUUIDFn:     func(_ any, _ ...string) (*model.Client, error) { return c, nil },
			createOrUpdateFn: func(_ *model.Client) (*model.Client, error) { return nil, errors.New("save err") },
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) { return actorUser(tenantID), nil },
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{},
			&mockAPIRepo{}, userRepo, &mockTenantRepo{})
		_, err := svc.Update(cUUID, tenantID, "test", "d", "pub", "ex.com", nil, "active", false, actorUUID)
		require.Error(t, err)
	})

	t.Run("success", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		c := clientWithIDP(tenantID)
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Client, error) { return c, nil },
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) { return actorUser(tenantID), nil },
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{},
			&mockAPIRepo{}, userRepo, &mockTenantRepo{})
		res, err := svc.Update(cUUID, tenantID, "test", "Test", "pub", "ex.com", nil, "active", false, actorUUID)
		require.NoError(t, err)
		assert.NotNil(t, res)
	})
}

// ===========================================================================
// SetStatusByUUID
// ===========================================================================

func TestClientService_SetStatusByUUID(t *testing.T) {
	cUUID := uuid.New()
	actorUUID := uuid.New()
	tenantID := int64(1)

	t.Run("client not found", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Client, error) { return nil, nil },
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{},
			&mockAPIRepo{}, &mockUserRepo{}, &mockTenantRepo{})
		_, err := svc.SetStatusByUUID(cUUID, tenantID, "inactive", actorUUID)
		require.Error(t, err)
	})

	t.Run("actor not found", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Client, error) { return clientWithIDP(tenantID), nil },
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) { return nil, nil },
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{},
			&mockAPIRepo{}, userRepo, &mockTenantRepo{})
		_, err := svc.SetStatusByUUID(cUUID, tenantID, "inactive", actorUUID)
		require.Error(t, err)
	})

	t.Run("default client cannot be updated", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		c := clientWithIDP(tenantID)
		c.IsDefault = true
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Client, error) { return c, nil },
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) { return actorUser(tenantID), nil },
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{},
			&mockAPIRepo{}, userRepo, &mockTenantRepo{})
		_, err := svc.SetStatusByUUID(cUUID, tenantID, "inactive", actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "default")
	})

	t.Run("system client cannot be updated", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		c := clientWithIDP(tenantID)
		c.IsSystem = true
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Client, error) { return c, nil },
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) { return actorUser(tenantID), nil },
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{},
			&mockAPIRepo{}, userRepo, &mockTenantRepo{})
		_, err := svc.SetStatusByUUID(cUUID, tenantID, "inactive", actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "system")
	})

	t.Run("save error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDFn:     func(_ any, _ ...string) (*model.Client, error) { return clientWithIDP(tenantID), nil },
			createOrUpdateFn: func(_ *model.Client) (*model.Client, error) { return nil, errors.New("save err") },
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) { return actorUser(tenantID), nil },
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{},
			&mockAPIRepo{}, userRepo, &mockTenantRepo{})
		_, err := svc.SetStatusByUUID(cUUID, tenantID, "inactive", actorUUID)
		require.Error(t, err)
	})

	t.Run("success", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Client, error) { return clientWithIDP(tenantID), nil },
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) { return actorUser(tenantID), nil },
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{},
			&mockAPIRepo{}, userRepo, &mockTenantRepo{})
		res, err := svc.SetStatusByUUID(cUUID, tenantID, "inactive", actorUUID)
		require.NoError(t, err)
		assert.NotNil(t, res)
	})
}

// ===========================================================================
// DeleteByUUID
// ===========================================================================

func TestClientService_DeleteByUUID(t *testing.T) {
	cUUID := uuid.New()
	actorUUID := uuid.New()
	tenantID := int64(1)

	t.Run("client not found", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Client, error) { return nil, nil },
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{},
			&mockAPIRepo{}, &mockUserRepo{}, &mockTenantRepo{})
		_, err := svc.DeleteByUUID(cUUID, tenantID, actorUUID)
		require.Error(t, err)
	})

	t.Run("actor not found", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Client, error) { return clientWithIDP(tenantID), nil },
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) { return nil, nil },
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{},
			&mockAPIRepo{}, userRepo, &mockTenantRepo{})
		_, err := svc.DeleteByUUID(cUUID, tenantID, actorUUID)
		require.Error(t, err)
	})

	t.Run("default client cannot be deleted", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		c := clientWithIDP(tenantID)
		c.IsDefault = true
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Client, error) { return c, nil },
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) { return actorUser(tenantID), nil },
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{},
			&mockAPIRepo{}, userRepo, &mockTenantRepo{})
		_, err := svc.DeleteByUUID(cUUID, tenantID, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "default")
	})

	t.Run("delete error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDFn:   func(_ any, _ ...string) (*model.Client, error) { return clientWithIDP(tenantID), nil },
			deleteByUUIDFn: func(_ any) error { return errors.New("del err") },
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) { return actorUser(tenantID), nil },
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{},
			&mockAPIRepo{}, userRepo, &mockTenantRepo{})
		_, err := svc.DeleteByUUID(cUUID, tenantID, actorUUID)
		require.Error(t, err)
	})

	t.Run("success", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Client, error) { return clientWithIDP(tenantID), nil },
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) { return actorUser(tenantID), nil },
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{},
			&mockAPIRepo{}, userRepo, &mockTenantRepo{})
		res, err := svc.DeleteByUUID(cUUID, tenantID, actorUUID)
		require.NoError(t, err)
		assert.NotNil(t, res)
	})
}

// ===========================================================================
// CreateURI / UpdateURI / DeleteURI
// ===========================================================================

func TestClientService_CreateURI(t *testing.T) {
	cUUID := uuid.New()
	actorUUID := uuid.New()
	tenantID := int64(1)

	t.Run("client not found", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewClientService(gormDB, &mockClientRepo{}, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{},
			&mockAPIRepo{}, &mockUserRepo{}, &mockTenantRepo{})
		_, err := svc.CreateURI(cUUID, tenantID, "https://cb.test", "redirect", actorUUID)
		require.Error(t, err)
	})

	t.Run("actor not found", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.Client, error) {
				return &model.Client{ClientID: 1, TenantID: tenantID}, nil
			},
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) { return nil, nil },
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{},
			&mockAPIRepo{}, userRepo, &mockTenantRepo{})
		_, err := svc.CreateURI(cUUID, tenantID, "https://cb.test", "redirect", actorUUID)
		require.Error(t, err)
	})

	t.Run("tenant mismatch → access denied", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.Client, error) {
				return &model.Client{ClientID: 1, TenantID: 999}, nil
			},
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) { return actorUser(tenantID), nil },
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{},
			&mockAPIRepo{}, userRepo, &mockTenantRepo{})
		_, err := svc.CreateURI(cUUID, tenantID, "https://cb.test", "redirect", actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "access denied")
	})

	t.Run("success", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		clientRepo := &mockClientRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.Client, error) {
				return &model.Client{ClientID: 1, TenantID: tenantID}, nil
			},
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) { return actorUser(tenantID), nil },
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{},
			&mockAPIRepo{}, userRepo, &mockTenantRepo{})
		res, err := svc.CreateURI(cUUID, tenantID, "https://cb.test", "redirect", actorUUID)
		require.NoError(t, err)
		assert.NotNil(t, res)
	})
}

func TestClientService_UpdateURI(t *testing.T) {
	cUUID := uuid.New()
	uriUUID := uuid.New()
	actorUUID := uuid.New()
	tenantID := int64(1)

	t.Run("client not found", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewClientService(gormDB, &mockClientRepo{}, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{},
			&mockAPIRepo{}, &mockUserRepo{}, &mockTenantRepo{})
		_, err := svc.UpdateURI(cUUID, tenantID, uriUUID, "https://new.test", "redirect", actorUUID)
		require.Error(t, err)
	})

	t.Run("actor not found", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.Client, error) {
				return &model.Client{ClientID: 1, TenantID: tenantID}, nil
			},
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) { return nil, nil },
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{},
			&mockAPIRepo{}, userRepo, &mockTenantRepo{})
		_, err := svc.UpdateURI(cUUID, tenantID, uriUUID, "https://new.test", "redirect", actorUUID)
		require.Error(t, err)
	})

	t.Run("URI not found", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.Client, error) {
				return &model.Client{ClientID: 1, TenantID: tenantID}, nil
			},
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) { return actorUser(tenantID), nil },
		}
		uriRepo := &mockClientURIRepo{
			findByUUIDAndTenantIDFn: func(_ string, _ int64) (*model.ClientURI, error) { return nil, nil },
		}
		svc := NewClientService(gormDB, clientRepo, uriRepo, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{},
			&mockAPIRepo{}, userRepo, &mockTenantRepo{})
		_, err := svc.UpdateURI(cUUID, tenantID, uriUUID, "https://new.test", "redirect", actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "URI not found")
	})

	t.Run("URI belongs to different client", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.Client, error) {
				return &model.Client{ClientID: 1, TenantID: tenantID}, nil
			},
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) { return actorUser(tenantID), nil },
		}
		uriRepo := &mockClientURIRepo{
			findByUUIDAndTenantIDFn: func(_ string, _ int64) (*model.ClientURI, error) {
				return &model.ClientURI{ClientID: 999}, nil // different client
			},
		}
		svc := NewClientService(gormDB, clientRepo, uriRepo, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{},
			&mockAPIRepo{}, userRepo, &mockTenantRepo{})
		_, err := svc.UpdateURI(cUUID, tenantID, uriUUID, "https://new.test", "redirect", actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "does not belong")
	})

	t.Run("success", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		clientRepo := &mockClientRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.Client, error) {
				return &model.Client{ClientID: 1, TenantID: tenantID}, nil
			},
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) { return actorUser(tenantID), nil },
		}
		uriRepo := &mockClientURIRepo{
			findByUUIDAndTenantIDFn: func(_ string, _ int64) (*model.ClientURI, error) {
				return &model.ClientURI{ClientID: 1}, nil
			},
		}
		svc := NewClientService(gormDB, clientRepo, uriRepo, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{},
			&mockAPIRepo{}, userRepo, &mockTenantRepo{})
		res, err := svc.UpdateURI(cUUID, tenantID, uriUUID, "https://new.test", "redirect", actorUUID)
		require.NoError(t, err)
		assert.NotNil(t, res)
	})
}

func TestClientService_DeleteURI(t *testing.T) {
	cUUID := uuid.New()
	uriUUID := uuid.New()
	actorUUID := uuid.New()
	tenantID := int64(1)

	t.Run("client not found", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewClientService(gormDB, &mockClientRepo{}, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{},
			&mockAPIRepo{}, &mockUserRepo{}, &mockTenantRepo{})
		_, err := svc.DeleteURI(cUUID, tenantID, uriUUID, actorUUID)
		require.Error(t, err)
	})

	t.Run("actor not found", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.Client, error) {
				return &model.Client{ClientID: 1, TenantID: tenantID}, nil
			},
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) { return nil, nil },
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{},
			&mockAPIRepo{}, userRepo, &mockTenantRepo{})
		_, err := svc.DeleteURI(cUUID, tenantID, uriUUID, actorUUID)
		require.Error(t, err)
	})

	t.Run("URI not found", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.Client, error) {
				return &model.Client{ClientID: 1, TenantID: tenantID}, nil
			},
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) { return actorUser(tenantID), nil },
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{},
			&mockAPIRepo{}, userRepo, &mockTenantRepo{})
		_, err := svc.DeleteURI(cUUID, tenantID, uriUUID, actorUUID)
		require.Error(t, err)
	})

	t.Run("URI belongs to different client", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.Client, error) {
				return &model.Client{ClientID: 1, TenantID: tenantID}, nil
			},
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) { return actorUser(tenantID), nil },
		}
		uriRepo := &mockClientURIRepo{
			findByUUIDAndTenantIDFn: func(_ string, _ int64) (*model.ClientURI, error) {
				return &model.ClientURI{ClientID: 999}, nil
			},
		}
		svc := NewClientService(gormDB, clientRepo, uriRepo, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{},
			&mockAPIRepo{}, userRepo, &mockTenantRepo{})
		_, err := svc.DeleteURI(cUUID, tenantID, uriUUID, actorUUID)
		require.Error(t, err)
	})

	t.Run("success", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		clientRepo := &mockClientRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.Client, error) {
				return &model.Client{ClientID: 1, TenantID: tenantID}, nil
			},
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) { return actorUser(tenantID), nil },
		}
		uriRepo := &mockClientURIRepo{
			findByUUIDAndTenantIDFn: func(_ string, _ int64) (*model.ClientURI, error) {
				return &model.ClientURI{ClientID: 1}, nil
			},
		}
		svc := NewClientService(gormDB, clientRepo, uriRepo, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{},
			&mockAPIRepo{}, userRepo, &mockTenantRepo{})
		res, err := svc.DeleteURI(cUUID, tenantID, uriUUID, actorUUID)
		require.NoError(t, err)
		assert.NotNil(t, res)
	})
}

// ===========================================================================
// GetClientAPIs / AddClientAPIs / RemoveClientAPI
// ===========================================================================

func TestClientService_GetClientAPIs(t *testing.T) {
	cUUID := uuid.New()

	t.Run("repo error", func(t *testing.T) {
		caRepo := &mockClientAPIRepo{
			findByClientUUIDFn: func(_ uuid.UUID) ([]model.ClientAPI, error) { return nil, errors.New("err") },
		}
		svc := buildFullClientService(t, &mockClientRepo{}, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, caRepo, &mockAPIRepo{}, &mockUserRepo{}, &mockTenantRepo{})
		_, err := svc.GetClientAPIs(1, cUUID)
		require.Error(t, err)
	})

	t.Run("success with permissions", func(t *testing.T) {
		perm := &model.Permission{PermissionUUID: uuid.New(), Name: "read"}
		cas := []model.ClientAPI{{
			ClientAPIUUID: uuid.New(),
			API:           model.API{APIUUID: uuid.New(), Name: "api1"},
			Permissions:   []model.ClientPermission{{Permission: perm}},
		}}
		caRepo := &mockClientAPIRepo{
			findByClientUUIDFn: func(_ uuid.UUID) ([]model.ClientAPI, error) { return cas, nil },
		}
		svc := buildFullClientService(t, &mockClientRepo{}, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, caRepo, &mockAPIRepo{}, &mockUserRepo{}, &mockTenantRepo{})
		results, err := svc.GetClientAPIs(1, cUUID)
		require.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Len(t, results[0].Permissions, 1)
	})

	t.Run("permission with nil Permission pointer", func(t *testing.T) {
		cas := []model.ClientAPI{{
			ClientAPIUUID: uuid.New(),
			API:           model.API{APIUUID: uuid.New(), Name: "api1"},
			Permissions:   []model.ClientPermission{{Permission: nil}},
		}}
		caRepo := &mockClientAPIRepo{
			findByClientUUIDFn: func(_ uuid.UUID) ([]model.ClientAPI, error) { return cas, nil },
		}
		svc := buildFullClientService(t, &mockClientRepo{}, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, caRepo, &mockAPIRepo{}, &mockUserRepo{}, &mockTenantRepo{})
		results, err := svc.GetClientAPIs(1, cUUID)
		require.NoError(t, err)
		assert.Len(t, results, 1)
	})
}

func TestClientService_AddClientAPIs(t *testing.T) {
	cUUID := uuid.New()
	apiUUID := uuid.New()
	tenantID := int64(1)

	t.Run("client not found", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Client, error) { return nil, nil },
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{},
			&mockAPIRepo{}, &mockUserRepo{}, &mockTenantRepo{})
		err := svc.AddClientAPIs(tenantID, cUUID, []uuid.UUID{apiUUID})
		require.Error(t, err)
	})

	t.Run("unauthorized tenant", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Client, error) {
				return &model.Client{ClientID: 1, IdentityProvider: &model.IdentityProvider{TenantID: 999}}, nil
			},
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{},
			&mockAPIRepo{}, &mockUserRepo{}, &mockTenantRepo{})
		err := svc.AddClientAPIs(tenantID, cUUID, []uuid.UUID{apiUUID})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unauthorized")
	})

	t.Run("API not found", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Client, error) {
				return &model.Client{ClientID: 1, IdentityProvider: &model.IdentityProvider{TenantID: tenantID}}, nil
			},
		}
		apiRepo := &mockAPIRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.API, error) { return nil, nil },
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{},
			apiRepo, &mockUserRepo{}, &mockTenantRepo{})
		err := svc.AddClientAPIs(tenantID, cUUID, []uuid.UUID{apiUUID})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "API not found")
	})

	t.Run("API already assigned", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Client, error) {
				return &model.Client{ClientID: 1, IdentityProvider: &model.IdentityProvider{TenantID: tenantID}}, nil
			},
		}
		apiRepo := &mockAPIRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.API, error) { return &model.API{APIID: 1}, nil },
		}
		caRepo := &mockClientAPIRepo{
			findByClientAndAPIFn: func(_, _ int64) (*model.ClientAPI, error) { return &model.ClientAPI{}, nil },
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, caRepo,
			apiRepo, &mockUserRepo{}, &mockTenantRepo{})
		err := svc.AddClientAPIs(tenantID, cUUID, []uuid.UUID{apiUUID})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "already assigned")
	})

	t.Run("success", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Client, error) {
				return &model.Client{ClientID: 1, IdentityProvider: &model.IdentityProvider{TenantID: tenantID}}, nil
			},
		}
		apiRepo := &mockAPIRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.API, error) { return &model.API{APIID: 1}, nil },
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{},
			apiRepo, &mockUserRepo{}, &mockTenantRepo{})
		err := svc.AddClientAPIs(tenantID, cUUID, []uuid.UUID{apiUUID})
		require.NoError(t, err)
	})
}

func TestClientService_RemoveClientAPI(t *testing.T) {
	cUUID := uuid.New()
	apiUUID := uuid.New()
	tenantID := int64(1)

	t.Run("client not found", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Client, error) { return nil, nil },
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{},
			&mockAPIRepo{}, &mockUserRepo{}, &mockTenantRepo{})
		err := svc.RemoveClientAPI(tenantID, cUUID, apiUUID)
		require.Error(t, err)
	})

	t.Run("unauthorized tenant", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Client, error) {
				return &model.Client{ClientID: 1, IdentityProvider: &model.IdentityProvider{TenantID: 999}}, nil
			},
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{},
			&mockAPIRepo{}, &mockUserRepo{}, &mockTenantRepo{})
		err := svc.RemoveClientAPI(tenantID, cUUID, apiUUID)
		require.Error(t, err)
	})

	t.Run("remove error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Client, error) {
				return &model.Client{ClientID: 1, IdentityProvider: &model.IdentityProvider{TenantID: tenantID}}, nil
			},
		}
		caRepo := &mockClientAPIRepo{
			removeByClientUUIDAndAPIUUIDFn: func(_, _ uuid.UUID) error { return errors.New("del err") },
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, caRepo,
			&mockAPIRepo{}, &mockUserRepo{}, &mockTenantRepo{})
		err := svc.RemoveClientAPI(tenantID, cUUID, apiUUID)
		require.Error(t, err)
	})

	t.Run("success", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Client, error) {
				return &model.Client{ClientID: 1, IdentityProvider: &model.IdentityProvider{TenantID: tenantID}}, nil
			},
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{},
			&mockAPIRepo{}, &mockUserRepo{}, &mockTenantRepo{})
		err := svc.RemoveClientAPI(tenantID, cUUID, apiUUID)
		require.NoError(t, err)
	})
}

// ===========================================================================
// GetClientAPIPermissions / AddClientAPIPermissions / RemoveClientAPIPermission
// ===========================================================================

func TestClientService_GetClientAPIPermissions(t *testing.T) {
	cUUID := uuid.New()
	apiUUID := uuid.New()
	tenantID := int64(1)

	t.Run("client API relationship not found", func(t *testing.T) {
		svc := buildFullClientService(t, &mockClientRepo{}, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{}, &mockAPIRepo{}, &mockUserRepo{}, &mockTenantRepo{})
		_, err := svc.GetClientAPIPermissions(tenantID, cUUID, apiUUID)
		require.Error(t, err)
	})

	t.Run("client not found", func(t *testing.T) {
		caRepo := &mockClientAPIRepo{
			findByClientUUIDAndAPIUUIDFn: func(_, _ uuid.UUID) (*model.ClientAPI, error) {
				return &model.ClientAPI{ClientAPIID: 1}, nil
			},
		}
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Client, error) { return nil, nil },
		}
		svc := buildFullClientService(t, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, caRepo, &mockAPIRepo{}, &mockUserRepo{}, &mockTenantRepo{})
		_, err := svc.GetClientAPIPermissions(tenantID, cUUID, apiUUID)
		require.Error(t, err)
	})

	t.Run("unauthorized tenant", func(t *testing.T) {
		caRepo := &mockClientAPIRepo{
			findByClientUUIDAndAPIUUIDFn: func(_, _ uuid.UUID) (*model.ClientAPI, error) {
				return &model.ClientAPI{ClientAPIID: 1}, nil
			},
		}
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Client, error) {
				return &model.Client{ClientID: 1, IdentityProvider: &model.IdentityProvider{TenantID: 999}}, nil
			},
		}
		svc := buildFullClientService(t, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, caRepo, &mockAPIRepo{}, &mockUserRepo{}, &mockTenantRepo{})
		_, err := svc.GetClientAPIPermissions(tenantID, cUUID, apiUUID)
		require.Error(t, err)
	})

	t.Run("success", func(t *testing.T) {
		caRepo := &mockClientAPIRepo{
			findByClientUUIDAndAPIUUIDFn: func(_, _ uuid.UUID) (*model.ClientAPI, error) {
				return &model.ClientAPI{ClientAPIID: 1}, nil
			},
		}
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Client, error) {
				return &model.Client{ClientID: 1, IdentityProvider: &model.IdentityProvider{TenantID: tenantID}}, nil
			},
		}
		cpRepo := &mockClientPermissionRepo{
			findByClientAPIIDFn: func(_ int64) ([]model.ClientPermission, error) {
				return []model.ClientPermission{{Permission: &model.Permission{PermissionUUID: uuid.New(), Name: "read"}}}, nil
			},
		}
		svc := buildFullClientService(t, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, cpRepo, caRepo, &mockAPIRepo{}, &mockUserRepo{}, &mockTenantRepo{})
		results, err := svc.GetClientAPIPermissions(tenantID, cUUID, apiUUID)
		require.NoError(t, err)
		assert.Len(t, results, 1)
	})
}

func TestClientService_AddClientAPIPermissions(t *testing.T) {
	cUUID := uuid.New()
	apiUUID := uuid.New()
	permUUID := uuid.New()
	tenantID := int64(1)

	t.Run("client not found", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Client, error) { return nil, nil },
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{},
			&mockAPIRepo{}, &mockUserRepo{}, &mockTenantRepo{})
		err := svc.AddClientAPIPermissions(tenantID, cUUID, apiUUID, []uuid.UUID{permUUID})
		require.Error(t, err)
	})

	t.Run("unauthorized tenant", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Client, error) {
				return &model.Client{ClientID: 1, IdentityProvider: &model.IdentityProvider{TenantID: 999}}, nil
			},
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{},
			&mockAPIRepo{}, &mockUserRepo{}, &mockTenantRepo{})
		err := svc.AddClientAPIPermissions(tenantID, cUUID, apiUUID, []uuid.UUID{permUUID})
		require.Error(t, err)
	})

	t.Run("client API not found", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Client, error) {
				return &model.Client{ClientID: 1, IdentityProvider: &model.IdentityProvider{TenantID: tenantID}}, nil
			},
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{},
			&mockAPIRepo{}, &mockUserRepo{}, &mockTenantRepo{})
		err := svc.AddClientAPIPermissions(tenantID, cUUID, apiUUID, []uuid.UUID{permUUID})
		require.Error(t, err)
	})

	t.Run("permission not found", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Client, error) {
				return &model.Client{ClientID: 1, IdentityProvider: &model.IdentityProvider{TenantID: tenantID}}, nil
			},
		}
		caRepo := &mockClientAPIRepo{
			findByClientUUIDAndAPIUUIDFn: func(_, _ uuid.UUID) (*model.ClientAPI, error) {
				return &model.ClientAPI{ClientAPIID: 1}, nil
			},
		}
		permRepo := &mockPermissionRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Permission, error) { return nil, nil },
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			permRepo, &mockClientPermissionRepo{}, caRepo,
			&mockAPIRepo{}, &mockUserRepo{}, &mockTenantRepo{})
		err := svc.AddClientAPIPermissions(tenantID, cUUID, apiUUID, []uuid.UUID{permUUID})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "permission not found")
	})

	t.Run("permission already assigned", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Client, error) {
				return &model.Client{ClientID: 1, IdentityProvider: &model.IdentityProvider{TenantID: tenantID}}, nil
			},
		}
		caRepo := &mockClientAPIRepo{
			findByClientUUIDAndAPIUUIDFn: func(_, _ uuid.UUID) (*model.ClientAPI, error) {
				return &model.ClientAPI{ClientAPIID: 1}, nil
			},
		}
		permRepo := &mockPermissionRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Permission, error) {
				return &model.Permission{PermissionID: 1}, nil
			},
		}
		cpRepo := &mockClientPermissionRepo{
			findByClientAPIAndPermissionFn: func(_, _ int64) (*model.ClientPermission, error) {
				return &model.ClientPermission{}, nil
			},
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			permRepo, cpRepo, caRepo,
			&mockAPIRepo{}, &mockUserRepo{}, &mockTenantRepo{})
		err := svc.AddClientAPIPermissions(tenantID, cUUID, apiUUID, []uuid.UUID{permUUID})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "already assigned")
	})

	t.Run("success", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Client, error) {
				return &model.Client{ClientID: 1, IdentityProvider: &model.IdentityProvider{TenantID: tenantID}}, nil
			},
		}
		caRepo := &mockClientAPIRepo{
			findByClientUUIDAndAPIUUIDFn: func(_, _ uuid.UUID) (*model.ClientAPI, error) {
				return &model.ClientAPI{ClientAPIID: 1}, nil
			},
		}
		permRepo := &mockPermissionRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Permission, error) {
				return &model.Permission{PermissionID: 1}, nil
			},
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			permRepo, &mockClientPermissionRepo{}, caRepo,
			&mockAPIRepo{}, &mockUserRepo{}, &mockTenantRepo{})
		err := svc.AddClientAPIPermissions(tenantID, cUUID, apiUUID, []uuid.UUID{permUUID})
		require.NoError(t, err)
	})
}

func TestClientService_RemoveClientAPIPermission(t *testing.T) {
	cUUID := uuid.New()
	apiUUID := uuid.New()
	permUUID := uuid.New()
	tenantID := int64(1)

	t.Run("client not found", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Client, error) { return nil, nil },
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{},
			&mockAPIRepo{}, &mockUserRepo{}, &mockTenantRepo{})
		err := svc.RemoveClientAPIPermission(tenantID, cUUID, apiUUID, permUUID)
		require.Error(t, err)
	})

	t.Run("unauthorized tenant", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Client, error) {
				return &model.Client{ClientID: 1, IdentityProvider: &model.IdentityProvider{TenantID: 999}}, nil
			},
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{},
			&mockAPIRepo{}, &mockUserRepo{}, &mockTenantRepo{})
		err := svc.RemoveClientAPIPermission(tenantID, cUUID, apiUUID, permUUID)
		require.Error(t, err)
	})

	t.Run("client API not found", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Client, error) {
				return &model.Client{ClientID: 1, IdentityProvider: &model.IdentityProvider{TenantID: tenantID}}, nil
			},
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{},
			&mockAPIRepo{}, &mockUserRepo{}, &mockTenantRepo{})
		err := svc.RemoveClientAPIPermission(tenantID, cUUID, apiUUID, permUUID)
		require.Error(t, err)
	})

	t.Run("permission not found", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Client, error) {
				return &model.Client{ClientID: 1, IdentityProvider: &model.IdentityProvider{TenantID: tenantID}}, nil
			},
		}
		caRepo := &mockClientAPIRepo{
			findByClientUUIDAndAPIUUIDFn: func(_, _ uuid.UUID) (*model.ClientAPI, error) {
				return &model.ClientAPI{ClientAPIID: 1}, nil
			},
		}
		permRepo := &mockPermissionRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Permission, error) { return nil, nil },
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			permRepo, &mockClientPermissionRepo{}, caRepo,
			&mockAPIRepo{}, &mockUserRepo{}, &mockTenantRepo{})
		err := svc.RemoveClientAPIPermission(tenantID, cUUID, apiUUID, permUUID)
		require.Error(t, err)
	})

	t.Run("success", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Client, error) {
				return &model.Client{ClientID: 1, IdentityProvider: &model.IdentityProvider{TenantID: tenantID}}, nil
			},
		}
		caRepo := &mockClientAPIRepo{
			findByClientUUIDAndAPIUUIDFn: func(_, _ uuid.UUID) (*model.ClientAPI, error) {
				return &model.ClientAPI{ClientAPIID: 1}, nil
			},
		}
		permRepo := &mockPermissionRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Permission, error) {
				return &model.Permission{PermissionID: 1}, nil
			},
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			permRepo, &mockClientPermissionRepo{}, caRepo,
			&mockAPIRepo{}, &mockUserRepo{}, &mockTenantRepo{})
		err := svc.RemoveClientAPIPermission(tenantID, cUUID, apiUUID, permUUID)
		require.NoError(t, err)
	})
}

// ===========================================================================
// ToClientServiceDataResult – nil
// ===========================================================================

func TestToClientServiceDataResult_Nil(t *testing.T) {
	assert.Nil(t, ToClientServiceDataResult(nil))
}

// ===========================================================================
// Additional edge case tests for 100% coverage
// ===========================================================================

func TestClientService_Update_ValidateTenantAccess(t *testing.T) {
	cUUID := uuid.New()
	actorUUID := uuid.New()
	tenantID := int64(1)

	t.Run("tenant access denied", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		c := clientWithIDP(tenantID)
		c.IdentityProvider.Tenant = &model.Tenant{TenantID: tenantID, IsDefault: false}
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Client, error) { return c, nil },
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return &model.User{UserID: 1, UserIdentities: []model.UserIdentity{
					{TenantID: 999, Tenant: &model.Tenant{TenantID: 999, IsDefault: false}},
				}}, nil
			},
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{},
			&mockAPIRepo{}, userRepo, &mockTenantRepo{})
		_, err := svc.Update(cUUID, tenantID, "n", "d", "pub", "ex.com", nil, "active", false, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "access denied")
	})

	t.Run("name changed and findByName returns error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		c := clientWithIDP(tenantID)
		c.Name = "old-name"
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Client, error) { return c, nil },
			findByNameAndIdentityProviderFn: func(_ string, _ int64, _ int64) (*model.Client, error) {
				return nil, errors.New("db err")
			},
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) { return actorUser(tenantID), nil },
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{},
			&mockAPIRepo{}, userRepo, &mockTenantRepo{})
		_, err := svc.Update(cUUID, tenantID, "new-name", "d", "pub", "ex.com", nil, "active", false, actorUUID)
		require.Error(t, err)
	})
}

func TestClientService_SetStatusByUUID_ValidateTenantAccess(t *testing.T) {
	cUUID := uuid.New()
	actorUUID := uuid.New()
	tenantID := int64(1)

	gormDB, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectRollback()
	c := clientWithIDP(tenantID)
	c.IdentityProvider.Tenant = &model.Tenant{TenantID: tenantID, IsDefault: false}
	clientRepo := &mockClientRepo{
		findByUUIDFn: func(_ any, _ ...string) (*model.Client, error) { return c, nil },
	}
	userRepo := &mockUserRepo{
		findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
			return &model.User{UserID: 1, UserIdentities: []model.UserIdentity{
				{TenantID: 999, Tenant: &model.Tenant{TenantID: 999, IsDefault: false}},
			}}, nil
		},
	}
	svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
		&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{},
		&mockAPIRepo{}, userRepo, &mockTenantRepo{})
	_, err := svc.SetStatusByUUID(cUUID, tenantID, "inactive", actorUUID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "access denied")
}

func TestClientService_DeleteByUUID_ValidateTenantAccess(t *testing.T) {
	cUUID := uuid.New()
	actorUUID := uuid.New()
	tenantID := int64(1)

	gormDB, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectRollback()
	c := clientWithIDP(tenantID)
	c.IdentityProvider.Tenant = &model.Tenant{TenantID: tenantID, IsDefault: false}
	clientRepo := &mockClientRepo{
		findByUUIDFn: func(_ any, _ ...string) (*model.Client, error) { return c, nil },
	}
	userRepo := &mockUserRepo{
		findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
			return &model.User{UserID: 1, UserIdentities: []model.UserIdentity{
				{TenantID: 999, Tenant: &model.Tenant{TenantID: 999, IsDefault: false}},
			}}, nil
		},
	}
	svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
		&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{},
		&mockAPIRepo{}, userRepo, &mockTenantRepo{})
	_, err := svc.DeleteByUUID(cUUID, tenantID, actorUUID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "access denied")
}

func TestClientService_CreateURI_EdgeCases(t *testing.T) {
	cUUID := uuid.New()
	actorUUID := uuid.New()
	tenantID := int64(1)

	t.Run("URI create error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.Client, error) {
				return &model.Client{ClientID: 1, TenantID: tenantID}, nil
			},
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) { return actorUser(tenantID), nil },
		}
		uriRepo := &mockClientURIRepo{
			createOrUpdateFn: func(_ *model.ClientURI) (*model.ClientURI, error) {
				return nil, errors.New("create err")
			},
		}
		svc := NewClientService(gormDB, clientRepo, uriRepo, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{},
			&mockAPIRepo{}, userRepo, &mockTenantRepo{})
		_, err := svc.CreateURI(cUUID, tenantID, "https://cb.test", "redirect", actorUUID)
		require.Error(t, err)
	})

	t.Run("post-save fetch error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		fetchCount := 0
		clientRepo := &mockClientRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.Client, error) {
				fetchCount++
				if fetchCount == 1 {
					return &model.Client{ClientID: 1, TenantID: tenantID}, nil
				}
				return nil, errors.New("fetch err")
			},
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) { return actorUser(tenantID), nil },
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{},
			&mockAPIRepo{}, userRepo, &mockTenantRepo{})
		_, err := svc.CreateURI(cUUID, tenantID, "https://cb.test", "redirect", actorUUID)
		require.Error(t, err)
	})

	t.Run("tenant mismatch", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.Client, error) {
				return &model.Client{ClientID: 1, TenantID: 999}, nil
			},
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) { return actorUser(tenantID), nil },
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{},
			&mockAPIRepo{}, userRepo, &mockTenantRepo{})
		_, err := svc.CreateURI(cUUID, tenantID, "https://cb.test", "redirect", actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "access denied")
	})
}

func TestClientService_UpdateURI_EdgeCases(t *testing.T) {
	cUUID := uuid.New()
	uriUUID := uuid.New()
	actorUUID := uuid.New()
	tenantID := int64(1)

	t.Run("tenant mismatch", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.Client, error) {
				return &model.Client{ClientID: 1, TenantID: 999}, nil
			},
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) { return actorUser(tenantID), nil },
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{},
			&mockAPIRepo{}, userRepo, &mockTenantRepo{})
		_, err := svc.UpdateURI(cUUID, tenantID, uriUUID, "https://new.test", "redirect", actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "access denied")
	})

	t.Run("URI save error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.Client, error) {
				return &model.Client{ClientID: 1, TenantID: tenantID}, nil
			},
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) { return actorUser(tenantID), nil },
		}
		uriRepo := &mockClientURIRepo{
			findByUUIDAndTenantIDFn: func(_ string, _ int64) (*model.ClientURI, error) {
				return &model.ClientURI{ClientID: 1}, nil
			},
			createOrUpdateFn: func(_ *model.ClientURI) (*model.ClientURI, error) {
				return nil, errors.New("save err")
			},
		}
		svc := NewClientService(gormDB, clientRepo, uriRepo, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{},
			&mockAPIRepo{}, userRepo, &mockTenantRepo{})
		_, err := svc.UpdateURI(cUUID, tenantID, uriUUID, "https://new.test", "redirect", actorUUID)
		require.Error(t, err)
	})

	t.Run("post-save fetch error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		fetchCount := 0
		clientRepo := &mockClientRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.Client, error) {
				fetchCount++
				if fetchCount == 1 {
					return &model.Client{ClientID: 1, TenantID: tenantID}, nil
				}
				return nil, errors.New("fetch err")
			},
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) { return actorUser(tenantID), nil },
		}
		uriRepo := &mockClientURIRepo{
			findByUUIDAndTenantIDFn: func(_ string, _ int64) (*model.ClientURI, error) {
				return &model.ClientURI{ClientID: 1}, nil
			},
		}
		svc := NewClientService(gormDB, clientRepo, uriRepo, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{},
			&mockAPIRepo{}, userRepo, &mockTenantRepo{})
		_, err := svc.UpdateURI(cUUID, tenantID, uriUUID, "https://new.test", "redirect", actorUUID)
		require.Error(t, err)
	})
}

func TestClientService_DeleteURI_EdgeCases(t *testing.T) {
	cUUID := uuid.New()
	uriUUID := uuid.New()
	actorUUID := uuid.New()
	tenantID := int64(1)

	t.Run("tenant mismatch", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.Client, error) {
				return &model.Client{ClientID: 1, TenantID: 999}, nil
			},
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) { return actorUser(tenantID), nil },
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{},
			&mockAPIRepo{}, userRepo, &mockTenantRepo{})
		_, err := svc.DeleteURI(cUUID, tenantID, uriUUID, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "access denied")
	})

	t.Run("delete error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.Client, error) {
				return &model.Client{ClientID: 1, TenantID: tenantID}, nil
			},
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) { return actorUser(tenantID), nil },
		}
		uriRepo := &mockClientURIRepo{
			findByUUIDAndTenantIDFn: func(_ string, _ int64) (*model.ClientURI, error) {
				return &model.ClientURI{ClientID: 1}, nil
			},
			deleteByUUIDAndTenantIDFn: func(_ string, _ int64) error { return errors.New("del err") },
		}
		svc := NewClientService(gormDB, clientRepo, uriRepo, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{},
			&mockAPIRepo{}, userRepo, &mockTenantRepo{})
		_, err := svc.DeleteURI(cUUID, tenantID, uriUUID, actorUUID)
		require.Error(t, err)
	})
}

func TestClientService_AddClientAPIs_EdgeCases(t *testing.T) {
	cUUID := uuid.New()
	apiUUID := uuid.New()
	tenantID := int64(1)

	t.Run("FindByClientAndAPI error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Client, error) {
				return &model.Client{ClientID: 1, IdentityProvider: &model.IdentityProvider{TenantID: tenantID}}, nil
			},
		}
		apiRepo := &mockAPIRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.API, error) { return &model.API{APIID: 1}, nil },
		}
		caRepo := &mockClientAPIRepo{
			findByClientAndAPIFn: func(_, _ int64) (*model.ClientAPI, error) { return nil, errors.New("db err") },
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, caRepo,
			apiRepo, &mockUserRepo{}, &mockTenantRepo{})
		err := svc.AddClientAPIs(tenantID, cUUID, []uuid.UUID{apiUUID})
		require.Error(t, err)
	})

	t.Run("findByUUID API error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Client, error) {
				return &model.Client{ClientID: 1, IdentityProvider: &model.IdentityProvider{TenantID: tenantID}}, nil
			},
		}
		apiRepo := &mockAPIRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.API, error) { return nil, errors.New("api err") },
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{},
			apiRepo, &mockUserRepo{}, &mockTenantRepo{})
		err := svc.AddClientAPIs(tenantID, cUUID, []uuid.UUID{apiUUID})
		require.Error(t, err)
	})

	t.Run("findByUUID client error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Client, error) { return nil, errors.New("db err") },
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{},
			&mockAPIRepo{}, &mockUserRepo{}, &mockTenantRepo{})
		err := svc.AddClientAPIs(tenantID, cUUID, []uuid.UUID{apiUUID})
		require.Error(t, err)
	})

	t.Run("Create unique constraint error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Client, error) {
				return &model.Client{ClientID: 1, IdentityProvider: &model.IdentityProvider{TenantID: tenantID}}, nil
			},
		}
		apiRepo := &mockAPIRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.API, error) { return &model.API{APIID: 1}, nil },
		}
		caRepo := &mockClientAPIRepo{
			createFn: func(_ *model.ClientAPI) (*model.ClientAPI, error) {
				return nil, errors.New("uq_client_apis_client_api violation")
			},
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, caRepo,
			apiRepo, &mockUserRepo{}, &mockTenantRepo{})
		err := svc.AddClientAPIs(tenantID, cUUID, []uuid.UUID{apiUUID})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "already assigned")
	})

	t.Run("Create generic error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Client, error) {
				return &model.Client{ClientID: 1, IdentityProvider: &model.IdentityProvider{TenantID: tenantID}}, nil
			},
		}
		apiRepo := &mockAPIRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.API, error) { return &model.API{APIID: 1}, nil },
		}
		caRepo := &mockClientAPIRepo{
			createFn: func(_ *model.ClientAPI) (*model.ClientAPI, error) {
				return nil, errors.New("generic db error")
			},
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, caRepo,
			apiRepo, &mockUserRepo{}, &mockTenantRepo{})
		err := svc.AddClientAPIs(tenantID, cUUID, []uuid.UUID{apiUUID})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "generic db error")
	})
}

func TestClientService_RemoveClientAPI_EdgeCases(t *testing.T) {
	cUUID := uuid.New()
	apiUUID := uuid.New()
	tenantID := int64(1)

	t.Run("findByUUID error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Client, error) { return nil, errors.New("db err") },
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{},
			&mockAPIRepo{}, &mockUserRepo{}, &mockTenantRepo{})
		err := svc.RemoveClientAPI(tenantID, cUUID, apiUUID)
		require.Error(t, err)
	})
}

func TestClientService_GetClientAPIPermissions_EdgeCases(t *testing.T) {
	cUUID := uuid.New()
	apiUUID := uuid.New()
	tenantID := int64(1)

	t.Run("findByClientUUIDAndAPIUUID error", func(t *testing.T) {
		caRepo := &mockClientAPIRepo{
			findByClientUUIDAndAPIUUIDFn: func(_, _ uuid.UUID) (*model.ClientAPI, error) {
				return nil, errors.New("db err")
			},
		}
		svc := buildFullClientService(t, &mockClientRepo{}, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, caRepo, &mockAPIRepo{}, &mockUserRepo{}, &mockTenantRepo{})
		_, err := svc.GetClientAPIPermissions(tenantID, cUUID, apiUUID)
		require.Error(t, err)
	})

	t.Run("client findByUUID error", func(t *testing.T) {
		caRepo := &mockClientAPIRepo{
			findByClientUUIDAndAPIUUIDFn: func(_, _ uuid.UUID) (*model.ClientAPI, error) {
				return &model.ClientAPI{ClientAPIID: 1}, nil
			},
		}
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Client, error) { return nil, errors.New("db err") },
		}
		svc := buildFullClientService(t, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, caRepo, &mockAPIRepo{}, &mockUserRepo{}, &mockTenantRepo{})
		_, err := svc.GetClientAPIPermissions(tenantID, cUUID, apiUUID)
		require.Error(t, err)
	})

	t.Run("FindByClientAPIID error", func(t *testing.T) {
		caRepo := &mockClientAPIRepo{
			findByClientUUIDAndAPIUUIDFn: func(_, _ uuid.UUID) (*model.ClientAPI, error) {
				return &model.ClientAPI{ClientAPIID: 1}, nil
			},
		}
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Client, error) {
				return &model.Client{ClientID: 1, IdentityProvider: &model.IdentityProvider{TenantID: tenantID}}, nil
			},
		}
		cpRepo := &mockClientPermissionRepo{
			findByClientAPIIDFn: func(_ int64) ([]model.ClientPermission, error) { return nil, errors.New("db err") },
		}
		svc := buildFullClientService(t, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, cpRepo, caRepo, &mockAPIRepo{}, &mockUserRepo{}, &mockTenantRepo{})
		_, err := svc.GetClientAPIPermissions(tenantID, cUUID, apiUUID)
		require.Error(t, err)
	})
}

func TestClientService_AddClientAPIPermissions_EdgeCases(t *testing.T) {
	cUUID := uuid.New()
	apiUUID := uuid.New()
	permUUID := uuid.New()
	tenantID := int64(1)

	t.Run("findByUUID client error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Client, error) { return nil, errors.New("db err") },
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{},
			&mockAPIRepo{}, &mockUserRepo{}, &mockTenantRepo{})
		err := svc.AddClientAPIPermissions(tenantID, cUUID, apiUUID, []uuid.UUID{permUUID})
		require.Error(t, err)
	})

	t.Run("findByClientUUIDAndAPIUUID error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Client, error) {
				return &model.Client{ClientID: 1, IdentityProvider: &model.IdentityProvider{TenantID: tenantID}}, nil
			},
		}
		caRepo := &mockClientAPIRepo{
			findByClientUUIDAndAPIUUIDFn: func(_, _ uuid.UUID) (*model.ClientAPI, error) {
				return nil, errors.New("db err")
			},
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, caRepo,
			&mockAPIRepo{}, &mockUserRepo{}, &mockTenantRepo{})
		err := svc.AddClientAPIPermissions(tenantID, cUUID, apiUUID, []uuid.UUID{permUUID})
		require.Error(t, err)
	})

	t.Run("findByUUID permission error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Client, error) {
				return &model.Client{ClientID: 1, IdentityProvider: &model.IdentityProvider{TenantID: tenantID}}, nil
			},
		}
		caRepo := &mockClientAPIRepo{
			findByClientUUIDAndAPIUUIDFn: func(_, _ uuid.UUID) (*model.ClientAPI, error) {
				return &model.ClientAPI{ClientAPIID: 1}, nil
			},
		}
		permRepo := &mockPermissionRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Permission, error) { return nil, errors.New("perm err") },
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			permRepo, &mockClientPermissionRepo{}, caRepo,
			&mockAPIRepo{}, &mockUserRepo{}, &mockTenantRepo{})
		err := svc.AddClientAPIPermissions(tenantID, cUUID, apiUUID, []uuid.UUID{permUUID})
		require.Error(t, err)
	})

	t.Run("FindByClientAPIAndPermission error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Client, error) {
				return &model.Client{ClientID: 1, IdentityProvider: &model.IdentityProvider{TenantID: tenantID}}, nil
			},
		}
		caRepo := &mockClientAPIRepo{
			findByClientUUIDAndAPIUUIDFn: func(_, _ uuid.UUID) (*model.ClientAPI, error) {
				return &model.ClientAPI{ClientAPIID: 1}, nil
			},
		}
		permRepo := &mockPermissionRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Permission, error) {
				return &model.Permission{PermissionID: 1}, nil
			},
		}
		cpRepo := &mockClientPermissionRepo{
			findByClientAPIAndPermissionFn: func(_, _ int64) (*model.ClientPermission, error) {
				return nil, errors.New("db err")
			},
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			permRepo, cpRepo, caRepo,
			&mockAPIRepo{}, &mockUserRepo{}, &mockTenantRepo{})
		err := svc.AddClientAPIPermissions(tenantID, cUUID, apiUUID, []uuid.UUID{permUUID})
		require.Error(t, err)
	})

	t.Run("Create unique constraint error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Client, error) {
				return &model.Client{ClientID: 1, IdentityProvider: &model.IdentityProvider{TenantID: tenantID}}, nil
			},
		}
		caRepo := &mockClientAPIRepo{
			findByClientUUIDAndAPIUUIDFn: func(_, _ uuid.UUID) (*model.ClientAPI, error) {
				return &model.ClientAPI{ClientAPIID: 1}, nil
			},
		}
		permRepo := &mockPermissionRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Permission, error) {
				return &model.Permission{PermissionID: 1}, nil
			},
		}
		cpRepo := &mockClientPermissionRepo{
			findByClientAPIAndPermissionFn: func(_, _ int64) (*model.ClientPermission, error) {
				return nil, nil
			},
			createFn: func(_ *model.ClientPermission) (*model.ClientPermission, error) {
				return nil, errors.New("uq_client_permissions_client_permission violation")
			},
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			permRepo, cpRepo, caRepo,
			&mockAPIRepo{}, &mockUserRepo{}, &mockTenantRepo{})
		err := svc.AddClientAPIPermissions(tenantID, cUUID, apiUUID, []uuid.UUID{permUUID})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "already assigned")
	})

	t.Run("Create generic error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Client, error) {
				return &model.Client{ClientID: 1, IdentityProvider: &model.IdentityProvider{TenantID: tenantID}}, nil
			},
		}
		caRepo := &mockClientAPIRepo{
			findByClientUUIDAndAPIUUIDFn: func(_, _ uuid.UUID) (*model.ClientAPI, error) {
				return &model.ClientAPI{ClientAPIID: 1}, nil
			},
		}
		permRepo := &mockPermissionRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Permission, error) {
				return &model.Permission{PermissionID: 1}, nil
			},
		}
		cpRepo := &mockClientPermissionRepo{
			findByClientAPIAndPermissionFn: func(_, _ int64) (*model.ClientPermission, error) {
				return nil, nil
			},
			createFn: func(_ *model.ClientPermission) (*model.ClientPermission, error) {
				return nil, errors.New("generic db error")
			},
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			permRepo, cpRepo, caRepo,
			&mockAPIRepo{}, &mockUserRepo{}, &mockTenantRepo{})
		err := svc.AddClientAPIPermissions(tenantID, cUUID, apiUUID, []uuid.UUID{permUUID})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "generic db error")
	})
}

func TestClientService_RemoveClientAPIPermission_EdgeCases(t *testing.T) {
	cUUID := uuid.New()
	apiUUID := uuid.New()
	permUUID := uuid.New()
	tenantID := int64(1)

	t.Run("findByUUID client error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Client, error) { return nil, errors.New("db err") },
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{},
			&mockAPIRepo{}, &mockUserRepo{}, &mockTenantRepo{})
		err := svc.RemoveClientAPIPermission(tenantID, cUUID, apiUUID, permUUID)
		require.Error(t, err)
	})

	t.Run("findByClientUUIDAndAPIUUID error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Client, error) {
				return &model.Client{ClientID: 1, IdentityProvider: &model.IdentityProvider{TenantID: tenantID}}, nil
			},
		}
		caRepo := &mockClientAPIRepo{
			findByClientUUIDAndAPIUUIDFn: func(_, _ uuid.UUID) (*model.ClientAPI, error) {
				return nil, errors.New("db err")
			},
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, caRepo,
			&mockAPIRepo{}, &mockUserRepo{}, &mockTenantRepo{})
		err := svc.RemoveClientAPIPermission(tenantID, cUUID, apiUUID, permUUID)
		require.Error(t, err)
	})

	t.Run("findByUUID permission error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Client, error) {
				return &model.Client{ClientID: 1, IdentityProvider: &model.IdentityProvider{TenantID: tenantID}}, nil
			},
		}
		caRepo := &mockClientAPIRepo{
			findByClientUUIDAndAPIUUIDFn: func(_, _ uuid.UUID) (*model.ClientAPI, error) {
				return &model.ClientAPI{ClientAPIID: 1}, nil
			},
		}
		permRepo := &mockPermissionRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Permission, error) { return nil, errors.New("db err") },
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			permRepo, &mockClientPermissionRepo{}, caRepo,
			&mockAPIRepo{}, &mockUserRepo{}, &mockTenantRepo{})
		err := svc.RemoveClientAPIPermission(tenantID, cUUID, apiUUID, permUUID)
		require.Error(t, err)
	})

	t.Run("RemoveByClientAPIAndPermission error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Client, error) {
				return &model.Client{ClientID: 1, IdentityProvider: &model.IdentityProvider{TenantID: tenantID}}, nil
			},
		}
		caRepo := &mockClientAPIRepo{
			findByClientUUIDAndAPIUUIDFn: func(_, _ uuid.UUID) (*model.ClientAPI, error) {
				return &model.ClientAPI{ClientAPIID: 1}, nil
			},
		}
		permRepo := &mockPermissionRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Permission, error) {
				return &model.Permission{PermissionID: 1}, nil
			},
		}
		cpRepo := &mockClientPermissionRepo{
			removeByClientAPIAndPermissionFn: func(_, _ int64) error { return errors.New("del err") },
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			permRepo, cpRepo, caRepo,
			&mockAPIRepo{}, &mockUserRepo{}, &mockTenantRepo{})
		err := svc.RemoveClientAPIPermission(tenantID, cUUID, apiUUID, permUUID)
		require.Error(t, err)
	})
}
