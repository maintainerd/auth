package idp

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/platform/crypto"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func newIDP(tenantID int64, name string) *IdentityProvider {
	return &IdentityProvider{
		IdentityProviderID:   1,
		IdentityProviderUUID: uuid.New(),
		TenantID:             tenantID,
		Name:                 name,
		DisplayName:          name,
		Provider:             "local",
		ProviderType:         "password",
		Status:               shared.StatusActive,
		Tenant:               &Tenant{TenantID: tenantID},
	}
}

func actorUserWithDefaultTenant(tenantID int64) *User {
	return &User{
		UserID: 1,
		UserIdentities: []UserIdentity{
			{TenantID: tenantID, Tenant: &Tenant{TenantID: tenantID, IsSystem: true}},
		},
	}
}

// createInput / updateInput keep the positional test call sites compact while the
// service now takes input structs.
func createInput(name, display, provider, providerType string, config datatypes.JSON, status, tenantUUID string, tenantID int64, actor uuid.UUID) IdentityProviderCreateInput {
	return IdentityProviderCreateInput{
		Name: name, DisplayName: display, Provider: provider, ProviderType: providerType,
		Config: config, Status: status, TenantUUID: tenantUUID, TenantID: tenantID, ActorUserUUID: actor,
	}
}

func updateInput(id uuid.UUID, name, display, provider, providerType string, config datatypes.JSON, status string, tenantID int64, actor uuid.UUID) IdentityProviderUpdateInput {
	return IdentityProviderUpdateInput{
		IdpUUID: id, Name: name, DisplayName: display, Provider: provider, ProviderType: providerType,
		Config: config, Status: status, TenantID: tenantID, ActorUserUUID: actor,
	}
}

// ---------------------------------------------------------------------------
// IdentityProviderService.Get
// ---------------------------------------------------------------------------

func TestIdentityProviderService_Get(t *testing.T) {
	tenantID := int64(1)

	t.Run("repo error → propagated", func(t *testing.T) {
		idpRepo := &mockIdentityProviderRepo{
			findPaginatedFn: func(_ IdentityProviderRepositoryGetFilter) (*PaginationResult[IdentityProvider], error) {
				return nil, errors.New("db error")
			},
		}
		svc := NewIdentityProviderService(nil, idpRepo, &mockIdentityProviderEmailDomainRepo{}, nil, &mockTenantRepo{}, &mockUserRepo{})
		_, err := svc.Get(context.Background(), IdentityProviderServiceGetFilter{TenantID: tenantID, Page: 1, Limit: 10})
		require.Error(t, err)
	})

	t.Run("success → returns mapped results", func(t *testing.T) {
		idp := newIDP(tenantID, "local")
		idpRepo := &mockIdentityProviderRepo{
			findPaginatedFn: func(_ IdentityProviderRepositoryGetFilter) (*PaginationResult[IdentityProvider], error) {
				return &PaginationResult[IdentityProvider]{
					Data: []IdentityProvider{*idp}, Total: 1, Page: 1, Limit: 10, TotalPages: 1,
				}, nil
			},
		}
		svc := NewIdentityProviderService(nil, idpRepo, &mockIdentityProviderEmailDomainRepo{}, nil, &mockTenantRepo{}, &mockUserRepo{})
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
			findByUUIDFn: func(_ any, _ ...string) (*IdentityProvider, error) { return nil, nil },
		}
		svc := NewIdentityProviderService(nil, idpRepo, &mockIdentityProviderEmailDomainRepo{}, nil, &mockTenantRepo{}, &mockUserRepo{})
		_, err := svc.GetByUUID(context.Background(), idpUUID, tenantID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("wrong tenant → access denied", func(t *testing.T) {
		idpRepo := &mockIdentityProviderRepo{
			findByUUIDFn: func(_ any, _ ...string) (*IdentityProvider, error) {
				return newIDP(999, "other"), nil // different tenant
			},
		}
		svc := NewIdentityProviderService(nil, idpRepo, &mockIdentityProviderEmailDomainRepo{}, nil, &mockTenantRepo{}, &mockUserRepo{})
		_, err := svc.GetByUUID(context.Background(), idpUUID, tenantID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "access denied")
	})

	t.Run("found, same tenant → success", func(t *testing.T) {
		idp := newIDP(tenantID, "local")
		idpRepo := &mockIdentityProviderRepo{
			findByUUIDFn: func(_ any, _ ...string) (*IdentityProvider, error) { return idp, nil },
		}
		svc := NewIdentityProviderService(nil, idpRepo, &mockIdentityProviderEmailDomainRepo{}, nil, &mockTenantRepo{}, &mockUserRepo{})
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
			findByUUIDFn: func(_ any, _ ...string) (*IdentityProvider, error) { return nil, nil },
		}
		svc := NewIdentityProviderService(nil, idpRepo, &mockIdentityProviderEmailDomainRepo{}, nil, &mockTenantRepo{}, &mockUserRepo{})
		_, err := svc.DeleteByUUID(context.Background(), idpUUID, tenantID, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("wrong tenant → access denied", func(t *testing.T) {
		idpRepo := &mockIdentityProviderRepo{
			findByUUIDFn: func(_ any, _ ...string) (*IdentityProvider, error) {
				return newIDP(999, "other"), nil
			},
		}
		svc := NewIdentityProviderService(nil, idpRepo, &mockIdentityProviderEmailDomainRepo{}, nil, &mockTenantRepo{}, &mockUserRepo{})
		_, err := svc.DeleteByUUID(context.Background(), idpUUID, tenantID, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "access denied")
	})

	t.Run("actor user not found → error", func(t *testing.T) {
		idpRepo := &mockIdentityProviderRepo{
			findByUUIDFn: func(_ any, _ ...string) (*IdentityProvider, error) {
				return newIDP(tenantID, "local"), nil
			},
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*User, error) { return nil, nil },
		}
		svc := NewIdentityProviderService(nil, idpRepo, &mockIdentityProviderEmailDomainRepo{}, nil, &mockTenantRepo{}, userRepo)
		_, err := svc.DeleteByUUID(context.Background(), idpUUID, tenantID, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "actor user not found")
	})

	t.Run("system idp → cannot delete", func(t *testing.T) {
		idp := newIDP(tenantID, "sys")
		idp.IsSystem = true
		idpRepo := &mockIdentityProviderRepo{
			findByUUIDFn: func(_ any, _ ...string) (*IdentityProvider, error) { return idp, nil },
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*User, error) {
				return actorUserWithDefaultTenant(tenantID), nil
			},
		}
		svc := NewIdentityProviderService(nil, idpRepo, &mockIdentityProviderEmailDomainRepo{}, nil, &mockTenantRepo{}, userRepo)
		_, err := svc.DeleteByUUID(context.Background(), idpUUID, tenantID, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "system idp")
	})

	t.Run("success → deleted clears child rows", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		idp := newIDP(tenantID, "local")
		var domainsCleared, audiencesCleared bool
		idpRepo := &mockIdentityProviderRepo{
			findByUUIDFn: func(_ any, _ ...string) (*IdentityProvider, error) { return idp, nil },
		}
		emailRepo := &mockIdentityProviderEmailDomainRepo{
			replaceForProviderFn: func(_ int64, _ int64, domains []string) error {
				domainsCleared = domains == nil
				return nil
			},
		}
		audienceRepo := &mockIdentityProviderAllowedAudienceRepo{
			replaceForProviderFn: func(_ int64, _ int64, audiences []string) error {
				audiencesCleared = audiences == nil
				return nil
			},
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*User, error) {
				return actorUserWithDefaultTenant(tenantID), nil
			},
		}
		svc := NewIdentityProviderService(gormDB, idpRepo, emailRepo, audienceRepo, &mockTenantRepo{}, userRepo)
		result, err := svc.DeleteByUUID(context.Background(), idpUUID, tenantID, actorUUID)
		require.NoError(t, err)
		assert.Equal(t, "local", result.Name)
		assert.True(t, domainsCleared, "email domains should be cleared on delete")
		assert.True(t, audiencesCleared, "allowed audiences should be cleared on delete")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("default idp → cannot delete", func(t *testing.T) {
		idp := newIDP(tenantID, "default-idp")
		idp.IsDefault = true
		idpRepo := &mockIdentityProviderRepo{
			findByUUIDFn: func(_ any, _ ...string) (*IdentityProvider, error) { return idp, nil },
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*User, error) {
				return actorUserWithDefaultTenant(tenantID), nil
			},
		}
		svc := NewIdentityProviderService(nil, idpRepo, &mockIdentityProviderEmailDomainRepo{}, nil, &mockTenantRepo{}, userRepo)
		_, err := svc.DeleteByUUID(context.Background(), idpUUID, tenantID, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "default idp")
	})

	t.Run("delete repo error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		idp := newIDP(tenantID, "local")
		idpRepo := &mockIdentityProviderRepo{
			findByUUIDFn:   func(_ any, _ ...string) (*IdentityProvider, error) { return idp, nil },
			deleteByUUIDFn: func(_ any) error { return errors.New("del err") },
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*User, error) {
				return actorUserWithDefaultTenant(tenantID), nil
			},
		}
		svc := NewIdentityProviderService(gormDB, idpRepo, &mockIdentityProviderEmailDomainRepo{}, nil, &mockTenantRepo{}, userRepo)
		_, err := svc.DeleteByUUID(context.Background(), idpUUID, tenantID, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "del err")
	})

	t.Run("ValidateTenantAccess error", func(t *testing.T) {
		idp := newIDP(tenantID, "local")
		idpRepo := &mockIdentityProviderRepo{
			findByUUIDFn: func(_ any, _ ...string) (*IdentityProvider, error) { return idp, nil },
		}
		// Non-default tenant user trying to access a different tenant
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*User, error) {
				return &User{
					UserID: 1,
					UserIdentities: []UserIdentity{
						{TenantID: 999, Tenant: &Tenant{TenantID: 999, IsSystem: false}},
					},
				}, nil
			},
		}
		svc := NewIdentityProviderService(nil, idpRepo, &mockIdentityProviderEmailDomainRepo{}, nil, &mockTenantRepo{}, userRepo)
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
		svc := NewIdentityProviderService(gormDB, &mockIdentityProviderRepo{}, &mockIdentityProviderEmailDomainRepo{}, nil, &mockTenantRepo{}, &mockUserRepo{})
		_, err := svc.Create(context.Background(), createInput("idp", "IDP", "local", "password", cfg, "active", "invalid-uuid", tenantID, actorUUID))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid tenant UUID")
	})

	t.Run("system provider type is rejected (reserved for the built-in)", func(t *testing.T) {
		// The 'system' type is seeded-only; it can never be created via the API,
		// which guarantees exactly one built-in provider per tenant. The guard
		// returns before the DB transaction, so no mock expectations are needed.
		gormDB, _ := newMockGormDB(t)
		svc := NewIdentityProviderService(gormDB, &mockIdentityProviderRepo{}, &mockIdentityProviderEmailDomainRepo{}, nil, &mockTenantRepo{}, &mockUserRepo{})
		_, err := svc.Create(context.Background(), createInput("maintainerd", "Built-in", shared.IDPProviderMaintainerd, shared.IDPTypeSystem, cfg, "active", tenantUUID.String(), tenantID, actorUUID))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "reserved for the built-in")
	})

	// FIX C: an omitted config must be normalized to '{}' so the JSONB NOT NULL
	// column is never written as SQL NULL.
	t.Run("empty config is normalized to {} before save", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		tenant := &Tenant{TenantID: tenantID, TenantUUID: tenantUUID, IsSystem: true}
		tenantRepo := &mockTenantRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Tenant, error) { return tenant, nil },
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*User, error) {
				return actorUserWithDefaultTenant(tenantID), nil
			},
		}
		var savedConfig datatypes.JSON
		idpRepo := &mockIdentityProviderRepo{
			createOrUpdateFn: func(e *IdentityProvider) (*IdentityProvider, error) {
				savedConfig = e.Config
				return e, nil
			},
			findByUUIDFn: func(_ any, _ ...string) (*IdentityProvider, error) {
				return &IdentityProvider{Name: "idp", TenantID: tenantID, Tenant: tenant}, nil
			},
		}
		svc := NewIdentityProviderService(gormDB, idpRepo, &mockIdentityProviderEmailDomainRepo{}, nil, tenantRepo, userRepo)
		_, err := svc.Create(context.Background(), createInput("idp", "IDP", "local", "password", nil, "active", tenantUUID.String(), tenantID, actorUUID))
		require.NoError(t, err)
		assert.Equal(t, "{}", string(savedConfig))
	})

	t.Run("tenant not found", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		tenantRepo := &mockTenantRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Tenant, error) { return nil, nil },
		}
		svc := NewIdentityProviderService(gormDB, &mockIdentityProviderRepo{}, &mockIdentityProviderEmailDomainRepo{}, nil, tenantRepo, &mockUserRepo{})
		_, err := svc.Create(context.Background(), createInput("idp", "IDP", "local", "password", cfg, "active", tenantUUID.String(), tenantID, actorUUID))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "tenant not found")
	})

	t.Run("tenant find error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		tenantRepo := &mockTenantRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Tenant, error) { return nil, errors.New("db err") },
		}
		svc := NewIdentityProviderService(gormDB, &mockIdentityProviderRepo{}, &mockIdentityProviderEmailDomainRepo{}, nil, tenantRepo, &mockUserRepo{})
		_, err := svc.Create(context.Background(), createInput("idp", "IDP", "local", "password", cfg, "active", tenantUUID.String(), tenantID, actorUUID))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "tenant not found")
	})

	t.Run("tenant ownership mismatch", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		tenantRepo := &mockTenantRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Tenant, error) {
				return &Tenant{TenantID: 999}, nil // different from tenantID=1
			},
		}
		svc := NewIdentityProviderService(gormDB, &mockIdentityProviderRepo{}, &mockIdentityProviderEmailDomainRepo{}, nil, tenantRepo, &mockUserRepo{})
		_, err := svc.Create(context.Background(), createInput("idp", "IDP", "local", "password", cfg, "active", tenantUUID.String(), tenantID, actorUUID))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "access denied")
	})

	t.Run("actor user not found", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		tenantRepo := &mockTenantRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Tenant, error) {
				return &Tenant{TenantID: tenantID, IsSystem: true}, nil
			},
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*User, error) { return nil, nil },
		}
		svc := NewIdentityProviderService(gormDB, &mockIdentityProviderRepo{}, &mockIdentityProviderEmailDomainRepo{}, nil, tenantRepo, userRepo)
		_, err := svc.Create(context.Background(), createInput("idp", "IDP", "local", "password", cfg, "active", tenantUUID.String(), tenantID, actorUUID))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "actor user not found")
	})

	t.Run("ValidateTenantAccess error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		tenantRepo := &mockTenantRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Tenant, error) {
				return &Tenant{TenantID: tenantID, IsSystem: false}, nil
			},
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*User, error) {
				return &User{
					UserID: 1,
					UserIdentities: []UserIdentity{
						{TenantID: 999, Tenant: &Tenant{TenantID: 999, IsSystem: false}},
					},
				}, nil
			},
		}
		svc := NewIdentityProviderService(gormDB, &mockIdentityProviderRepo{}, &mockIdentityProviderEmailDomainRepo{}, nil, tenantRepo, userRepo)
		_, err := svc.Create(context.Background(), createInput("idp", "IDP", "local", "password", cfg, "active", tenantUUID.String(), tenantID, actorUUID))
		require.Error(t, err)
	})

	t.Run("findByName error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		tenantRepo := &mockTenantRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Tenant, error) {
				return &Tenant{TenantID: tenantID, IsSystem: true}, nil
			},
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*User, error) {
				return actorUserWithDefaultTenant(tenantID), nil
			},
		}
		idpRepo := &mockIdentityProviderRepo{
			findByNameFn: func(_ string, _ int64) (*IdentityProvider, error) {
				return nil, errors.New("db err")
			},
		}
		svc := NewIdentityProviderService(gormDB, idpRepo, &mockIdentityProviderEmailDomainRepo{}, nil, tenantRepo, userRepo)
		_, err := svc.Create(context.Background(), createInput("idp", "IDP", "local", "password", cfg, "active", tenantUUID.String(), tenantID, actorUUID))
		require.Error(t, err)
	})

	t.Run("idp already exists", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		tenantRepo := &mockTenantRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Tenant, error) {
				return &Tenant{TenantID: tenantID, IsSystem: true}, nil
			},
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*User, error) {
				return actorUserWithDefaultTenant(tenantID), nil
			},
		}
		idpRepo := &mockIdentityProviderRepo{
			findByNameFn: func(_ string, _ int64) (*IdentityProvider, error) {
				return &IdentityProvider{Name: "idp"}, nil
			},
		}
		svc := NewIdentityProviderService(gormDB, idpRepo, &mockIdentityProviderEmailDomainRepo{}, nil, tenantRepo, userRepo)
		_, err := svc.Create(context.Background(), createInput("idp", "IDP", "local", "password", cfg, "active", tenantUUID.String(), tenantID, actorUUID))
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
			findByUUIDFn: func(_ any, _ ...string) (*Tenant, error) {
				return &Tenant{TenantID: tenantID, IsSystem: true}, nil
			},
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*User, error) {
				return actorUserWithDefaultTenant(tenantID), nil
			},
		}
		idpRepo := &mockIdentityProviderRepo{}
		svc := NewIdentityProviderService(gormDB, idpRepo, &mockIdentityProviderEmailDomainRepo{}, nil, tenantRepo, userRepo)
		_, err := svc.Create(context.Background(), createInput("idp", "IDP", "local", "password", cfg, "active", tenantUUID.String(), tenantID, actorUUID))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "rand failure")
	})

	t.Run("createOrUpdate error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		tenantRepo := &mockTenantRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Tenant, error) {
				return &Tenant{TenantID: tenantID, IsSystem: true}, nil
			},
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*User, error) {
				return actorUserWithDefaultTenant(tenantID), nil
			},
		}
		idpRepo := &mockIdentityProviderRepo{
			createOrUpdateFn: func(_ *IdentityProvider) (*IdentityProvider, error) {
				return nil, errors.New("create err")
			},
		}
		svc := NewIdentityProviderService(gormDB, idpRepo, &mockIdentityProviderEmailDomainRepo{}, nil, tenantRepo, userRepo)
		_, err := svc.Create(context.Background(), createInput("idp", "IDP", "local", "password", cfg, "active", tenantUUID.String(), tenantID, actorUUID))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "create err")
	})

	t.Run("client secret encryption failure", func(t *testing.T) {
		orig := crypto.EncryptAtRest
		defer func() { crypto.EncryptAtRest = orig }()
		crypto.EncryptAtRest = func(string) (string, error) { return "", errors.New("encrypt failure") }

		cfgWithSecret := datatypes.JSON(json.RawMessage(`{"client_secret":"secret"}`))
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		tenant := &Tenant{TenantID: tenantID, TenantUUID: tenantUUID, IsSystem: true}
		tenantRepo := &mockTenantRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Tenant, error) { return tenant, nil },
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*User, error) {
				return actorUserWithDefaultTenant(tenantID), nil
			},
		}
		svc := NewIdentityProviderService(gormDB, &mockIdentityProviderRepo{}, &mockIdentityProviderEmailDomainRepo{}, nil, tenantRepo, userRepo)
		in := createInput("idp", "IDP", "local", "password", cfgWithSecret, "active", tenantUUID.String(), tenantID, actorUUID)
		in.ProviderClientSecret = "secret"
		_, err := svc.Create(context.Background(), in)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "encrypt failure")
	})

	t.Run("findByUUID after create error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		tenantRepo := &mockTenantRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Tenant, error) {
				return &Tenant{TenantID: tenantID, IsSystem: true}, nil
			},
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*User, error) {
				return actorUserWithDefaultTenant(tenantID), nil
			},
		}
		idpRepo := &mockIdentityProviderRepo{
			findByUUIDFn: func(_ any, _ ...string) (*IdentityProvider, error) {
				return nil, errors.New("fetch err")
			},
		}
		svc := NewIdentityProviderService(gormDB, idpRepo, &mockIdentityProviderEmailDomainRepo{}, nil, tenantRepo, userRepo)
		_, err := svc.Create(context.Background(), createInput("idp", "IDP", "local", "password", cfg, "active", tenantUUID.String(), tenantID, actorUUID))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "fetch err")
	})

	t.Run("success", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		tenant := &Tenant{TenantID: tenantID, TenantUUID: tenantUUID, IsSystem: true}
		tenantRepo := &mockTenantRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Tenant, error) { return tenant, nil },
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*User, error) {
				return actorUserWithDefaultTenant(tenantID), nil
			},
		}
		idpRepo := &mockIdentityProviderRepo{
			findByUUIDFn: func(_ any, _ ...string) (*IdentityProvider, error) {
				return &IdentityProvider{
					Name: "idp", DisplayName: "IDP", TenantID: tenantID,
					Tenant: tenant,
				}, nil
			},
		}
		svc := NewIdentityProviderService(gormDB, idpRepo, &mockIdentityProviderEmailDomainRepo{}, nil, tenantRepo, userRepo)
		res, err := svc.Create(context.Background(), createInput("idp", "IDP", "local", "password", cfg, "active", tenantUUID.String(), tenantID, actorUUID))
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
			findByUUIDFn: func(_ any, _ ...string) (*IdentityProvider, error) { return nil, nil },
		}
		svc := NewIdentityProviderService(gormDB, idpRepo, &mockIdentityProviderEmailDomainRepo{}, nil, &mockTenantRepo{}, &mockUserRepo{})
		_, err := svc.Update(context.Background(), updateInput(idpUUID, "n", "d", "local", "password", cfg, "active", tenantID, actorUUID))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("wrong tenant", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		idpRepo := &mockIdentityProviderRepo{
			findByUUIDFn: func(_ any, _ ...string) (*IdentityProvider, error) {
				return newIDP(999, "other"), nil
			},
		}
		svc := NewIdentityProviderService(gormDB, idpRepo, &mockIdentityProviderEmailDomainRepo{}, nil, &mockTenantRepo{}, &mockUserRepo{})
		_, err := svc.Update(context.Background(), updateInput(idpUUID, "n", "d", "local", "password", cfg, "active", tenantID, actorUUID))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "access denied")
	})

	t.Run("actor user not found", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		idpRepo := &mockIdentityProviderRepo{
			findByUUIDFn: func(_ any, _ ...string) (*IdentityProvider, error) {
				return newIDP(tenantID, "local"), nil
			},
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*User, error) { return nil, nil },
		}
		svc := NewIdentityProviderService(gormDB, idpRepo, &mockIdentityProviderEmailDomainRepo{}, nil, &mockTenantRepo{}, userRepo)
		_, err := svc.Update(context.Background(), updateInput(idpUUID, "n", "d", "local", "password", cfg, "active", tenantID, actorUUID))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "actor user not found")
	})

	t.Run("ValidateTenantAccess error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		idpRepo := &mockIdentityProviderRepo{
			findByUUIDFn: func(_ any, _ ...string) (*IdentityProvider, error) {
				return newIDP(tenantID, "local"), nil
			},
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*User, error) {
				return &User{
					UserID: 1,
					UserIdentities: []UserIdentity{
						{TenantID: 999, Tenant: &Tenant{TenantID: 999, IsSystem: false}},
					},
				}, nil
			},
		}
		svc := NewIdentityProviderService(gormDB, idpRepo, &mockIdentityProviderEmailDomainRepo{}, nil, &mockTenantRepo{}, userRepo)
		_, err := svc.Update(context.Background(), updateInput(idpUUID, "n", "d", "local", "password", cfg, "active", tenantID, actorUUID))
		require.Error(t, err)
	})

	t.Run("system idp blocked", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		idp := newIDP(tenantID, "sys")
		idp.IsSystem = true
		idpRepo := &mockIdentityProviderRepo{
			findByUUIDFn: func(_ any, _ ...string) (*IdentityProvider, error) { return idp, nil },
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*User, error) {
				return actorUserWithDefaultTenant(tenantID), nil
			},
		}
		svc := NewIdentityProviderService(gormDB, idpRepo, &mockIdentityProviderEmailDomainRepo{}, nil, &mockTenantRepo{}, userRepo)
		_, err := svc.Update(context.Background(), updateInput(idpUUID, "n", "d", "local", "password", cfg, "active", tenantID, actorUUID))
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
			findByUUIDFn: func(_ any, _ ...string) (*IdentityProvider, error) { return idp, nil },
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*User, error) {
				return actorUserWithDefaultTenant(tenantID), nil
			},
		}
		svc := NewIdentityProviderService(gormDB, idpRepo, &mockIdentityProviderEmailDomainRepo{}, nil, &mockTenantRepo{}, userRepo)
		_, err := svc.Update(context.Background(), updateInput(idpUUID, "n", "d", "local", "password", cfg, "active", tenantID, actorUUID))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "default idp")
	})

	t.Run("duplicate name error from findByName", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		idp := newIDP(tenantID, "old-name")
		idpRepo := &mockIdentityProviderRepo{
			findByUUIDFn: func(_ any, _ ...string) (*IdentityProvider, error) { return idp, nil },
			findByNameFn: func(_ string, _ int64) (*IdentityProvider, error) {
				return nil, errors.New("db err")
			},
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*User, error) {
				return actorUserWithDefaultTenant(tenantID), nil
			},
		}
		svc := NewIdentityProviderService(gormDB, idpRepo, &mockIdentityProviderEmailDomainRepo{}, nil, &mockTenantRepo{}, userRepo)
		_, err := svc.Update(context.Background(), updateInput(idpUUID, "new-name", "d", "local", "password", cfg, "active", tenantID, actorUUID))
		require.Error(t, err)
	})

	t.Run("duplicate name exists", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		idp := newIDP(tenantID, "old-name")
		otherUUID := uuid.New()
		idpRepo := &mockIdentityProviderRepo{
			findByUUIDFn: func(_ any, _ ...string) (*IdentityProvider, error) { return idp, nil },
			findByNameFn: func(_ string, _ int64) (*IdentityProvider, error) {
				return &IdentityProvider{IdentityProviderUUID: otherUUID}, nil
			},
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*User, error) {
				return actorUserWithDefaultTenant(tenantID), nil
			},
		}
		svc := NewIdentityProviderService(gormDB, idpRepo, &mockIdentityProviderEmailDomainRepo{}, nil, &mockTenantRepo{}, userRepo)
		_, err := svc.Update(context.Background(), updateInput(idpUUID, "new-name", "d", "local", "password", cfg, "active", tenantID, actorUUID))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
	})

	t.Run("createOrUpdate error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		idp := newIDP(tenantID, "local")
		idpRepo := &mockIdentityProviderRepo{
			findByUUIDFn: func(_ any, _ ...string) (*IdentityProvider, error) { return idp, nil },
			createOrUpdateFn: func(_ *IdentityProvider) (*IdentityProvider, error) {
				return nil, errors.New("save err")
			},
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*User, error) {
				return actorUserWithDefaultTenant(tenantID), nil
			},
		}
		svc := NewIdentityProviderService(gormDB, idpRepo, &mockIdentityProviderEmailDomainRepo{}, nil, &mockTenantRepo{}, userRepo)
		_, err := svc.Update(context.Background(), updateInput(idpUUID, "local", "d", "local", "password", cfg, "active", tenantID, actorUUID))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "save err")
	})

	t.Run("success same name", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		idp := newIDP(tenantID, "local")
		idpRepo := &mockIdentityProviderRepo{
			findByUUIDFn: func(_ any, _ ...string) (*IdentityProvider, error) { return idp, nil },
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*User, error) {
				return actorUserWithDefaultTenant(tenantID), nil
			},
		}
		svc := NewIdentityProviderService(gormDB, idpRepo, &mockIdentityProviderEmailDomainRepo{}, nil, &mockTenantRepo{}, userRepo)
		res, err := svc.Update(context.Background(), updateInput(idpUUID, "local", "New Display", "local", "password", cfg, "active", tenantID, actorUUID))
		require.NoError(t, err)
		assert.Equal(t, "New Display", res.DisplayName)
	})

	t.Run("success different name no conflict", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		idp := newIDP(tenantID, "old-name")
		idpRepo := &mockIdentityProviderRepo{
			findByUUIDFn: func(_ any, _ ...string) (*IdentityProvider, error) { return idp, nil },
			findByNameFn: func(_ string, _ int64) (*IdentityProvider, error) { return nil, nil },
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*User, error) {
				return actorUserWithDefaultTenant(tenantID), nil
			},
		}
		svc := NewIdentityProviderService(gormDB, idpRepo, &mockIdentityProviderEmailDomainRepo{}, nil, &mockTenantRepo{}, userRepo)
		res, err := svc.Update(context.Background(), updateInput(idpUUID, "new-name", "d", "local", "password", cfg, "active", tenantID, actorUUID))
		require.NoError(t, err)
		assert.Equal(t, "new-name", res.Name)
	})

	// FIX C: config is JSONB NOT NULL. An omitted/empty config must be normalized to
	// '{}' before the row is saved, otherwise GORM writes SQL NULL and the update 500s.
	t.Run("empty config is normalized to {} before save", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		idp := newIDP(tenantID, "local")
		var savedConfig datatypes.JSON
		idpRepo := &mockIdentityProviderRepo{
			findByUUIDFn: func(_ any, _ ...string) (*IdentityProvider, error) { return idp, nil },
			createOrUpdateFn: func(e *IdentityProvider) (*IdentityProvider, error) {
				savedConfig = e.Config
				return e, nil
			},
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*User, error) {
				return actorUserWithDefaultTenant(tenantID), nil
			},
		}
		svc := NewIdentityProviderService(gormDB, idpRepo, &mockIdentityProviderEmailDomainRepo{}, nil, &mockTenantRepo{}, userRepo)
		// nil config in the input must not reach the DB as NULL.
		res, err := svc.Update(context.Background(), updateInput(idpUUID, "local", "d", "local", "password", nil, "active", tenantID, actorUUID))
		require.NoError(t, err)
		assert.Equal(t, "{}", string(savedConfig))
		assert.Equal(t, "local", res.Name)
	})

	t.Run("client secret encryption failure", func(t *testing.T) {
		orig := crypto.EncryptAtRest
		defer func() { crypto.EncryptAtRest = orig }()
		crypto.EncryptAtRest = func(string) (string, error) { return "", errors.New("encrypt failure") }

		cfgWithSecret := datatypes.JSON(json.RawMessage(`{"client_secret":"secret"}`))
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		idp := newIDP(tenantID, "local")
		idpRepo := &mockIdentityProviderRepo{
			findByUUIDFn: func(_ any, _ ...string) (*IdentityProvider, error) { return idp, nil },
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*User, error) {
				return actorUserWithDefaultTenant(tenantID), nil
			},
		}
		svc := NewIdentityProviderService(gormDB, idpRepo, &mockIdentityProviderEmailDomainRepo{}, nil, &mockTenantRepo{}, userRepo)
		in := updateInput(idpUUID, "local", "d", "local", "password", cfgWithSecret, "active", tenantID, actorUUID)
		in.ProviderClientSecret = "secret"
		_, err := svc.Update(context.Background(), in)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "encrypt failure")
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
			findByUUIDFn: func(_ any, _ ...string) (*IdentityProvider, error) { return nil, nil },
		}
		svc := NewIdentityProviderService(gormDB, idpRepo, &mockIdentityProviderEmailDomainRepo{}, nil, &mockTenantRepo{}, &mockUserRepo{})
		_, err := svc.SetStatusByUUID(context.Background(), idpUUID, "active", tenantID, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("wrong tenant", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		idpRepo := &mockIdentityProviderRepo{
			findByUUIDFn: func(_ any, _ ...string) (*IdentityProvider, error) {
				return newIDP(999, "other"), nil
			},
		}
		svc := NewIdentityProviderService(gormDB, idpRepo, &mockIdentityProviderEmailDomainRepo{}, nil, &mockTenantRepo{}, &mockUserRepo{})
		_, err := svc.SetStatusByUUID(context.Background(), idpUUID, "active", tenantID, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "access denied")
	})

	t.Run("actor user not found", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		idpRepo := &mockIdentityProviderRepo{
			findByUUIDFn: func(_ any, _ ...string) (*IdentityProvider, error) {
				return newIDP(tenantID, "local"), nil
			},
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*User, error) { return nil, nil },
		}
		svc := NewIdentityProviderService(gormDB, idpRepo, &mockIdentityProviderEmailDomainRepo{}, nil, &mockTenantRepo{}, userRepo)
		_, err := svc.SetStatusByUUID(context.Background(), idpUUID, "active", tenantID, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "actor user not found")
	})

	t.Run("ValidateTenantAccess error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		idpRepo := &mockIdentityProviderRepo{
			findByUUIDFn: func(_ any, _ ...string) (*IdentityProvider, error) {
				return newIDP(tenantID, "local"), nil
			},
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*User, error) {
				return &User{
					UserID: 1,
					UserIdentities: []UserIdentity{
						{TenantID: 999, Tenant: &Tenant{TenantID: 999, IsSystem: false}},
					},
				}, nil
			},
		}
		svc := NewIdentityProviderService(gormDB, idpRepo, &mockIdentityProviderEmailDomainRepo{}, nil, &mockTenantRepo{}, userRepo)
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
			findByUUIDFn: func(_ any, _ ...string) (*IdentityProvider, error) { return idp, nil },
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*User, error) {
				return actorUserWithDefaultTenant(tenantID), nil
			},
		}
		svc := NewIdentityProviderService(gormDB, idpRepo, &mockIdentityProviderEmailDomainRepo{}, nil, &mockTenantRepo{}, userRepo)
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
			findByUUIDFn: func(_ any, _ ...string) (*IdentityProvider, error) { return idp, nil },
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*User, error) {
				return actorUserWithDefaultTenant(tenantID), nil
			},
		}
		svc := NewIdentityProviderService(gormDB, idpRepo, &mockIdentityProviderEmailDomainRepo{}, nil, &mockTenantRepo{}, userRepo)
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
			findByUUIDFn: func(_ any, _ ...string) (*IdentityProvider, error) { return idp, nil },
			createOrUpdateFn: func(_ *IdentityProvider) (*IdentityProvider, error) {
				return nil, errors.New("save err")
			},
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*User, error) {
				return actorUserWithDefaultTenant(tenantID), nil
			},
		}
		svc := NewIdentityProviderService(gormDB, idpRepo, &mockIdentityProviderEmailDomainRepo{}, nil, &mockTenantRepo{}, userRepo)
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
			findByUUIDFn: func(_ any, _ ...string) (*IdentityProvider, error) { return idp, nil },
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*User, error) {
				return actorUserWithDefaultTenant(tenantID), nil
			},
		}
		svc := NewIdentityProviderService(gormDB, idpRepo, &mockIdentityProviderEmailDomainRepo{}, nil, &mockTenantRepo{}, userRepo)
		res, err := svc.SetStatusByUUID(context.Background(), idpUUID, "inactive", tenantID, actorUUID)
		require.NoError(t, err)
		assert.Equal(t, "local", res.Name)
	})

	// FIX B: activating a provider must re-run the structural validation the
	// create/update DTOs enforce, so a draft can't be flipped ACTIVE unconfigured.

	// draftExternal builds a stored, inactive enterprise google provider missing
	// its issuer/client_id — a draft that must NOT be activatable as-is.
	draftExternal := func() *IdentityProvider {
		idp := newIDP(tenantID, "google-draft")
		idp.Provider = shared.IDPProviderGoogle
		idp.ProviderType = shared.IDPTypeEnterprise
		idp.Status = shared.StatusInactive
		idp.Issuer = nil
		idp.ProviderClientID = nil
		idp.Config = nil
		return idp
	}

	t.Run("activate draft missing issuer/config is rejected", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		idpRepo := &mockIdentityProviderRepo{
			findByUUIDFn: func(_ any, _ ...string) (*IdentityProvider, error) { return draftExternal(), nil },
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*User, error) {
				return actorUserWithDefaultTenant(tenantID), nil
			},
		}
		svc := NewIdentityProviderService(gormDB, idpRepo, &mockIdentityProviderEmailDomainRepo{}, nil, &mockTenantRepo{}, userRepo)
		_, err := svc.SetStatusByUUID(context.Background(), idpUUID, shared.StatusActive, tenantID, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "issuer is required")
	})

	t.Run("deactivate draft is always allowed", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		idp := draftExternal()
		idp.Status = shared.StatusActive // pretend it was somehow active; deactivate it
		idpRepo := &mockIdentityProviderRepo{
			findByUUIDFn: func(_ any, _ ...string) (*IdentityProvider, error) { return idp, nil },
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*User, error) {
				return actorUserWithDefaultTenant(tenantID), nil
			},
		}
		svc := NewIdentityProviderService(gormDB, idpRepo, &mockIdentityProviderEmailDomainRepo{}, nil, &mockTenantRepo{}, userRepo)
		_, err := svc.SetStatusByUUID(context.Background(), idpUUID, shared.StatusInactive, tenantID, actorUUID)
		require.NoError(t, err)
	})

	t.Run("activate fully-configured provider succeeds", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		issuer := "https://accounts.google.com"
		clientID := "client-abc"
		idp := draftExternal()
		idp.Issuer = &issuer
		idp.ProviderClientID = &clientID
		idp.Config = datatypes.JSON(`{"scopes":["openid","email"]}`)
		idpRepo := &mockIdentityProviderRepo{
			findByUUIDFn: func(_ any, _ ...string) (*IdentityProvider, error) { return idp, nil },
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*User, error) {
				return actorUserWithDefaultTenant(tenantID), nil
			},
		}
		svc := NewIdentityProviderService(gormDB, idpRepo, &mockIdentityProviderEmailDomainRepo{}, nil, &mockTenantRepo{}, userRepo)
		_, err := svc.SetStatusByUUID(context.Background(), idpUUID, shared.StatusActive, tenantID, actorUUID)
		require.NoError(t, err)
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
		idp := &IdentityProvider{
			Name:   "idp",
			Tenant: &Tenant{TenantID: 1, Name: "t"},
		}
		res := toIdpServiceDataResult(idp)
		require.NotNil(t, res.Tenant)
		assert.Equal(t, "t", res.Tenant.Name)
	})

	t.Run("without tenant", func(t *testing.T) {
		idp := &IdentityProvider{Name: "idp"}
		res := toIdpServiceDataResult(idp)
		assert.Nil(t, res.Tenant)
	})
}

// ---------------------------------------------------------------------------
// External-active validation (gated on status=active)
// ---------------------------------------------------------------------------

func TestValidateExternalProviderColumns(t *testing.T) {
	// Drafts (inactive) are never constrained — they can be created before being
	// fully configured.
	require.NoError(t, validateExternalProviderColumns(shared.IDPTypeSocial, shared.StatusInactive, "", ""))
	require.NoError(t, validateExternalProviderColumns(shared.IDPTypeEnterprise, shared.StatusInactive, "", ""))
	// System providers never need upstream creds.
	require.NoError(t, validateExternalProviderColumns(shared.IDPTypeSystem, shared.StatusActive, "", ""))
	// Active social/enterprise require both issuer and client_id.
	err := validateExternalProviderColumns(shared.IDPTypeSocial, shared.StatusActive, "", "client")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "issuer is required")
	err = validateExternalProviderColumns(shared.IDPTypeEnterprise, shared.StatusActive, "https://idp", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "client_id is required")
	// Complete active external config passes.
	require.NoError(t, validateExternalProviderColumns(shared.IDPTypeSocial, shared.StatusActive, "https://idp", "client"))
}

func TestIdentityProviderService_Create_ExternalActiveRejected(t *testing.T) {
	tenantID := int64(1)
	actorUUID := uuid.New()
	tenantUUID := uuid.New()
	cfg := datatypes.JSON([]byte(`{}`))
	// nil db is safe: the guard runs before any transaction.
	svc := NewIdentityProviderService(nil, &mockIdentityProviderRepo{}, &mockIdentityProviderEmailDomainRepo{}, nil, &mockTenantRepo{}, &mockUserRepo{})

	in := createInput("g", "Google Display", shared.IDPProviderGoogle, shared.IDPTypeSocial, cfg, shared.StatusActive, tenantUUID.String(), tenantID, actorUUID)
	in.ProviderClientID = "client-1" // issuer still missing
	_, err := svc.Create(context.Background(), in)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "issuer is required")
}

// ---------------------------------------------------------------------------
// Secret never appears in read/list responses (column-level omit)
// ---------------------------------------------------------------------------

func TestIdpResult_NeverExposesSecret(t *testing.T) {
	issuer := "https://idp.example.com"
	clientID := "client-1"
	enc := "ENCRYPTED_SECRET_VALUE"
	idp := &IdentityProvider{
		Name:                          "google",
		Issuer:                        &issuer,
		ProviderClientID:              &clientID,
		ProviderClientSecretEncrypted: &enc,
		AllowJITProvisioning:          true,
		Config:                        datatypes.JSON([]byte(`{"scopes":["openid"]}`)),
		EmailDomains: []IdentityProviderEmailDomain{
			{Domain: "example.com"},
			{Domain: "foo.com"},
		},
	}

	res := toIdpServiceDataResult(idp)
	require.NotNil(t, res)
	assert.Equal(t, issuer, res.Issuer)
	assert.Equal(t, clientID, res.ProviderClientID)
	assert.True(t, res.AllowJITProvisioning)
	assert.ElementsMatch(t, []string{"example.com", "foo.com"}, res.EmailDomains)

	// The detail DTO is exactly what GET returns; it must never carry the secret.
	detail := toIdpDetailResponseDTO(*res)
	raw, err := json.Marshal(detail)
	require.NoError(t, err)
	assert.NotContains(t, string(raw), enc)
	assert.NotContains(t, string(raw), "client_secret")
	assert.NotContains(t, string(raw), "client_secret_encrypted")
}

// ---------------------------------------------------------------------------
// Email-domain membership is replaced transactionally on create
// ---------------------------------------------------------------------------

func TestIdentityProviderService_Create_ReplacesEmailDomains(t *testing.T) {
	tenantID := int64(1)
	actorUUID := uuid.New()
	tenantUUID := uuid.New()
	cfg := datatypes.JSON([]byte(`{}`))

	gormDB, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectCommit()
	tenant := &Tenant{TenantID: tenantID, TenantUUID: tenantUUID, IsSystem: true}
	tenantRepo := &mockTenantRepo{findByUUIDFn: func(_ any, _ ...string) (*Tenant, error) { return tenant, nil }}
	userRepo := &mockUserRepo{findByUUIDFn: func(_ any, _ ...string) (*User, error) {
		return actorUserWithDefaultTenant(tenantID), nil
	}}
	idpRepo := &mockIdentityProviderRepo{
		findByUUIDFn: func(_ any, _ ...string) (*IdentityProvider, error) {
			return &IdentityProvider{Name: "idp", TenantID: tenantID, Tenant: tenant}, nil
		},
	}
	var captured []string
	emailRepo := &mockIdentityProviderEmailDomainRepo{
		replaceForProviderFn: func(_ int64, _ int64, domains []string) error {
			captured = domains
			return nil
		},
	}
	svc := NewIdentityProviderService(gormDB, idpRepo, emailRepo, nil, tenantRepo, userRepo)
	in := createInput("idp", "IDP", "local", "password", cfg, "active", tenantUUID.String(), tenantID, actorUUID)
	in.Issuer = "https://idp.example.com"
	in.ProviderClientID = "client-1"
	in.EmailDomains = []string{"example.com", "foo.com"}
	_, err := svc.Create(context.Background(), in)
	require.NoError(t, err)
	assert.Equal(t, []string{"example.com", "foo.com"}, captured)
}

func TestIdentityProviderService_Create_EmailDomainReplaceError(t *testing.T) {
	tenantID := int64(1)
	actorUUID := uuid.New()
	tenantUUID := uuid.New()
	cfg := datatypes.JSON([]byte(`{}`))

	gormDB, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectRollback()
	tenant := &Tenant{TenantID: tenantID, TenantUUID: tenantUUID, IsSystem: true}
	tenantRepo := &mockTenantRepo{findByUUIDFn: func(_ any, _ ...string) (*Tenant, error) { return tenant, nil }}
	userRepo := &mockUserRepo{findByUUIDFn: func(_ any, _ ...string) (*User, error) {
		return actorUserWithDefaultTenant(tenantID), nil
	}}
	emailRepo := &mockIdentityProviderEmailDomainRepo{
		replaceForProviderFn: func(_ int64, _ int64, _ []string) error {
			return errors.New("domain err")
		},
	}
	svc := NewIdentityProviderService(gormDB, &mockIdentityProviderRepo{}, emailRepo, nil, tenantRepo, userRepo)
	in := createInput("idp", "IDP", "local", "password", cfg, "active", tenantUUID.String(), tenantID, actorUUID)
	in.EmailDomains = []string{"example.com"}
	_, err := svc.Create(context.Background(), in)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "domain err")
}
