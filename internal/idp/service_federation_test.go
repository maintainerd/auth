package idp

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/authevent"
	"github.com/maintainerd/auth/internal/authn"
	"github.com/maintainerd/auth/internal/platform/apperror"
	"github.com/maintainerd/auth/internal/platform/config"
	"github.com/maintainerd/auth/internal/platform/jwt"
	"github.com/maintainerd/auth/internal/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func initTestJWTKeysService(t *testing.T) {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	privPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})
	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PUBLIC KEY", Bytes: x509.MarshalPKCS1PublicKey(&priv.PublicKey)})
	config.JWTPrivateKey = privPEM
	config.JWTPublicKey = pubPEM
	require.NoError(t, jwt.InitJWTKeys())
}

type mockFederationUserIdentityRepo struct {
	mockBaseRepo[UserIdentity]
	findByUserIDFn            func(int64) ([]UserIdentity, error)
	findByUserIDAndProviderFn func(int64, string) (*UserIdentity, error)
	findByProviderAndSubFn    func(string, string) (*UserIdentity, error)
	createFn                  func(*UserIdentity) (*UserIdentity, error)
	deleteByIDFn              func(any) error
}

func (m *mockFederationUserIdentityRepo) WithTx(_ *gorm.DB) UserIdentityRepository {
	return m
}

func (m *mockFederationUserIdentityRepo) FindByUserID(userID int64) ([]UserIdentity, error) {
	if m.findByUserIDFn != nil {
		return m.findByUserIDFn(userID)
	}
	return nil, nil
}

func (m *mockFederationUserIdentityRepo) FindByProviderAndSub(provider, sub string) (*UserIdentity, error) {
	if m.findByProviderAndSubFn != nil {
		return m.findByProviderAndSubFn(provider, sub)
	}
	return nil, nil
}

func (m *mockFederationUserIdentityRepo) FindByUserIDAndProvider(userID int64, provider string) (*UserIdentity, error) {
	if m.findByUserIDAndProviderFn != nil {
		return m.findByUserIDAndProviderFn(userID, provider)
	}
	return nil, nil
}

func (m *mockFederationUserIdentityRepo) DeleteByUserID(_ int64) error {
	return nil
}

func (m *mockFederationUserIdentityRepo) DeleteByID(id any) error {
	if m.deleteByIDFn != nil {
		return m.deleteByIDFn(id)
	}
	return nil
}

func (m *mockFederationUserIdentityRepo) Create(identity *UserIdentity) (*UserIdentity, error) {
	if m.createFn != nil {
		return m.createFn(identity)
	}
	return identity, nil
}

type mockAuthEventService struct {
	logFn func(ctx context.Context, input authevent.AuthEventInput)
}

func (m *mockAuthEventService) Log(ctx context.Context, input authevent.AuthEventInput) {
	if m.logFn != nil {
		m.logFn(ctx, input)
	}
}

func (m *mockAuthEventService) FindPaginated(ctx context.Context, filter authevent.AuthEventRepositoryGetFilter) (*PaginationResult[authevent.AuthEventServiceDataResult], error) {
	return nil, nil
}

func (m *mockAuthEventService) FindByUUID(ctx context.Context, tenantID int64, eventUUID uuid.UUID) (*authevent.AuthEventServiceDataResult, error) {
	return nil, nil
}

func (m *mockAuthEventService) CountByEventType(ctx context.Context, eventType string, tenantID int64) (int64, error) {
	return 0, nil
}

func (m *mockAuthEventService) DeleteOlderThan(ctx context.Context, cutoff time.Time) (int64, error) {
	return 0, nil
}

func (m *mockAuthEventService) Shutdown() {}

type mockSessionService struct{}

func (m *mockSessionService) ListSessions(ctx context.Context, userID int64) ([]*authn.SessionDataResult, error) {
	return nil, nil
}
func (m *mockSessionService) RevokeSession(ctx context.Context, userID int64, sessionUUID uuid.UUID) error {
	return nil
}
func (m *mockSessionService) RevokeAllSessions(ctx context.Context, userID int64) error {
	return nil
}
func (m *mockSessionService) CreateSession(ctx context.Context, userID int64, ipAddress, userAgent string) (*authn.UserToken, error) {
	return &authn.UserToken{}, nil
}
func (m *mockSessionService) EnforceConcurrentLimit(ctx context.Context, userUUID uuid.UUID, userID int64) error {
	return nil
}
func (m *mockSessionService) ValidateAndTouch(ctx context.Context, sessionUUID uuid.UUID, userID int64) error {
	return nil
}

func validOIDCConfigJSON() datatypes.JSON {
	return datatypes.JSON(json.RawMessage(`{"issuer":"https://accounts.google.com","client_id":"test-client","scopes":["openid","email"]}`))
}

func validOAuth2ConfigJSON() datatypes.JSON {
	return datatypes.JSON(json.RawMessage(`{"issuer":"https://auth.example.com","client_id":"test-client","client_secret":"secret","scopes":["openid","email"],"userinfo_endpoint":"https://auth.example.com/userinfo"}`))
}

func activeOIDCProvider(identifier string) *IdentityProvider {
	return &IdentityProvider{
		IdentityProviderID: 1,
		Identifier:         identifier,
		Provider:           "google",
		Status:             "active",
		TenantID:           1,
		Config:             validOIDCConfigJSON(),
	}
}

func TestFederationServiceProvisionUser_UnverifiedEmailDoesNotMergeExistingAccount(t *testing.T) {
	gormDB, _ := newMockGormDB(t)

	var emailLookupCalled bool
	createdUser := &User{UserID: 200, Email: "owner@example.com"}
	userRepo := &mockUserRepo{
		findByEmailFn: func(string) (*User, error) {
			t.Fatal("global email lookup must not be used for federation merge")
			return nil, nil
		},
		findByEmailAndTenantIDFn: func(string, int64) (*User, error) {
			emailLookupCalled = true
			return &User{UserID: 100, Email: "owner@example.com"}, nil
		},
		createFn: func(user *User) (*User, error) {
			assert.Equal(t, "owner@example.com", user.Email)
			assert.False(t, user.IsEmailVerified)
			return createdUser, nil
		},
	}

	var externalIdentity *UserIdentity
	identityRepo := &mockFederationUserIdentityRepo{
		createFn: func(identity *UserIdentity) (*UserIdentity, error) {
			if identity.Provider == "google" {
				externalIdentity = identity
			}
			return identity, nil
		},
	}

	svc := &federationService{
		userRepo:         userRepo,
		userIdentityRepo: identityRepo,
		idpRepo:          &mockIdentityProviderRepo{},
		roleRepo:         &mockRoleRepo{},
	}

	user, isNew, err := svc.provisionUser(context.Background(), gormDB, &IdentityProvider{
		IdentityProviderID: 10,
		TenantID:           20,
		Provider:           "google",
	}, "external-sub", "owner@example.com", IdentityMetadata{
		Email:         "owner@example.com",
		EmailVerified: false,
	})

	require.NoError(t, err)
	require.NotNil(t, user)
	assert.True(t, isNew)
	assert.Equal(t, int64(200), user.UserID)
	assert.False(t, emailLookupCalled)
	require.NotNil(t, externalIdentity)
	assert.Equal(t, int64(200), externalIdentity.UserID)
}

func TestFederationServiceProvisionUser_VerifiedEmailMergesTenantAccount(t *testing.T) {
	gormDB, _ := newMockGormDB(t)

	existingUser := &User{UserID: 100, Email: "owner@example.com", IsEmailVerified: true}
	var createUserCalled bool
	userRepo := &mockUserRepo{
		findByEmailFn: func(string) (*User, error) {
			t.Fatal("global email lookup must not be used for federation merge")
			return nil, nil
		},
		findByEmailAndTenantIDFn: func(email string, tenantID int64) (*User, error) {
			assert.Equal(t, "owner@example.com", email)
			assert.Equal(t, int64(20), tenantID)
			return existingUser, nil
		},
		createFn: func(user *User) (*User, error) {
			createUserCalled = true
			return user, nil
		},
	}

	var externalIdentity *UserIdentity
	identityRepo := &mockFederationUserIdentityRepo{
		createFn: func(identity *UserIdentity) (*UserIdentity, error) {
			if identity.Provider == "google" {
				externalIdentity = identity
			}
			return identity, nil
		},
	}

	svc := &federationService{
		userRepo:         userRepo,
		userIdentityRepo: identityRepo,
		idpRepo:          &mockIdentityProviderRepo{},
		roleRepo:         &mockRoleRepo{},
	}

	user, isNew, err := svc.provisionUser(context.Background(), gormDB, &IdentityProvider{
		IdentityProviderID: 10,
		TenantID:           20,
		Provider:           "google",
	}, "external-sub", "owner@example.com", IdentityMetadata{
		Email:         "owner@example.com",
		EmailVerified: true,
	})

	require.NoError(t, err)
	require.Same(t, existingUser, user)
	assert.False(t, isNew)
	assert.False(t, createUserCalled)
	require.NotNil(t, externalIdentity)
	assert.Equal(t, int64(100), externalIdentity.UserID)
}

func TestFederationServiceProvisionUser_VerifiedEmailLookupErrorFailsClosed(t *testing.T) {
	gormDB, _ := newMockGormDB(t)
	lookupErr := errors.New("database unavailable")

	userRepo := &mockUserRepo{
		findByEmailAndTenantIDFn: func(string, int64) (*User, error) {
			return nil, lookupErr
		},
		createFn: func(user *User) (*User, error) {
			t.Fatal("user must not be created when verified email lookup fails")
			return user, nil
		},
	}

	svc := &federationService{
		userRepo:         userRepo,
		userIdentityRepo: &mockFederationUserIdentityRepo{},
		idpRepo:          &mockIdentityProviderRepo{},
		roleRepo:         &mockRoleRepo{},
	}

	user, isNew, err := svc.provisionUser(context.Background(), gormDB, &IdentityProvider{
		TenantID: 20,
		Provider: "google",
	}, "external-sub", "owner@example.com", IdentityMetadata{
		Email:         "owner@example.com",
		EmailVerified: true,
	})

	require.Error(t, err)
	assert.Nil(t, user)
	assert.False(t, isNew)
	var internalErr *apperror.InternalError
	assert.ErrorAs(t, err, &internalErr)
}

// ---------------------------------------------------------------------------
// NewFederationService
// ---------------------------------------------------------------------------

func TestNewFederationService(t *testing.T) {
	t.Run("without session service", func(t *testing.T) {
		svc := NewFederationService(nil, &mockUserRepo{}, &mockFederationUserIdentityRepo{}, &mockIdentityProviderRepo{}, &mockClientRepo{}, nil, &mockRoleRepo{}, &mockAuthEventService{})
		require.NotNil(t, svc)
	})

	t.Run("with session service", func(t *testing.T) {
		svc := NewFederationService(nil, &mockUserRepo{}, &mockFederationUserIdentityRepo{}, &mockIdentityProviderRepo{}, &mockClientRepo{}, nil, &mockRoleRepo{}, &mockAuthEventService{}, &mockSessionService{})
		require.NotNil(t, svc)
	})
}

// ---------------------------------------------------------------------------
// ExchangeExternalToken — early error paths
// ---------------------------------------------------------------------------

func TestFederationService_ExchangeExternalToken_ErrorPaths(t *testing.T) {
	req := FederationTokenRequestDTO{
		ProviderIdentifier: "idp-1",
		ExternalToken:      "tok",
		ClientID:           "app",
	}

	t.Run("provider not found", func(t *testing.T) {
		idpRepo := &mockIdentityProviderRepo{
			findByIdentifierFn: func(string) (*IdentityProvider, error) { return nil, nil },
		}
		svc := &federationService{idpRepo: idpRepo}
		_, err := svc.ExchangeExternalToken(context.Background(), req)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("provider not active", func(t *testing.T) {
		idp := activeOIDCProvider("idp-1")
		idp.Status = "inactive"
		idpRepo := &mockIdentityProviderRepo{
			findByIdentifierFn: func(string) (*IdentityProvider, error) { return idp, nil },
		}
		svc := &federationService{idpRepo: idpRepo}
		_, err := svc.ExchangeExternalToken(context.Background(), req)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not active")
	})

	t.Run("invalid OIDC config (bad JSON)", func(t *testing.T) {
		idp := activeOIDCProvider("idp-1")
		idp.Config = datatypes.JSON(json.RawMessage(`not-json`))
		idpRepo := &mockIdentityProviderRepo{
			findByIdentifierFn: func(string) (*IdentityProvider, error) { return idp, nil },
		}
		svc := &federationService{idpRepo: idpRepo}
		_, err := svc.ExchangeExternalToken(context.Background(), req)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not configured for OIDC")
	})

	t.Run("empty issuer", func(t *testing.T) {
		idp := activeOIDCProvider("idp-1")
		idp.Config = datatypes.JSON(json.RawMessage(`{}`))
		idpRepo := &mockIdentityProviderRepo{
			findByIdentifierFn: func(string) (*IdentityProvider, error) { return idp, nil },
		}
		svc := &federationService{idpRepo: idpRepo}
		_, err := svc.ExchangeExternalToken(context.Background(), req)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not configured for OIDC")
	})
}

// ---------------------------------------------------------------------------
// ExchangeOAuth2Code — early error paths
// ---------------------------------------------------------------------------

func TestFederationService_ExchangeOAuth2Code_ErrorPaths(t *testing.T) {
	req := FederationOAuth2CallbackDTO{
		ProviderIdentifier: "idp-1",
		Code:               "c",
		RedirectURI:        "https://example.com",
		ClientID:           "app",
	}

	t.Run("provider not found", func(t *testing.T) {
		idpRepo := &mockIdentityProviderRepo{
			findByIdentifierFn: func(string) (*IdentityProvider, error) { return nil, nil },
		}
		svc := &federationService{idpRepo: idpRepo}
		_, err := svc.ExchangeOAuth2Code(context.Background(), req)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("provider not active", func(t *testing.T) {
		idp := activeOIDCProvider("idp-1")
		idp.Status = "inactive"
		idpRepo := &mockIdentityProviderRepo{
			findByIdentifierFn: func(string) (*IdentityProvider, error) { return idp, nil },
		}
		svc := &federationService{idpRepo: idpRepo}
		_, err := svc.ExchangeOAuth2Code(context.Background(), req)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not active")
	})

	t.Run("invalid config JSON", func(t *testing.T) {
		idp := activeOIDCProvider("idp-1")
		idp.Config = datatypes.JSON(json.RawMessage(`not-json`))
		idpRepo := &mockIdentityProviderRepo{
			findByIdentifierFn: func(string) (*IdentityProvider, error) { return idp, nil },
		}
		svc := &federationService{idpRepo: idpRepo}
		_, err := svc.ExchangeOAuth2Code(context.Background(), req)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "configuration is invalid")
	})

	t.Run("missing client credentials", func(t *testing.T) {
		idp := activeOIDCProvider("idp-1")
		idp.Config = datatypes.JSON(json.RawMessage(`{"issuer":"https://auth.example.com"}`))
		idpRepo := &mockIdentityProviderRepo{
			findByIdentifierFn: func(string) (*IdentityProvider, error) { return idp, nil },
		}
		svc := &federationService{idpRepo: idpRepo}
		_, err := svc.ExchangeOAuth2Code(context.Background(), req)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing OAuth2 client credentials")
	})

	t.Run("missing userinfo endpoint", func(t *testing.T) {
		idp := activeOIDCProvider("idp-1")
		idp.Config = datatypes.JSON(json.RawMessage(`{"client_id":"c","client_secret":"s"}`))
		idpRepo := &mockIdentityProviderRepo{
			findByIdentifierFn: func(string) (*IdentityProvider, error) { return idp, nil },
		}
		svc := &federationService{idpRepo: idpRepo}
		_, err := svc.ExchangeOAuth2Code(context.Background(), req)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing userinfo endpoint")
	})
}

// ---------------------------------------------------------------------------
// LinkIdentity — early error paths
// ---------------------------------------------------------------------------

func TestFederationService_LinkIdentity_ErrorPaths(t *testing.T) {
	req := LinkIdentityRequestDTO{
		ProviderIdentifier: "idp-1",
		ExternalToken:      "tok",
	}

	t.Run("provider not found", func(t *testing.T) {
		idpRepo := &mockIdentityProviderRepo{
			findByIdentifierFn: func(string) (*IdentityProvider, error) { return nil, nil },
		}
		svc := &federationService{idpRepo: idpRepo}
		_, err := svc.LinkIdentity(context.Background(), 1, req)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("invalid OIDC config", func(t *testing.T) {
		idp := activeOIDCProvider("idp-1")
		idp.Config = datatypes.JSON(json.RawMessage(`{"client_id":"c"}`))
		idpRepo := &mockIdentityProviderRepo{
			findByIdentifierFn: func(string) (*IdentityProvider, error) { return idp, nil },
		}
		svc := &federationService{idpRepo: idpRepo}
		_, err := svc.LinkIdentity(context.Background(), 1, req)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not configured for OIDC")
	})
}

// ---------------------------------------------------------------------------
// UnlinkIdentity
// ---------------------------------------------------------------------------

func TestFederationService_UnlinkIdentity(t *testing.T) {
	identUUID := uuid.New()
	userID := int64(1)

	t.Run("FindByUserID error", func(t *testing.T) {
		identityRepo := &mockFederationUserIdentityRepo{
			findByUserIDFn: func(int64) ([]UserIdentity, error) {
				return nil, errors.New("db error")
			},
		}
		svc := &federationService{
			userIdentityRepo:  identityRepo,
			authEventService:  &mockAuthEventService{},
		}
		err := svc.UnlinkIdentity(context.Background(), userID, identUUID.String())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "identity lookup failed")
	})

	t.Run("identity not found", func(t *testing.T) {
		identityRepo := &mockFederationUserIdentityRepo{
			findByUserIDFn: func(int64) ([]UserIdentity, error) {
				return []UserIdentity{{UserIdentityUUID: uuid.New()}}, nil
			},
		}
		svc := &federationService{
			userIdentityRepo:  identityRepo,
			authEventService:  &mockAuthEventService{},
		}
		err := svc.UnlinkIdentity(context.Background(), userID, identUUID.String())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "identity not found")
	})

	t.Run("built-in identity cannot be unlinked", func(t *testing.T) {
		defaultIdent := UserIdentity{
			UserIdentityID:   10,
			UserIdentityUUID: identUUID,
			Provider:         shared.ProviderDefault,
			UserID:           userID,
		}
		identityRepo := &mockFederationUserIdentityRepo{
			findByUserIDFn: func(int64) ([]UserIdentity, error) {
				return []UserIdentity{defaultIdent}, nil
			},
		}
		svc := &federationService{
			userIdentityRepo:  identityRepo,
			authEventService:  &mockAuthEventService{},
		}
		err := svc.UnlinkIdentity(context.Background(), userID, identUUID.String())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "built-in identity")
	})

	t.Run("delete fails", func(t *testing.T) {
		extIdent := UserIdentity{
			UserIdentityID:   10,
			UserIdentityUUID: identUUID,
			Provider:         "google",
			UserID:           userID,
		}
		identityRepo := &mockFederationUserIdentityRepo{
			findByUserIDFn: func(int64) ([]UserIdentity, error) {
				return []UserIdentity{extIdent}, nil
			},
			deleteByIDFn: func(any) error { return errors.New("del err") },
		}
		svc := &federationService{
			userIdentityRepo:  identityRepo,
			authEventService:  &mockAuthEventService{},
		}
		err := svc.UnlinkIdentity(context.Background(), userID, identUUID.String())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to unlink")
	})

	t.Run("success", func(t *testing.T) {
		extIdent := UserIdentity{
			UserIdentityID:   10,
			UserIdentityUUID: identUUID,
			Provider:         "google",
			UserID:           userID,
		}
		identityRepo := &mockFederationUserIdentityRepo{
			findByUserIDFn: func(int64) ([]UserIdentity, error) {
				return []UserIdentity{extIdent}, nil
			},
		}
		svc := &federationService{
			userIdentityRepo:  identityRepo,
			authEventService:  &mockAuthEventService{},
		}
		err := svc.UnlinkIdentity(context.Background(), userID, identUUID.String())
		require.NoError(t, err)
	})
}

// ---------------------------------------------------------------------------
// GetUserIdentities
// ---------------------------------------------------------------------------

func TestFederationService_GetUserIdentities(t *testing.T) {
	t.Run("FindByUserID error", func(t *testing.T) {
		identityRepo := &mockFederationUserIdentityRepo{
			findByUserIDFn: func(int64) ([]UserIdentity, error) {
				return nil, errors.New("db error")
			},
		}
		svc := &federationService{userIdentityRepo: identityRepo}
		_, err := svc.GetUserIdentities(context.Background(), 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "identity lookup failed")
	})

	t.Run("success with identities", func(t *testing.T) {
		now := time.Unix(1, 0).UTC()
		identityRepo := &mockFederationUserIdentityRepo{
			findByUserIDFn: func(int64) ([]UserIdentity, error) {
				return []UserIdentity{
					{
						UserIdentityUUID: uuid.New(),
						Provider:         shared.ProviderDefault,
						Sub:              "sub-1",
						CreatedAt:        now,
					},
					{
						UserIdentityUUID: uuid.New(),
						Provider:         "google",
						Sub:              "sub-2",
						CreatedAt:        now,
					},
				}, nil
			},
		}
		svc := &federationService{userIdentityRepo: identityRepo}
		idents, err := svc.GetUserIdentities(context.Background(), 1)
		require.NoError(t, err)
		assert.Len(t, idents, 2)
		assert.True(t, idents[0].IsDefault)
		assert.False(t, idents[1].IsDefault)
		assert.Equal(t, "google", idents[1].Provider)
	})
}

// ---------------------------------------------------------------------------
// HomeRealmDiscovery
// ---------------------------------------------------------------------------

func TestFederationService_HomeRealmDiscovery(t *testing.T) {
	t.Run("invalid email", func(t *testing.T) {
		svc := &federationService{idpRepo: &mockIdentityProviderRepo{}}
		_, err := svc.HomeRealmDiscovery(context.Background(), 1, "bad")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid email")
	})

	t.Run("FindAllByTenantID error", func(t *testing.T) {
		idpRepo := &mockIdentityProviderRepo{
			findAllByTenantIDFn: func(int64) ([]IdentityProvider, error) {
				return nil, errors.New("db error")
			},
		}
		svc := &federationService{idpRepo: idpRepo}
		_, err := svc.HomeRealmDiscovery(context.Background(), 1, "user@example.com")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "provider lookup failed")
	})

	t.Run("matching domain returns provider", func(t *testing.T) {
		idpRepo := &mockIdentityProviderRepo{
			findAllByTenantIDFn: func(int64) ([]IdentityProvider, error) {
				return []IdentityProvider{{
					Identifier:  "idp-1",
					Provider:    "google",
					DisplayName: "Google",
					Config:      datatypes.JSON(json.RawMessage(`{"email_domains":["example.com"]}`)),
				}}, nil
			},
		}
		svc := &federationService{idpRepo: idpRepo}
		res, err := svc.HomeRealmDiscovery(context.Background(), 1, "user@example.com")
		require.NoError(t, err)
		assert.Equal(t, "idp-1", res.ProviderIdentifier)
	})

	t.Run("no matching domain falls back to default", func(t *testing.T) {
		idpRepo := &mockIdentityProviderRepo{
			findAllByTenantIDFn: func(int64) ([]IdentityProvider, error) {
				return []IdentityProvider{{
					Identifier:  "idp-1",
					Provider:    "google",
					DisplayName: "Google",
					Config:      datatypes.JSON(json.RawMessage(`{"email_domains":["other.com"]}`)),
				}}, nil
			},
			findDefaultByTenantIDFn: func(int64) (*IdentityProvider, error) {
				return &IdentityProvider{
					Identifier:  "default-idp",
					Provider:    "maintainerd",
					DisplayName: "Maintainerd",
				}, nil
			},
		}
		svc := &federationService{idpRepo: idpRepo}
		res, err := svc.HomeRealmDiscovery(context.Background(), 1, "user@example.com")
		require.NoError(t, err)
		assert.Equal(t, "default-idp", res.ProviderIdentifier)
	})

	t.Run("no default IDP returns error", func(t *testing.T) {
		idpRepo := &mockIdentityProviderRepo{
			findAllByTenantIDFn: func(int64) ([]IdentityProvider, error) {
				return nil, nil
			},
			findDefaultByTenantIDFn: func(int64) (*IdentityProvider, error) {
				return nil, nil
			},
		}
		svc := &federationService{idpRepo: idpRepo}
		_, err := svc.HomeRealmDiscovery(context.Background(), 1, "user@example.com")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no identity provider found")
	})

	t.Run("config unmarshal error skips provider", func(t *testing.T) {
		idpRepo := &mockIdentityProviderRepo{
			findAllByTenantIDFn: func(int64) ([]IdentityProvider, error) {
				return []IdentityProvider{{
					Identifier:  "idp-1",
					Provider:    "bad",
					DisplayName: "Bad",
					Config:      datatypes.JSON(json.RawMessage(`bad-json`)),
				}}, nil
			},
			findDefaultByTenantIDFn: func(int64) (*IdentityProvider, error) {
				return &IdentityProvider{
					Identifier:  "default-idp",
					Provider:    "maintainerd",
					DisplayName: "Maintainerd",
				}, nil
			},
		}
		svc := &federationService{idpRepo: idpRepo}
		res, err := svc.HomeRealmDiscovery(context.Background(), 1, "user@example.com")
		require.NoError(t, err)
		assert.Equal(t, "default-idp", res.ProviderIdentifier)
	})
}

// ---------------------------------------------------------------------------
// findDefaultRole
// ---------------------------------------------------------------------------

func TestFederationService_FindDefaultRole(t *testing.T) {
	t.Run("FindPaginated empty falls back to FindByNameAndTenantID", func(t *testing.T) {
		roleRepo := &mockRoleRepo{
			findPaginatedFn: func(RoleRepositoryGetFilter) (*PaginationResult[Role], error) {
				return &PaginationResult[Role]{Data: nil}, nil
			},
			findByNameAndTenantIDFn: func(string, int64) (*Role, error) {
				return &Role{RoleID: 99, Name: shared.RoleRegistered}, nil
			},
		}
		svc := &federationService{}
		role, err := svc.findDefaultRole(roleRepo, 1)
		require.NoError(t, err)
		require.NotNil(t, role)
		assert.Equal(t, int64(99), role.RoleID)
	})

	t.Run("FindPaginated returns data", func(t *testing.T) {
		roleRepo := &mockRoleRepo{
			findPaginatedFn: func(RoleRepositoryGetFilter) (*PaginationResult[Role], error) {
				return &PaginationResult[Role]{Data: []Role{{RoleID: 42}}}, nil
			},
		}
		svc := &federationService{}
		role, err := svc.findDefaultRole(roleRepo, 1)
		require.NoError(t, err)
		require.NotNil(t, role)
		assert.Equal(t, int64(42), role.RoleID)
	})
}

// ---------------------------------------------------------------------------
// provisionUser — additional branch coverage
// ---------------------------------------------------------------------------

func TestFederationServiceProvisionUser_CreateUserFails(t *testing.T) {
	gormDB, _ := newMockGormDB(t)

	userRepo := &mockUserRepo{
		createFn: func(user *User) (*User, error) {
			return nil, errors.New("create failed")
		},
	}

	svc := &federationService{
		userRepo:         userRepo,
		userIdentityRepo: &mockFederationUserIdentityRepo{},
		idpRepo:          &mockIdentityProviderRepo{},
		roleRepo:         &mockRoleRepo{},
	}

	user, isNew, err := svc.provisionUser(context.Background(), gormDB, &IdentityProvider{
		IdentityProviderID: 10,
		TenantID:           20,
		Provider:           "google",
	}, "external-sub", "user@example.com", IdentityMetadata{
		Email:         "user@example.com",
		EmailVerified: false,
	})

	require.Error(t, err)
	assert.Nil(t, user)
	assert.False(t, isNew)
}

func TestFederationServiceProvisionUser_CreateUserReturnsNil(t *testing.T) {
	gormDB, _ := newMockGormDB(t)

	userRepo := &mockUserRepo{
		createFn: func(user *User) (*User, error) {
			return nil, nil
		},
	}

	svc := &federationService{
		userRepo:         userRepo,
		userIdentityRepo: &mockFederationUserIdentityRepo{},
		idpRepo:          &mockIdentityProviderRepo{},
		roleRepo:         &mockRoleRepo{},
	}

	user, isNew, err := svc.provisionUser(context.Background(), gormDB, &IdentityProvider{
		IdentityProviderID: 10,
		TenantID:           20,
		Provider:           "google",
	}, "external-sub", "user@example.com", IdentityMetadata{})

	require.Error(t, err)
	assert.Nil(t, user)
	assert.False(t, isNew)
}

func TestFederationServiceProvisionUser_ExternalIdentityCreateFails(t *testing.T) {
	gormDB, _ := newMockGormDB(t)

	createdUser := &User{UserID: 300, Email: "user@example.com"}
	userRepo := &mockUserRepo{
		createFn: func(user *User) (*User, error) {
			return createdUser, nil
		},
	}

	callCount := 0
	identityRepo := &mockFederationUserIdentityRepo{
		createFn: func(identity *UserIdentity) (*UserIdentity, error) {
			callCount++
			if callCount >= 2 {
				return nil, errors.New("external identity create failed")
			}
			return identity, nil
		},
	}

	svc := &federationService{
		userRepo:         userRepo,
		userIdentityRepo: identityRepo,
		idpRepo:          &mockIdentityProviderRepo{},
		roleRepo:         &mockRoleRepo{},
	}

	user, isNew, err := svc.provisionUser(context.Background(), gormDB, &IdentityProvider{
		IdentityProviderID: 10,
		TenantID:           20,
		Provider:           "google",
	}, "external-sub", "user@example.com", IdentityMetadata{})

	require.Error(t, err)
	assert.Nil(t, user)
	assert.False(t, isNew)
}

func TestFederationServiceProvisionUser_WithDefaultRole(t *testing.T) {
	gormDB, _ := newMockGormDB(t)

	createdUser := &User{UserID: 400, Email: "user@example.com"}
	userRepo := &mockUserRepo{
		createFn: func(user *User) (*User, error) {
			return createdUser, nil
		},
	}

	identityRepo := &mockFederationUserIdentityRepo{
		createFn: func(identity *UserIdentity) (*UserIdentity, error) {
			return identity, nil
		},
	}

	roleRepo := &mockRoleRepo{
		findPaginatedFn: func(RoleRepositoryGetFilter) (*PaginationResult[Role], error) {
			return &PaginationResult[Role]{Data: []Role{{RoleID: 10, Name: "admin"}}}, nil
		},
	}

	svc := &federationService{
		db:               gormDB,
		userRepo:         userRepo,
		userIdentityRepo: identityRepo,
		idpRepo: &mockIdentityProviderRepo{
			findDefaultByTenantIDFn: func(tenantID int64) (*IdentityProvider, error) {
				return &IdentityProvider{
					IdentityProviderID: int64(99),
					Identifier:         "default-idp",
				}, nil
			},
		},
		roleRepo: roleRepo,
	}

	user, isNew, err := svc.provisionUser(context.Background(), gormDB, &IdentityProvider{
		IdentityProviderID: 10,
		TenantID:           20,
		Provider:           "google",
	}, "external-sub", "user@example.com", IdentityMetadata{})

	require.NoError(t, err)
	require.NotNil(t, user)
	assert.True(t, isNew)
	assert.Equal(t, int64(400), user.UserID)
}

// ---------------------------------------------------------------------------
// generateTokens
// ---------------------------------------------------------------------------

func TestFederationService_GenerateTokens_WithSession(t *testing.T) {
	initTestJWTKeysService(t)

	svc := &federationService{
		sessionService: &mockSessionService{},
	}

	user := &User{
		UserID:          1,
		UserUUID:        uuid.New(),
		Email:           "test@example.com",
		IsEmailVerified: true,
	}

	domain := "example.com"
	identifier := "app"
	client := &Client{
		Domain:     &domain,
		Identifier: &identifier,
		IdentityProvider: &IdentityProvider{
			Identifier: "default-idp",
		},
	}

	resp, err := svc.generateTokens(context.Background(), "sub-1", user, client)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.NotEmpty(t, resp.AccessToken)
	assert.NotEmpty(t, resp.IDToken)
	assert.NotEmpty(t, resp.RefreshToken)
	assert.Equal(t, "Bearer", resp.TokenType)
	assert.EqualValues(t, shared.DefaultAccessTokenExpiresIn, resp.ExpiresIn)
	assert.NotNil(t, resp.SessionID)
}

func TestFederationService_GenerateTokens_WithoutSession(t *testing.T) {
	initTestJWTKeysService(t)

	svc := &federationService{
		sessionService: nil,
	}

	user := &User{
		UserID:          1,
		UserUUID:        uuid.New(),
		Email:           "test@example.com",
		IsEmailVerified: true,
	}

	domain := "example.com"
	identifier := "app"
	client := &Client{
		Domain:     &domain,
		Identifier: &identifier,
		IdentityProvider: &IdentityProvider{
			Identifier: "default-idp",
		},
	}

	resp, err := svc.generateTokens(context.Background(), "sub-1", user, client)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.NotEmpty(t, resp.AccessToken)
	assert.NotEmpty(t, resp.IDToken)
	assert.NotEmpty(t, resp.RefreshToken)
	assert.Equal(t, "Bearer", resp.TokenType)
	assert.Nil(t, resp.SessionID)
}

func TestFederationService_GenerateTokens_AccessTokenError(t *testing.T) {
	origPriv := config.JWTPrivateKey
	origPub := config.JWTPublicKey
	defer func() {
		config.JWTPrivateKey = origPriv
		config.JWTPublicKey = origPub
		_ = jwt.InitJWTKeys()
	}()
	jwt.ResetJWTKeys()
	config.JWTPrivateKey = []byte("invalid")
	config.JWTPublicKey = []byte("invalid")

	svc := &federationService{}

	user := &User{
		UserID:          1,
		UserUUID:        uuid.New(),
		Email:           "test@example.com",
		IsEmailVerified: true,
	}

	domain := "example.com"
	identifier := "app"
	client := &Client{
		Domain:     &domain,
		Identifier: &identifier,
		IdentityProvider: &IdentityProvider{
			Identifier: "default-idp",
		},
	}

	resp, err := svc.generateTokens(context.Background(), "sub-1", user, client)
	require.Error(t, err)
	require.Nil(t, resp)
}

type mockErrorSessionService struct{}

func (m *mockErrorSessionService) ListSessions(ctx context.Context, userID int64) ([]*authn.SessionDataResult, error) {
	return nil, nil
}
func (m *mockErrorSessionService) RevokeSession(ctx context.Context, userID int64, sessionUUID uuid.UUID) error {
	return nil
}
func (m *mockErrorSessionService) RevokeAllSessions(ctx context.Context, userID int64) error {
	return nil
}
func (m *mockErrorSessionService) CreateSession(ctx context.Context, userID int64, ipAddress, userAgent string) (*authn.UserToken, error) {
	return nil, errors.New("session create failed")
}
func (m *mockErrorSessionService) EnforceConcurrentLimit(ctx context.Context, userUUID uuid.UUID, userID int64) error {
	return nil
}
func (m *mockErrorSessionService) ValidateAndTouch(ctx context.Context, sessionUUID uuid.UUID, userID int64) error {
	return nil
}

func TestFederationService_GenerateTokens_SessionError(t *testing.T) {
	svc := &federationService{
		sessionService: &mockErrorSessionService{},
	}

	user := &User{
		UserID:          1,
		UserUUID:        uuid.New(),
		Email:           "test@example.com",
		IsEmailVerified: true,
	}

	domain := "example.com"
	identifier := "app"
	client := &Client{
		Domain:     &domain,
		Identifier: &identifier,
		IdentityProvider: &IdentityProvider{
			Identifier: "default-idp",
		},
	}

	resp, err := svc.generateTokens(context.Background(), "sub-1", user, client)
	require.Error(t, err)
	require.Nil(t, resp)
}

func TestFederationService_GenerateTokens_ConcurrentLimitError(t *testing.T) {
	svc := &federationService{
		sessionService: &mockConcurrentLimitErrorService{},
	}

	user := &User{
		UserID:          1,
		UserUUID:        uuid.New(),
		Email:           "test@example.com",
		IsEmailVerified: true,
	}

	domain := "example.com"
	identifier := "app"
	client := &Client{
		Domain:     &domain,
		Identifier: &identifier,
		IdentityProvider: &IdentityProvider{
			Identifier: "default-idp",
		},
	}

	resp, err := svc.generateTokens(context.Background(), "sub-1", user, client)
	require.Error(t, err)
	require.Nil(t, resp)
}

type mockConcurrentLimitErrorService struct{}

func (m *mockConcurrentLimitErrorService) ListSessions(ctx context.Context, userID int64) ([]*authn.SessionDataResult, error) {
	return nil, nil
}
func (m *mockConcurrentLimitErrorService) RevokeSession(ctx context.Context, userID int64, sessionUUID uuid.UUID) error {
	return nil
}
func (m *mockConcurrentLimitErrorService) RevokeAllSessions(ctx context.Context, userID int64) error {
	return nil
}
func (m *mockConcurrentLimitErrorService) CreateSession(ctx context.Context, userID int64, ipAddress, userAgent string) (*authn.UserToken, error) {
	return &authn.UserToken{}, nil
}
func (m *mockConcurrentLimitErrorService) EnforceConcurrentLimit(ctx context.Context, userUUID uuid.UUID, userID int64) error {
	return errors.New("too many sessions")
}
func (m *mockConcurrentLimitErrorService) ValidateAndTouch(ctx context.Context, sessionUUID uuid.UUID, userID int64) error {
	return nil
}

// ---------------------------------------------------------------------------
// Federation pure helpers
// ---------------------------------------------------------------------------

func TestFederationPureHelpers(t *testing.T) {
	claims := map[string]interface{}{
		"email":          "User@Example.COM",
		"email_verified": true,
		"name":           "Test User",
		"given_name":     "Test",
		"family_name":    "User",
		"picture":        "https://example.com/avatar.png",
		"locale":         "en",
		"non_string":     123,
		"non_bool":       "yes",
	}

	assert.Equal(t, "Test User", stringClaim(claims, "name"))
	assert.Empty(t, stringClaim(claims, "missing"))
	assert.Empty(t, stringClaim(claims, "non_string"))
	assert.True(t, boolClaim(claims, "email_verified"))
	assert.False(t, boolClaim(claims, "missing"))
	assert.False(t, boolClaim(claims, "non_bool"))

	meta := extractMetadata(claims, map[string]string{"email": "email", "name": "name"})
	assert.Equal(t, "User@Example.COM", meta.Email)
	assert.True(t, meta.EmailVerified)
	assert.Equal(t, "Test User", meta.Name)

	assert.Equal(t, "test_user", deriveUsername(meta, "fallback@example.com"))
	assert.Equal(t, "fallback", deriveUsername(IdentityMetadata{}, "fallback@example.com"))
	assert.True(t, len(deriveUsername(IdentityMetadata{}, "")) > len("user_"))
	assert.Equal(t, "example.com", emailDomain("User@Example.COM"))
	assert.Empty(t, emailDomain("bad-email"))

	metadata, err := json.Marshal(IdentityMetadata{Email: "user@example.com", Name: "User", Picture: "pic"})
	require.NoError(t, err)
	dto := identityToDTO(&UserIdentity{
		UserIdentityUUID: uuid.New(),
		Provider:         shared.ProviderDefault,
		Sub:              "sub",
		Metadata:         datatypes.JSON(metadata),
		CreatedAt:        time.Unix(1, 0).UTC(),
	})
	require.NotNil(t, dto)
	assert.True(t, dto.IsDefault)
	assert.Equal(t, "user@example.com", *dto.Email)

	idp := &IdentityProvider{Identifier: "google", Provider: "google", DisplayName: "Google"}
	hrd := hrdResponseFrom(idp)
	assert.Equal(t, "google", hrd.ProviderIdentifier)
	assert.Equal(t, "Google", hrd.DisplayName)
}

func TestFederationIdentityToDTO_EmptyMetadata(t *testing.T) {
	now := time.Unix(1, 0).UTC()
	dto := identityToDTO(&UserIdentity{
		UserIdentityUUID: uuid.New(),
		Provider:         "google",
		Sub:              "sub-1",
		Metadata:         nil,
		CreatedAt:        now,
	})
	require.NotNil(t, dto)
	assert.False(t, dto.IsDefault)
	assert.Nil(t, dto.Email)
	assert.Nil(t, dto.Name)
	assert.Nil(t, dto.Picture)
}

func TestFederationExtractMetadata_NilMapping(t *testing.T) {
	claims := map[string]interface{}{
		"email": "test@example.com",
	}
	meta := extractMetadata(claims, nil)
	assert.Equal(t, "test@example.com", meta.Email)
}

func TestFederationExtractMetadata_MissingClaims(t *testing.T) {
	meta := extractMetadata(map[string]interface{}{}, nil)
	assert.Empty(t, meta.Email)
	assert.False(t, meta.EmailVerified)
	assert.Empty(t, meta.Name)
}

func TestEmailDomain_WithSubdomain(t *testing.T) {
	assert.Equal(t, "sub.example.com", emailDomain("user@sub.example.com"))
}

func TestEmailDomain_NoAt(t *testing.T) {
	assert.Empty(t, emailDomain("noatsign"))
}

func TestEmailDomain_EmptyDomain(t *testing.T) {
	assert.Empty(t, emailDomain("user@"))
}
