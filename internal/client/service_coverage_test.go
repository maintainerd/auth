package client

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPIKeyService_RemainingBranches(t *testing.T) {
	t.Run("generate api key returns random error", func(t *testing.T) {
		original := apiKeyRandRead
		apiKeyRandRead = func([]byte) (int, error) { return 0, errors.New("entropy error") }
		t.Cleanup(func() { apiKeyRandRead = original })

		svc := &apiKeyService{}
		plain, hash, prefix, err := svc.generateAPIKey()

		require.Error(t, err)
		assert.Empty(t, plain)
		assert.Empty(t, hash)
		assert.Empty(t, prefix)
	})

	t.Run("create returns generator error", func(t *testing.T) {
		original := apiKeyRandRead
		apiKeyRandRead = func([]byte) (int, error) { return 0, errors.New("entropy error") }
		t.Cleanup(func() { apiKeyRandRead = original })

		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()

		svc := NewAPIKeyService(db, &mockAPIKeyRepo{}, &mockAPIKeyAPIRepo{}, &mockAPIKeyPermissionRepo{}, &mockAPIRepo{}, &mockUserRepo{}, &mockPermissionRepo{}, nil)
		result, plain, err := svc.Create(context.Background(), tenantID, "key", "desc", nil, nil, shared.StatusActive)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Empty(t, plain)
	})

	akUUID := uuid.New()
	apiUUID := uuid.New()
	permUUID := uuid.New()

	for _, tc := range []struct {
		name string
		run  func(APIKeyService) error
	}{
		{
			name: "get api key apis api key find error",
			run: func(s APIKeyService) error {
				_, err := s.GetAPIKeyAPIs(context.Background(), tenantID, akUUID, 0, 0, "", "")
				return err
			},
		},
		{
			name: "remove api key api api key find error",
			run: func(s APIKeyService) error {
				return s.RemoveAPIKeyAPI(context.Background(), tenantID, akUUID, apiUUID)
			},
		},
		{
			name: "get api key api permissions api key find error",
			run: func(s APIKeyService) error {
				_, err := s.GetAPIKeyAPIPermissions(context.Background(), tenantID, akUUID, apiUUID)
				return err
			},
		},
		{
			name: "add api key api permissions api key find error",
			run: func(s APIKeyService) error {
				return s.AddAPIKeyAPIPermissions(context.Background(), tenantID, akUUID, apiUUID, []uuid.UUID{permUUID})
			},
		},
		{
			name: "remove api key api permission api key find error",
			run: func(s APIKeyService) error {
				return s.RemoveAPIKeyAPIPermission(context.Background(), tenantID, akUUID, apiUUID, permUUID)
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db, mock := newMockGormDB(t)
			if tc.name != "get api key apis api key find error" && tc.name != "get api key api permissions api key find error" {
				mock.ExpectBegin()
				mock.ExpectRollback()
			}
			akRepo := &mockAPIKeyRepo{
				findByUUIDAndTenantIDFn: func(_ string, _ int64) (*APIKey, error) {
					return nil, errors.New("api key find error")
				},
			}
			svc := NewAPIKeyService(db, akRepo, &mockAPIKeyAPIRepo{}, &mockAPIKeyPermissionRepo{}, &mockAPIRepo{}, &mockUserRepo{}, &mockPermissionRepo{}, nil)

			err := tc.run(svc)

			require.Error(t, err)
			assert.Contains(t, err.Error(), "api key find error")
		})
	}

	for _, tc := range []struct {
		name string
		run  func(APIKeyService) error
	}{
		{
			name: "get api key apis api key not found",
			run: func(s APIKeyService) error {
				_, err := s.GetAPIKeyAPIs(context.Background(), tenantID, akUUID, 0, 0, "", "")
				return err
			},
		},
		{
			name: "remove api key api api key not found",
			run: func(s APIKeyService) error {
				return s.RemoveAPIKeyAPI(context.Background(), tenantID, akUUID, apiUUID)
			},
		},
		{
			name: "get api key api permissions api key not found",
			run: func(s APIKeyService) error {
				_, err := s.GetAPIKeyAPIPermissions(context.Background(), tenantID, akUUID, apiUUID)
				return err
			},
		},
		{
			name: "add api key api permissions api key not found",
			run: func(s APIKeyService) error {
				return s.AddAPIKeyAPIPermissions(context.Background(), tenantID, akUUID, apiUUID, []uuid.UUID{permUUID})
			},
		},
		{
			name: "remove api key api permission api key not found",
			run: func(s APIKeyService) error {
				return s.RemoveAPIKeyAPIPermission(context.Background(), tenantID, akUUID, apiUUID, permUUID)
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db, mock := newMockGormDB(t)
			if tc.name != "get api key apis api key not found" && tc.name != "get api key api permissions api key not found" {
				mock.ExpectBegin()
				mock.ExpectRollback()
			}
			svc := NewAPIKeyService(db, &mockAPIKeyRepo{}, &mockAPIKeyAPIRepo{}, &mockAPIKeyPermissionRepo{}, &mockAPIRepo{}, &mockUserRepo{}, &mockPermissionRepo{}, nil)

			err := tc.run(svc)

			require.Error(t, err)
			assert.Contains(t, err.Error(), "not found")
		})
	}

	t.Run("add api key apis rejects cross tenant api", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		akRepo := &mockAPIKeyRepo{findByUUIDAndTenantIDFn: func(string, int64) (*APIKey, error) {
			return &APIKey{APIKeyID: 1, APIKeyUUID: akUUID, TenantID: tenantID}, nil
		}}
		apiRepo := &mockAPIRepo{findByUUIDFn: func(any, ...string) (*API, error) {
			return &API{APIID: 1, TenantID: tenantID + 1}, nil
		}}
		svc := NewAPIKeyService(db, akRepo, &mockAPIKeyAPIRepo{}, &mockAPIKeyPermissionRepo{}, apiRepo, &mockUserRepo{}, &mockPermissionRepo{}, nil)

		err := svc.AddAPIKeyAPIs(context.Background(), tenantID, akUUID, []uuid.UUID{apiUUID})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "access denied")
	})

	for _, tc := range []struct {
		name      string
		apiTenant int64
		perm      *Permission
		run       func(APIKeyService) error
	}{
		{
			name:      "add permissions rejects cross tenant api",
			apiTenant: tenantID + 1,
			perm:      &Permission{PermissionID: 1, APIID: 1, TenantID: tenantID},
			run: func(s APIKeyService) error {
				return s.AddAPIKeyAPIPermissions(context.Background(), tenantID, akUUID, apiUUID, []uuid.UUID{permUUID})
			},
		},
		{
			name:      "add permissions rejects cross tenant permission",
			apiTenant: tenantID,
			perm:      &Permission{PermissionID: 1, APIID: 1, TenantID: tenantID + 1},
			run: func(s APIKeyService) error {
				return s.AddAPIKeyAPIPermissions(context.Background(), tenantID, akUUID, apiUUID, []uuid.UUID{permUUID})
			},
		},
		{
			name:      "remove permission rejects cross tenant api",
			apiTenant: tenantID + 1,
			perm:      &Permission{PermissionID: 1, APIID: 1, TenantID: tenantID},
			run: func(s APIKeyService) error {
				return s.RemoveAPIKeyAPIPermission(context.Background(), tenantID, akUUID, apiUUID, permUUID)
			},
		},
		{
			name:      "remove permission rejects cross tenant permission",
			apiTenant: tenantID,
			perm:      &Permission{PermissionID: 1, APIID: 1, TenantID: tenantID + 1},
			run: func(s APIKeyService) error {
				return s.RemoveAPIKeyAPIPermission(context.Background(), tenantID, akUUID, apiUUID, permUUID)
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db, mock := newMockGormDB(t)
			mock.ExpectBegin()
			mock.ExpectRollback()
			akRepo := &mockAPIKeyRepo{findByUUIDAndTenantIDFn: func(string, int64) (*APIKey, error) {
				return &APIKey{APIKeyID: 1, APIKeyUUID: akUUID, TenantID: tenantID}, nil
			}}
			akaRepo := &mockAPIKeyAPIRepo{findByAPIKeyUUIDAndAPIUUIDFn: func(uuid.UUID, uuid.UUID) (*APIKeyAPI, error) {
				return &APIKeyAPI{APIKeyAPIID: 1}, nil
			}}
			apiRepo := &mockAPIRepo{findByUUIDFn: func(any, ...string) (*API, error) {
				return &API{APIID: 1, TenantID: tc.apiTenant}, nil
			}}
			permRepo := &mockPermissionRepo{findByUUIDFn: func(any, ...string) (*Permission, error) {
				return tc.perm, nil
			}}
			svc := NewAPIKeyService(db, akRepo, akaRepo, &mockAPIKeyPermissionRepo{}, apiRepo, &mockUserRepo{}, permRepo, nil)

			err := tc.run(svc)

			require.Error(t, err)
			assert.Contains(t, err.Error(), "access denied")
		})
	}
}

func TestClientService_RemainingBranches(t *testing.T) {
	t.Run("create returns hash error", func(t *testing.T) {
		original := hashClientSecret
		hashClientSecret = func(context.Context, string) (string, error) { return "", errors.New("hash error") }
		t.Cleanup(func() { hashClientSecret = original })

		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		idpRepo := &mockIdentityProviderRepo{
			findByUUIDFn: func(any, ...string) (*IdentityProvider, error) {
				return &IdentityProvider{IdentityProviderID: 1, Tenant: &Tenant{TenantID: tenantID, IsSystem: true}}, nil
			},
		}
		userRepo := &mockUserRepo{findByUUIDFn: func(any, ...string) (*User, error) { return actorUser(tenantID), nil }}
		svc := NewClientService(db, &mockClientRepo{}, &mockClientURIRepo{}, idpRepo, &mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{}, &mockAPIRepo{}, userRepo, &mockTenantRepo{}, nil, nil)

		result, err := svc.Create(context.Background(), tenantID, "name", "Name", "public", "example.com", nil, shared.StatusActive, false, uuid.NewString(), nil, true, uuid.New())

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "hash error")
	})

	t.Run("create returns encrypt error", func(t *testing.T) {
		original := encryptClientSecret
		encryptClientSecret = func(string) (string, error) { return "", errors.New("encrypt error") }
		t.Cleanup(func() { encryptClientSecret = original })

		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		idpRepo := &mockIdentityProviderRepo{
			findByUUIDFn: func(any, ...string) (*IdentityProvider, error) {
				return &IdentityProvider{IdentityProviderID: 1, Tenant: &Tenant{TenantID: tenantID, IsSystem: true}}, nil
			},
		}
		userRepo := &mockUserRepo{findByUUIDFn: func(any, ...string) (*User, error) { return actorUser(tenantID), nil }}
		svc := NewClientService(db, &mockClientRepo{}, &mockClientURIRepo{}, idpRepo, &mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{}, &mockAPIRepo{}, userRepo, &mockTenantRepo{}, nil, nil)

		result, err := svc.Create(context.Background(), tenantID, "name", "Name", "public", "example.com", nil, shared.StatusActive, false, uuid.NewString(), nil, true, uuid.New())

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "encrypt error")
	})

	t.Run("create success includes generated identifier", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		expectClientIdentityProviderConnectionInsert(mock)
		mock.ExpectCommit()
		idpRepo := &mockIdentityProviderRepo{
			findByUUIDFn: func(any, ...string) (*IdentityProvider, error) {
				return &IdentityProvider{IdentityProviderID: 1, Tenant: &Tenant{TenantID: tenantID, IsSystem: true}}, nil
			},
		}
		userRepo := &mockUserRepo{findByUUIDFn: func(any, ...string) (*User, error) { return actorUser(tenantID), nil }}
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(any, ...string) (*Client, error) {
				identifier := "client-id"
				return &Client{ClientUUID: uuid.New(), Name: "name", Identifier: &identifier, IdentityProvider: &IdentityProvider{}, TenantID: tenantID}, nil
			},
		}
		svc := NewClientService(db, clientRepo, &mockClientURIRepo{}, idpRepo, &mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{}, &mockAPIRepo{}, userRepo, &mockTenantRepo{}, nil, nil)

		result, err := svc.Create(context.Background(), tenantID, "name", "Name", "public", "example.com", nil, shared.StatusActive, false, uuid.NewString(), nil, true, uuid.New())

		require.NoError(t, err)
		assert.Equal(t, "client-id", result.ClientIdentifier)
	})

	t.Run("rotate secret returns generator error", func(t *testing.T) {
		original := generateClientIdentifier
		generateClientIdentifier = func(int) (string, error) { return "", errors.New("generator error") }
		t.Cleanup(func() { generateClientIdentifier = original })

		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{findByUUIDAndTenantIDFn: func(uuid.UUID, int64) (*Client, error) {
			return &Client{ClientUUID: uuid.New(), TenantID: tenantID}, nil
		}}
		svc := NewClientService(db, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{}, &mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{}, &mockAPIRepo{}, &mockUserRepo{}, &mockTenantRepo{}, nil, nil)

		secret, err := svc.RotateSecret(context.Background(), uuid.New(), tenantID, uuid.New(), 0)

		require.Error(t, err)
		assert.Empty(t, secret)
	})

	t.Run("rotate secret returns hash error", func(t *testing.T) {
		original := hashClientSecret
		hashClientSecret = func(context.Context, string) (string, error) { return "", errors.New("hash error") }
		t.Cleanup(func() { hashClientSecret = original })

		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{findByUUIDAndTenantIDFn: func(uuid.UUID, int64) (*Client, error) {
			return &Client{ClientUUID: uuid.New(), TenantID: tenantID}, nil
		}}
		svc := NewClientService(db, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{}, &mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{}, &mockAPIRepo{}, &mockUserRepo{}, &mockTenantRepo{}, nil, nil)

		secret, err := svc.RotateSecret(context.Background(), uuid.New(), tenantID, uuid.New(), 0)

		require.Error(t, err)
		assert.Empty(t, secret)
	})

	t.Run("rotate secret returns encrypt error", func(t *testing.T) {
		original := encryptClientSecret
		encryptClientSecret = func(string) (string, error) { return "", errors.New("encrypt error") }
		t.Cleanup(func() { encryptClientSecret = original })

		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{findByUUIDAndTenantIDFn: func(uuid.UUID, int64) (*Client, error) {
			return &Client{ClientUUID: uuid.New(), TenantID: tenantID}, nil
		}}
		svc := NewClientService(db, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{}, &mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{}, &mockAPIRepo{}, &mockUserRepo{}, &mockTenantRepo{}, nil, nil)

		secret, err := svc.RotateSecret(context.Background(), uuid.New(), tenantID, uuid.New(), 0)

		require.Error(t, err)
		assert.Empty(t, secret)
	})

	t.Run("create uri rejects invalid redirect uri", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{findByUUIDAndTenantIDFn: func(uuid.UUID, int64) (*Client, error) {
			return &Client{ClientID: 1, ClientUUID: uuid.New(), TenantID: tenantID}, nil
		}}
		userRepo := &mockUserRepo{findByUUIDFn: func(any, ...string) (*User, error) { return actorUser(tenantID), nil }}
		svc := NewClientService(db, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{}, &mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{}, &mockAPIRepo{}, userRepo, &mockTenantRepo{}, nil, nil)

		result, err := svc.CreateURI(context.Background(), uuid.New(), tenantID, "javascript:alert(1)", shared.ClientURITypeRedirect, uuid.New())

		require.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("update uri rejects invalid redirect uri", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{findByUUIDAndTenantIDFn: func(uuid.UUID, int64) (*Client, error) {
			return &Client{ClientID: 1, ClientUUID: uuid.New(), TenantID: tenantID}, nil
		}}
		uriRepo := &mockClientURIRepo{findByUUIDAndTenantIDFn: func(string, int64) (*ClientURI, error) {
			return &ClientURI{ClientID: 1}, nil
		}}
		userRepo := &mockUserRepo{findByUUIDFn: func(any, ...string) (*User, error) { return actorUser(tenantID), nil }}
		svc := NewClientService(db, clientRepo, uriRepo, &mockIdentityProviderRepo{}, &mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{}, &mockAPIRepo{}, userRepo, &mockTenantRepo{}, nil, nil)

		result, err := svc.UpdateURI(context.Background(), uuid.New(), tenantID, uuid.New(), "javascript:alert(1)", shared.ClientURITypeRedirect, uuid.New())

		require.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("get client apis client find error", func(t *testing.T) {
		clientRepo := &mockClientRepo{findByUUIDAndTenantIDFn: func(uuid.UUID, int64) (*Client, error) {
			return nil, errors.New("client find error")
		}}
		svc := buildFullClientService(t, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{}, &mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{}, &mockAPIRepo{}, &mockUserRepo{}, &mockTenantRepo{}, nil)

		result, err := svc.GetClientAPIs(context.Background(), tenantID, uuid.New())

		require.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("get client apis client not found", func(t *testing.T) {
		svc := buildFullClientService(t, &mockClientRepo{}, &mockClientURIRepo{}, &mockIdentityProviderRepo{}, &mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{}, &mockAPIRepo{}, &mockUserRepo{}, &mockTenantRepo{}, nil)

		result, err := svc.GetClientAPIs(context.Background(), tenantID, uuid.New())

		require.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("validate tenant access edge cases", func(t *testing.T) {
		assert.Error(t, ValidateTenantAccess(nil, &Tenant{TenantID: tenantID}))
		assert.Error(t, ValidateTenantAccess(&User{}, nil))
		// Same-tenant identity is allowed.
		assert.NoError(t, ValidateTenantAccess(&User{UserIdentities: []UserIdentity{{TenantID: tenantID, Tenant: &Tenant{TenantID: tenantID}}}}, &Tenant{TenantID: tenantID}))
		// Lockdown: a system-tenant identity no longer grants cross-tenant access.
		assert.Error(t, ValidateTenantAccess(&User{UserIdentities: []UserIdentity{{TenantID: 2, Tenant: &Tenant{TenantID: 2, IsSystem: true}}}}, &Tenant{TenantID: tenantID}))
	})
}
