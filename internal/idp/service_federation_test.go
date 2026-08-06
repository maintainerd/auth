package idp

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	jwtlib "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/authevent"
	"github.com/maintainerd/maintainerd-auth/internal/authn"
	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	"github.com/maintainerd/maintainerd-auth/internal/platform/config"
	"github.com/maintainerd/maintainerd-auth/internal/platform/crypto"
	"github.com/maintainerd/maintainerd-auth/internal/platform/jwt"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
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
	findByTenantAndSubFn func(int64, string) (*UserIdentity, error)
	mockBaseRepo[UserIdentity]
	findByUserIDFn               func(int64) ([]UserIdentity, error)
	findByUserIDAndProviderFn    func(int64, string) (*UserIdentity, error)
	findByUserIDAndIDPIDFn       func(int64, int64) (*UserIdentity, error)
	findByTenantProviderAndSubFn func(int64, string, string) (*UserIdentity, error)
	createFn                     func(*UserIdentity) (*UserIdentity, error)
	createIfAbsentFn             func(*UserIdentity) (*UserIdentity, bool, error)
	deleteByIDFn                 func(any) error
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

func (m *mockFederationUserIdentityRepo) FindByTenantProviderAndSub(tenantID int64, provider, sub string) (*UserIdentity, error) {
	if m.findByTenantProviderAndSubFn != nil {
		return m.findByTenantProviderAndSubFn(tenantID, provider, sub)
	}
	return nil, nil
}

func (m *mockFederationUserIdentityRepo) FindByTenantAndSub(tenantID int64, sub string) (*UserIdentity, error) {
	if m.findByTenantAndSubFn != nil {
		return m.findByTenantAndSubFn(tenantID, sub)
	}
	return nil, nil
}

func (m *mockFederationUserIdentityRepo) FindByUserIDAndProvider(userID int64, provider string) (*UserIdentity, error) {
	if m.findByUserIDAndProviderFn != nil {
		return m.findByUserIDAndProviderFn(userID, provider)
	}
	return nil, nil
}

func (m *mockFederationUserIdentityRepo) FindByUserIDAndIdentityProviderID(userID int64, idpID int64) (*UserIdentity, error) {
	if m.findByUserIDAndIDPIDFn != nil {
		return m.findByUserIDAndIDPIDFn(userID, idpID)
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

func (m *mockFederationUserIdentityRepo) CreateByTenantProviderSubIfAbsent(identity *UserIdentity) (*UserIdentity, bool, error) {
	if m.createIfAbsentFn != nil {
		return m.createIfAbsentFn(identity)
	}
	if m.createFn != nil {
		created, err := m.createFn(identity)
		return created, err == nil, err
	}
	return identity, true, nil
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

type mockSessionService struct {
	// revokedAllUserIDs records every user whose sessions were globally revoked,
	// so a logout test can assert the sessions were actually ended rather than
	// just that the call returned a redirect.
	revokedAllUserIDs []int64
	revokeAllErr      error
}

func (m *mockSessionService) ListSessions(ctx context.Context, userID int64) ([]*authn.SessionDataResult, error) {
	return nil, nil
}
func (m *mockSessionService) RevokeSession(ctx context.Context, userID int64, sessionUUID uuid.UUID) error {
	return nil
}
func (m *mockSessionService) RevokeAllSessions(ctx context.Context, userID int64, reason string) error {
	m.revokedAllUserIDs = append(m.revokedAllUserIDs, userID)
	return m.revokeAllErr
}
func (m *mockSessionService) CreateSession(ctx context.Context, userID, tenantID int64, ipAddress, userAgent string) (*authn.UserSession, error) {
	return &authn.UserSession{}, nil
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

// promoteIDPConfigColumns mirrors production storage: issuer / client_id /
// client_secret / allow_jit_provisioning live in dedicated columns, not the
// config JSONB. It parses those keys out of the legacy config JSON and sets the
// columns. The secret is ENCRYPTED at rest (as production does), so the strict
// federation decrypt path round-trips it back to plaintext. The package TestMain
// installs the AES key that makes this work.
func promoteIDPConfigColumns(idp *IdentityProvider) {
	if len(idp.Config) == 0 {
		return
	}
	var m map[string]any
	if json.Unmarshal(idp.Config, &m) != nil {
		return
	}
	if v, ok := m["issuer"].(string); ok && v != "" {
		s := v
		idp.Issuer = &s
	}
	if v, ok := m["client_id"].(string); ok && v != "" {
		s := v
		idp.ProviderClientID = &s
	}
	if v, ok := m["client_secret"].(string); ok && v != "" {
		if enc, err := crypto.EncryptAtRest(v); err == nil {
			idp.ProviderClientSecretEncrypted = &enc
		}
	}
	if v, ok := m["allow_jit_provisioning"].(bool); ok {
		idp.AllowJITProvisioning = v
	}
}

// setIDPConfigJSON replaces a fixture's config and re-derives the promoted
// columns, clearing any previously promoted values first.
func setIDPConfigJSON(idp *IdentityProvider, cfg datatypes.JSON) {
	idp.Issuer = nil
	idp.ProviderClientID = nil
	idp.ProviderClientSecretEncrypted = nil
	idp.AllowJITProvisioning = false
	idp.Config = cfg
	promoteIDPConfigColumns(idp)
}

func TestResolveTokenEndpoint(t *testing.T) {
	ctx := context.Background()
	// Explicit token_endpoint wins (Google/Facebook-style).
	assert.Equal(t, "https://custom.example.com/oauth2/token",
		resolveTokenEndpoint(ctx, "https://auth.example.com", OIDCProviderConfig{TokenEndpoint: "https://custom.example.com/oauth2/token"}))
	// Falls back to OIDC discovery when issuer is set.
	orig := idpOIDCDiscover
	idpOIDCDiscover = func(context.Context, string) (string, string, error) {
		return "https://accounts.google.com/o/oauth2/auth", "https://oauth2.googleapis.com/token", nil
	}
	t.Cleanup(func() { idpOIDCDiscover = orig })
	assert.Equal(t, "https://oauth2.googleapis.com/token",
		resolveTokenEndpoint(ctx, "https://accounts.google.com", OIDCProviderConfig{}))
	// Falls back to legacy when discovery fails.
	idpOIDCDiscover = func(context.Context, string) (string, string, error) {
		return "", "", errors.New("discovery down")
	}
	assert.Equal(t, "https://accounts.google.com/oauth/token",
		resolveTokenEndpoint(ctx, "https://accounts.google.com/", OIDCProviderConfig{}))
	// Legacy default when nothing else works.
	assert.Equal(t, "https://auth.example.com/oauth/token",
		resolveTokenEndpoint(ctx, "https://auth.example.com/", OIDCProviderConfig{}))
}

func TestFederationService_ResolveBrokerProvider(t *testing.T) {
	mkProvider := func(config string) *IdentityProvider {
		idp := activeOIDCProvider("google")
		setIDPConfigJSON(idp, datatypes.JSON(config))
		return idp
	}
	svcWith := func(idp *IdentityProvider, findErr error) *federationService {
		return &federationService{idpRepo: &mockIdentityProviderRepo{
			findByIdentifierFn: func(string) (*IdentityProvider, error) { return idp, findErr },
		}}
	}

	t.Run("explicit authorization_endpoint", func(t *testing.T) {
		svc := svcWith(mkProvider(`{"client_id":"upstream","authorization_endpoint":"https://idp.example.com/authorize","token_endpoint":"https://idp.example.com/token","scopes":["openid","email"]}`), nil)
		info, err := svc.ResolveBrokerProvider(context.Background(), "google")
		require.NoError(t, err)
		assert.Equal(t, "https://idp.example.com/authorize", info.AuthorizationEndpoint)
		assert.Equal(t, "upstream", info.ClientID)
		assert.Equal(t, []string{"openid", "email"}, info.Scopes)
	})

	t.Run("discovery fallback", func(t *testing.T) {
		orig := idpOIDCDiscover
		idpOIDCDiscover = func(context.Context, string) (string, string, error) {
			return "https://accounts.google.com/o/oauth2/v2/auth", "https://oauth2.googleapis.com/token", nil
		}
		t.Cleanup(func() { idpOIDCDiscover = orig })
		svc := svcWith(mkProvider(`{"client_id":"upstream","issuer":"https://accounts.google.com"}`), nil)
		info, err := svc.ResolveBrokerProvider(context.Background(), "google")
		require.NoError(t, err)
		assert.Equal(t, "https://accounts.google.com/o/oauth2/v2/auth", info.AuthorizationEndpoint)
	})

	t.Run("discovery error", func(t *testing.T) {
		orig := idpOIDCDiscover
		idpOIDCDiscover = func(context.Context, string) (string, string, error) {
			return "", "", errors.New("discovery failed")
		}
		t.Cleanup(func() { idpOIDCDiscover = orig })
		svc := svcWith(mkProvider(`{"client_id":"upstream","issuer":"https://accounts.google.com"}`), nil)
		_, err := svc.ResolveBrokerProvider(context.Background(), "google")
		require.Error(t, err)
	})

	t.Run("missing client_id", func(t *testing.T) {
		svc := svcWith(mkProvider(`{"issuer":"https://idp.example.com"}`), nil)
		_, err := svc.ResolveBrokerProvider(context.Background(), "google")
		require.Error(t, err)
	})

	t.Run("missing endpoint and issuer", func(t *testing.T) {
		svc := svcWith(mkProvider(`{"client_id":"upstream"}`), nil)
		_, err := svc.ResolveBrokerProvider(context.Background(), "google")
		require.Error(t, err)
	})

	t.Run("provider not found", func(t *testing.T) {
		svc := svcWith(nil, nil)
		_, err := svc.ResolveBrokerProvider(context.Background(), "google")
		require.Error(t, err)
	})

	t.Run("inactive provider", func(t *testing.T) {
		idp := mkProvider(`{"client_id":"upstream","issuer":"https://idp.example.com"}`)
		idp.Status = "inactive"
		svc := svcWith(idp, nil)
		_, err := svc.ResolveBrokerProvider(context.Background(), "google")
		require.Error(t, err)
	})
}

// systemIDPStub returns the tenant's built-in system IdP. Its id matches
// activeOIDCProvider's id (1) so the single identity created in most fixtures is
// reachable both by the external-IdP id and the system-IdP id.
func systemIDPStub(int64) (*IdentityProvider, error) {
	return &IdentityProvider{IdentityProviderID: 1, IsSystem: true, ProviderType: shared.IDPTypeSystem}, nil
}

func activeOIDCProvider(identifier string) *IdentityProvider {
	idp := &IdentityProvider{
		IdentityProviderID: 1,
		Identifier:         identifier,
		Provider:           "google",
		Status:             "active",
		TenantID:           1,
		Config:             validOIDCConfigJSON(),
	}
	promoteIDPConfigColumns(idp)
	return idp
}

func jitOIDCConfigJSON() datatypes.JSON {
	return datatypes.JSON(json.RawMessage(`{
		"issuer":"https://accounts.google.com",
		"client_id":"test-client",
		"allow_jit_provisioning":true,
		"attribute_mapping":{"email":"email","name":"name","email_verified":"email_verified"}
	}`))
}

func federationClient() *Client {
	domain := "https://auth.example.com"
	identifier := "app"
	return &Client{
		ClientID:   10,
		TenantID:   1,
		Domain:     &domain,
		Identifier: &identifier,
		IdentityProvider: &IdentityProvider{
			Identifier: "default-idp",
		},
	}
}

func stubFederationTokenHooks(t *testing.T) {
	t.Helper()
	origAccess := idpGenerateAccessTokenWithOptionsContext
	origID := idpGenerateIDTokenWithContext
	origRefresh := idpGenerateRefreshTokenWithContext
	idpGenerateAccessTokenWithOptionsContext = func(context.Context, string, string, string, string, string, string, *jwt.AccessTokenOptions) (string, error) {
		return "access-token", nil
	}
	idpGenerateIDTokenWithContext = func(context.Context, string, string, string, string, *jwt.UserProfile, string, *jwt.IDTokenParams) (string, error) {
		return "id-token", nil
	}
	idpGenerateRefreshTokenWithContext = func(context.Context, string, string, string, string) (string, error) {
		return "refresh-token", nil
	}
	t.Cleanup(func() {
		idpGenerateAccessTokenWithOptionsContext = origAccess
		idpGenerateIDTokenWithContext = origID
		idpGenerateRefreshTokenWithContext = origRefresh
	})
}

func stubOIDCClaims(t *testing.T, claims map[string]interface{}) {
	t.Helper()
	orig := idpValidateOIDCToken
	idpValidateOIDCToken = func(*federationService, context.Context, string, string, string) (map[string]interface{}, error) {
		return claims, nil
	}
	t.Cleanup(func() { idpValidateOIDCToken = orig })
}

func stubOAuth2Userinfo(t *testing.T, body string) {
	t.Helper()
	origExchange := idpOAuth2Exchange
	origUserinfo := idpOAuth2GetUserinfo
	idpOAuth2Exchange = func(context.Context, *oauth2.Config, string) (*oauth2.Token, error) {
		return &oauth2.Token{AccessToken: "provider-access"}, nil
	}
	idpOAuth2GetUserinfo = func(context.Context, *oauth2.Config, *oauth2.Token, string) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewBufferString(body))}, nil
	}
	t.Cleanup(func() {
		idpOAuth2Exchange = origExchange
		idpOAuth2GetUserinfo = origUserinfo
	})
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
		idpRepo:          &mockIdentityProviderRepo{findSystemByTenantIDFn: systemIDPStub},
		roleRepo:         &mockRoleRepo{},
	}

	user, isNew, err := svc.provisionUser(context.Background(), gormDB, &IdentityProvider{
		IdentityProviderID: 10,
		TenantID:           20,
		Provider:           "google",
	}, "external-sub", "owner@example.com", IdentityMetadata{
		Email:         "owner@example.com",
		EmailVerified: false,
	}, int64Ptr(10))

	require.NoError(t, err)
	require.NotNil(t, user)
	assert.True(t, isNew)
	assert.Equal(t, int64(200), user.UserID)
	assert.False(t, emailLookupCalled)
	require.NotNil(t, externalIdentity)
	assert.Equal(t, int64(200), externalIdentity.UserID)
}

// F3: a verified-email collision with a pre-existing account must fail closed —
// provisionUser surfaces errEmailCollision instead of silently merging, even
// when no account-link service is wired.
func TestFederationServiceProvisionUser_VerifiedEmailCollisionFailsClosed(t *testing.T) {
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

	var externalIdentityCreated bool
	identityRepo := &mockFederationUserIdentityRepo{
		createFn: func(identity *UserIdentity) (*UserIdentity, error) {
			externalIdentityCreated = true
			return identity, nil
		},
	}

	// accountLinkSvc intentionally left nil — the old code silently merged in this
	// case; the fix must instead fail closed with a collision.
	svc := &federationService{
		userRepo:         userRepo,
		userIdentityRepo: identityRepo,
		idpRepo:          &mockIdentityProviderRepo{findSystemByTenantIDFn: systemIDPStub},
		roleRepo:         &mockRoleRepo{},
	}

	user, isNew, err := svc.provisionUser(context.Background(), gormDB, &IdentityProvider{
		IdentityProviderID: 10,
		TenantID:           20,
		Provider:           "google",
	}, "external-sub", "owner@example.com", IdentityMetadata{
		Email:         "owner@example.com",
		EmailVerified: true,
	}, int64Ptr(10))

	require.Error(t, err)
	var collision *errEmailCollision
	require.ErrorAs(t, err, &collision)
	assert.Equal(t, int64(100), collision.existingUserID)
	assert.Equal(t, "owner@example.com", collision.providerEmail)
	assert.Nil(t, user)
	assert.False(t, isNew)
	assert.False(t, createUserCalled, "must not create a new user on collision")
	assert.False(t, externalIdentityCreated, "must not silently link into the existing account")
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
	}, int64Ptr(10))

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
		svc := NewFederationService(nil, &mockUserRepo{}, &mockFederationUserIdentityRepo{}, &mockIdentityProviderRepo{}, &mockIdentityProviderEmailDomainRepo{}, &mockClientRepo{}, nil, &mockRoleRepo{}, &mockAuthEventService{}, nil, nil, nil)
		require.NotNil(t, svc)
	})

	t.Run("with session service", func(t *testing.T) {
		svc := NewFederationService(nil, &mockUserRepo{}, &mockFederationUserIdentityRepo{}, &mockIdentityProviderRepo{}, &mockIdentityProviderEmailDomainRepo{}, &mockClientRepo{}, nil, &mockRoleRepo{}, &mockAuthEventService{}, nil, nil, nil, &mockSessionService{})
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
		setIDPConfigJSON(idp, datatypes.JSON(json.RawMessage(`not-json`)))
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
		setIDPConfigJSON(idp, datatypes.JSON(json.RawMessage(`{}`)))
		idpRepo := &mockIdentityProviderRepo{
			findByIdentifierFn: func(string) (*IdentityProvider, error) { return idp, nil },
		}
		svc := &federationService{idpRepo: idpRepo}
		_, err := svc.ExchangeExternalToken(context.Background(), req)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not configured for OIDC")
	})
}

func TestFederationService_ExchangeExternalToken_Branches(t *testing.T) {
	req := FederationTokenRequestDTO{ProviderIdentifier: "idp-1", ExternalToken: "tok", ClientID: "app"}

	t.Run("OIDC validation error", func(t *testing.T) {
		orig := idpValidateOIDCToken
		idpValidateOIDCToken = func(*federationService, context.Context, string, string, string) (map[string]interface{}, error) {
			return nil, errors.New("bad token")
		}
		t.Cleanup(func() { idpValidateOIDCToken = orig })

		idp := activeOIDCProvider("idp-1")
		svc := &federationService{idpRepo: &mockIdentityProviderRepo{
			findByIdentifierFn: func(string) (*IdentityProvider, error) { return idp, nil },
		}}

		_, err := svc.ExchangeExternalToken(context.Background(), req)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "external token validation failed")
	})

	t.Run("missing sub claim", func(t *testing.T) {
		stubOIDCClaims(t, map[string]interface{}{"email": "user@example.com"})
		idp := activeOIDCProvider("idp-1")
		svc := &federationService{idpRepo: &mockIdentityProviderRepo{
			findByIdentifierFn: func(string) (*IdentityProvider, error) { return idp, nil },
		}}

		_, err := svc.ExchangeExternalToken(context.Background(), req)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing 'sub'")
	})

	t.Run("JIT disabled for unknown identity", func(t *testing.T) {
		stubOIDCClaims(t, map[string]interface{}{"sub": "external-sub", "email": "user@example.com"})
		gdb, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()

		idp := activeOIDCProvider("idp-1")
		svc := &federationService{
			db: gdb,
			idpRepo: &mockIdentityProviderRepo{
				findByIdentifierFn: func(string) (*IdentityProvider, error) { return idp, nil },
			},
			userIdentityRepo: &mockFederationUserIdentityRepo{},
			userRepo:         &mockUserRepo{},
		}

		_, err := svc.ExchangeExternalToken(context.Background(), req)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "JIT provisioning is disabled")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("success with JIT provisioned user", func(t *testing.T) {
		stubOIDCClaims(t, map[string]interface{}{
			"sub":            "external-sub",
			"email":          "user@example.com",
			"email_verified": true,
			"name":           "User Name",
		})
		stubFederationTokenHooks(t)
		gdb, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()

		idp := activeOIDCProvider("idp-1")
		setIDPConfigJSON(idp, jitOIDCConfigJSON())
		createdUser := &User{UserID: 10, UserUUID: uuid.New(), Email: "user@example.com", IsEmailVerified: true}
		var logged bool
		svc := &federationService{
			db: gdb,
			idpRepo: &mockIdentityProviderRepo{
				findByIdentifierFn:     func(string) (*IdentityProvider, error) { return idp, nil },
				findSystemByTenantIDFn: systemIDPStub,
			},
			userIdentityRepo: &mockFederationUserIdentityRepo{
				findByUserIDAndIDPIDFn: func(userID int64, idpID int64) (*UserIdentity, error) {
					return &UserIdentity{UserID: userID, Provider: shared.ProviderMaintainerd, Sub: "internal-sub"}, nil
				},
			},
			userRepo: &mockUserRepo{
				createFn: func(user *User) (*User, error) {
					assert.Equal(t, "user@example.com", user.Email)
					return createdUser, nil
				},
			},
			roleRepo: &mockRoleRepo{},
			clientRepo: &mockClientRepo{
				findByClientIDAndIdentityProviderFn: func(clientID, provider string) (*Client, error) {
					assert.Equal(t, "app", clientID)
					assert.Equal(t, "idp-1", provider)
					return federationClient(), nil
				},
			},
			authEventService: &mockAuthEventService{
				logFn: func(context.Context, authevent.AuthEventInput) { logged = true },
			},
		}

		resp, err := svc.ExchangeExternalToken(context.Background(), req)
		require.NoError(t, err)
		assert.Equal(t, "access-token", resp.AccessToken)
		assert.True(t, logged)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	// Inverted: this used to assert that a client NOT connected to the requesting
	// provider was still accepted if it was connected to the tenant's default
	// one. That defeats the client↔provider connection model, and once the client
	// lookup began rejecting disabled connections it became the way to route
	// around a connection an admin had just disabled. It must now be refused.
	t.Run("client not connected to the requesting provider is refused", func(t *testing.T) {
		stubOIDCClaims(t, map[string]interface{}{"sub": "external-sub"})
		stubFederationTokenHooks(t)
		gdb, mock := newMockGormDB(t)
		// Client resolution happens inside the provisioning transaction, so the
		// refusal must roll it back — no user is left behind.
		mock.ExpectBegin()
		mock.ExpectRollback()

		idp := activeOIDCProvider("idp-1")
		setIDPConfigJSON(idp, jitOIDCConfigJSON())
		svc := &federationService{
			db: gdb,
			idpRepo: &mockIdentityProviderRepo{
				findByIdentifierFn:     func(string) (*IdentityProvider, error) { return idp, nil },
				findSystemByTenantIDFn: systemIDPStub,
				findDefaultByTenantIDFn: func(int64) (*IdentityProvider, error) {
					return &IdentityProvider{Identifier: "default-idp"}, nil
				},
			},
			userIdentityRepo: &mockFederationUserIdentityRepo{
				findByUserIDAndIDPIDFn: func(userID int64, idpID int64) (*UserIdentity, error) {
					return &UserIdentity{UserID: userID, Sub: "internal-sub"}, nil
				},
			},
			userRepo: &mockUserRepo{
				createFn: func(user *User) (*User, error) {
					user.UserID = 11
					user.UserUUID = uuid.New()
					return user, nil
				},
			},
			roleRepo: &mockRoleRepo{},
			clientRepo: &mockClientRepo{
				findByClientIDAndIdentityProviderFn: func(_ string, provider string) (*Client, error) {
					if provider == "default-idp" {
						return federationClient(), nil
					}
					return nil, nil
				},
			},
			authEventService: &mockAuthEventService{},
		}

		_, err := svc.ExchangeExternalToken(context.Background(), req)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "client not found for this provider")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("identity lookup error", func(t *testing.T) {
		stubOIDCClaims(t, map[string]interface{}{"sub": "external-sub"})
		gdb, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		idp := activeOIDCProvider("idp-1")
		svc := &federationService{
			db: gdb,
			idpRepo: &mockIdentityProviderRepo{
				findByIdentifierFn: func(string) (*IdentityProvider, error) { return idp, nil },
			},
			userIdentityRepo: &mockFederationUserIdentityRepo{
				findByTenantProviderAndSubFn: func(int64, string, string) (*UserIdentity, error) {
					return nil, errors.New("identity lookup error")
				},
			},
			userRepo: &mockUserRepo{},
		}
		_, err := svc.ExchangeExternalToken(context.Background(), req)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "identity lookup failed")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("existing identity user missing", func(t *testing.T) {
		stubOIDCClaims(t, map[string]interface{}{"sub": "external-sub"})
		gdb, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		idp := activeOIDCProvider("idp-1")
		svc := &federationService{
			db: gdb,
			idpRepo: &mockIdentityProviderRepo{
				findByIdentifierFn: func(string) (*IdentityProvider, error) { return idp, nil },
			},
			userIdentityRepo: &mockFederationUserIdentityRepo{
				findByTenantProviderAndSubFn: func(int64, string, string) (*UserIdentity, error) {
					return &UserIdentity{UserID: 10, Provider: "google", Sub: "external-sub"}, nil
				},
			},
			userRepo: &mockUserRepo{
				findByIDFn: func(any, ...string) (*User, error) { return nil, nil },
			},
		}
		_, err := svc.ExchangeExternalToken(context.Background(), req)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("default identity lookup error", func(t *testing.T) {
		stubOIDCClaims(t, map[string]interface{}{"sub": "external-sub"})
		gdb, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		idp := activeOIDCProvider("idp-1")
		svc := &federationService{
			db: gdb,
			idpRepo: &mockIdentityProviderRepo{
				findByIdentifierFn:     func(string) (*IdentityProvider, error) { return idp, nil },
				findSystemByTenantIDFn: systemIDPStub,
			},
			userIdentityRepo: &mockFederationUserIdentityRepo{
				findByTenantProviderAndSubFn: func(int64, string, string) (*UserIdentity, error) {
					return nil, nil
				},
				findByUserIDAndIDPIDFn: func(int64, int64) (*UserIdentity, error) {
					return nil, errors.New("default lookup error")
				},
			},
			userRepo: &mockUserRepo{
				createFn: func(user *User) (*User, error) {
					user.UserID = 20
					user.UserUUID = uuid.New()
					return user, nil
				},
			},
			roleRepo: &mockRoleRepo{},
			clientRepo: &mockClientRepo{
				findByClientIDAndIdentityProviderFn: func(string, string) (*Client, error) {
					return federationClient(), nil
				},
			},
		}
		setIDPConfigJSON(idp, jitOIDCConfigJSON())
		_, err := svc.ExchangeExternalToken(context.Background(), req)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "default identity lookup failed")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("client missing after fallback", func(t *testing.T) {
		stubOIDCClaims(t, map[string]interface{}{"sub": "external-sub"})
		gdb, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		idp := activeOIDCProvider("idp-1")
		setIDPConfigJSON(idp, jitOIDCConfigJSON())
		svc := &federationService{
			db: gdb,
			idpRepo: &mockIdentityProviderRepo{
				findByIdentifierFn:      func(string) (*IdentityProvider, error) { return idp, nil },
				findDefaultByTenantIDFn: func(int64) (*IdentityProvider, error) { return nil, nil },
			},
			userIdentityRepo: &mockFederationUserIdentityRepo{
				findByUserIDAndProviderFn: func(userID int64, provider string) (*UserIdentity, error) {
					return &UserIdentity{UserID: userID, Provider: provider, Sub: "internal-sub"}, nil
				},
			},
			userRepo:   &mockUserRepo{},
			roleRepo:   &mockRoleRepo{},
			clientRepo: &mockClientRepo{},
		}
		_, err := svc.ExchangeExternalToken(context.Background(), req)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "client not found")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("existing identity refreshes metadata", func(t *testing.T) {
		stubOIDCClaims(t, map[string]interface{}{"sub": "external-sub", "email": "user@example.com"})
		stubFederationTokenHooks(t)
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE "user_identities" SET "metadata"=.*"updated_at"=.*WHERE user_identity_id = .*`).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		idp := activeOIDCProvider("idp-1")
		svc := &federationService{
			db: gdb,
			idpRepo: &mockIdentityProviderRepo{
				findByIdentifierFn:     func(string) (*IdentityProvider, error) { return idp, nil },
				findSystemByTenantIDFn: systemIDPStub,
			},
			userIdentityRepo: &mockFederationUserIdentityRepo{
				findByTenantProviderAndSubFn: func(int64, string, string) (*UserIdentity, error) {
					return &UserIdentity{UserIdentityID: 1, UserID: 30, Provider: "google", Sub: "external-sub"}, nil
				},
				findByUserIDAndIDPIDFn: func(userID int64, idpID int64) (*UserIdentity, error) {
					return &UserIdentity{UserID: userID, Sub: "internal-sub"}, nil
				},
			},
			userRepo: &mockUserRepo{
				findByIDFn: func(any, ...string) (*User, error) {
					return &User{UserID: 30, UserUUID: uuid.New(), Email: "user@example.com"}, nil
				},
			},
			clientRepo: &mockClientRepo{
				findByClientIDAndIdentityProviderFn: func(string, string) (*Client, error) {
					return federationClient(), nil
				},
			},
			authEventService: &mockAuthEventService{},
		}
		resp, err := svc.ExchangeExternalToken(context.Background(), req)
		require.NoError(t, err)
		assert.Equal(t, "access-token", resp.AccessToken)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("provision error is returned", func(t *testing.T) {
		stubOIDCClaims(t, map[string]interface{}{"sub": "external-sub", "email": "user@example.com"})
		gdb, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		idp := activeOIDCProvider("idp-1")
		setIDPConfigJSON(idp, jitOIDCConfigJSON())
		svc := &federationService{
			db: gdb,
			idpRepo: &mockIdentityProviderRepo{
				findByIdentifierFn: func(string) (*IdentityProvider, error) { return idp, nil },
			},
			userIdentityRepo: &mockFederationUserIdentityRepo{},
			userRepo: &mockUserRepo{
				createFn: func(*User) (*User, error) { return nil, errors.New("create user error") },
			},
			roleRepo: &mockRoleRepo{},
			clientRepo: &mockClientRepo{
				findByClientIDAndIdentityProviderFn: func(string, string) (*Client, error) {
					return federationClient(), nil
				},
			},
		}
		_, err := svc.ExchangeExternalToken(context.Background(), req)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to provision user")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("default identity missing", func(t *testing.T) {
		stubOIDCClaims(t, map[string]interface{}{"sub": "external-sub"})
		gdb, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		idp := activeOIDCProvider("idp-1")
		setIDPConfigJSON(idp, jitOIDCConfigJSON())
		svc := &federationService{
			db: gdb,
			idpRepo: &mockIdentityProviderRepo{
				findByIdentifierFn:     func(string) (*IdentityProvider, error) { return idp, nil },
				findSystemByTenantIDFn: systemIDPStub,
			},
			userIdentityRepo: &mockFederationUserIdentityRepo{},
			userRepo: &mockUserRepo{
				createFn: func(user *User) (*User, error) {
					user.UserID = 31
					user.UserUUID = uuid.New()
					return user, nil
				},
			},
			roleRepo: &mockRoleRepo{},
			clientRepo: &mockClientRepo{
				findByClientIDAndIdentityProviderFn: func(string, string) (*Client, error) {
					return federationClient(), nil
				},
			},
		}
		_, err := svc.ExchangeExternalToken(context.Background(), req)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no default identity")
		assert.NoError(t, mock.ExpectationsWereMet())
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
		setIDPConfigJSON(idp, datatypes.JSON(json.RawMessage(`not-json`)))
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
		setIDPConfigJSON(idp, datatypes.JSON(json.RawMessage(`{"issuer":"https://auth.example.com"}`)))
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
		setIDPConfigJSON(idp, datatypes.JSON(json.RawMessage(`{"client_id":"c","client_secret":"s"}`)))
		idpRepo := &mockIdentityProviderRepo{
			findByIdentifierFn: func(string) (*IdentityProvider, error) { return idp, nil },
		}
		svc := &federationService{idpRepo: idpRepo}
		_, err := svc.ExchangeOAuth2Code(context.Background(), req)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing userinfo endpoint")
	})

	// FIX E: an undecryptable stored secret must fail the exchange CLOSED — the raw
	// ciphertext must never be POSTed upstream as the client secret. The stored
	// value below is not valid ciphertext for the configured key, so strict decrypt
	// errors and the exchange aborts before any upstream call.
	t.Run("undecryptable secret fails closed", func(t *testing.T) {
		idp := activeOIDCProvider("idp-1")
		setIDPConfigJSON(idp, datatypes.JSON(json.RawMessage(`{"issuer":"https://auth.example.com","client_id":"test-client","userinfo_endpoint":"https://auth.example.com/userinfo"}`)))
		garbage := "this-is-not-valid-ciphertext"
		idp.ProviderClientSecretEncrypted = &garbage

		origExchange := idpOAuth2Exchange
		idpOAuth2Exchange = func(context.Context, *oauth2.Config, string) (*oauth2.Token, error) {
			t.Fatal("upstream token exchange must not be called when the secret is undecryptable")
			return nil, nil
		}
		t.Cleanup(func() { idpOAuth2Exchange = origExchange })

		idpRepo := &mockIdentityProviderRepo{
			findByIdentifierFn: func(string) (*IdentityProvider, error) { return idp, nil },
		}
		svc := &federationService{idpRepo: idpRepo}
		_, err := svc.ExchangeOAuth2Code(context.Background(), req)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "provider client secret unavailable")
	})
}

type readErrorCloser struct{}

func (readErrorCloser) Read([]byte) (int, error) { return 0, errors.New("read failed") }
func (readErrorCloser) Close() error             { return nil }

func TestFederationService_ExchangeOAuth2Code_Branches(t *testing.T) {
	req := FederationOAuth2CallbackDTO{
		ProviderIdentifier: "idp-1",
		Code:               "c",
		RedirectURI:        "https://example.com",
		ClientID:           "app",
	}
	provider := func() *IdentityProvider {
		idp := activeOIDCProvider("idp-1")
		setIDPConfigJSON(idp, validOAuth2ConfigJSON())
		return idp
	}

	t.Run("OAuth2 code exchange error", func(t *testing.T) {
		orig := idpOAuth2Exchange
		idpOAuth2Exchange = func(context.Context, *oauth2.Config, string) (*oauth2.Token, error) {
			return nil, errors.New("exchange failed")
		}
		t.Cleanup(func() { idpOAuth2Exchange = orig })

		svc := &federationService{idpRepo: &mockIdentityProviderRepo{
			findByIdentifierFn: func(string) (*IdentityProvider, error) { return provider(), nil },
		}}

		_, err := svc.ExchangeOAuth2Code(context.Background(), req)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to exchange")
	})

	t.Run("userinfo fetch error", func(t *testing.T) {
		origExchange := idpOAuth2Exchange
		origUserinfo := idpOAuth2GetUserinfo
		idpOAuth2Exchange = func(context.Context, *oauth2.Config, string) (*oauth2.Token, error) {
			return &oauth2.Token{AccessToken: "provider-access"}, nil
		}
		idpOAuth2GetUserinfo = func(context.Context, *oauth2.Config, *oauth2.Token, string) (*http.Response, error) {
			return nil, errors.New("userinfo error")
		}
		t.Cleanup(func() {
			idpOAuth2Exchange = origExchange
			idpOAuth2GetUserinfo = origUserinfo
		})

		svc := &federationService{idpRepo: &mockIdentityProviderRepo{
			findByIdentifierFn: func(string) (*IdentityProvider, error) { return provider(), nil },
		}}

		_, err := svc.ExchangeOAuth2Code(context.Background(), req)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to fetch")
	})

	t.Run("userinfo read error", func(t *testing.T) {
		origExchange := idpOAuth2Exchange
		origUserinfo := idpOAuth2GetUserinfo
		idpOAuth2Exchange = func(context.Context, *oauth2.Config, string) (*oauth2.Token, error) {
			return &oauth2.Token{AccessToken: "provider-access"}, nil
		}
		idpOAuth2GetUserinfo = func(context.Context, *oauth2.Config, *oauth2.Token, string) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusOK, Body: readErrorCloser{}}, nil
		}
		t.Cleanup(func() {
			idpOAuth2Exchange = origExchange
			idpOAuth2GetUserinfo = origUserinfo
		})

		svc := &federationService{idpRepo: &mockIdentityProviderRepo{
			findByIdentifierFn: func(string) (*IdentityProvider, error) { return provider(), nil },
		}}

		_, err := svc.ExchangeOAuth2Code(context.Background(), req)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read")
	})

	t.Run("userinfo parse error", func(t *testing.T) {
		stubOAuth2Userinfo(t, `not-json`)
		svc := &federationService{idpRepo: &mockIdentityProviderRepo{
			findByIdentifierFn: func(string) (*IdentityProvider, error) { return provider(), nil },
		}}

		_, err := svc.ExchangeOAuth2Code(context.Background(), req)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse")
	})

	t.Run("missing user identifier", func(t *testing.T) {
		stubOAuth2Userinfo(t, `{"email":"user@example.com"}`)
		svc := &federationService{idpRepo: &mockIdentityProviderRepo{
			findByIdentifierFn: func(string) (*IdentityProvider, error) { return provider(), nil },
		}}

		_, err := svc.ExchangeOAuth2Code(context.Background(), req)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing user identifier")
	})

	t.Run("success provisions user using id fallback", func(t *testing.T) {
		stubOAuth2Userinfo(t, `{"id":"external-id","email":"user@example.com","email_verified":true}`)
		stubFederationTokenHooks(t)
		gdb, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()

		idp := provider()
		setIDPConfigJSON(idp, datatypes.JSON(json.RawMessage(`{"issuer":"https://auth.example.com","client_id":"test-client","client_secret":"secret","allow_jit_provisioning":true}`)))
		svc := &federationService{
			db: gdb,
			idpRepo: &mockIdentityProviderRepo{
				findByIdentifierFn:     func(string) (*IdentityProvider, error) { return idp, nil },
				findSystemByTenantIDFn: systemIDPStub,
			},
			userIdentityRepo: &mockFederationUserIdentityRepo{
				findByUserIDAndIDPIDFn: func(userID int64, idpID int64) (*UserIdentity, error) {
					if idpID == idp.IdentityProviderID {
						return &UserIdentity{UserID: userID, Provider: idp.Provider, Sub: "external-id"}, nil
					}
					return nil, nil
				},
			},
			userRepo: &mockUserRepo{
				createFn: func(user *User) (*User, error) {
					user.UserID = 12
					user.UserUUID = uuid.New()
					return user, nil
				},
			},
			roleRepo: &mockRoleRepo{},
			clientRepo: &mockClientRepo{
				findByClientIDAndIdentityProviderFn: func(string, string) (*Client, error) {
					return federationClient(), nil
				},
			},
		}

		resp, err := svc.ExchangeOAuth2Code(context.Background(), req)
		require.NoError(t, err)
		assert.Equal(t, "access-token", resp.AccessToken)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("client not found", func(t *testing.T) {
		stubOAuth2Userinfo(t, `{"sub":"external-sub"}`)
		gdb, mock := newMockGormDB(t)

		idp := provider()
		setIDPConfigJSON(idp, datatypes.JSON(json.RawMessage(`{"issuer":"https://auth.example.com","client_id":"test-client","client_secret":"secret","allow_jit_provisioning":true}`)))
		svc := &federationService{
			db: gdb,
			idpRepo: &mockIdentityProviderRepo{
				findByIdentifierFn: func(string) (*IdentityProvider, error) { return idp, nil },
			},
			userIdentityRepo: &mockFederationUserIdentityRepo{
				findByUserIDAndProviderFn: func(userID int64, provider string) (*UserIdentity, error) {
					return &UserIdentity{UserID: userID, Provider: provider, Sub: "external-sub"}, nil
				},
			},
			userRepo: &mockUserRepo{
				createFn: func(user *User) (*User, error) {
					user.UserID = 13
					user.UserUUID = uuid.New()
					return user, nil
				},
			},
			roleRepo:   &mockRoleRepo{},
			clientRepo: &mockClientRepo{},
		}

		_, err := svc.ExchangeOAuth2Code(context.Background(), req)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "client not found")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("identity lookup error in transaction", func(t *testing.T) {
		stubOAuth2Userinfo(t, `{"sub":"external-sub"}`)
		gdb, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		idp := provider()
		svc := &federationService{
			db: gdb,
			idpRepo: &mockIdentityProviderRepo{
				findByIdentifierFn: func(string) (*IdentityProvider, error) { return idp, nil },
			},
			userIdentityRepo: &mockFederationUserIdentityRepo{
				findByTenantProviderAndSubFn: func(int64, string, string) (*UserIdentity, error) {
					return nil, errors.New("identity lookup error")
				},
			},
			userRepo: &mockUserRepo{},
			clientRepo: &mockClientRepo{
				findByClientIDAndIdentityProviderFn: func(string, string) (*Client, error) {
					return federationClient(), nil
				},
			},
		}
		_, err := svc.ExchangeOAuth2Code(context.Background(), req)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "identity lookup failed")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("existing identity user lookup error", func(t *testing.T) {
		stubOAuth2Userinfo(t, `{"sub":"external-sub"}`)
		gdb, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		idp := provider()
		svc := &federationService{
			db: gdb,
			idpRepo: &mockIdentityProviderRepo{
				findByIdentifierFn: func(string) (*IdentityProvider, error) { return idp, nil },
			},
			userIdentityRepo: &mockFederationUserIdentityRepo{
				findByTenantProviderAndSubFn: func(int64, string, string) (*UserIdentity, error) {
					return &UserIdentity{UserID: 10, Provider: "google", Sub: "external-sub"}, nil
				},
			},
			userRepo: &mockUserRepo{
				findByIDFn: func(any, ...string) (*User, error) { return nil, errors.New("user lookup error") },
			},
			clientRepo: &mockClientRepo{
				findByClientIDAndIdentityProviderFn: func(string, string) (*Client, error) {
					return federationClient(), nil
				},
			},
		}
		_, err := svc.ExchangeOAuth2Code(context.Background(), req)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user lookup failed")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("identity resolution error", func(t *testing.T) {
		stubOAuth2Userinfo(t, `{"sub":"external-sub"}`)
		gdb, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		idp := provider()
		setIDPConfigJSON(idp, datatypes.JSON(json.RawMessage(`{"issuer":"https://auth.example.com","client_id":"test-client","client_secret":"secret","allow_jit_provisioning":true}`)))
		svc := &federationService{
			db: gdb,
			idpRepo: &mockIdentityProviderRepo{
				findByIdentifierFn:     func(string) (*IdentityProvider, error) { return idp, nil },
				findSystemByTenantIDFn: systemIDPStub,
			},
			userIdentityRepo: &mockFederationUserIdentityRepo{
				findByUserIDAndIDPIDFn: func(int64, int64) (*UserIdentity, error) {
					return nil, errors.New("identity resolution error")
				},
			},
			userRepo: &mockUserRepo{
				createFn: func(user *User) (*User, error) {
					user.UserID = 14
					user.UserUUID = uuid.New()
					return user, nil
				},
			},
			roleRepo: &mockRoleRepo{},
			clientRepo: &mockClientRepo{
				findByClientIDAndIdentityProviderFn: func(string, string) (*Client, error) {
					return federationClient(), nil
				},
			},
		}
		_, err := svc.ExchangeOAuth2Code(context.Background(), req)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "default identity lookup failed")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("existing identity refreshes metadata", func(t *testing.T) {
		stubOAuth2Userinfo(t, `{"sub":"external-sub","email":"user@example.com"}`)
		stubFederationTokenHooks(t)
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE "user_identities" SET "metadata"=.*"updated_at"=.*WHERE user_identity_id = .*`).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()
		idp := provider()
		svc := &federationService{
			db: gdb,
			idpRepo: &mockIdentityProviderRepo{
				findByIdentifierFn: func(string) (*IdentityProvider, error) { return idp, nil },
			},
			userIdentityRepo: &mockFederationUserIdentityRepo{
				findByTenantProviderAndSubFn: func(int64, string, string) (*UserIdentity, error) {
					return &UserIdentity{UserIdentityID: 1, UserID: 40, Provider: "google", Sub: "external-sub"}, nil
				},
				findByUserIDAndIDPIDFn: func(userID int64, idpID int64) (*UserIdentity, error) {
					return &UserIdentity{UserID: userID, Sub: "external-sub"}, nil
				},
			},
			userRepo: &mockUserRepo{
				findByIDFn: func(any, ...string) (*User, error) {
					return &User{UserID: 40, UserUUID: uuid.New(), Email: "user@example.com"}, nil
				},
			},
			clientRepo: &mockClientRepo{
				findByClientIDAndIdentityProviderFn: func(string, string) (*Client, error) {
					return federationClient(), nil
				},
			},
		}
		resp, err := svc.ExchangeOAuth2Code(context.Background(), req)
		require.NoError(t, err)
		assert.Equal(t, "access-token", resp.AccessToken)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("provision error is returned", func(t *testing.T) {
		stubOAuth2Userinfo(t, `{"sub":"external-sub","email":"user@example.com"}`)
		gdb, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		idp := provider()
		setIDPConfigJSON(idp, datatypes.JSON(json.RawMessage(`{"issuer":"https://auth.example.com","client_id":"test-client","client_secret":"secret","allow_jit_provisioning":true}`)))
		svc := &federationService{
			db: gdb,
			idpRepo: &mockIdentityProviderRepo{
				findByIdentifierFn: func(string) (*IdentityProvider, error) { return idp, nil },
			},
			userIdentityRepo: &mockFederationUserIdentityRepo{},
			userRepo: &mockUserRepo{
				createFn: func(*User) (*User, error) { return nil, errors.New("create user error") },
			},
			roleRepo: &mockRoleRepo{},
			clientRepo: &mockClientRepo{
				findByClientIDAndIdentityProviderFn: func(string, string) (*Client, error) {
					return federationClient(), nil
				},
			},
		}
		_, err := svc.ExchangeOAuth2Code(context.Background(), req)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to provision user")
		assert.NoError(t, mock.ExpectationsWereMet())
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
		setIDPConfigJSON(idp, datatypes.JSON(json.RawMessage(`{"client_id":"c"}`)))
		idpRepo := &mockIdentityProviderRepo{
			findByIdentifierFn: func(string) (*IdentityProvider, error) { return idp, nil },
		}
		svc := &federationService{idpRepo: idpRepo}
		_, err := svc.LinkIdentity(context.Background(), 1, req)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not configured for OIDC")
	})
}

func TestFederationService_LinkIdentity_Branches(t *testing.T) {
	req := LinkIdentityRequestDTO{ProviderIdentifier: "idp-1", ExternalToken: "tok"}

	t.Run("OIDC validation error", func(t *testing.T) {
		orig := idpValidateOIDCToken
		idpValidateOIDCToken = func(*federationService, context.Context, string, string, string) (map[string]interface{}, error) {
			return nil, errors.New("bad token")
		}
		t.Cleanup(func() { idpValidateOIDCToken = orig })

		idp := activeOIDCProvider("idp-1")
		svc := &federationService{idpRepo: &mockIdentityProviderRepo{
			findByIdentifierFn: func(string) (*IdentityProvider, error) { return idp, nil },
		}}

		_, err := svc.LinkIdentity(context.Background(), 1, req)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "external token validation failed")
	})

	t.Run("missing sub claim", func(t *testing.T) {
		stubOIDCClaims(t, map[string]interface{}{"email": "user@example.com"})
		idp := activeOIDCProvider("idp-1")
		svc := &federationService{idpRepo: &mockIdentityProviderRepo{
			findByIdentifierFn: func(string) (*IdentityProvider, error) { return idp, nil },
		}}

		_, err := svc.LinkIdentity(context.Background(), 1, req)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing 'sub'")
	})

	t.Run("identity lookup error", func(t *testing.T) {
		stubOIDCClaims(t, map[string]interface{}{"sub": "external-sub"})
		idp := activeOIDCProvider("idp-1")
		svc := &federationService{
			idpRepo: &mockIdentityProviderRepo{
				findByIdentifierFn: func(string) (*IdentityProvider, error) { return idp, nil },
			},
			userIdentityRepo: &mockFederationUserIdentityRepo{
				findByTenantProviderAndSubFn: func(int64, string, string) (*UserIdentity, error) {
					return nil, errors.New("lookup error")
				},
			},
		}

		_, err := svc.LinkIdentity(context.Background(), 1, req)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "lookup error")
	})

	t.Run("already linked to another user", func(t *testing.T) {
		stubOIDCClaims(t, map[string]interface{}{"sub": "external-sub"})
		idp := activeOIDCProvider("idp-1")
		svc := &federationService{
			idpRepo: &mockIdentityProviderRepo{
				findByIdentifierFn: func(string) (*IdentityProvider, error) { return idp, nil },
			},
			userIdentityRepo: &mockFederationUserIdentityRepo{
				findByTenantProviderAndSubFn: func(int64, string, string) (*UserIdentity, error) {
					return &UserIdentity{UserID: 99, Provider: "google", Sub: "external-sub"}, nil
				},
			},
		}

		_, err := svc.LinkIdentity(context.Background(), 1, req)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "different account")
	})

	t.Run("already linked to same user is idempotent", func(t *testing.T) {
		stubOIDCClaims(t, map[string]interface{}{"sub": "external-sub"})
		idp := activeOIDCProvider("idp-1")
		svc := &federationService{
			idpRepo: &mockIdentityProviderRepo{
				findByIdentifierFn: func(string) (*IdentityProvider, error) { return idp, nil },
			},
			userIdentityRepo: &mockFederationUserIdentityRepo{
				findByTenantProviderAndSubFn: func(int64, string, string) (*UserIdentity, error) {
					return &UserIdentity{UserID: 1, Provider: "google", Sub: "external-sub"}, nil
				},
			},
		}

		dto, err := svc.LinkIdentity(context.Background(), 1, req)
		require.NoError(t, err)
		require.NotNil(t, dto)
		assert.Equal(t, "google", dto.Provider)
	})

	t.Run("create error", func(t *testing.T) {
		stubOIDCClaims(t, map[string]interface{}{"sub": "external-sub"})
		gdb, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		idp := activeOIDCProvider("idp-1")
		svc := &federationService{
			db: gdb,
			idpRepo: &mockIdentityProviderRepo{
				findByIdentifierFn: func(string) (*IdentityProvider, error) { return idp, nil },
			},
			userIdentityRepo: &mockFederationUserIdentityRepo{
				createFn: func(*UserIdentity) (*UserIdentity, error) {
					return nil, errors.New("create error")
				},
			},
		}

		_, err := svc.LinkIdentity(context.Background(), 1, req)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to link identity")
	})

	t.Run("success creates identity", func(t *testing.T) {
		stubOIDCClaims(t, map[string]interface{}{"sub": "external-sub", "email": "user@example.com"})
		gdb, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		idp := activeOIDCProvider("idp-1")
		var logged bool
		svc := &federationService{
			db: gdb,
			idpRepo: &mockIdentityProviderRepo{
				findByIdentifierFn: func(string) (*IdentityProvider, error) { return idp, nil },
			},
			userIdentityRepo: &mockFederationUserIdentityRepo{
				createFn: func(identity *UserIdentity) (*UserIdentity, error) {
					identity.UserIdentityUUID = uuid.New()
					return identity, nil
				},
			},
			authEventService: &mockAuthEventService{
				logFn: func(context.Context, authevent.AuthEventInput) { logged = true },
			},
		}

		dto, err := svc.LinkIdentity(context.Background(), 1, req)
		require.NoError(t, err)
		assert.Equal(t, "google", dto.Provider)
		assert.True(t, logged)
	})
}

func TestFederationService_ValidateOIDCToken(t *testing.T) {
	t.Run("discovery error", func(t *testing.T) {
		svc := &federationService{}
		claims, err := svc.validateOIDCToken(context.Background(), "http://127.0.0.1:1", "client", "token")
		require.Error(t, err)
		assert.Nil(t, claims)
		assert.Contains(t, err.Error(), "OIDC discovery failed")
	})

	t.Run("missing client id", func(t *testing.T) {
		var issuer string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/.well-known/openid-configuration" {
				_, _ = w.Write([]byte(`{"issuer":"` + issuer + `","jwks_uri":"` + issuer + `/keys"}`))
				return
			}
			http.NotFound(w, r)
		}))
		defer server.Close()
		issuer = server.URL
		origClientFactory := idpHTTPClientFactory
		idpHTTPClientFactory = server.Client
		t.Cleanup(func() { idpHTTPClientFactory = origClientFactory })

		svc := &federationService{}
		claims, err := svc.validateOIDCToken(context.Background(), issuer, "", "token")
		require.Error(t, err)
		assert.Nil(t, claims)
		assert.Contains(t, err.Error(), "client_id is required")
	})

	t.Run("verification error", func(t *testing.T) {
		var issuer string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/.well-known/openid-configuration":
				_, _ = w.Write([]byte(`{"issuer":"` + issuer + `","jwks_uri":"` + issuer + `/keys"}`))
			case "/keys":
				_, _ = w.Write([]byte(`{"keys":[]}`))
			default:
				http.NotFound(w, r)
			}
		}))
		defer server.Close()
		issuer = server.URL
		origClientFactory := idpHTTPClientFactory
		idpHTTPClientFactory = server.Client
		t.Cleanup(func() { idpHTTPClientFactory = origClientFactory })

		svc := &federationService{}
		claims, err := svc.validateOIDCToken(context.Background(), issuer, "client", "bad-token")
		require.Error(t, err)
		assert.Nil(t, claims)
		assert.Contains(t, err.Error(), "token verification failed")
	})

	t.Run("success", func(t *testing.T) {
		key, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)
		kid := "key-1"
		var issuer string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/.well-known/openid-configuration":
				_, _ = w.Write([]byte(`{"issuer":"` + issuer + `","jwks_uri":"` + issuer + `/keys"}`))
			case "/keys":
				_, _ = w.Write([]byte(`{"keys":[` + rsaPublicJWK(key, kid) + `]}`))
			default:
				http.NotFound(w, r)
			}
		}))
		defer server.Close()
		issuer = server.URL
		origClientFactory := idpHTTPClientFactory
		idpHTTPClientFactory = server.Client
		t.Cleanup(func() { idpHTTPClientFactory = origClientFactory })

		token := jwtlib.NewWithClaims(jwtlib.SigningMethodRS256, jwtlib.MapClaims{
			"iss": issuer,
			"aud": "client",
			"sub": "external-sub",
			"exp": time.Now().Add(time.Hour).Unix(),
			"iat": time.Now().Unix(),
		})
		token.Header["kid"] = kid
		raw, err := token.SignedString(key)
		require.NoError(t, err)

		svc := &federationService{}
		claims, err := svc.validateOIDCToken(context.Background(), issuer, "client", raw)
		require.NoError(t, err)
		assert.Equal(t, "external-sub", claims["sub"])
	})
}

func rsaPublicJWK(key *rsa.PrivateKey, kid string) string {
	n := base64.RawURLEncoding.EncodeToString(key.N.Bytes())
	e := base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes())
	return `{"kty":"RSA","kid":"` + kid + `","alg":"RS256","use":"sig","n":"` + n + `","e":"` + e + `"}`
}

func TestFederationDefaultHookWrappers(t *testing.T) {
	origExchange := idpOAuth2Exchange
	origUserinfo := idpOAuth2GetUserinfo
	defer func() {
		idpOAuth2Exchange = origExchange
		idpOAuth2GetUserinfo = origUserinfo
	}()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	cfg := &oauth2.Config{Endpoint: oauth2.Endpoint{TokenURL: "https://example.com/token"}}
	_, err := origExchange(ctx, cfg, "code")
	require.Error(t, err)

	userinfoServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{}`))
	}))
	defer userinfoServer.Close()

	resp, _ := origUserinfo(context.Background(), cfg, &oauth2.Token{AccessToken: "token"}, userinfoServer.URL)
	if resp != nil && resp.Body != nil {
		_ = resp.Body.Close()
	}
}

func TestFederationService_RefreshMetadata(t *testing.T) {
	gdb, mock := newMockGormDBRegex(t)
	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE "user_identities" SET "metadata"=.*"updated_at"=.*WHERE user_identity_id = .*`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	svc := &federationService{}
	err := gdb.Transaction(func(tx *gorm.DB) error {
		return svc.refreshMetadata(tx, &UserIdentity{UserIdentityID: 1}, IdentityMetadata{Email: "user@example.com"})
	})
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
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
			userIdentityRepo: identityRepo,
			authEventService: &mockAuthEventService{},
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
			userIdentityRepo: identityRepo,
			authEventService: &mockAuthEventService{},
		}
		err := svc.UnlinkIdentity(context.Background(), userID, identUUID.String())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "identity not found")
	})

	t.Run("built-in identity cannot be unlinked", func(t *testing.T) {
		defaultIdent := UserIdentity{
			UserIdentityID:   10,
			UserIdentityUUID: identUUID,
			Provider:         shared.ProviderMaintainerd,
			UserID:           userID,
		}
		identityRepo := &mockFederationUserIdentityRepo{
			findByUserIDFn: func(int64) ([]UserIdentity, error) {
				return []UserIdentity{defaultIdent}, nil
			},
		}
		svc := &federationService{
			userIdentityRepo: identityRepo,
			authEventService: &mockAuthEventService{},
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
		gdb, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := &federationService{
			db:               gdb,
			userIdentityRepo: identityRepo,
			authEventService: &mockAuthEventService{},
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
		gdb, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		svc := &federationService{
			db:               gdb,
			userIdentityRepo: identityRepo,
			authEventService: &mockAuthEventService{},
		}
		err := svc.UnlinkIdentity(context.Background(), userID, identUUID.String())
		require.NoError(t, err)
	})

	// An EXTERNAL maintainerd federated identity (provider="maintainerd" but
	// linked to a non-system IdP) can be unlinked — the built-in guard must key
	// off the tenant's system IdP id, not the provider string.
	t.Run("external maintainerd federated identity is unlinkable", func(t *testing.T) {
		extIDPID := int64(2)
		extMaintainerd := UserIdentity{
			UserIdentityID:     10,
			UserIdentityUUID:   identUUID,
			Provider:           shared.ProviderMaintainerd,
			IdentityProviderID: extIDPID,
			TenantID:           1,
			UserID:             userID,
		}
		identityRepo := &mockFederationUserIdentityRepo{
			findByUserIDFn: func(int64) ([]UserIdentity, error) {
				return []UserIdentity{extMaintainerd}, nil
			},
		}
		idpRepo := &mockIdentityProviderRepo{
			findSystemByTenantIDFn: func(int64) (*IdentityProvider, error) {
				return &IdentityProvider{IdentityProviderID: 1, IsSystem: true, ProviderType: shared.IDPTypeSystem}, nil
			},
		}
		gdb, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		svc := &federationService{
			db:               gdb,
			userIdentityRepo: identityRepo,
			idpRepo:          idpRepo,
			authEventService: &mockAuthEventService{},
		}
		err := svc.UnlinkIdentity(context.Background(), userID, identUUID.String())
		require.NoError(t, err)
	})

	// The real built-in system identity (linked to the tenant's system IdP) must
	// never be unlinkable, decided by is_system — not the provider string.
	t.Run("built-in system identity cannot be unlinked (by is_system)", func(t *testing.T) {
		systemIDPID := int64(1)
		builtin := UserIdentity{
			UserIdentityID:     10,
			UserIdentityUUID:   identUUID,
			Provider:           shared.ProviderMaintainerd,
			IdentityProviderID: systemIDPID,
			TenantID:           1,
			UserID:             userID,
		}
		identityRepo := &mockFederationUserIdentityRepo{
			findByUserIDFn: func(int64) ([]UserIdentity, error) {
				return []UserIdentity{builtin}, nil
			},
		}
		idpRepo := &mockIdentityProviderRepo{
			findSystemByTenantIDFn: func(int64) (*IdentityProvider, error) {
				return &IdentityProvider{IdentityProviderID: 1, IsSystem: true, ProviderType: shared.IDPTypeSystem}, nil
			},
		}
		svc := &federationService{
			userIdentityRepo: identityRepo,
			idpRepo:          idpRepo,
			authEventService: &mockAuthEventService{},
		}
		err := svc.UnlinkIdentity(context.Background(), userID, identUUID.String())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "built-in identity")
	})
}

// ---------------------------------------------------------------------------
// Identity disambiguation — a user holding BOTH a built-in maintainerd identity
// and an external (enterprise) maintainerd identity must resolve each distinctly
// by identity_provider_id, never by the shared "maintainerd" provider string.
// ---------------------------------------------------------------------------

func TestFederationService_ResolveExistingUserIdentity_DisambiguatesMaintainerd(t *testing.T) {
	const (
		systemIDPID  = int64(1)
		externalIDID = int64(2)
		theUserID    = int64(50)
	)
	// External IdP: provider slug is "maintainerd" (a peer Maintainerd instance)
	// but it is an enterprise IdP with its own identity_provider_id.
	externalIDP := &IdentityProvider{
		IdentityProviderID: externalIDID,
		TenantID:           1,
		Provider:           shared.ProviderMaintainerd,
		ProviderType:       shared.IDPTypeEnterprise,
	}
	systemIdentity := &UserIdentity{UserID: theUserID, Provider: shared.ProviderMaintainerd, Sub: "system-sub", IdentityProviderID: systemIDPID}
	externalIdentity := &UserIdentity{UserID: theUserID, Provider: shared.ProviderMaintainerd, Sub: "external-sub", IdentityProviderID: externalIDID}

	newSvc := func() *federationService {
		return &federationService{
			userRepo: &mockUserRepo{
				findByIDFn: func(any, ...string) (*User, error) {
					return &User{UserID: theUserID, UserUUID: uuid.New()}, nil
				},
			},
			userIdentityRepo: &mockFederationUserIdentityRepo{
				// Keyed by sub — unambiguous even though both rows share the
				// "maintainerd" provider slug.
				findByTenantProviderAndSubFn: func(_ int64, provider, sub string) (*UserIdentity, error) {
					if provider == shared.ProviderMaintainerd && sub == "external-sub" {
						return externalIdentity, nil
					}
					return nil, nil
				},
				findByUserIDAndIDPIDFn: func(_ int64, idpID int64) (*UserIdentity, error) {
					switch idpID {
					case systemIDPID:
						return systemIdentity, nil
					case externalIDID:
						return externalIdentity, nil
					}
					return nil, nil
				},
			},
			idpRepo: &mockIdentityProviderRepo{
				findSystemByTenantIDFn: func(int64) (*IdentityProvider, error) {
					return &IdentityProvider{IdentityProviderID: systemIDPID, IsSystem: true, ProviderType: shared.IDPTypeSystem}, nil
				},
			},
		}
	}

	t.Run("useSystemIdentity resolves the built-in identity by system IdP id", func(t *testing.T) {
		user, sub, err := newSvc().resolveExistingUserIdentity(externalIDP, "external-sub", true)
		require.NoError(t, err)
		require.NotNil(t, user)
		assert.Equal(t, "system-sub", sub)
	})

	t.Run("without useSystemIdentity resolves the external identity's own sub", func(t *testing.T) {
		user, sub, err := newSvc().resolveExistingUserIdentity(externalIDP, "external-sub", false)
		require.NoError(t, err)
		require.NotNil(t, user)
		assert.Equal(t, "external-sub", sub)
	})

	t.Run("built-in system identity is not the external one", func(t *testing.T) {
		// Sanity: the two identities are distinct rows differing only by
		// identity_provider_id / sub — resolving each id yields a different sub.
		svc := newSvc()
		sys, err := svc.userIdentityRepo.FindByUserIDAndIdentityProviderID(theUserID, systemIDPID)
		require.NoError(t, err)
		ext, err := svc.userIdentityRepo.FindByUserIDAndIdentityProviderID(theUserID, externalIDID)
		require.NoError(t, err)
		assert.NotEqual(t, sys.Sub, ext.Sub)
	})
}

// ---------------------------------------------------------------------------
// AdminUnlinkIdentity
// ---------------------------------------------------------------------------

func TestFederationService_AdminUnlinkIdentity(t *testing.T) {
	tenantID := int64(1)
	actorUserID := int64(42)
	targetUserID := int64(7)
	targetUserUUID := uuid.New()
	identUUID := uuid.New()

	t.Run("user not found", func(t *testing.T) {
		userRepo := &mockUserRepo{
			findByUUIDFn: func(any, ...string) (*User, error) { return nil, nil },
		}
		svc := &federationService{
			userRepo:         userRepo,
			userIdentityRepo: &mockFederationUserIdentityRepo{},
			authEventService: &mockAuthEventService{},
		}
		err := svc.AdminUnlinkIdentity(context.Background(), tenantID, actorUserID, targetUserUUID, identUUID.String())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")
	})

	t.Run("user lookup error", func(t *testing.T) {
		userRepo := &mockUserRepo{
			findByUUIDFn: func(any, ...string) (*User, error) { return nil, errors.New("db error") },
		}
		svc := &federationService{
			userRepo:         userRepo,
			userIdentityRepo: &mockFederationUserIdentityRepo{},
			authEventService: &mockAuthEventService{},
		}
		err := svc.AdminUnlinkIdentity(context.Background(), tenantID, actorUserID, targetUserUUID, identUUID.String())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user lookup failed")
	})

	t.Run("cross-tenant target returns not found", func(t *testing.T) {
		userRepo := &mockUserRepo{
			findByUUIDFn: func(any, ...string) (*User, error) {
				return &User{UserID: targetUserID, UserUUID: targetUserUUID, TenantID: 999}, nil
			},
		}
		svc := &federationService{
			userRepo:         userRepo,
			userIdentityRepo: &mockFederationUserIdentityRepo{},
			authEventService: &mockAuthEventService{},
		}
		err := svc.AdminUnlinkIdentity(context.Background(), tenantID, actorUserID, targetUserUUID, identUUID.String())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")
	})

	t.Run("FindByUserID error", func(t *testing.T) {
		userRepo := &mockUserRepo{
			findByUUIDFn: func(any, ...string) (*User, error) {
				return &User{UserID: targetUserID, UserUUID: targetUserUUID, TenantID: tenantID}, nil
			},
		}
		identityRepo := &mockFederationUserIdentityRepo{
			findByUserIDFn: func(int64) ([]UserIdentity, error) { return nil, errors.New("db error") },
		}
		svc := &federationService{
			userRepo:         userRepo,
			userIdentityRepo: identityRepo,
			authEventService: &mockAuthEventService{},
		}
		err := svc.AdminUnlinkIdentity(context.Background(), tenantID, actorUserID, targetUserUUID, identUUID.String())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "identity lookup failed")
	})

	t.Run("identity not found", func(t *testing.T) {
		userRepo := &mockUserRepo{
			findByUUIDFn: func(any, ...string) (*User, error) {
				return &User{UserID: targetUserID, UserUUID: targetUserUUID, TenantID: tenantID}, nil
			},
		}
		identityRepo := &mockFederationUserIdentityRepo{
			findByUserIDFn: func(int64) ([]UserIdentity, error) {
				return []UserIdentity{{UserIdentityUUID: uuid.New(), TenantID: tenantID}}, nil
			},
		}
		svc := &federationService{
			userRepo:         userRepo,
			userIdentityRepo: identityRepo,
			authEventService: &mockAuthEventService{},
		}
		err := svc.AdminUnlinkIdentity(context.Background(), tenantID, actorUserID, targetUserUUID, identUUID.String())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "identity not found")
	})

	t.Run("identity belongs to another tenant returns not found", func(t *testing.T) {
		userRepo := &mockUserRepo{
			findByUUIDFn: func(any, ...string) (*User, error) {
				return &User{UserID: targetUserID, UserUUID: targetUserUUID, TenantID: tenantID}, nil
			},
		}
		identityRepo := &mockFederationUserIdentityRepo{
			findByUserIDFn: func(int64) ([]UserIdentity, error) {
				return []UserIdentity{{
					UserIdentityID:   10,
					UserIdentityUUID: identUUID,
					Provider:         "google",
					UserID:           targetUserID,
					TenantID:         999,
				}}, nil
			},
		}
		svc := &federationService{
			userRepo:         userRepo,
			userIdentityRepo: identityRepo,
			authEventService: &mockAuthEventService{},
		}
		err := svc.AdminUnlinkIdentity(context.Background(), tenantID, actorUserID, targetUserUUID, identUUID.String())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "identity not found")
	})

	t.Run("built-in identity cannot be unlinked", func(t *testing.T) {
		userRepo := &mockUserRepo{
			findByUUIDFn: func(any, ...string) (*User, error) {
				return &User{UserID: targetUserID, UserUUID: targetUserUUID, TenantID: tenantID}, nil
			},
		}
		identityRepo := &mockFederationUserIdentityRepo{
			findByUserIDFn: func(int64) ([]UserIdentity, error) {
				return []UserIdentity{{
					UserIdentityID:   10,
					UserIdentityUUID: identUUID,
					Provider:         shared.ProviderMaintainerd,
					UserID:           targetUserID,
					TenantID:         tenantID,
				}}, nil
			},
		}
		svc := &federationService{
			userRepo:         userRepo,
			userIdentityRepo: identityRepo,
			authEventService: &mockAuthEventService{},
		}
		err := svc.AdminUnlinkIdentity(context.Background(), tenantID, actorUserID, targetUserUUID, identUUID.String())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "built-in identity")
	})

	t.Run("delete fails rolls back", func(t *testing.T) {
		userRepo := &mockUserRepo{
			findByUUIDFn: func(any, ...string) (*User, error) {
				return &User{UserID: targetUserID, UserUUID: targetUserUUID, TenantID: tenantID}, nil
			},
		}
		identityRepo := &mockFederationUserIdentityRepo{
			findByUserIDFn: func(int64) ([]UserIdentity, error) {
				return []UserIdentity{{
					UserIdentityID:   10,
					UserIdentityUUID: identUUID,
					Provider:         "google",
					UserID:           targetUserID,
					TenantID:         tenantID,
				}}, nil
			},
			deleteByIDFn: func(any) error { return errors.New("del err") },
		}
		gdb, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := &federationService{
			db:               gdb,
			userRepo:         userRepo,
			userIdentityRepo: identityRepo,
			authEventService: &mockAuthEventService{},
		}
		err := svc.AdminUnlinkIdentity(context.Background(), tenantID, actorUserID, targetUserUUID, identUUID.String())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to unlink")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("success logs admin actor", func(t *testing.T) {
		userRepo := &mockUserRepo{
			findByUUIDFn: func(any, ...string) (*User, error) {
				return &User{UserID: targetUserID, UserUUID: targetUserUUID, TenantID: tenantID}, nil
			},
		}
		var deletedID any
		identityRepo := &mockFederationUserIdentityRepo{
			findByUserIDFn: func(int64) ([]UserIdentity, error) {
				return []UserIdentity{{
					UserIdentityID:   10,
					UserIdentityUUID: identUUID,
					Provider:         "google",
					UserID:           targetUserID,
					TenantID:         tenantID,
				}}, nil
			},
			deleteByIDFn: func(id any) error { deletedID = id; return nil },
		}
		var logged authevent.AuthEventInput
		authSvc := &mockAuthEventService{logFn: func(_ context.Context, in authevent.AuthEventInput) { logged = in }}
		gdb, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		svc := &federationService{
			db:               gdb,
			userRepo:         userRepo,
			userIdentityRepo: identityRepo,
			authEventService: authSvc,
		}
		err := svc.AdminUnlinkIdentity(context.Background(), tenantID, actorUserID, targetUserUUID, identUUID.String())
		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
		assert.Equal(t, int64(10), deletedID)
		require.NotNil(t, logged.ActorUserID)
		assert.Equal(t, actorUserID, *logged.ActorUserID)
		require.NotNil(t, logged.Description)
		assert.Contains(t, *logged.Description, targetUserUUID.String())
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
						Provider:         shared.ProviderMaintainerd,
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
		svc := &federationService{idpRepo: &mockIdentityProviderRepo{}, emailDomainRepo: &mockIdentityProviderEmailDomainRepo{}}
		_, err := svc.HomeRealmDiscovery(context.Background(), 1, "bad")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid email")
	})

	t.Run("email domain lookup error", func(t *testing.T) {
		svc := &federationService{
			idpRepo: &mockIdentityProviderRepo{},
			emailDomainRepo: &mockIdentityProviderEmailDomainRepo{
				findByTenantAndDomainFn: func(int64, string) (*IdentityProviderEmailDomain, error) {
					return nil, errors.New("db error")
				},
			},
		}
		_, err := svc.HomeRealmDiscovery(context.Background(), 1, "user@example.com")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "provider lookup failed")
	})

	t.Run("matching domain returns provider", func(t *testing.T) {
		svc := &federationService{
			idpRepo: &mockIdentityProviderRepo{
				findByIDFn: func(any, ...string) (*IdentityProvider, error) {
					return &IdentityProvider{Identifier: "idp-1", Provider: "google", DisplayName: "Google"}, nil
				},
			},
			emailDomainRepo: &mockIdentityProviderEmailDomainRepo{
				findByTenantAndDomainFn: func(_ int64, domain string) (*IdentityProviderEmailDomain, error) {
					assert.Equal(t, "example.com", domain)
					return &IdentityProviderEmailDomain{IdentityProviderID: 7}, nil
				},
			},
		}
		res, err := svc.HomeRealmDiscovery(context.Background(), 1, "user@example.com")
		require.NoError(t, err)
		assert.Equal(t, "idp-1", res.ProviderIdentifier)
	})

	t.Run("no matching domain falls back to default", func(t *testing.T) {
		svc := &federationService{
			idpRepo: &mockIdentityProviderRepo{
				findDefaultByTenantIDFn: func(int64) (*IdentityProvider, error) {
					return &IdentityProvider{Identifier: "default-idp", Provider: "maintainerd", DisplayName: "Maintainerd"}, nil
				},
			},
			emailDomainRepo: &mockIdentityProviderEmailDomainRepo{
				findByTenantAndDomainFn: func(int64, string) (*IdentityProviderEmailDomain, error) { return nil, nil },
			},
		}
		res, err := svc.HomeRealmDiscovery(context.Background(), 1, "user@example.com")
		require.NoError(t, err)
		assert.Equal(t, "default-idp", res.ProviderIdentifier)
	})

	t.Run("no default IDP returns error", func(t *testing.T) {
		svc := &federationService{
			idpRepo: &mockIdentityProviderRepo{
				findDefaultByTenantIDFn: func(int64) (*IdentityProvider, error) { return nil, nil },
			},
			emailDomainRepo: &mockIdentityProviderEmailDomainRepo{
				findByTenantAndDomainFn: func(int64, string) (*IdentityProviderEmailDomain, error) { return nil, nil },
			},
		}
		_, err := svc.HomeRealmDiscovery(context.Background(), 1, "user@example.com")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no identity provider found")
	})

	t.Run("domain matched but provider missing falls back to default", func(t *testing.T) {
		svc := &federationService{
			idpRepo: &mockIdentityProviderRepo{
				findByIDFn: func(any, ...string) (*IdentityProvider, error) { return nil, nil },
				findDefaultByTenantIDFn: func(int64) (*IdentityProvider, error) {
					return &IdentityProvider{Identifier: "default-idp", Provider: "maintainerd", DisplayName: "Maintainerd"}, nil
				},
			},
			emailDomainRepo: &mockIdentityProviderEmailDomainRepo{
				findByTenantAndDomainFn: func(int64, string) (*IdentityProviderEmailDomain, error) {
					return &IdentityProviderEmailDomain{IdentityProviderID: 7}, nil
				},
			},
		}
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
	}, int64Ptr(10))

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
	}, "external-sub", "user@example.com", IdentityMetadata{}, int64Ptr(10))

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
	}, "external-sub", "user@example.com", IdentityMetadata{}, int64Ptr(10))

	require.Error(t, err)
	assert.Nil(t, user)
	assert.False(t, isNew)
}

func TestFederationServiceProvisionUser_WithDefaultRole(t *testing.T) {
	gormDB, mock := newMockGormDBRegex(t)
	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO "user_roles"`).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

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
			findSystemByTenantIDFn: func(tenantID int64) (*IdentityProvider, error) {
				return &IdentityProvider{
					IdentityProviderID: int64(99),
					Identifier:         "system-idp",
					IsSystem:           true,
					ProviderType:       shared.IDPTypeSystem,
				}, nil
			},
		},
		roleRepo: roleRepo,
	}

	user, isNew, err := svc.provisionUser(context.Background(), gormDB, &IdentityProvider{
		IdentityProviderID: 10,
		TenantID:           20,
		Provider:           "google",
	}, "external-sub", "user@example.com", IdentityMetadata{}, int64Ptr(10))

	require.NoError(t, err)
	require.NotNil(t, user)
	assert.True(t, isNew)
	assert.Equal(t, int64(400), user.UserID)
	assert.NoError(t, mock.ExpectationsWereMet())
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
	assert.EqualValues(t, int64(15*60), resp.ExpiresIn)
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

func TestFederationService_GenerateTokens_IDTokenError(t *testing.T) {
	initTestJWTKeysService(t)
	orig := idpGenerateIDTokenWithContext
	idpGenerateIDTokenWithContext = func(context.Context, string, string, string, string, *jwt.UserProfile, string, *jwt.IDTokenParams) (string, error) {
		return "", errors.New("id token error")
	}
	t.Cleanup(func() { idpGenerateIDTokenWithContext = orig })

	user := &User{UserID: 1, UserUUID: uuid.New(), Email: "test@example.com"}
	domain := "example.com"
	identifier := "app"
	client := &Client{
		Domain:     &domain,
		Identifier: &identifier,
		IdentityProvider: &IdentityProvider{
			Identifier: "default-idp",
		},
	}
	svc := &federationService{}

	resp, err := svc.generateTokens(context.Background(), "sub-1", user, client)
	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "id token generation failed")
}

func TestFederationService_GenerateTokens_RefreshTokenError(t *testing.T) {
	initTestJWTKeysService(t)
	orig := idpGenerateRefreshTokenWithContext
	idpGenerateRefreshTokenWithContext = func(context.Context, string, string, string, string) (string, error) {
		return "", errors.New("refresh token error")
	}
	t.Cleanup(func() { idpGenerateRefreshTokenWithContext = orig })

	user := &User{UserID: 1, UserUUID: uuid.New(), Email: "test@example.com"}
	domain := "example.com"
	identifier := "app"
	client := &Client{
		Domain:     &domain,
		Identifier: &identifier,
		IdentityProvider: &IdentityProvider{
			Identifier: "default-idp",
		},
	}
	svc := &federationService{}

	resp, err := svc.generateTokens(context.Background(), "sub-1", user, client)
	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "refresh token generation failed")
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
		Provider:         shared.ProviderMaintainerd,
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

// F3: handleEmailCollision surfaces the collision (never silently merges) and
// does not panic when no account-link service is wired.
func TestFederationService_HandleEmailCollision(t *testing.T) {
	t.Run("collision with nil account-link service surfaces conflict", func(t *testing.T) {
		svc := &federationService{}
		err := svc.handleEmailCollision(context.Background(), &errEmailCollision{
			tenantID:       1,
			existingUserID: 2,
			providerName:   "google",
			providerEmail:  "owner@example.com",
		})
		require.Error(t, err)
		var conflict *apperror.ConflictError
		require.ErrorAs(t, err, &conflict)
	})

	t.Run("non-collision error returns nil (handled by caller)", func(t *testing.T) {
		svc := &federationService{}
		assert.NoError(t, svc.handleEmailCollision(context.Background(), errors.New("some other error")))
	})

	t.Run("nil error returns nil", func(t *testing.T) {
		svc := &federationService{}
		assert.NoError(t, svc.handleEmailCollision(context.Background(), nil))
	})
}
