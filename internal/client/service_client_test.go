package client

import (
	"context"
	"errors"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/authevent"
	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func buildClientService(t *testing.T, clientRepo *mockClientRepo, idpRepo *mockIdentityProviderRepo, userRepo *mockUserRepo) ClientService {
	t.Helper()
	db, _ := newMockGormDB(t)
	return NewClientService(db, clientRepo, &mockClientURIRepo{}, idpRepo,
		&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{}, &mockClientRoleRepo{}, &mockRoleRepo{},
		&mockAPIRepo{}, userRepo, &mockTenantRepo{}, nil, nil)
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
	authEventSvc authevent.AuthEventService,
) ClientService {
	t.Helper()
	db, _ := newMockGormDB(t)
	return NewClientService(db, clientRepo, clientURIRepo, idpRepo, permRepo, cpRepo, caRepo, &mockClientRoleRepo{}, &mockRoleRepo{}, apiRepo, userRepo, tenantRepo, authEventSvc, nil)
}

func clientWithIDP(tenantID int64) *Client {
	idp := IdentityProvider{
		IdentityProviderID: 1,
		TenantID:           tenantID,
		Tenant:             &Tenant{TenantID: tenantID, IsSystem: true},
	}
	connections := []ClientIdentityProvider{
		{IdentityProviderID: idp.IdentityProviderID, IdentityProvider: &idp, IsDefault: true, Enabled: boolPtrTest(true)},
	}
	return &Client{
		ClientID:           1,
		ClientUUID:         uuid.New(),
		Name:               "test",
		TenantID:           tenantID,
		Status:             shared.StatusActive,
		IdentityProviderID: idp.IdentityProviderID,
		IdentityProvider:   &idp,
		ConnectedProviders: &connections,
	}
}

func actorUser(tenantID int64) *User {
	return &User{
		UserID: 1,
		UserIdentities: []UserIdentity{
			{TenantID: tenantID, Tenant: &Tenant{TenantID: tenantID, IsSystem: true}},
		},
	}
}

// actorUserRepo returns a user repo whose actor holds an identity in tenantID, which
// is what requireActorTenantAccess demands on every grant mutation.
func actorUserRepo(tenantID int64) *mockUserRepo {
	return &mockUserRepo{
		findByUUIDFn: func(_ any, _ ...string) (*User, error) { return actorUser(tenantID), nil },
	}
}

func TestClientService_GetPublicConsoleByTenantIdentifier(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		identifier := "console-client"
		clientRepo := &mockClientRepo{
			findSystemByTenantIdentifierNameFn: func(tenantIdentifier, name string) (*Client, error) {
				assert.Equal(t, "acme", tenantIdentifier)
				assert.Equal(t, shared.SystemClientNameAuthConsole, name)
				return &Client{
					Name:        shared.SystemClientNameAuthConsole,
					DisplayName: "Maintainerd Auth Console",
					ClientType:  shared.ClientTypeSPA,
					Identifier:  &identifier,
					Status:      shared.StatusActive,
					Tenant:      &Tenant{Name: "acme"},
				}, nil
			},
		}
		svc := buildClientService(t, clientRepo, &mockIdentityProviderRepo{}, &mockUserRepo{})

		result, err := svc.(*clientService).GetPublicConsoleByTenantIdentifier(context.Background(), " acme ")

		require.NoError(t, err)
		assert.Equal(t, identifier, result.ClientID)
		assert.Equal(t, "acme", result.TenantIdentifier)
	})

	t.Run("missing client returns not found", func(t *testing.T) {
		svc := buildClientService(t, &mockClientRepo{}, &mockIdentityProviderRepo{}, &mockUserRepo{})

		result, err := svc.(*clientService).GetPublicConsoleByTenantIdentifier(context.Background(), "acme")

		require.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestClientService_GetPublicIdentityByTenantIdentifier(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		identifier := "identity-client"
		clientRepo := &mockClientRepo{
			findSystemByTenantIdentifierNameFn: func(tenantIdentifier, name string) (*Client, error) {
				assert.Equal(t, "acme", tenantIdentifier)
				assert.Equal(t, shared.SystemClientNameAuthIdentity, name)
				return &Client{
					Name:        shared.SystemClientNameAuthIdentity,
					DisplayName: "Maintainerd Auth Identity",
					ClientType:  shared.ClientTypeSPA,
					Identifier:  &identifier,
					Status:      shared.StatusActive,
					Tenant:      &Tenant{Name: "acme"},
				}, nil
			},
		}
		svc := buildClientService(t, clientRepo, &mockIdentityProviderRepo{}, &mockUserRepo{})

		result, err := svc.(*clientService).GetPublicIdentityByTenantIdentifier(context.Background(), " acme ")

		require.NoError(t, err)
		assert.Equal(t, identifier, result.ClientID)
		assert.Equal(t, "acme", result.TenantIdentifier)
	})

	t.Run("missing client returns not found", func(t *testing.T) {
		svc := buildClientService(t, &mockClientRepo{}, &mockIdentityProviderRepo{}, &mockUserRepo{})

		result, err := svc.(*clientService).GetPublicIdentityByTenantIdentifier(context.Background(), "acme")

		require.Error(t, err)
		assert.Nil(t, result)
	})
}

// ===========================================================================
// Get
// ===========================================================================

func TestClientService_Get(t *testing.T) {
	idpUUID := uuid.New().String()

	t.Run("idp not found - returns empty result", func(t *testing.T) {
		idpRepo := &mockIdentityProviderRepo{
			findByUUIDFn: func(_ any, _ ...string) (*IdentityProvider, error) { return nil, nil },
		}
		svc := buildClientService(t, &mockClientRepo{}, idpRepo, &mockUserRepo{})
		res, err := svc.Get(context.Background(), ClientServiceGetFilter{IdentityProviderUUID: &idpUUID, TenantID: 1})
		require.NoError(t, err)
		assert.NotNil(t, res)
		assert.Empty(t, res.Data)
	})

	t.Run("paginate error", func(t *testing.T) {
		clientRepo := &mockClientRepo{
			findPaginatedFn: func(_ ClientRepositoryGetFilter) (*PaginationResult[Client], error) {
				return nil, errors.New("db error")
			},
		}
		svc := buildClientService(t, clientRepo, &mockIdentityProviderRepo{}, &mockUserRepo{})
		_, err := svc.Get(context.Background(), ClientServiceGetFilter{TenantID: 1})
		require.Error(t, err)
	})

	t.Run("success with no filter - returns empty list", func(t *testing.T) {
		svc := buildClientService(t, &mockClientRepo{}, &mockIdentityProviderRepo{}, &mockUserRepo{})
		res, err := svc.Get(context.Background(), ClientServiceGetFilter{TenantID: 1})
		require.NoError(t, err)
		assert.NotNil(t, res)
	})

	t.Run("success with data and IDP preloaded", func(t *testing.T) {
		c := clientWithIDP(1)
		uris := []ClientURI{{ClientURIUUID: uuid.New(), URI: "https://cb.example.com", Type: "redirect"}}
		c.ClientURIs = &uris
		clientRepo := &mockClientRepo{
			findPaginatedFn: func(_ ClientRepositoryGetFilter) (*PaginationResult[Client], error) {
				return &PaginationResult[Client]{
					Data: []Client{*c}, Total: 1, Page: 1, Limit: 10, TotalPages: 1,
				}, nil
			},
		}
		idpRepo := &mockIdentityProviderRepo{
			findByUUIDFn: func(_ any, _ ...string) (*IdentityProvider, error) {
				return &IdentityProvider{IdentityProviderID: 1}, nil
			},
		}
		svc := buildClientService(t, clientRepo, idpRepo, &mockUserRepo{})
		res, err := svc.Get(context.Background(), ClientServiceGetFilter{TenantID: 1, IdentityProviderUUID: &idpUUID})
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
		_, err := svc.GetByUUID(context.Background(), cUUID, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("repo error returns error", func(t *testing.T) {
		clientRepo := &mockClientRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*Client, error) {
				return nil, errors.New("db error")
			},
		}
		svc := buildClientService(t, clientRepo, &mockIdentityProviderRepo{}, &mockUserRepo{})
		_, err := svc.GetByUUID(context.Background(), cUUID, 1)
		require.Error(t, err)
	})

	t.Run("success", func(t *testing.T) {
		clientRepo := &mockClientRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*Client, error) {
				return &Client{ClientUUID: cUUID}, nil
			},
		}
		svc := buildClientService(t, clientRepo, &mockIdentityProviderRepo{}, &mockUserRepo{})
		res, err := svc.GetByUUID(context.Background(), cUUID, 1)
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
		_, err := svc.GetSecretByUUID(context.Background(), cUUID, 1)
		require.Error(t, err)
	})

	t.Run("repo error → propagated", func(t *testing.T) {
		clientRepo := &mockClientRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*Client, error) {
				return nil, errors.New("db err")
			},
		}
		svc := buildClientService(t, clientRepo, &mockIdentityProviderRepo{}, &mockUserRepo{})
		_, err := svc.GetSecretByUUID(context.Background(), cUUID, 1)
		require.Error(t, err)
	})

	t.Run("always errors — secret not retrievable after creation", func(t *testing.T) {
		id := "client-id"
		secret := "client-secret"
		clientRepo := &mockClientRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*Client, error) {
				return &Client{Identifier: &id, SecretHash: &secret}, nil
			},
		}
		svc := buildClientService(t, clientRepo, &mockIdentityProviderRepo{}, &mockUserRepo{})
		_, err := svc.GetSecretByUUID(context.Background(), cUUID, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "rotate-secret")
	})
}

// ===========================================================================
// GetConfigByUUID
// ===========================================================================

func TestClientService_GetConfigByUUID(t *testing.T) {
	cUUID := uuid.New()

	t.Run("not found → error", func(t *testing.T) {
		svc := buildClientService(t, &mockClientRepo{}, &mockIdentityProviderRepo{}, &mockUserRepo{})
		_, err := svc.GetConfigByUUID(context.Background(), cUUID, 1)
		require.Error(t, err)
	})

	t.Run("repo error → propagated", func(t *testing.T) {
		clientRepo := &mockClientRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*Client, error) {
				return nil, errors.New("db err")
			},
		}
		svc := buildClientService(t, clientRepo, &mockIdentityProviderRepo{}, &mockUserRepo{})
		_, err := svc.GetConfigByUUID(context.Background(), cUUID, 1)
		require.Error(t, err)
	})

	t.Run("success", func(t *testing.T) {
		clientRepo := &mockClientRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*Client, error) {
				return &Client{Config: []byte(`{"key":"value"}`)}, nil
			},
		}
		svc := buildClientService(t, clientRepo, &mockIdentityProviderRepo{}, &mockUserRepo{})
		cfg, err := svc.GetConfigByUUID(context.Background(), cUUID, 1)
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
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{}, &mockClientRoleRepo{}, &mockRoleRepo{},
			&mockAPIRepo{}, actorUserRepo(tenantID), &mockTenantRepo{}, nil, nil)
		_, err := svc.Create(context.Background(), tenantID, "test", "Test", "public", "example.com", nil, "active", false, "not-a-valid-uuid", nil, true, nil, nil, nil, nil, actorUUID, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid identity provider UUID")
	})

	t.Run("identity provider not found", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		idpRepo := &mockIdentityProviderRepo{
			findByUUIDFn: func(_ any, _ ...string) (*IdentityProvider, error) { return nil, nil },
		}
		svc := NewClientService(gormDB, &mockClientRepo{}, &mockClientURIRepo{}, idpRepo,
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{}, &mockClientRoleRepo{}, &mockRoleRepo{},
			&mockAPIRepo{}, actorUserRepo(tenantID), &mockTenantRepo{}, nil, nil)
		_, err := svc.Create(context.Background(), tenantID, "test", "Test", "public", "example.com", nil, "active", false, uuid.New().String(), nil, true, nil, nil, nil, nil, actorUUID, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "identity provider not found")
	})

	t.Run("actor user not found", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		idpRepo := &mockIdentityProviderRepo{
			findByUUIDFn: func(_ any, _ ...string) (*IdentityProvider, error) {
				return &IdentityProvider{IdentityProviderID: 1, Tenant: &Tenant{TenantID: tenantID, IsSystem: true}}, nil
			},
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*User, error) { return nil, nil },
		}
		svc := NewClientService(gormDB, &mockClientRepo{}, &mockClientURIRepo{}, idpRepo,
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{}, &mockClientRoleRepo{}, &mockRoleRepo{},
			&mockAPIRepo{}, userRepo, &mockTenantRepo{}, nil, nil)
		_, err := svc.Create(context.Background(), tenantID, "test", "Test", "public", "example.com", nil, "active", false, uuid.New().String(), nil, true, nil, nil, nil, nil, actorUUID, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "actor user not found")
	})

	t.Run("tenant access denied", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		idpRepo := &mockIdentityProviderRepo{
			findByUUIDFn: func(_ any, _ ...string) (*IdentityProvider, error) {
				return &IdentityProvider{IdentityProviderID: 1, Tenant: &Tenant{TenantID: tenantID, IsSystem: false}}, nil
			},
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*User, error) {
				return &User{UserID: 1, UserIdentities: []UserIdentity{
					{TenantID: 999, Tenant: &Tenant{TenantID: 999, IsSystem: false}},
				}}, nil
			},
		}
		svc := NewClientService(gormDB, &mockClientRepo{}, &mockClientURIRepo{}, idpRepo,
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{}, &mockClientRoleRepo{}, &mockRoleRepo{},
			&mockAPIRepo{}, userRepo, &mockTenantRepo{}, nil, nil)
		_, err := svc.Create(context.Background(), tenantID, "test", "Test", "public", "example.com", nil, "active", false, uuid.New().String(), nil, true, nil, nil, nil, nil, actorUUID, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "access denied")
	})

	t.Run("client name already exists", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		idpRepo := &mockIdentityProviderRepo{
			findByUUIDFn: func(_ any, _ ...string) (*IdentityProvider, error) {
				return &IdentityProvider{IdentityProviderID: 1, Tenant: &Tenant{TenantID: tenantID, IsSystem: true}}, nil
			},
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*User, error) { return actorUser(tenantID), nil },
		}
		clientRepo := &mockClientRepo{
			findByNameAndIdentityProviderFn: func(_ string, _ int64, _ int64) (*Client, error) {
				return &Client{}, nil
			},
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, idpRepo,
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{}, &mockClientRoleRepo{}, &mockRoleRepo{},
			&mockAPIRepo{}, userRepo, &mockTenantRepo{}, nil, nil)
		_, err := svc.Create(context.Background(), tenantID, "test", "Test", "public", "example.com", nil, "active", false, uuid.New().String(), nil, true, nil, nil, nil, nil, actorUUID, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
	})

	t.Run("findByNameAndIdentityProvider error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		idpRepo := &mockIdentityProviderRepo{
			findByUUIDFn: func(_ any, _ ...string) (*IdentityProvider, error) {
				return &IdentityProvider{IdentityProviderID: 1, Tenant: &Tenant{TenantID: tenantID, IsSystem: true}}, nil
			},
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*User, error) { return actorUser(tenantID), nil },
		}
		clientRepo := &mockClientRepo{
			findByNameAndIdentityProviderFn: func(_ string, _ int64, _ int64) (*Client, error) {
				return nil, errors.New("db err")
			},
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, idpRepo,
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{}, &mockClientRoleRepo{}, &mockRoleRepo{},
			&mockAPIRepo{}, userRepo, &mockTenantRepo{}, nil, nil)
		_, err := svc.Create(context.Background(), tenantID, "test", "Test", "public", "example.com", nil, "active", false, uuid.New().String(), nil, true, nil, nil, nil, nil, actorUUID, nil)
		require.Error(t, err)
	})

	t.Run("GenerateIdentifier failure on clientId", func(t *testing.T) {
		orig := generateClientIdentifier
		defer func() { generateClientIdentifier = orig }()
		generateClientIdentifier = func(int) (string, error) { return "", errors.New("rand failure") }

		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		idpRepo := &mockIdentityProviderRepo{
			findByUUIDFn: func(_ any, _ ...string) (*IdentityProvider, error) {
				return &IdentityProvider{IdentityProviderID: 1, Tenant: &Tenant{TenantID: tenantID, IsSystem: true}}, nil
			},
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*User, error) { return actorUser(tenantID), nil },
		}
		svc := NewClientService(gormDB, &mockClientRepo{}, &mockClientURIRepo{}, idpRepo,
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{}, &mockClientRoleRepo{}, &mockRoleRepo{},
			&mockAPIRepo{}, userRepo, &mockTenantRepo{}, nil, nil)
		_, err := svc.Create(context.Background(), tenantID, "test", "Test", "public", "example.com", nil, "active", false, uuid.New().String(), nil, true, nil, nil, nil, nil, actorUUID, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "rand failure")
	})

	t.Run("GenerateIdentifier failure on clientSecret", func(t *testing.T) {
		orig := generateClientIdentifier
		defer func() { generateClientIdentifier = orig }()
		callCount := 0
		generateClientIdentifier = func(n int) (string, error) {
			callCount++
			if callCount == 1 {
				return "fake-client-id", nil
			}
			return "", errors.New("rand failure on secret")
		}

		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		idpRepo := &mockIdentityProviderRepo{
			findByUUIDFn: func(_ any, _ ...string) (*IdentityProvider, error) {
				return &IdentityProvider{IdentityProviderID: 1, Tenant: &Tenant{TenantID: tenantID, IsSystem: true}}, nil
			},
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*User, error) { return actorUser(tenantID), nil },
		}
		svc := NewClientService(gormDB, &mockClientRepo{}, &mockClientURIRepo{}, idpRepo,
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{}, &mockClientRoleRepo{}, &mockRoleRepo{},
			&mockAPIRepo{}, userRepo, &mockTenantRepo{}, nil, nil)
		_, err := svc.Create(context.Background(), tenantID, "test", "Test", "public", "example.com", nil, "active", false, uuid.New().String(), nil, true, nil, nil, nil, nil, actorUUID, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "rand failure on secret")
	})

	t.Run("createOrUpdate error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		idpRepo := &mockIdentityProviderRepo{
			findByUUIDFn: func(_ any, _ ...string) (*IdentityProvider, error) {
				return &IdentityProvider{IdentityProviderID: 1, Tenant: &Tenant{TenantID: tenantID, IsSystem: true}}, nil
			},
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*User, error) { return actorUser(tenantID), nil },
		}
		clientRepo := &mockClientRepo{
			createOrUpdateFn: func(_ *Client) (*Client, error) { return nil, errors.New("save err") },
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, idpRepo,
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{}, &mockClientRoleRepo{}, &mockRoleRepo{},
			&mockAPIRepo{}, userRepo, &mockTenantRepo{}, nil, nil)
		_, err := svc.Create(context.Background(), tenantID, "test", "Test", "public", "example.com", nil, "active", false, uuid.New().String(), nil, true, nil, nil, nil, nil, actorUUID, nil)
		require.Error(t, err)
	})

	t.Run("fetch after save error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		idpRepo := &mockIdentityProviderRepo{
			findByUUIDFn: func(_ any, _ ...string) (*IdentityProvider, error) {
				return &IdentityProvider{IdentityProviderID: 1, Tenant: &Tenant{TenantID: tenantID, IsSystem: true}}, nil
			},
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*User, error) { return actorUser(tenantID), nil },
		}
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Client, error) { return nil, errors.New("fetch err") },
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, idpRepo,
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{}, &mockClientRoleRepo{}, &mockRoleRepo{},
			&mockAPIRepo{}, userRepo, &mockTenantRepo{}, nil, nil)
		_, err := svc.Create(context.Background(), tenantID, "test", "Test", "public", "example.com", nil, "active", false, uuid.New().String(), nil, true, nil, nil, nil, nil, actorUUID, nil)
		require.Error(t, err)
	})

	t.Run("success", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		expectClientIdentityProviderConnectionInsert(mock)
		mock.ExpectCommit()
		idpRepo := &mockIdentityProviderRepo{
			findByUUIDFn: func(_ any, _ ...string) (*IdentityProvider, error) {
				return &IdentityProvider{IdentityProviderID: 1, Tenant: &Tenant{TenantID: tenantID, IsSystem: true}}, nil
			},
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*User, error) { return actorUser(tenantID), nil },
		}
		created := clientWithIDP(tenantID)
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Client, error) { return created, nil },
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, idpRepo,
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{}, &mockClientRoleRepo{}, &mockRoleRepo{},
			&mockAPIRepo{}, userRepo, &mockTenantRepo{}, nil, nil)
		res, err := svc.Create(context.Background(), tenantID, "test", "Test", "public", "example.com", nil, "active", false, uuid.New().String(), nil, true, nil, nil, nil, nil, actorUUID, nil)
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
			findByUUIDFn: func(_ any, _ ...string) (*Client, error) { return nil, nil },
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{}, &mockClientRoleRepo{}, &mockRoleRepo{},
			&mockAPIRepo{}, actorUserRepo(tenantID), &mockTenantRepo{}, nil, nil)
		_, err := svc.Update(context.Background(), cUUID, tenantID, "n", "d", "pub", "ex.com", nil, "active", false, nil, testBoolPtr(true), nil, nil, nil, nil, actorUUID, nil, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("actor user not found", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Client, error) { return clientWithIDP(tenantID), nil },
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*User, error) { return nil, nil },
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{}, &mockClientRoleRepo{}, &mockRoleRepo{},
			&mockAPIRepo{}, userRepo, &mockTenantRepo{}, nil, nil)
		_, err := svc.Update(context.Background(), cUUID, tenantID, "n", "d", "pub", "ex.com", nil, "active", false, nil, testBoolPtr(true), nil, nil, nil, nil, actorUUID, nil, nil)
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
			findByUUIDFn: func(_ any, _ ...string) (*Client, error) { return c, nil },
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*User, error) { return actorUser(tenantID), nil },
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{}, &mockClientRoleRepo{}, &mockRoleRepo{},
			&mockAPIRepo{}, userRepo, &mockTenantRepo{}, nil, nil)
		_, err := svc.Update(context.Background(), cUUID, tenantID, "n", "d", "pub", "ex.com", nil, "active", false, nil, testBoolPtr(true), nil, nil, nil, nil, actorUUID, nil, nil)
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
			findByUUIDFn: func(_ any, _ ...string) (*Client, error) { return c, nil },
			findByNameAndIdentityProviderFn: func(_ string, _ int64, _ int64) (*Client, error) {
				other := clientWithIDP(tenantID)
				other.ClientUUID = uuid.New()
				return other, nil
			},
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*User, error) { return actorUser(tenantID), nil },
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{}, &mockClientRoleRepo{}, &mockRoleRepo{},
			&mockAPIRepo{}, userRepo, &mockTenantRepo{}, nil, nil)
		_, err := svc.Update(context.Background(), cUUID, tenantID, "new-name", "d", "pub", "ex.com", nil, "active", false, nil, testBoolPtr(true), nil, nil, nil, nil, actorUUID, nil, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
	})

	t.Run("save error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		c := clientWithIDP(tenantID)
		clientRepo := &mockClientRepo{
			findByUUIDFn:     func(_ any, _ ...string) (*Client, error) { return c, nil },
			createOrUpdateFn: func(_ *Client) (*Client, error) { return nil, errors.New("save err") },
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*User, error) { return actorUser(tenantID), nil },
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{}, &mockClientRoleRepo{}, &mockRoleRepo{},
			&mockAPIRepo{}, userRepo, &mockTenantRepo{}, nil, nil)
		_, err := svc.Update(context.Background(), cUUID, tenantID, "test", "d", "pub", "ex.com", nil, "active", false, nil, testBoolPtr(true), nil, nil, nil, nil, actorUUID, nil, nil)
		require.Error(t, err)
	})

	t.Run("success", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		c := clientWithIDP(tenantID)
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Client, error) { return c, nil },
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*User, error) { return actorUser(tenantID), nil },
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{}, &mockClientRoleRepo{}, &mockRoleRepo{},
			&mockAPIRepo{}, userRepo, &mockTenantRepo{}, nil, nil)
		res, err := svc.Update(context.Background(), cUUID, tenantID, "test", "Test", "pub", "ex.com", nil, "active", false, nil, testBoolPtr(true), nil, nil, nil, nil, actorUUID, nil, nil)
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
			findByUUIDFn: func(_ any, _ ...string) (*Client, error) { return nil, nil },
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{}, &mockClientRoleRepo{}, &mockRoleRepo{},
			&mockAPIRepo{}, actorUserRepo(tenantID), &mockTenantRepo{}, nil, nil)
		_, err := svc.SetStatusByUUID(context.Background(), cUUID, tenantID, "inactive", actorUUID)
		require.Error(t, err)
	})

	t.Run("actor not found", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Client, error) { return clientWithIDP(tenantID), nil },
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*User, error) { return nil, nil },
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{}, &mockClientRoleRepo{}, &mockRoleRepo{},
			&mockAPIRepo{}, userRepo, &mockTenantRepo{}, nil, nil)
		_, err := svc.SetStatusByUUID(context.Background(), cUUID, tenantID, "inactive", actorUUID)
		require.Error(t, err)
	})

	t.Run("default client cannot be updated", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		c := clientWithIDP(tenantID)
		c.IsDefault = true
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Client, error) { return c, nil },
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*User, error) { return actorUser(tenantID), nil },
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{}, &mockClientRoleRepo{}, &mockRoleRepo{},
			&mockAPIRepo{}, userRepo, &mockTenantRepo{}, nil, nil)
		_, err := svc.SetStatusByUUID(context.Background(), cUUID, tenantID, "inactive", actorUUID)
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
			findByUUIDFn: func(_ any, _ ...string) (*Client, error) { return c, nil },
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*User, error) { return actorUser(tenantID), nil },
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{}, &mockClientRoleRepo{}, &mockRoleRepo{},
			&mockAPIRepo{}, userRepo, &mockTenantRepo{}, nil, nil)
		_, err := svc.SetStatusByUUID(context.Background(), cUUID, tenantID, "inactive", actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "system")
	})

	t.Run("save error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDFn:     func(_ any, _ ...string) (*Client, error) { return clientWithIDP(tenantID), nil },
			createOrUpdateFn: func(_ *Client) (*Client, error) { return nil, errors.New("save err") },
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*User, error) { return actorUser(tenantID), nil },
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{}, &mockClientRoleRepo{}, &mockRoleRepo{},
			&mockAPIRepo{}, userRepo, &mockTenantRepo{}, nil, nil)
		_, err := svc.SetStatusByUUID(context.Background(), cUUID, tenantID, "inactive", actorUUID)
		require.Error(t, err)
	})

	t.Run("success", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Client, error) { return clientWithIDP(tenantID), nil },
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*User, error) { return actorUser(tenantID), nil },
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{}, &mockClientRoleRepo{}, &mockRoleRepo{},
			&mockAPIRepo{}, userRepo, &mockTenantRepo{}, nil, nil)
		res, err := svc.SetStatusByUUID(context.Background(), cUUID, tenantID, "inactive", actorUUID)
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
			findByUUIDFn: func(_ any, _ ...string) (*Client, error) { return nil, nil },
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{}, &mockClientRoleRepo{}, &mockRoleRepo{},
			&mockAPIRepo{}, actorUserRepo(tenantID), &mockTenantRepo{}, nil, nil)
		_, err := svc.DeleteByUUID(context.Background(), cUUID, tenantID, actorUUID)
		require.Error(t, err)
	})

	t.Run("actor not found", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Client, error) { return clientWithIDP(tenantID), nil },
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*User, error) { return nil, nil },
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{}, &mockClientRoleRepo{}, &mockRoleRepo{},
			&mockAPIRepo{}, userRepo, &mockTenantRepo{}, nil, nil)
		_, err := svc.DeleteByUUID(context.Background(), cUUID, tenantID, actorUUID)
		require.Error(t, err)
	})

	t.Run("default client cannot be deleted", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		c := clientWithIDP(tenantID)
		c.IsDefault = true
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Client, error) { return c, nil },
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*User, error) { return actorUser(tenantID), nil },
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{}, &mockClientRoleRepo{}, &mockRoleRepo{},
			&mockAPIRepo{}, userRepo, &mockTenantRepo{}, nil, nil)
		_, err := svc.DeleteByUUID(context.Background(), cUUID, tenantID, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "default")
	})

	t.Run("delete error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDFn:   func(_ any, _ ...string) (*Client, error) { return clientWithIDP(tenantID), nil },
			deleteByUUIDFn: func(_ any) error { return errors.New("del err") },
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*User, error) { return actorUser(tenantID), nil },
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{}, &mockClientRoleRepo{}, &mockRoleRepo{},
			&mockAPIRepo{}, userRepo, &mockTenantRepo{}, nil, nil)
		_, err := svc.DeleteByUUID(context.Background(), cUUID, tenantID, actorUUID)
		require.Error(t, err)
	})

	t.Run("success", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		// The client's children are cleared first: clients is soft-deleted, so the
		// FK ON DELETE CASCADE never fires and these rows would otherwise outlive
		// the client and keep resolving. Soft delete where the model has a
		// DeletedAt (uris, identity-provider connections), hard delete where it
		// does not (permissions, apis, roles).
		mock.ExpectExec(`UPDATE "client_uris" SET "deleted_at"=`).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(`UPDATE "client_identity_providers" SET "deleted_at"=`).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(`DELETE FROM "client_permissions"`).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(`DELETE FROM "client_apis"`).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(`DELETE FROM "client_roles"`).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Client, error) { return clientWithIDP(tenantID), nil },
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*User, error) { return actorUser(tenantID), nil },
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{}, &mockClientRoleRepo{}, &mockRoleRepo{},
			&mockAPIRepo{}, userRepo, &mockTenantRepo{}, nil, nil)
		res, err := svc.DeleteByUUID(context.Background(), cUUID, tenantID, actorUUID)
		require.NoError(t, err)
		assert.NotNil(t, res)
		assert.NoError(t, mock.ExpectationsWereMet())
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
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{}, &mockClientRoleRepo{}, &mockRoleRepo{},
			&mockAPIRepo{}, actorUserRepo(tenantID), &mockTenantRepo{}, nil, nil)
		_, err := svc.CreateURI(context.Background(), cUUID, tenantID, "https://cb.test", "redirect", actorUUID)
		require.Error(t, err)
	})

	t.Run("actor not found", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*Client, error) {
				return &Client{ClientID: 1, TenantID: tenantID}, nil
			},
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*User, error) { return nil, nil },
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{}, &mockClientRoleRepo{}, &mockRoleRepo{},
			&mockAPIRepo{}, userRepo, &mockTenantRepo{}, nil, nil)
		_, err := svc.CreateURI(context.Background(), cUUID, tenantID, "https://cb.test", "redirect", actorUUID)
		require.Error(t, err)
	})

	t.Run("tenant mismatch → access denied", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*Client, error) {
				return &Client{ClientID: 1, TenantID: 999}, nil
			},
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*User, error) { return actorUser(tenantID), nil },
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{}, &mockClientRoleRepo{}, &mockRoleRepo{},
			&mockAPIRepo{}, userRepo, &mockTenantRepo{}, nil, nil)
		_, err := svc.CreateURI(context.Background(), cUUID, tenantID, "https://cb.test", "redirect", actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "access denied")
	})

	t.Run("success", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		clientRepo := &mockClientRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*Client, error) {
				return &Client{ClientID: 1, TenantID: tenantID}, nil
			},
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*User, error) { return actorUser(tenantID), nil },
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{}, &mockClientRoleRepo{}, &mockRoleRepo{},
			&mockAPIRepo{}, userRepo, &mockTenantRepo{}, nil, nil)
		res, err := svc.CreateURI(context.Background(), cUUID, tenantID, "https://cb.test", "redirect", actorUUID)
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
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{}, &mockClientRoleRepo{}, &mockRoleRepo{},
			&mockAPIRepo{}, actorUserRepo(tenantID), &mockTenantRepo{}, nil, nil)
		_, err := svc.UpdateURI(context.Background(), cUUID, tenantID, uriUUID, "https://new.test", "redirect", actorUUID)
		require.Error(t, err)
	})

	t.Run("actor not found", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*Client, error) {
				return &Client{ClientID: 1, TenantID: tenantID}, nil
			},
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*User, error) { return nil, nil },
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{}, &mockClientRoleRepo{}, &mockRoleRepo{},
			&mockAPIRepo{}, userRepo, &mockTenantRepo{}, nil, nil)
		_, err := svc.UpdateURI(context.Background(), cUUID, tenantID, uriUUID, "https://new.test", "redirect", actorUUID)
		require.Error(t, err)
	})

	t.Run("URI not found", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*Client, error) {
				return &Client{ClientID: 1, TenantID: tenantID}, nil
			},
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*User, error) { return actorUser(tenantID), nil },
		}
		uriRepo := &mockClientURIRepo{
			findByUUIDAndTenantIDFn: func(_ string, _ int64) (*ClientURI, error) { return nil, nil },
		}
		svc := NewClientService(gormDB, clientRepo, uriRepo, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{}, &mockClientRoleRepo{}, &mockRoleRepo{},
			&mockAPIRepo{}, userRepo, &mockTenantRepo{}, nil, nil)
		_, err := svc.UpdateURI(context.Background(), cUUID, tenantID, uriUUID, "https://new.test", "redirect", actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "URI not found")
	})

	t.Run("URI belongs to different client", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*Client, error) {
				return &Client{ClientID: 1, TenantID: tenantID}, nil
			},
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*User, error) { return actorUser(tenantID), nil },
		}
		uriRepo := &mockClientURIRepo{
			findByUUIDAndTenantIDFn: func(_ string, _ int64) (*ClientURI, error) {
				return &ClientURI{ClientID: 999}, nil // different client
			},
		}
		svc := NewClientService(gormDB, clientRepo, uriRepo, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{}, &mockClientRoleRepo{}, &mockRoleRepo{},
			&mockAPIRepo{}, userRepo, &mockTenantRepo{}, nil, nil)
		_, err := svc.UpdateURI(context.Background(), cUUID, tenantID, uriUUID, "https://new.test", "redirect", actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "does not belong")
	})

	t.Run("success", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		clientRepo := &mockClientRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*Client, error) {
				return &Client{ClientID: 1, TenantID: tenantID}, nil
			},
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*User, error) { return actorUser(tenantID), nil },
		}
		uriRepo := &mockClientURIRepo{
			findByUUIDAndTenantIDFn: func(_ string, _ int64) (*ClientURI, error) {
				return &ClientURI{ClientID: 1}, nil
			},
		}
		svc := NewClientService(gormDB, clientRepo, uriRepo, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{}, &mockClientRoleRepo{}, &mockRoleRepo{},
			&mockAPIRepo{}, userRepo, &mockTenantRepo{}, nil, nil)
		res, err := svc.UpdateURI(context.Background(), cUUID, tenantID, uriUUID, "https://new.test", "redirect", actorUUID)
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
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{}, &mockClientRoleRepo{}, &mockRoleRepo{},
			&mockAPIRepo{}, actorUserRepo(tenantID), &mockTenantRepo{}, nil, nil)
		_, err := svc.DeleteURI(context.Background(), cUUID, tenantID, uriUUID, actorUUID)
		require.Error(t, err)
	})

	t.Run("actor not found", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*Client, error) {
				return &Client{ClientID: 1, TenantID: tenantID}, nil
			},
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*User, error) { return nil, nil },
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{}, &mockClientRoleRepo{}, &mockRoleRepo{},
			&mockAPIRepo{}, userRepo, &mockTenantRepo{}, nil, nil)
		_, err := svc.DeleteURI(context.Background(), cUUID, tenantID, uriUUID, actorUUID)
		require.Error(t, err)
	})

	t.Run("URI not found", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*Client, error) {
				return &Client{ClientID: 1, TenantID: tenantID}, nil
			},
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*User, error) { return actorUser(tenantID), nil },
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{}, &mockClientRoleRepo{}, &mockRoleRepo{},
			&mockAPIRepo{}, userRepo, &mockTenantRepo{}, nil, nil)
		_, err := svc.DeleteURI(context.Background(), cUUID, tenantID, uriUUID, actorUUID)
		require.Error(t, err)
	})

	t.Run("URI belongs to different client", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*Client, error) {
				return &Client{ClientID: 1, TenantID: tenantID}, nil
			},
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*User, error) { return actorUser(tenantID), nil },
		}
		uriRepo := &mockClientURIRepo{
			findByUUIDAndTenantIDFn: func(_ string, _ int64) (*ClientURI, error) {
				return &ClientURI{ClientID: 999}, nil
			},
		}
		svc := NewClientService(gormDB, clientRepo, uriRepo, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{}, &mockClientRoleRepo{}, &mockRoleRepo{},
			&mockAPIRepo{}, userRepo, &mockTenantRepo{}, nil, nil)
		_, err := svc.DeleteURI(context.Background(), cUUID, tenantID, uriUUID, actorUUID)
		require.Error(t, err)
	})

	t.Run("success", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		clientRepo := &mockClientRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*Client, error) {
				return &Client{ClientID: 1, TenantID: tenantID}, nil
			},
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*User, error) { return actorUser(tenantID), nil },
		}
		uriRepo := &mockClientURIRepo{
			findByUUIDAndTenantIDFn: func(_ string, _ int64) (*ClientURI, error) {
				return &ClientURI{ClientID: 1}, nil
			},
		}
		svc := NewClientService(gormDB, clientRepo, uriRepo, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{}, &mockClientRoleRepo{}, &mockRoleRepo{},
			&mockAPIRepo{}, userRepo, &mockTenantRepo{}, nil, nil)
		res, err := svc.DeleteURI(context.Background(), cUUID, tenantID, uriUUID, actorUUID)
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
			findByClientUUIDFn: func(_ uuid.UUID) ([]ClientAPI, error) { return nil, errors.New("err") },
		}
		clientRepo := &mockClientRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, tenantID int64) (*Client, error) {
				return &Client{ClientID: 1, ClientUUID: cUUID, TenantID: tenantID}, nil
			},
		}
		svc := buildFullClientService(t, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, caRepo, &mockAPIRepo{}, &mockUserRepo{}, &mockTenantRepo{}, nil)
		_, err := svc.GetClientAPIs(context.Background(), 1, cUUID)
		require.Error(t, err)
	})

	t.Run("success with permissions", func(t *testing.T) {
		perm := &Permission{PermissionUUID: uuid.New(), Name: "read"}
		cas := []ClientAPI{{
			ClientAPIUUID: uuid.New(),
			API:           API{APIUUID: uuid.New(), Name: "api1"},
			Permissions:   []ClientPermission{{Permission: perm}},
		}}
		caRepo := &mockClientAPIRepo{
			findByClientUUIDFn: func(_ uuid.UUID) ([]ClientAPI, error) { return cas, nil },
		}
		clientRepo := &mockClientRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, tenantID int64) (*Client, error) {
				return &Client{ClientID: 1, ClientUUID: cUUID, TenantID: tenantID}, nil
			},
		}
		svc := buildFullClientService(t, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, caRepo, &mockAPIRepo{}, &mockUserRepo{}, &mockTenantRepo{}, nil)
		results, err := svc.GetClientAPIs(context.Background(), 1, cUUID)
		require.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Len(t, results[0].Permissions, 1)
	})

	t.Run("permission with nil Permission pointer", func(t *testing.T) {
		cas := []ClientAPI{{
			ClientAPIUUID: uuid.New(),
			API:           API{APIUUID: uuid.New(), Name: "api1"},
			Permissions:   []ClientPermission{{Permission: nil}},
		}}
		caRepo := &mockClientAPIRepo{
			findByClientUUIDFn: func(_ uuid.UUID) ([]ClientAPI, error) { return cas, nil },
		}
		clientRepo := &mockClientRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, tenantID int64) (*Client, error) {
				return &Client{ClientID: 1, ClientUUID: cUUID, TenantID: tenantID}, nil
			},
		}
		svc := buildFullClientService(t, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, caRepo, &mockAPIRepo{}, &mockUserRepo{}, &mockTenantRepo{}, nil)
		results, err := svc.GetClientAPIs(context.Background(), 1, cUUID)
		require.NoError(t, err)
		assert.Len(t, results, 1)
	})
}

func TestClientService_AddClientAPIs(t *testing.T) {
	actorUUID := uuid.New()
	cUUID := uuid.New()
	apiUUID := uuid.New()
	tenantID := int64(1)

	t.Run("client not found", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Client, error) { return nil, nil },
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{}, &mockClientRoleRepo{}, &mockRoleRepo{},
			&mockAPIRepo{}, actorUserRepo(tenantID), &mockTenantRepo{}, nil, nil)
		err := svc.AddClientAPIs(context.Background(), tenantID, cUUID, []uuid.UUID{apiUUID}, actorUUID)
		require.Error(t, err)
	})

	t.Run("cross-tenant client is not returned", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*Client, error) {
				return nil, nil
			},
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{}, &mockClientRoleRepo{}, &mockRoleRepo{},
			&mockAPIRepo{}, actorUserRepo(tenantID), &mockTenantRepo{}, nil, nil)
		err := svc.AddClientAPIs(context.Background(), tenantID, cUUID, []uuid.UUID{apiUUID}, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "auth client not found")
	})

	t.Run("API not found", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Client, error) {
				return &Client{ClientID: 1, IdentityProvider: &IdentityProvider{TenantID: tenantID}}, nil
			},
		}
		apiRepo := &mockAPIRepo{
			findByUUIDFn: func(_ any, _ ...string) (*API, error) { return nil, nil },
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{}, &mockClientRoleRepo{}, &mockRoleRepo{},
			apiRepo, actorUserRepo(tenantID), &mockTenantRepo{}, nil, nil)
		err := svc.AddClientAPIs(context.Background(), tenantID, cUUID, []uuid.UUID{apiUUID}, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "API not found")
	})

	t.Run("API already assigned", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Client, error) {
				return &Client{ClientID: 1, IdentityProvider: &IdentityProvider{TenantID: tenantID}}, nil
			},
		}
		apiRepo := &mockAPIRepo{
			findByUUIDFn: func(_ any, _ ...string) (*API, error) { return &API{APIID: 1, TenantID: 1}, nil },
		}
		caRepo := &mockClientAPIRepo{
			findByClientAndAPIFn: func(_, _ int64) (*ClientAPI, error) { return &ClientAPI{}, nil },
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, caRepo, &mockClientRoleRepo{}, &mockRoleRepo{},
			apiRepo, actorUserRepo(tenantID), &mockTenantRepo{}, nil, nil)
		err := svc.AddClientAPIs(context.Background(), tenantID, cUUID, []uuid.UUID{apiUUID}, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "already assigned")
	})

	t.Run("success", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Client, error) {
				return &Client{ClientID: 1, IdentityProvider: &IdentityProvider{TenantID: tenantID}}, nil
			},
		}
		apiRepo := &mockAPIRepo{
			findByUUIDFn: func(_ any, _ ...string) (*API, error) { return &API{APIID: 1, TenantID: 1}, nil },
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{}, &mockClientRoleRepo{}, &mockRoleRepo{},
			apiRepo, actorUserRepo(tenantID), &mockTenantRepo{}, nil, nil)
		err := svc.AddClientAPIs(context.Background(), tenantID, cUUID, []uuid.UUID{apiUUID}, actorUUID)
		require.NoError(t, err)
	})
}

// A grant mutation must be attributable AND the actor must belong to the tenant:
// before this, the middleware-supplied tenant was the only trust boundary, and the
// gRPC surface could pass no actor at all.
func TestClientService_GrantMutations_RequireActorTenantAccess(t *testing.T) {
	cUUID := uuid.New()
	apiUUID := uuid.New()
	permUUID := uuid.New()
	actorUUID := uuid.New()
	tenantID := int64(1)

	buildSvc := func(t *testing.T, userRepo UserRepository) ClientService {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		return NewClientService(gormDB, &mockClientRepo{}, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{}, &mockClientRoleRepo{}, &mockRoleRepo{},
			&mockAPIRepo{}, userRepo, &mockTenantRepo{}, nil, nil)
	}

	// An unresolvable actor is what an empty actor UUID looks like by the time it
	// reaches the service.
	unknownActor := &mockUserRepo{
		findByUUIDFn: func(_ any, _ ...string) (*User, error) { return nil, nil },
	}
	// An actor whose only identity is in another, non-system tenant.
	foreignActor := &mockUserRepo{
		findByUUIDFn: func(_ any, _ ...string) (*User, error) {
			return &User{
				UserID: 7,
				UserIdentities: []UserIdentity{
					{TenantID: 99, Tenant: &Tenant{TenantID: 99, IsSystem: false}},
				},
			}, nil
		},
	}

	for name, userRepo := range map[string]UserRepository{
		"unknown actor": unknownActor,
		"foreign actor": foreignActor,
	} {
		t.Run(name+" cannot add APIs", func(t *testing.T) {
			err := buildSvc(t, userRepo).AddClientAPIs(context.Background(), tenantID, cUUID, []uuid.UUID{apiUUID}, actorUUID)
			require.Error(t, err)
		})
		t.Run(name+" cannot remove an API", func(t *testing.T) {
			err := buildSvc(t, userRepo).RemoveClientAPI(context.Background(), tenantID, cUUID, apiUUID, actorUUID)
			require.Error(t, err)
		})
		t.Run(name+" cannot add API permissions", func(t *testing.T) {
			err := buildSvc(t, userRepo).AddClientAPIPermissions(context.Background(), tenantID, cUUID, apiUUID, []uuid.UUID{permUUID}, actorUUID)
			require.Error(t, err)
		})
		t.Run(name+" cannot remove an API permission", func(t *testing.T) {
			err := buildSvc(t, userRepo).RemoveClientAPIPermission(context.Background(), tenantID, cUUID, apiUUID, permUUID, actorUUID)
			require.Error(t, err)
		})
	}
}

func TestClientService_RemoveClientAPI(t *testing.T) {
	actorUUID := uuid.New()
	cUUID := uuid.New()
	apiUUID := uuid.New()
	tenantID := int64(1)

	t.Run("client not found", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Client, error) { return nil, nil },
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{}, &mockClientRoleRepo{}, &mockRoleRepo{},
			&mockAPIRepo{}, actorUserRepo(tenantID), &mockTenantRepo{}, nil, nil)
		err := svc.RemoveClientAPI(context.Background(), tenantID, cUUID, apiUUID, actorUUID)
		require.Error(t, err)
	})

	t.Run("unauthorized tenant", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Client, error) {
				return &Client{ClientID: 1, IdentityProvider: &IdentityProvider{TenantID: 999}}, nil
			},
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{}, &mockClientRoleRepo{}, &mockRoleRepo{},
			&mockAPIRepo{}, actorUserRepo(tenantID), &mockTenantRepo{}, nil, nil)
		err := svc.RemoveClientAPI(context.Background(), tenantID, cUUID, apiUUID, actorUUID)
		require.Error(t, err)
	})

	t.Run("remove error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Client, error) {
				return &Client{ClientID: 1, IdentityProvider: &IdentityProvider{TenantID: tenantID}}, nil
			},
		}
		caRepo := &mockClientAPIRepo{
			removeByClientUUIDAndAPIUUIDFn: func(_, _ uuid.UUID) error { return errors.New("del err") },
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, caRepo, &mockClientRoleRepo{}, &mockRoleRepo{},
			&mockAPIRepo{}, actorUserRepo(tenantID), &mockTenantRepo{}, nil, nil)
		err := svc.RemoveClientAPI(context.Background(), tenantID, cUUID, apiUUID, actorUUID)
		require.Error(t, err)
	})

	t.Run("success", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Client, error) {
				return &Client{ClientID: 1, IdentityProvider: &IdentityProvider{TenantID: tenantID}}, nil
			},
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{}, &mockClientRoleRepo{}, &mockRoleRepo{},
			&mockAPIRepo{}, actorUserRepo(tenantID), &mockTenantRepo{}, nil, nil)
		err := svc.RemoveClientAPI(context.Background(), tenantID, cUUID, apiUUID, actorUUID)
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
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{}, &mockAPIRepo{}, &mockUserRepo{}, &mockTenantRepo{}, nil)
		_, err := svc.GetClientAPIPermissions(context.Background(), tenantID, cUUID, apiUUID)
		require.Error(t, err)
	})

	t.Run("client not found", func(t *testing.T) {
		caRepo := &mockClientAPIRepo{
			findByClientUUIDAndAPIUUIDFn: func(_, _ uuid.UUID) (*ClientAPI, error) {
				return &ClientAPI{ClientAPIID: 1}, nil
			},
		}
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Client, error) { return nil, nil },
		}
		svc := buildFullClientService(t, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, caRepo, &mockAPIRepo{}, &mockUserRepo{}, &mockTenantRepo{}, nil)
		_, err := svc.GetClientAPIPermissions(context.Background(), tenantID, cUUID, apiUUID)
		require.Error(t, err)
	})

	t.Run("cross-tenant client is not returned", func(t *testing.T) {
		caRepo := &mockClientAPIRepo{
			findByClientUUIDAndAPIUUIDFn: func(_, _ uuid.UUID) (*ClientAPI, error) {
				return &ClientAPI{ClientAPIID: 1}, nil
			},
		}
		clientRepo := &mockClientRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*Client, error) {
				return nil, nil
			},
		}
		svc := buildFullClientService(t, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, caRepo, &mockAPIRepo{}, &mockUserRepo{}, &mockTenantRepo{}, nil)
		_, err := svc.GetClientAPIPermissions(context.Background(), tenantID, cUUID, apiUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "auth client not found")
	})

	t.Run("success", func(t *testing.T) {
		caRepo := &mockClientAPIRepo{
			findByClientUUIDAndAPIUUIDFn: func(_, _ uuid.UUID) (*ClientAPI, error) {
				return &ClientAPI{ClientAPIID: 1}, nil
			},
		}
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Client, error) {
				return &Client{ClientID: 1, IdentityProvider: &IdentityProvider{TenantID: tenantID}}, nil
			},
		}
		cpRepo := &mockClientPermissionRepo{
			findByClientAPIIDFn: func(_ int64) ([]ClientPermission, error) {
				return []ClientPermission{{Permission: &Permission{PermissionUUID: uuid.New(), Name: "read"}}}, nil
			},
		}
		svc := buildFullClientService(t, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, cpRepo, caRepo, &mockAPIRepo{}, &mockUserRepo{}, &mockTenantRepo{}, nil)
		results, err := svc.GetClientAPIPermissions(context.Background(), tenantID, cUUID, apiUUID)
		require.NoError(t, err)
		assert.Len(t, results, 1)
	})
}

func TestClientService_AddClientAPIPermissions(t *testing.T) {
	actorUUID := uuid.New()
	cUUID := uuid.New()
	apiUUID := uuid.New()
	permUUID := uuid.New()
	tenantID := int64(1)

	t.Run("client not found", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Client, error) { return nil, nil },
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{}, &mockClientRoleRepo{}, &mockRoleRepo{},
			&mockAPIRepo{}, actorUserRepo(tenantID), &mockTenantRepo{}, nil, nil)
		err := svc.AddClientAPIPermissions(context.Background(), tenantID, cUUID, apiUUID, []uuid.UUID{permUUID}, actorUUID)
		require.Error(t, err)
	})

	t.Run("unauthorized tenant", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Client, error) {
				return &Client{ClientID: 1, IdentityProvider: &IdentityProvider{TenantID: 999}}, nil
			},
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{}, &mockClientRoleRepo{}, &mockRoleRepo{},
			&mockAPIRepo{}, actorUserRepo(tenantID), &mockTenantRepo{}, nil, nil)
		err := svc.AddClientAPIPermissions(context.Background(), tenantID, cUUID, apiUUID, []uuid.UUID{permUUID}, actorUUID)
		require.Error(t, err)
	})

	t.Run("client API not found", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Client, error) {
				return &Client{ClientID: 1, IdentityProvider: &IdentityProvider{TenantID: tenantID}}, nil
			},
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{}, &mockClientRoleRepo{}, &mockRoleRepo{},
			&mockAPIRepo{}, actorUserRepo(tenantID), &mockTenantRepo{}, nil, nil)
		err := svc.AddClientAPIPermissions(context.Background(), tenantID, cUUID, apiUUID, []uuid.UUID{permUUID}, actorUUID)
		require.Error(t, err)
	})

	t.Run("permission not found", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Client, error) {
				return &Client{ClientID: 1, IdentityProvider: &IdentityProvider{TenantID: tenantID}}, nil
			},
		}
		caRepo := &mockClientAPIRepo{
			findByClientUUIDAndAPIUUIDFn: func(_, _ uuid.UUID) (*ClientAPI, error) {
				return &ClientAPI{ClientAPIID: 1}, nil
			},
		}
		permRepo := &mockPermissionRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Permission, error) { return nil, nil },
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			permRepo, &mockClientPermissionRepo{}, caRepo, &mockClientRoleRepo{}, &mockRoleRepo{},
			&mockAPIRepo{}, actorUserRepo(tenantID), &mockTenantRepo{}, nil, nil)
		err := svc.AddClientAPIPermissions(context.Background(), tenantID, cUUID, apiUUID, []uuid.UUID{permUUID}, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "permission not found")
	})

	t.Run("permission already assigned", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Client, error) {
				return &Client{ClientID: 1, IdentityProvider: &IdentityProvider{TenantID: tenantID}}, nil
			},
		}
		caRepo := &mockClientAPIRepo{
			findByClientUUIDAndAPIUUIDFn: func(_, _ uuid.UUID) (*ClientAPI, error) {
				return &ClientAPI{ClientAPIID: 1}, nil
			},
		}
		permRepo := &mockPermissionRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Permission, error) {
				return &Permission{PermissionID: 1, TenantID: 1}, nil
			},
		}
		cpRepo := &mockClientPermissionRepo{
			findByClientAPIAndPermissionFn: func(_, _ int64) (*ClientPermission, error) {
				return &ClientPermission{}, nil
			},
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			permRepo, cpRepo, caRepo, &mockClientRoleRepo{}, &mockRoleRepo{},
			&mockAPIRepo{}, actorUserRepo(tenantID), &mockTenantRepo{}, nil, nil)
		err := svc.AddClientAPIPermissions(context.Background(), tenantID, cUUID, apiUUID, []uuid.UUID{permUUID}, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "already assigned")
	})

	t.Run("success", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Client, error) {
				return &Client{ClientID: 1, IdentityProvider: &IdentityProvider{TenantID: tenantID}}, nil
			},
		}
		caRepo := &mockClientAPIRepo{
			findByClientUUIDAndAPIUUIDFn: func(_, _ uuid.UUID) (*ClientAPI, error) {
				return &ClientAPI{ClientAPIID: 1}, nil
			},
		}
		permRepo := &mockPermissionRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Permission, error) {
				return &Permission{PermissionID: 1, TenantID: 1}, nil
			},
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			permRepo, &mockClientPermissionRepo{}, caRepo, &mockClientRoleRepo{}, &mockRoleRepo{},
			&mockAPIRepo{}, actorUserRepo(tenantID), &mockTenantRepo{}, nil, nil)
		err := svc.AddClientAPIPermissions(context.Background(), tenantID, cUUID, apiUUID, []uuid.UUID{permUUID}, actorUUID)
		require.NoError(t, err)
	})
}

func TestClientService_RemoveClientAPIPermission(t *testing.T) {
	actorUUID := uuid.New()
	cUUID := uuid.New()
	apiUUID := uuid.New()
	permUUID := uuid.New()
	tenantID := int64(1)

	t.Run("client not found", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Client, error) { return nil, nil },
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{}, &mockClientRoleRepo{}, &mockRoleRepo{},
			&mockAPIRepo{}, actorUserRepo(tenantID), &mockTenantRepo{}, nil, nil)
		err := svc.RemoveClientAPIPermission(context.Background(), tenantID, cUUID, apiUUID, permUUID, actorUUID)
		require.Error(t, err)
	})

	t.Run("unauthorized tenant", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Client, error) {
				return &Client{ClientID: 1, IdentityProvider: &IdentityProvider{TenantID: 999}}, nil
			},
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{}, &mockClientRoleRepo{}, &mockRoleRepo{},
			&mockAPIRepo{}, actorUserRepo(tenantID), &mockTenantRepo{}, nil, nil)
		err := svc.RemoveClientAPIPermission(context.Background(), tenantID, cUUID, apiUUID, permUUID, actorUUID)
		require.Error(t, err)
	})

	t.Run("client API not found", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Client, error) {
				return &Client{ClientID: 1, IdentityProvider: &IdentityProvider{TenantID: tenantID}}, nil
			},
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{}, &mockClientRoleRepo{}, &mockRoleRepo{},
			&mockAPIRepo{}, actorUserRepo(tenantID), &mockTenantRepo{}, nil, nil)
		err := svc.RemoveClientAPIPermission(context.Background(), tenantID, cUUID, apiUUID, permUUID, actorUUID)
		require.Error(t, err)
	})

	t.Run("permission not found", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Client, error) {
				return &Client{ClientID: 1, IdentityProvider: &IdentityProvider{TenantID: tenantID}}, nil
			},
		}
		caRepo := &mockClientAPIRepo{
			findByClientUUIDAndAPIUUIDFn: func(_, _ uuid.UUID) (*ClientAPI, error) {
				return &ClientAPI{ClientAPIID: 1}, nil
			},
		}
		permRepo := &mockPermissionRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Permission, error) { return nil, nil },
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			permRepo, &mockClientPermissionRepo{}, caRepo, &mockClientRoleRepo{}, &mockRoleRepo{},
			&mockAPIRepo{}, actorUserRepo(tenantID), &mockTenantRepo{}, nil, nil)
		err := svc.RemoveClientAPIPermission(context.Background(), tenantID, cUUID, apiUUID, permUUID, actorUUID)
		require.Error(t, err)
	})

	t.Run("success", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Client, error) {
				return &Client{ClientID: 1, IdentityProvider: &IdentityProvider{TenantID: tenantID}}, nil
			},
		}
		caRepo := &mockClientAPIRepo{
			findByClientUUIDAndAPIUUIDFn: func(_, _ uuid.UUID) (*ClientAPI, error) {
				return &ClientAPI{ClientAPIID: 1}, nil
			},
		}
		permRepo := &mockPermissionRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Permission, error) {
				return &Permission{PermissionID: 1, TenantID: 1}, nil
			},
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			permRepo, &mockClientPermissionRepo{}, caRepo, &mockClientRoleRepo{}, &mockRoleRepo{},
			&mockAPIRepo{}, actorUserRepo(tenantID), &mockTenantRepo{}, nil, nil)
		err := svc.RemoveClientAPIPermission(context.Background(), tenantID, cUUID, apiUUID, permUUID, actorUUID)
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
		c.IdentityProvider.Tenant = &Tenant{TenantID: tenantID, IsSystem: false}
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Client, error) { return c, nil },
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*User, error) {
				return &User{UserID: 1, UserIdentities: []UserIdentity{
					{TenantID: 999, Tenant: &Tenant{TenantID: 999, IsSystem: false}},
				}}, nil
			},
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{}, &mockClientRoleRepo{}, &mockRoleRepo{},
			&mockAPIRepo{}, userRepo, &mockTenantRepo{}, nil, nil)
		_, err := svc.Update(context.Background(), cUUID, tenantID, "n", "d", "pub", "ex.com", nil, "active", false, nil, testBoolPtr(true), nil, nil, nil, nil, actorUUID, nil, nil)
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
			findByUUIDFn: func(_ any, _ ...string) (*Client, error) { return c, nil },
			findByNameAndIdentityProviderFn: func(_ string, _ int64, _ int64) (*Client, error) {
				return nil, errors.New("db err")
			},
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*User, error) { return actorUser(tenantID), nil },
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{}, &mockClientRoleRepo{}, &mockRoleRepo{},
			&mockAPIRepo{}, userRepo, &mockTenantRepo{}, nil, nil)
		_, err := svc.Update(context.Background(), cUUID, tenantID, "new-name", "d", "pub", "ex.com", nil, "active", false, nil, testBoolPtr(true), nil, nil, nil, nil, actorUUID, nil, nil)
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
	c.IdentityProvider.Tenant = &Tenant{TenantID: tenantID, IsSystem: false}
	clientRepo := &mockClientRepo{
		findByUUIDFn: func(_ any, _ ...string) (*Client, error) { return c, nil },
	}
	userRepo := &mockUserRepo{
		findByUUIDFn: func(_ any, _ ...string) (*User, error) {
			return &User{UserID: 1, UserIdentities: []UserIdentity{
				{TenantID: 999, Tenant: &Tenant{TenantID: 999, IsSystem: false}},
			}}, nil
		},
	}
	svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
		&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{}, &mockClientRoleRepo{}, &mockRoleRepo{},
		&mockAPIRepo{}, userRepo, &mockTenantRepo{}, nil, nil)
	_, err := svc.SetStatusByUUID(context.Background(), cUUID, tenantID, "inactive", actorUUID)
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
	c.IdentityProvider.Tenant = &Tenant{TenantID: tenantID, IsSystem: false}
	clientRepo := &mockClientRepo{
		findByUUIDFn: func(_ any, _ ...string) (*Client, error) { return c, nil },
	}
	userRepo := &mockUserRepo{
		findByUUIDFn: func(_ any, _ ...string) (*User, error) {
			return &User{UserID: 1, UserIdentities: []UserIdentity{
				{TenantID: 999, Tenant: &Tenant{TenantID: 999, IsSystem: false}},
			}}, nil
		},
	}
	svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
		&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{}, &mockClientRoleRepo{}, &mockRoleRepo{},
		&mockAPIRepo{}, userRepo, &mockTenantRepo{}, nil, nil)
	_, err := svc.DeleteByUUID(context.Background(), cUUID, tenantID, actorUUID)
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
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*Client, error) {
				return &Client{ClientID: 1, TenantID: tenantID}, nil
			},
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*User, error) { return actorUser(tenantID), nil },
		}
		uriRepo := &mockClientURIRepo{
			createOrUpdateFn: func(_ *ClientURI) (*ClientURI, error) {
				return nil, errors.New("create err")
			},
		}
		svc := NewClientService(gormDB, clientRepo, uriRepo, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{}, &mockClientRoleRepo{}, &mockRoleRepo{},
			&mockAPIRepo{}, userRepo, &mockTenantRepo{}, nil, nil)
		_, err := svc.CreateURI(context.Background(), cUUID, tenantID, "https://cb.test", "redirect", actorUUID)
		require.Error(t, err)
	})

	t.Run("post-save fetch error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		fetchCount := 0
		clientRepo := &mockClientRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*Client, error) {
				fetchCount++
				if fetchCount == 1 {
					return &Client{ClientID: 1, TenantID: tenantID}, nil
				}
				return nil, errors.New("fetch err")
			},
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*User, error) { return actorUser(tenantID), nil },
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{}, &mockClientRoleRepo{}, &mockRoleRepo{},
			&mockAPIRepo{}, userRepo, &mockTenantRepo{}, nil, nil)
		_, err := svc.CreateURI(context.Background(), cUUID, tenantID, "https://cb.test", "redirect", actorUUID)
		require.Error(t, err)
	})

	t.Run("tenant mismatch", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*Client, error) {
				return &Client{ClientID: 1, TenantID: 999}, nil
			},
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*User, error) { return actorUser(tenantID), nil },
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{}, &mockClientRoleRepo{}, &mockRoleRepo{},
			&mockAPIRepo{}, userRepo, &mockTenantRepo{}, nil, nil)
		_, err := svc.CreateURI(context.Background(), cUUID, tenantID, "https://cb.test", "redirect", actorUUID)
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
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*Client, error) {
				return &Client{ClientID: 1, TenantID: 999}, nil
			},
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*User, error) { return actorUser(tenantID), nil },
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{}, &mockClientRoleRepo{}, &mockRoleRepo{},
			&mockAPIRepo{}, userRepo, &mockTenantRepo{}, nil, nil)
		_, err := svc.UpdateURI(context.Background(), cUUID, tenantID, uriUUID, "https://new.test", "redirect", actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "access denied")
	})

	t.Run("URI save error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*Client, error) {
				return &Client{ClientID: 1, TenantID: tenantID}, nil
			},
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*User, error) { return actorUser(tenantID), nil },
		}
		uriRepo := &mockClientURIRepo{
			findByUUIDAndTenantIDFn: func(_ string, _ int64) (*ClientURI, error) {
				return &ClientURI{ClientID: 1}, nil
			},
			createOrUpdateFn: func(_ *ClientURI) (*ClientURI, error) {
				return nil, errors.New("save err")
			},
		}
		svc := NewClientService(gormDB, clientRepo, uriRepo, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{}, &mockClientRoleRepo{}, &mockRoleRepo{},
			&mockAPIRepo{}, userRepo, &mockTenantRepo{}, nil, nil)
		_, err := svc.UpdateURI(context.Background(), cUUID, tenantID, uriUUID, "https://new.test", "redirect", actorUUID)
		require.Error(t, err)
	})

	t.Run("post-save fetch error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		fetchCount := 0
		clientRepo := &mockClientRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*Client, error) {
				fetchCount++
				if fetchCount == 1 {
					return &Client{ClientID: 1, TenantID: tenantID}, nil
				}
				return nil, errors.New("fetch err")
			},
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*User, error) { return actorUser(tenantID), nil },
		}
		uriRepo := &mockClientURIRepo{
			findByUUIDAndTenantIDFn: func(_ string, _ int64) (*ClientURI, error) {
				return &ClientURI{ClientID: 1}, nil
			},
		}
		svc := NewClientService(gormDB, clientRepo, uriRepo, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{}, &mockClientRoleRepo{}, &mockRoleRepo{},
			&mockAPIRepo{}, userRepo, &mockTenantRepo{}, nil, nil)
		_, err := svc.UpdateURI(context.Background(), cUUID, tenantID, uriUUID, "https://new.test", "redirect", actorUUID)
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
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*Client, error) {
				return &Client{ClientID: 1, TenantID: 999}, nil
			},
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*User, error) { return actorUser(tenantID), nil },
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{}, &mockClientRoleRepo{}, &mockRoleRepo{},
			&mockAPIRepo{}, userRepo, &mockTenantRepo{}, nil, nil)
		_, err := svc.DeleteURI(context.Background(), cUUID, tenantID, uriUUID, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "access denied")
	})

	t.Run("delete error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*Client, error) {
				return &Client{ClientID: 1, TenantID: tenantID}, nil
			},
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*User, error) { return actorUser(tenantID), nil },
		}
		uriRepo := &mockClientURIRepo{
			findByUUIDAndTenantIDFn: func(_ string, _ int64) (*ClientURI, error) {
				return &ClientURI{ClientID: 1}, nil
			},
			deleteByUUIDAndTenantIDFn: func(_ string, _ int64) error { return errors.New("del err") },
		}
		svc := NewClientService(gormDB, clientRepo, uriRepo, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{}, &mockClientRoleRepo{}, &mockRoleRepo{},
			&mockAPIRepo{}, userRepo, &mockTenantRepo{}, nil, nil)
		_, err := svc.DeleteURI(context.Background(), cUUID, tenantID, uriUUID, actorUUID)
		require.Error(t, err)
	})
}

func TestClientService_AddClientAPIs_EdgeCases(t *testing.T) {
	actorUUID := uuid.New()
	cUUID := uuid.New()
	apiUUID := uuid.New()
	tenantID := int64(1)

	t.Run("FindByClientAndAPI error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Client, error) {
				return &Client{ClientID: 1, IdentityProvider: &IdentityProvider{TenantID: tenantID}}, nil
			},
		}
		apiRepo := &mockAPIRepo{
			findByUUIDFn: func(_ any, _ ...string) (*API, error) { return &API{APIID: 1, TenantID: 1}, nil },
		}
		caRepo := &mockClientAPIRepo{
			findByClientAndAPIFn: func(_, _ int64) (*ClientAPI, error) { return nil, errors.New("db err") },
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, caRepo, &mockClientRoleRepo{}, &mockRoleRepo{},
			apiRepo, actorUserRepo(tenantID), &mockTenantRepo{}, nil, nil)
		err := svc.AddClientAPIs(context.Background(), tenantID, cUUID, []uuid.UUID{apiUUID}, actorUUID)
		require.Error(t, err)
	})

	t.Run("findByUUID API error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Client, error) {
				return &Client{ClientID: 1, IdentityProvider: &IdentityProvider{TenantID: tenantID}}, nil
			},
		}
		apiRepo := &mockAPIRepo{
			findByUUIDFn: func(_ any, _ ...string) (*API, error) { return nil, errors.New("api err") },
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{}, &mockClientRoleRepo{}, &mockRoleRepo{},
			apiRepo, actorUserRepo(tenantID), &mockTenantRepo{}, nil, nil)
		err := svc.AddClientAPIs(context.Background(), tenantID, cUUID, []uuid.UUID{apiUUID}, actorUUID)
		require.Error(t, err)
	})

	t.Run("findByUUID client error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Client, error) { return nil, errors.New("db err") },
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{}, &mockClientRoleRepo{}, &mockRoleRepo{},
			&mockAPIRepo{}, actorUserRepo(tenantID), &mockTenantRepo{}, nil, nil)
		err := svc.AddClientAPIs(context.Background(), tenantID, cUUID, []uuid.UUID{apiUUID}, actorUUID)
		require.Error(t, err)
	})

	t.Run("Create unique constraint error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Client, error) {
				return &Client{ClientID: 1, IdentityProvider: &IdentityProvider{TenantID: tenantID}}, nil
			},
		}
		apiRepo := &mockAPIRepo{
			findByUUIDFn: func(_ any, _ ...string) (*API, error) { return &API{APIID: 1, TenantID: 1}, nil },
		}
		caRepo := &mockClientAPIRepo{
			createFn: func(_ *ClientAPI) (*ClientAPI, error) {
				return nil, errors.New("uq_client_apis_client_api violation")
			},
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, caRepo, &mockClientRoleRepo{}, &mockRoleRepo{},
			apiRepo, actorUserRepo(tenantID), &mockTenantRepo{}, nil, nil)
		err := svc.AddClientAPIs(context.Background(), tenantID, cUUID, []uuid.UUID{apiUUID}, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "already assigned")
	})

	t.Run("Create generic error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Client, error) {
				return &Client{ClientID: 1, IdentityProvider: &IdentityProvider{TenantID: tenantID}}, nil
			},
		}
		apiRepo := &mockAPIRepo{
			findByUUIDFn: func(_ any, _ ...string) (*API, error) { return &API{APIID: 1, TenantID: 1}, nil },
		}
		caRepo := &mockClientAPIRepo{
			createFn: func(_ *ClientAPI) (*ClientAPI, error) {
				return nil, errors.New("generic db error")
			},
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, caRepo, &mockClientRoleRepo{}, &mockRoleRepo{},
			apiRepo, actorUserRepo(tenantID), &mockTenantRepo{}, nil, nil)
		err := svc.AddClientAPIs(context.Background(), tenantID, cUUID, []uuid.UUID{apiUUID}, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "generic db error")
	})
}

func TestClientService_RemoveClientAPI_EdgeCases(t *testing.T) {
	actorUUID := uuid.New()
	cUUID := uuid.New()
	apiUUID := uuid.New()
	tenantID := int64(1)

	t.Run("findByUUID error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Client, error) { return nil, errors.New("db err") },
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{}, &mockClientRoleRepo{}, &mockRoleRepo{},
			&mockAPIRepo{}, actorUserRepo(tenantID), &mockTenantRepo{}, nil, nil)
		err := svc.RemoveClientAPI(context.Background(), tenantID, cUUID, apiUUID, actorUUID)
		require.Error(t, err)
	})
}

func TestClientService_GetClientAPIPermissions_EdgeCases(t *testing.T) {
	cUUID := uuid.New()
	apiUUID := uuid.New()
	tenantID := int64(1)

	t.Run("findByClientUUIDAndAPIUUID error", func(t *testing.T) {
		caRepo := &mockClientAPIRepo{
			findByClientUUIDAndAPIUUIDFn: func(_, _ uuid.UUID) (*ClientAPI, error) {
				return nil, errors.New("db err")
			},
		}
		svc := buildFullClientService(t, &mockClientRepo{}, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, caRepo, &mockAPIRepo{}, &mockUserRepo{}, &mockTenantRepo{}, nil)
		_, err := svc.GetClientAPIPermissions(context.Background(), tenantID, cUUID, apiUUID)
		require.Error(t, err)
	})

	t.Run("client findByUUID error", func(t *testing.T) {
		caRepo := &mockClientAPIRepo{
			findByClientUUIDAndAPIUUIDFn: func(_, _ uuid.UUID) (*ClientAPI, error) {
				return &ClientAPI{ClientAPIID: 1}, nil
			},
		}
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Client, error) { return nil, errors.New("db err") },
		}
		svc := buildFullClientService(t, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, caRepo, &mockAPIRepo{}, &mockUserRepo{}, &mockTenantRepo{}, nil)
		_, err := svc.GetClientAPIPermissions(context.Background(), tenantID, cUUID, apiUUID)
		require.Error(t, err)
	})

	t.Run("FindByClientAPIID error", func(t *testing.T) {
		caRepo := &mockClientAPIRepo{
			findByClientUUIDAndAPIUUIDFn: func(_, _ uuid.UUID) (*ClientAPI, error) {
				return &ClientAPI{ClientAPIID: 1}, nil
			},
		}
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Client, error) {
				return &Client{ClientID: 1, IdentityProvider: &IdentityProvider{TenantID: tenantID}}, nil
			},
		}
		cpRepo := &mockClientPermissionRepo{
			findByClientAPIIDFn: func(_ int64) ([]ClientPermission, error) { return nil, errors.New("db err") },
		}
		svc := buildFullClientService(t, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, cpRepo, caRepo, &mockAPIRepo{}, &mockUserRepo{}, &mockTenantRepo{}, nil)
		_, err := svc.GetClientAPIPermissions(context.Background(), tenantID, cUUID, apiUUID)
		require.Error(t, err)
	})
}

func TestClientService_AddClientAPIPermissions_EdgeCases(t *testing.T) {
	actorUUID := uuid.New()
	cUUID := uuid.New()
	apiUUID := uuid.New()
	permUUID := uuid.New()
	tenantID := int64(1)

	t.Run("findByUUID client error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Client, error) { return nil, errors.New("db err") },
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{}, &mockClientRoleRepo{}, &mockRoleRepo{},
			&mockAPIRepo{}, actorUserRepo(tenantID), &mockTenantRepo{}, nil, nil)
		err := svc.AddClientAPIPermissions(context.Background(), tenantID, cUUID, apiUUID, []uuid.UUID{permUUID}, actorUUID)
		require.Error(t, err)
	})

	t.Run("findByClientUUIDAndAPIUUID error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Client, error) {
				return &Client{ClientID: 1, IdentityProvider: &IdentityProvider{TenantID: tenantID}}, nil
			},
		}
		caRepo := &mockClientAPIRepo{
			findByClientUUIDAndAPIUUIDFn: func(_, _ uuid.UUID) (*ClientAPI, error) {
				return nil, errors.New("db err")
			},
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, caRepo, &mockClientRoleRepo{}, &mockRoleRepo{},
			&mockAPIRepo{}, actorUserRepo(tenantID), &mockTenantRepo{}, nil, nil)
		err := svc.AddClientAPIPermissions(context.Background(), tenantID, cUUID, apiUUID, []uuid.UUID{permUUID}, actorUUID)
		require.Error(t, err)
	})

	t.Run("findByUUID permission error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Client, error) {
				return &Client{ClientID: 1, IdentityProvider: &IdentityProvider{TenantID: tenantID}}, nil
			},
		}
		caRepo := &mockClientAPIRepo{
			findByClientUUIDAndAPIUUIDFn: func(_, _ uuid.UUID) (*ClientAPI, error) {
				return &ClientAPI{ClientAPIID: 1}, nil
			},
		}
		permRepo := &mockPermissionRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Permission, error) { return nil, errors.New("perm err") },
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			permRepo, &mockClientPermissionRepo{}, caRepo, &mockClientRoleRepo{}, &mockRoleRepo{},
			&mockAPIRepo{}, actorUserRepo(tenantID), &mockTenantRepo{}, nil, nil)
		err := svc.AddClientAPIPermissions(context.Background(), tenantID, cUUID, apiUUID, []uuid.UUID{permUUID}, actorUUID)
		require.Error(t, err)
	})

	t.Run("FindByClientAPIAndPermission error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Client, error) {
				return &Client{ClientID: 1, IdentityProvider: &IdentityProvider{TenantID: tenantID}}, nil
			},
		}
		caRepo := &mockClientAPIRepo{
			findByClientUUIDAndAPIUUIDFn: func(_, _ uuid.UUID) (*ClientAPI, error) {
				return &ClientAPI{ClientAPIID: 1}, nil
			},
		}
		permRepo := &mockPermissionRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Permission, error) {
				return &Permission{PermissionID: 1, TenantID: 1}, nil
			},
		}
		cpRepo := &mockClientPermissionRepo{
			findByClientAPIAndPermissionFn: func(_, _ int64) (*ClientPermission, error) {
				return nil, errors.New("db err")
			},
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			permRepo, cpRepo, caRepo, &mockClientRoleRepo{}, &mockRoleRepo{},
			&mockAPIRepo{}, actorUserRepo(tenantID), &mockTenantRepo{}, nil, nil)
		err := svc.AddClientAPIPermissions(context.Background(), tenantID, cUUID, apiUUID, []uuid.UUID{permUUID}, actorUUID)
		require.Error(t, err)
	})

	t.Run("Create unique constraint error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Client, error) {
				return &Client{ClientID: 1, IdentityProvider: &IdentityProvider{TenantID: tenantID}}, nil
			},
		}
		caRepo := &mockClientAPIRepo{
			findByClientUUIDAndAPIUUIDFn: func(_, _ uuid.UUID) (*ClientAPI, error) {
				return &ClientAPI{ClientAPIID: 1}, nil
			},
		}
		permRepo := &mockPermissionRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Permission, error) {
				return &Permission{PermissionID: 1, TenantID: 1}, nil
			},
		}
		cpRepo := &mockClientPermissionRepo{
			findByClientAPIAndPermissionFn: func(_, _ int64) (*ClientPermission, error) {
				return nil, nil
			},
			createFn: func(_ *ClientPermission) (*ClientPermission, error) {
				return nil, errors.New("uq_client_permissions_client_permission violation")
			},
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			permRepo, cpRepo, caRepo, &mockClientRoleRepo{}, &mockRoleRepo{},
			&mockAPIRepo{}, actorUserRepo(tenantID), &mockTenantRepo{}, nil, nil)
		err := svc.AddClientAPIPermissions(context.Background(), tenantID, cUUID, apiUUID, []uuid.UUID{permUUID}, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "already assigned")
	})

	t.Run("Create generic error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Client, error) {
				return &Client{ClientID: 1, IdentityProvider: &IdentityProvider{TenantID: tenantID}}, nil
			},
		}
		caRepo := &mockClientAPIRepo{
			findByClientUUIDAndAPIUUIDFn: func(_, _ uuid.UUID) (*ClientAPI, error) {
				return &ClientAPI{ClientAPIID: 1}, nil
			},
		}
		permRepo := &mockPermissionRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Permission, error) {
				return &Permission{PermissionID: 1, TenantID: 1}, nil
			},
		}
		cpRepo := &mockClientPermissionRepo{
			findByClientAPIAndPermissionFn: func(_, _ int64) (*ClientPermission, error) {
				return nil, nil
			},
			createFn: func(_ *ClientPermission) (*ClientPermission, error) {
				return nil, errors.New("generic db error")
			},
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			permRepo, cpRepo, caRepo, &mockClientRoleRepo{}, &mockRoleRepo{},
			&mockAPIRepo{}, actorUserRepo(tenantID), &mockTenantRepo{}, nil, nil)
		err := svc.AddClientAPIPermissions(context.Background(), tenantID, cUUID, apiUUID, []uuid.UUID{permUUID}, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "generic db error")
	})
}

// ===========================================================================
// RotateSecret
// ===========================================================================

func TestClientService_RotateSecret(t *testing.T) {
	cUUID := uuid.New()
	actorUUID := uuid.New()
	tenantID := int64(1)

	t.Run("FindByUUIDAndTenantID error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*Client, error) {
				return nil, errors.New("db error")
			},
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{}, &mockClientRoleRepo{}, &mockRoleRepo{},
			&mockAPIRepo{}, actorUserRepo(tenantID), &mockTenantRepo{}, nil, nil)
		_, err := svc.RotateSecret(context.Background(), cUUID, tenantID, actorUUID, 0)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "db error")
	})

	t.Run("client not found", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*Client, error) {
				return nil, nil
			},
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{}, &mockClientRoleRepo{}, &mockRoleRepo{},
			&mockAPIRepo{}, actorUserRepo(tenantID), &mockTenantRepo{}, nil, nil)
		_, err := svc.RotateSecret(context.Background(), cUUID, tenantID, actorUUID, 0)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("CreateOrUpdate error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*Client, error) {
				return &Client{ClientID: 1, ClientUUID: cUUID, TenantID: tenantID}, nil
			},
			createOrUpdateFn: func(_ *Client) (*Client, error) {
				return nil, errors.New("save err")
			},
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{}, &mockClientRoleRepo{}, &mockRoleRepo{},
			&mockAPIRepo{}, actorUserRepo(tenantID), &mockTenantRepo{}, nil, nil)
		_, err := svc.RotateSecret(context.Background(), cUUID, tenantID, actorUUID, 0)
		require.Error(t, err)
	})

	t.Run("success with grace period", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		clientRepo := &mockClientRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*Client, error) {
				return &Client{ClientID: 1, ClientUUID: cUUID, TenantID: tenantID}, nil
			},
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{}, &mockClientRoleRepo{}, &mockRoleRepo{},
			&mockAPIRepo{}, &mockUserRepo{findByUUIDFn: func(any, ...string) (*User, error) { return actorUser(tenantID), nil }}, &mockTenantRepo{}, nil, nil)
		secret, err := svc.RotateSecret(context.Background(), cUUID, tenantID, actorUUID, 24)
		require.NoError(t, err)
		assert.NotEmpty(t, secret)
	})

	t.Run("success with zero grace period (immediate revoke)", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		clientRepo := &mockClientRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*Client, error) {
				return &Client{ClientID: 1, ClientUUID: cUUID, TenantID: tenantID}, nil
			},
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{}, &mockClientRoleRepo{}, &mockRoleRepo{},
			&mockAPIRepo{}, &mockUserRepo{findByUUIDFn: func(any, ...string) (*User, error) { return actorUser(tenantID), nil }}, &mockTenantRepo{}, nil, nil)
		secret, err := svc.RotateSecret(context.Background(), cUUID, tenantID, actorUUID, 0)
		require.NoError(t, err)
		assert.NotEmpty(t, secret)
	})
}

func TestClientService_RemoveClientAPIPermission_EdgeCases(t *testing.T) {
	actorUUID := uuid.New()
	cUUID := uuid.New()
	apiUUID := uuid.New()
	permUUID := uuid.New()
	tenantID := int64(1)

	t.Run("findByUUID client error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Client, error) { return nil, errors.New("db err") },
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{}, &mockClientRoleRepo{}, &mockRoleRepo{},
			&mockAPIRepo{}, actorUserRepo(tenantID), &mockTenantRepo{}, nil, nil)
		err := svc.RemoveClientAPIPermission(context.Background(), tenantID, cUUID, apiUUID, permUUID, actorUUID)
		require.Error(t, err)
	})

	t.Run("findByClientUUIDAndAPIUUID error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Client, error) {
				return &Client{ClientID: 1, IdentityProvider: &IdentityProvider{TenantID: tenantID}}, nil
			},
		}
		caRepo := &mockClientAPIRepo{
			findByClientUUIDAndAPIUUIDFn: func(_, _ uuid.UUID) (*ClientAPI, error) {
				return nil, errors.New("db err")
			},
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, caRepo, &mockClientRoleRepo{}, &mockRoleRepo{},
			&mockAPIRepo{}, actorUserRepo(tenantID), &mockTenantRepo{}, nil, nil)
		err := svc.RemoveClientAPIPermission(context.Background(), tenantID, cUUID, apiUUID, permUUID, actorUUID)
		require.Error(t, err)
	})

	t.Run("findByUUID permission error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Client, error) {
				return &Client{ClientID: 1, IdentityProvider: &IdentityProvider{TenantID: tenantID}}, nil
			},
		}
		caRepo := &mockClientAPIRepo{
			findByClientUUIDAndAPIUUIDFn: func(_, _ uuid.UUID) (*ClientAPI, error) {
				return &ClientAPI{ClientAPIID: 1}, nil
			},
		}
		permRepo := &mockPermissionRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Permission, error) { return nil, errors.New("db err") },
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			permRepo, &mockClientPermissionRepo{}, caRepo, &mockClientRoleRepo{}, &mockRoleRepo{},
			&mockAPIRepo{}, actorUserRepo(tenantID), &mockTenantRepo{}, nil, nil)
		err := svc.RemoveClientAPIPermission(context.Background(), tenantID, cUUID, apiUUID, permUUID, actorUUID)
		require.Error(t, err)
	})

	t.Run("RemoveByClientAPIAndPermission error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Client, error) {
				return &Client{ClientID: 1, IdentityProvider: &IdentityProvider{TenantID: tenantID}}, nil
			},
		}
		caRepo := &mockClientAPIRepo{
			findByClientUUIDAndAPIUUIDFn: func(_, _ uuid.UUID) (*ClientAPI, error) {
				return &ClientAPI{ClientAPIID: 1}, nil
			},
		}
		permRepo := &mockPermissionRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Permission, error) {
				return &Permission{PermissionID: 1, TenantID: 1}, nil
			},
		}
		cpRepo := &mockClientPermissionRepo{
			removeByClientAPIAndPermissionFn: func(_, _ int64) error { return errors.New("del err") },
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			permRepo, cpRepo, caRepo, &mockClientRoleRepo{}, &mockRoleRepo{},
			&mockAPIRepo{}, actorUserRepo(tenantID), &mockTenantRepo{}, nil, nil)
		err := svc.RemoveClientAPIPermission(context.Background(), tenantID, cUUID, apiUUID, permUUID, actorUUID)
		require.Error(t, err)
	})
}

// ---------------------------------------------------------------------------
// Identity provider connections
// ---------------------------------------------------------------------------

func TestClientService_GetConnections(t *testing.T) {
	clientUUID := uuid.New()

	t.Run("client not found", func(t *testing.T) {
		gormDB, _ := newMockGormDB(t)
		clientRepo := &mockClientRepo{findByUUIDAndTenantIDFn: func(uuid.UUID, int64) (*Client, error) { return nil, nil }}
		svc := buildConnSvc(gormDB, clientRepo, &mockIdentityProviderRepo{}, &mockUserRepo{})
		result, err := svc.GetConnections(context.Background(), clientUUID, 1)
		require.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("repo error", func(t *testing.T) {
		gormDB, _ := newMockGormDB(t)
		clientRepo := &mockClientRepo{findByUUIDAndTenantIDFn: func(uuid.UUID, int64) (*Client, error) { return nil, errors.New("db error") }}
		svc := buildConnSvc(gormDB, clientRepo, &mockIdentityProviderRepo{}, &mockUserRepo{})
		result, err := svc.GetConnections(context.Background(), clientUUID, 1)
		require.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("success with connections", func(t *testing.T) {
		gormDB, _ := newMockGormDB(t)
		idpModel := IdentityProvider{IdentityProviderID: 1, IdentityProviderUUID: uuid.New(), Name: "google"}
		connections := []ClientIdentityProvider{{ClientIdentityProviderUUID: uuid.New(), Enabled: boolPtrTest(true), IsDefault: true, IdentityProvider: &idpModel}}
		clientRepo := &mockClientRepo{findByUUIDAndTenantIDFn: func(uuid.UUID, int64) (*Client, error) {
			return &Client{ClientID: 1, TenantID: 1, ConnectedProviders: &connections}, nil
		}}
		svc := buildConnSvc(gormDB, clientRepo, &mockIdentityProviderRepo{}, &mockUserRepo{})
		result, err := svc.GetConnections(context.Background(), clientUUID, 1)
		require.NoError(t, err)
		assert.Len(t, result, 1)
	})

	t.Run("success no connections", func(t *testing.T) {
		gormDB, _ := newMockGormDB(t)
		clientRepo := &mockClientRepo{findByUUIDAndTenantIDFn: func(uuid.UUID, int64) (*Client, error) {
			return &Client{ClientID: 1, TenantID: 1}, nil
		}}
		svc := buildConnSvc(gormDB, clientRepo, &mockIdentityProviderRepo{}, &mockUserRepo{})
		result, err := svc.GetConnections(context.Background(), clientUUID, 1)
		require.NoError(t, err)
		assert.Empty(t, result)
	})
}

func TestClientService_AddConnection(t *testing.T) {
	clientUUID := uuid.New()
	idpUUID := uuid.New()
	actorUUID := uuid.New()

	activeClient := func() *Client { return &Client{ClientID: 1, TenantID: 1, Name: "c1"} }
	// Must hold an identity in the tenant: every connection/URI mutation now
	// asserts tenant membership rather than only stamping an audit id.
	activeUser := func() *User { return actorUser(1) }
	activeIDP := func() *IdentityProvider { return &IdentityProvider{IdentityProviderID: 1, TenantID: 1, Name: "google"} }

	t.Run("client not found", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{findByUUIDAndTenantIDFn: func(uuid.UUID, int64) (*Client, error) { return nil, nil }}
		svc := buildConnSvc(gormDB, clientRepo, &mockIdentityProviderRepo{}, &mockUserRepo{})
		_, err := svc.AddConnection(context.Background(), clientUUID, 1, idpUUID, false, true, 0, actorUUID)
		require.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("actor not found", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{findByUUIDAndTenantIDFn: func(uuid.UUID, int64) (*Client, error) { return activeClient(), nil }}
		userRepo := &mockUserRepo{findByUUIDFn: func(any, ...string) (*User, error) { return nil, nil }}
		svc := buildConnSvc(gormDB, clientRepo, &mockIdentityProviderRepo{}, userRepo)
		_, err := svc.AddConnection(context.Background(), clientUUID, 1, idpUUID, false, true, 0, actorUUID)
		require.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("tenant mismatch", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{findByUUIDAndTenantIDFn: func(uuid.UUID, int64) (*Client, error) { return &Client{ClientID: 1, TenantID: 99}, nil }}
		userRepo := &mockUserRepo{findByUUIDFn: func(any, ...string) (*User, error) { return activeUser(), nil }}
		svc := buildConnSvc(gormDB, clientRepo, &mockIdentityProviderRepo{}, userRepo)
		_, err := svc.AddConnection(context.Background(), clientUUID, 1, idpUUID, false, true, 0, actorUUID)
		require.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("identity provider not found", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{findByUUIDAndTenantIDFn: func(uuid.UUID, int64) (*Client, error) { return activeClient(), nil }}
		userRepo := &mockUserRepo{findByUUIDFn: func(any, ...string) (*User, error) { return activeUser(), nil }}
		idpRepo := &mockIdentityProviderRepo{findByUUIDFn: func(any, ...string) (*IdentityProvider, error) { return nil, nil }}
		svc := buildConnSvc(gormDB, clientRepo, idpRepo, userRepo)
		_, err := svc.AddConnection(context.Background(), clientUUID, 1, idpUUID, false, true, 0, actorUUID)
		require.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("identity provider tenant mismatch", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{findByUUIDAndTenantIDFn: func(uuid.UUID, int64) (*Client, error) { return activeClient(), nil }}
		userRepo := &mockUserRepo{findByUUIDFn: func(any, ...string) (*User, error) { return activeUser(), nil }}
		idpRepo := &mockIdentityProviderRepo{findByUUIDFn: func(any, ...string) (*IdentityProvider, error) {
			return &IdentityProvider{IdentityProviderID: 1, TenantID: 99, Name: "google"}, nil
		}}
		svc := buildConnSvc(gormDB, clientRepo, idpRepo, userRepo)
		_, err := svc.AddConnection(context.Background(), clientUUID, 1, idpUUID, false, true, 0, actorUUID)
		require.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("duplicate connection", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT \* FROM "client_identity_providers" WHERE.*client_id = \$1.*identity_provider_id = \$2`).
			WithArgs(int64(1), int64(1), 1).
			WillReturnRows(cipRows())
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{findByUUIDAndTenantIDFn: func(uuid.UUID, int64) (*Client, error) { return activeClient(), nil }}
		userRepo := &mockUserRepo{findByUUIDFn: func(any, ...string) (*User, error) { return activeUser(), nil }}
		idpRepo := &mockIdentityProviderRepo{findByUUIDFn: func(any, ...string) (*IdentityProvider, error) { return activeIDP(), nil }}
		svc := buildConnSvc(gormDB, clientRepo, idpRepo, userRepo)
		_, err := svc.AddConnection(context.Background(), clientUUID, 1, idpUUID, false, true, 0, actorUUID)
		require.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("success", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT \* FROM "client_identity_providers" WHERE.*client_id = \$1.*identity_provider_id = \$2`).
			WithArgs(int64(1), int64(1), 1).
			WillReturnRows(cipEmptyRows())
		mock.ExpectQuery(`INSERT INTO "client_identity_providers"`).
			WillReturnRows(cipInsertRow())
		mock.ExpectCommit()
		clientRepo := &mockClientRepo{findByUUIDAndTenantIDFn: func(uuid.UUID, int64) (*Client, error) { return activeClient(), nil }}
		userRepo := &mockUserRepo{findByUUIDFn: func(any, ...string) (*User, error) { return activeUser(), nil }}
		idpRepo := &mockIdentityProviderRepo{findByUUIDFn: func(any, ...string) (*IdentityProvider, error) { return activeIDP(), nil }}
		svc := buildConnSvc(gormDB, clientRepo, idpRepo, userRepo)
		result, err := svc.AddConnection(context.Background(), clientUUID, 1, idpUUID, false, true, 0, actorUUID)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("success as default unsets others", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT \* FROM "client_identity_providers" WHERE.*client_id = \$1.*identity_provider_id = \$2`).
			WithArgs(int64(1), int64(1), 1).
			WillReturnRows(cipEmptyRows())
		mock.ExpectExec(`UPDATE "client_identity_providers" SET.*is_default`).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectQuery(`INSERT INTO "client_identity_providers"`).
			WillReturnRows(cipInsertRow())
		mock.ExpectCommit()
		clientRepo := &mockClientRepo{findByUUIDAndTenantIDFn: func(uuid.UUID, int64) (*Client, error) { return activeClient(), nil }}
		userRepo := &mockUserRepo{findByUUIDFn: func(any, ...string) (*User, error) { return activeUser(), nil }}
		idpRepo := &mockIdentityProviderRepo{findByUUIDFn: func(any, ...string) (*IdentityProvider, error) { return activeIDP(), nil }}
		svc := buildConnSvc(gormDB, clientRepo, idpRepo, userRepo)
		result, err := svc.AddConnection(context.Background(), clientUUID, 1, idpUUID, true, true, 0, actorUUID)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestClientService_UpdateConnection(t *testing.T) {
	clientUUID := uuid.New()
	connUUID := uuid.New()
	actorUUID := uuid.New()

	clientID10 := func() *Client { return &Client{ClientID: 10, TenantID: 1, Name: "c1"} }
	// Must hold an identity in the tenant: every connection/URI mutation now
	// asserts tenant membership rather than only stamping an audit id.
	activeUser := func() *User { return actorUser(1) }

	t.Run("client not found", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{findByUUIDAndTenantIDFn: func(uuid.UUID, int64) (*Client, error) { return nil, nil }}
		svc := buildConnSvc(gormDB, clientRepo, &mockIdentityProviderRepo{}, &mockUserRepo{})
		_, err := svc.UpdateConnection(context.Background(), clientUUID, 1, connUUID, boolPtrTest(false), boolPtrTest(true), intPtrTest(0), actorUUID)
		require.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("connection not found", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT \* FROM "client_identity_providers" WHERE.*client_identity_provider_uuid = \$1.*tenant_id = \$2`).
			WithArgs(connUUID.String(), int64(1), 1).
			WillReturnRows(cipEmptyRows())
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{findByUUIDAndTenantIDFn: func(uuid.UUID, int64) (*Client, error) { return clientID10(), nil }}
		userRepo := &mockUserRepo{findByUUIDFn: func(any, ...string) (*User, error) { return activeUser(), nil }}
		svc := buildConnSvc(gormDB, clientRepo, &mockIdentityProviderRepo{}, userRepo)
		_, err := svc.UpdateConnection(context.Background(), clientUUID, 1, connUUID, boolPtrTest(false), boolPtrTest(true), intPtrTest(0), actorUUID)
		require.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("connection belongs to another client", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT \* FROM "client_identity_providers" WHERE.*client_identity_provider_uuid = \$1.*tenant_id = \$2`).
			WithArgs(connUUID.String(), int64(1), 1).
			WillReturnRows(cipRows())
		mock.ExpectQuery(`SELECT \* FROM "identity_providers"`).WillReturnRows(idpPreloadRows())
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{findByUUIDAndTenantIDFn: func(uuid.UUID, int64) (*Client, error) { return &Client{ClientID: 1, TenantID: 1}, nil }}
		userRepo := &mockUserRepo{findByUUIDFn: func(any, ...string) (*User, error) { return activeUser(), nil }}
		svc := buildConnSvc(gormDB, clientRepo, &mockIdentityProviderRepo{}, userRepo)
		_, err := svc.UpdateConnection(context.Background(), clientUUID, 1, connUUID, boolPtrTest(false), boolPtrTest(true), intPtrTest(0), actorUUID)
		require.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("success", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT \* FROM "client_identity_providers" WHERE.*client_identity_provider_uuid = \$1.*tenant_id = \$2`).
			WithArgs(connUUID.String(), int64(1), 1).
			WillReturnRows(cipRows())
		mock.ExpectQuery(`SELECT \* FROM "identity_providers"`).WillReturnRows(idpPreloadRows())
		mock.ExpectExec(`UPDATE "client_identity_providers" SET`).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()
		clientRepo := &mockClientRepo{findByUUIDAndTenantIDFn: func(uuid.UUID, int64) (*Client, error) { return clientID10(), nil }}
		userRepo := &mockUserRepo{findByUUIDFn: func(any, ...string) (*User, error) { return activeUser(), nil }}
		svc := buildConnSvc(gormDB, clientRepo, &mockIdentityProviderRepo{}, userRepo)
		result, err := svc.UpdateConnection(context.Background(), clientUUID, 1, connUUID, boolPtrTest(false), boolPtrTest(true), intPtrTest(5), actorUUID)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestClientService_RemoveConnection(t *testing.T) {
	clientUUID := uuid.New()
	connUUID := uuid.New()
	actorUUID := uuid.New()

	clientID10 := func() *Client { return &Client{ClientID: 10, TenantID: 1, Name: "c1"} }
	// Must hold an identity in the tenant: every connection/URI mutation now
	// asserts tenant membership rather than only stamping an audit id.
	activeUser := func() *User { return actorUser(1) }

	t.Run("client not found", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{findByUUIDAndTenantIDFn: func(uuid.UUID, int64) (*Client, error) { return nil, nil }}
		svc := buildConnSvc(gormDB, clientRepo, &mockIdentityProviderRepo{}, &mockUserRepo{})
		_, err := svc.RemoveConnection(context.Background(), clientUUID, 1, connUUID, actorUUID)
		require.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("connection not found", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT \* FROM "client_identity_providers" WHERE.*client_identity_provider_uuid = \$1.*tenant_id = \$2`).
			WithArgs(connUUID.String(), int64(1), 1).
			WillReturnRows(cipEmptyRows())
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{findByUUIDAndTenantIDFn: func(uuid.UUID, int64) (*Client, error) { return clientID10(), nil }}
		userRepo := &mockUserRepo{findByUUIDFn: func(any, ...string) (*User, error) { return activeUser(), nil }}
		svc := buildConnSvc(gormDB, clientRepo, &mockIdentityProviderRepo{}, userRepo)
		_, err := svc.RemoveConnection(context.Background(), clientUUID, 1, connUUID, actorUUID)
		require.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("system connection cannot be removed", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT \* FROM "client_identity_providers" WHERE.*client_identity_provider_uuid = \$1.*tenant_id = \$2`).
			WithArgs(connUUID.String(), int64(1), 1).
			WillReturnRows(cipRows())
		mock.ExpectQuery(`SELECT \* FROM "identity_providers"`).WillReturnRows(idpPreloadSystemRows())
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{findByUUIDAndTenantIDFn: func(uuid.UUID, int64) (*Client, error) { return clientID10(), nil }}
		userRepo := &mockUserRepo{findByUUIDFn: func(any, ...string) (*User, error) { return activeUser(), nil }}
		svc := buildConnSvc(gormDB, clientRepo, &mockIdentityProviderRepo{}, userRepo)
		_, err := svc.RemoveConnection(context.Background(), clientUUID, 1, connUUID, actorUUID)
		require.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("success", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT \* FROM "client_identity_providers" WHERE.*client_identity_provider_uuid = \$1.*tenant_id = \$2`).
			WithArgs(connUUID.String(), int64(1), 1).
			WillReturnRows(cipRows())
		mock.ExpectQuery(`SELECT \* FROM "identity_providers"`).WillReturnRows(idpPreloadRows())
		// The client-stays-usable invariant counts the remaining enabled
		// connections before allowing the removal.
		mock.ExpectQuery(`SELECT \* FROM "client_identity_providers" WHERE client_id = \$1`).
			WillReturnRows(cipTwoEnabledRows())
		// FindByClientID preloads the provider relation for each connection.
		mock.ExpectQuery(`SELECT \* FROM "identity_providers"`).WillReturnRows(idpPreloadRows())
		mock.ExpectExec(`UPDATE "client_identity_providers" SET "deleted_at"=`).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()
		clientRepo := &mockClientRepo{findByUUIDAndTenantIDFn: func(uuid.UUID, int64) (*Client, error) { return clientID10(), nil }}
		userRepo := &mockUserRepo{findByUUIDFn: func(any, ...string) (*User, error) { return activeUser(), nil }}
		svc := buildConnSvc(gormDB, clientRepo, &mockIdentityProviderRepo{}, userRepo)
		result, err := svc.RemoveConnection(context.Background(), clientUUID, 1, connUUID, actorUUID)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	// Removing the last enabled connection would leave the client with no way to
	// sign in — the hosted login page would render no options at all.
	t.Run("only enabled connection cannot be removed", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT \* FROM "client_identity_providers" WHERE.*client_identity_provider_uuid = \$1.*tenant_id = \$2`).
			WithArgs(connUUID.String(), int64(1), 1).
			WillReturnRows(cipRows())
		mock.ExpectQuery(`SELECT \* FROM "identity_providers"`).WillReturnRows(idpPreloadRows())
		mock.ExpectQuery(`SELECT \* FROM "client_identity_providers" WHERE client_id = \$1`).
			WillReturnRows(cipOnlyOneEnabledRow())
		// FindByClientID preloads the provider relation for each connection.
		mock.ExpectQuery(`SELECT \* FROM "identity_providers"`).WillReturnRows(idpPreloadRows())
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{findByUUIDAndTenantIDFn: func(uuid.UUID, int64) (*Client, error) { return clientID10(), nil }}
		userRepo := &mockUserRepo{findByUUIDFn: func(any, ...string) (*User, error) { return activeUser(), nil }}
		svc := buildConnSvc(gormDB, clientRepo, &mockIdentityProviderRepo{}, userRepo)
		_, err := svc.RemoveConnection(context.Background(), clientUUID, 1, connUUID, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "only enabled identity provider")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestClientService_IsManagementClient(t *testing.T) {
	tests := []struct {
		name   string
		repoFn func(string) (*Client, error)
		input  string
		want   bool
	}{
		{
			name: "auth-console system client is a management client",
			repoFn: func(string) (*Client, error) {
				return &Client{Name: shared.SystemClientNameAuthConsole, IsSystem: true}, nil
			},
			input: "id",
			want:  true,
		},
		{
			name: "non-system client named auth-console is rejected",
			repoFn: func(string) (*Client, error) {
				return &Client{Name: shared.SystemClientNameAuthConsole, IsSystem: false}, nil
			},
			input: "id",
			want:  false,
		},
		{
			name: "other system client is rejected",
			repoFn: func(string) (*Client, error) {
				return &Client{Name: shared.SystemClientNameAuthIdentity, IsSystem: true}, nil
			},
			input: "id",
			want:  false,
		},
		{
			name:   "client not found is rejected",
			repoFn: func(string) (*Client, error) { return nil, nil },
			input:  "id",
			want:   false,
		},
		{
			name:   "repo error is rejected",
			repoFn: func(string) (*Client, error) { return nil, errors.New("db error") },
			input:  "id",
			want:   false,
		},
		{
			name: "empty identifier is rejected without a lookup",
			repoFn: func(string) (*Client, error) {
				t.Fatalf("repo must not be queried for an empty identifier")
				return nil, nil
			},
			input: "  ",
			want:  false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := &clientService{clientRepo: &mockClientRepo{findByIdentifierFn: tc.repoFn}}
			assert.Equal(t, tc.want, s.IsManagementClient(context.Background(), tc.input))
		})
	}
}

func testBoolPtr(b bool) *bool { return &b }

// boolPtrTest builds the *bool the client models now use, where nil means
// "fall back to the DB default".
func boolPtrTest(v bool) *bool { return &v }

// intPtrTest builds the *int the connection DTO now uses, where nil means
// "leave the stored value alone".
func intPtrTest(v int) *int { return &v }

// A client role grants the client permissions, so these two mutations are the most
// sensitive in the domain — and they had no service-level coverage at all.
func TestClientService_ClientRoleGrants(t *testing.T) {
	cUUID := uuid.New()
	roleUUID := uuid.New()
	actorUUID := uuid.New()
	tenantID := int64(1)

	buildSvc := func(clientRepo *mockClientRepo, roleRepo *mockRoleRepo, clientRoleRepo *mockClientRoleRepo) ClientService {
		gormDB, _ := newMockGormDB(t)
		return NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{}, clientRoleRepo, roleRepo,
			&mockAPIRepo{}, actorUserRepo(tenantID), &mockTenantRepo{}, nil, nil)
	}
	inTenantClient := &mockClientRepo{
		findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*Client, error) {
			return &Client{ClientID: 10, TenantID: tenantID}, nil
		},
	}
	inTenantRole := &mockRoleRepo{
		findByUUIDFn: func(_ any, _ ...string) (*Role, error) {
			return &Role{RoleID: 20, TenantID: tenantID}, nil
		},
	}

	t.Run("assign stamps the actor as created_by", func(t *testing.T) {
		var gotCreatedBy *int64
		var gotClientID, gotRoleID int64
		clientRoleRepo := &mockClientRoleRepo{
			assignRoleFn: func(clientID, roleID int64, createdBy *int64) (*ClientRole, error) {
				gotClientID, gotRoleID, gotCreatedBy = clientID, roleID, createdBy
				return &ClientRole{ClientRoleUUID: uuid.New()}, nil
			},
		}
		svc := buildSvc(inTenantClient, inTenantRole, clientRoleRepo)

		result, err := svc.AssignClientRole(context.Background(), cUUID, roleUUID, tenantID, actorUUID)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, int64(10), gotClientID)
		assert.Equal(t, int64(20), gotRoleID)
		// The handler used to pass nil here, so every grant was recorded authorless.
		require.NotNil(t, gotCreatedBy)
		assert.Equal(t, int64(1), *gotCreatedBy)
	})

	t.Run("remove resolves the client and role", func(t *testing.T) {
		removed := false
		clientRoleRepo := &mockClientRoleRepo{
			removeRoleFn: func(clientID, roleID int64) error {
				removed = clientID == 10 && roleID == 20
				return nil
			},
		}
		svc := buildSvc(inTenantClient, inTenantRole, clientRoleRepo)

		require.NoError(t, svc.RemoveClientRole(context.Background(), cUUID, roleUUID, tenantID, actorUUID))
		assert.True(t, removed)
	})

	// A distinct "belongs to another tenant" answer confirms the UUID exists, which
	// makes these endpoints an existence oracle across tenants.
	t.Run("a cross-tenant client is not found, not forbidden", func(t *testing.T) {
		crossTenant := &mockClientRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*Client, error) { return nil, nil },
		}
		svc := buildSvc(crossTenant, inTenantRole, &mockClientRoleRepo{})

		_, err := svc.AssignClientRole(context.Background(), cUUID, roleUUID, tenantID, actorUUID)
		require.Error(t, err)
		var notFound *apperror.NotFoundError
		assert.ErrorAs(t, err, &notFound)
		assert.Contains(t, err.Error(), "not found or access denied")

		err = svc.RemoveClientRole(context.Background(), cUUID, roleUUID, tenantID, actorUUID)
		require.Error(t, err)
		assert.ErrorAs(t, err, &notFound)
	})

	t.Run("a cross-tenant role is not found, and is never granted", func(t *testing.T) {
		foreignRole := &mockRoleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Role, error) {
				return &Role{RoleID: 20, TenantID: 99}, nil
			},
		}
		assigned, removedCalled := false, false
		clientRoleRepo := &mockClientRoleRepo{
			assignRoleFn: func(int64, int64, *int64) (*ClientRole, error) {
				assigned = true
				return &ClientRole{}, nil
			},
			removeRoleFn: func(int64, int64) error { removedCalled = true; return nil },
		}
		svc := buildSvc(inTenantClient, foreignRole, clientRoleRepo)

		_, err := svc.AssignClientRole(context.Background(), cUUID, roleUUID, tenantID, actorUUID)
		require.Error(t, err)
		var notFound *apperror.NotFoundError
		assert.ErrorAs(t, err, &notFound)
		assert.False(t, assigned, "a role from another tenant must never be granted")

		// RemoveClientRole had no tenant check at all before this.
		err = svc.RemoveClientRole(context.Background(), cUUID, roleUUID, tenantID, actorUUID)
		require.Error(t, err)
		assert.False(t, removedCalled)
	})

	t.Run("an actor outside the tenant cannot grant a role", func(t *testing.T) {
		gormDB, _ := newMockGormDB(t)
		foreignActor := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*User, error) {
				return &User{UserID: 7, UserIdentities: []UserIdentity{
					{TenantID: 99, Tenant: &Tenant{TenantID: 99, IsSystem: false}},
				}}, nil
			},
		}
		svc := NewClientService(gormDB, inTenantClient, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{}, &mockClientRoleRepo{}, inTenantRole,
			&mockAPIRepo{}, foreignActor, &mockTenantRepo{}, nil, nil)

		_, err := svc.AssignClientRole(context.Background(), cUUID, roleUUID, tenantID, actorUUID)
		require.Error(t, err)
		err = svc.RemoveClientRole(context.Background(), cUUID, roleUUID, tenantID, actorUUID)
		require.Error(t, err)
	})
}
