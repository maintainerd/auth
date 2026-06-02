package oauth

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/maintainerd/auth/internal/platform/cache"
	"github.com/maintainerd/auth/internal/platform/crypto"
	"github.com/maintainerd/auth/internal/platform/jwt"
	"github.com/maintainerd/auth/internal/platform/security"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// ── helpers ─────────────────────────────────────────────────────────────────

func newOAuthTokenSvc(
	db *gorm.DB,
	clientRepo *mockClientRepo,
	authCodeRepo *mockOAuthAuthCodeRepo,
	refreshTokenRepo *mockOAuthRefreshTokenRepo,
	userRepo *mockUserRepo,
	userIdentityRepo *mockUserIdentityRepo,
	authEventSvc *mockAuthEventService,
) OAuthTokenService {
	return NewOAuthTokenService(db, clientRepo, authCodeRepo, refreshTokenRepo, userRepo, userIdentityRepo, authEventSvc, cache.NopJTIDenylister{})
}

type recordingJTIDenylister struct {
	jti string
	ttl time.Duration
	err error
}

func (r *recordingJTIDenylister) DenyJTI(_ context.Context, jti string, ttl time.Duration) error {
	r.jti = jti
	r.ttl = ttl
	return r.err
}

func (r *recordingJTIDenylister) IsJTIDenied(_ context.Context, _ string) (bool, error) {
	return false, nil
}

func mockClientRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"client_id", "client_uuid", "tenant_id", "identity_provider_id", "name", "display_name",
		"client_type", "domain", "identifier", "secret", "status",
		"is_default", "is_system", "token_endpoint_auth_method",
		"grant_types", "response_types", "access_token_ttl", "refresh_token_ttl",
		"require_consent", "created_at", "updated_at",
	}).AddRow(
		10, uuid.New(), 1, int64(100), "test-client", "Test Client",
		"spa", "https://auth.example.com", "my-client", nil, "active",
		false, false, "none",
		`{authorization_code,refresh_token}`, `{code}`, nil, nil,
		true, time.Now(), time.Now(),
	)
}

// mockIDPRows returns sqlmock rows for the IdentityProvider preload.
func mockIDPRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"identity_provider_id", "identity_provider_uuid", "tenant_id",
		"name", "display_name", "provider", "provider_type",
		"identifier", "config", "status", "is_default", "is_system",
		"created_at", "updated_at",
	}).AddRow(
		100, uuid.New(), 1,
		"default", "Default Provider", "local", "local",
		"default-provider", `{}`, "active", true, false,
		time.Now(), time.Now(),
	)
}

// expectClientLookup sets up sqlmock expectations for findActiveClientByIdentifier.
// Matches the main query + Preload("IdentityProvider") + Preload("IdentityProvider.Tenant").
func expectClientLookup(mock sqlmock.Sqlmock, rows *sqlmock.Rows) {
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).WillReturnRows(rows)
	// Preload IdentityProvider — returns a provider row.
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).WillReturnRows(mockIDPRows())
	// Preload IdentityProvider.Tenant (nested) — return empty since not needed.
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).WillReturnRows(sqlmock.NewRows(nil))
}

func expectClientNotFound(mock sqlmock.Sqlmock) {
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).WillReturnError(gorm.ErrRecordNotFound)
}

// ── TestOAuthTokenService_Exchange ──────────────────────────────────────────

func TestOAuthTokenService_Exchange(t *testing.T) {
	ctx := context.Background()

	t.Run("unsupported grant type", func(t *testing.T) {
		db, _ := newMockDB(t)
		svc := newOAuthTokenSvc(db, &mockClientRepo{}, &mockOAuthAuthCodeRepo{}, &mockOAuthRefreshTokenRepo{}, &mockUserRepo{}, &mockUserIdentityRepo{findByUserIDAndClientIDFn: func(_, _ int64) (*UserIdentity, error) { return nil, nil }}, &mockAuthEventService{})

		_, oerr := svc.Exchange(ctx, OAuthTokenRequestDTO{GrantType: "implicit"}, OAuthClientCredentials{})
		require.NotNil(t, oerr)
		assert.Equal(t, "unsupported_grant_type", oerr.Code)
	})

	t.Run("authorization_code — missing code", func(t *testing.T) {
		db, _ := newMockDB(t)
		svc := newOAuthTokenSvc(db, &mockClientRepo{}, &mockOAuthAuthCodeRepo{}, &mockOAuthRefreshTokenRepo{}, &mockUserRepo{}, &mockUserIdentityRepo{findByUserIDAndClientIDFn: func(_, _ int64) (*UserIdentity, error) { return nil, nil }}, &mockAuthEventService{})

		_, oerr := svc.Exchange(ctx, OAuthTokenRequestDTO{
			GrantType:    "authorization_code",
			RedirectURI:  "https://example.com/callback",
			CodeVerifier: "abc",
		}, OAuthClientCredentials{ClientID: "my-client"})
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_request", oerr.Code)
		assert.Contains(t, oerr.Description, "code is required")
	})

	t.Run("authorization_code — missing redirect_uri", func(t *testing.T) {
		db, _ := newMockDB(t)
		svc := newOAuthTokenSvc(db, &mockClientRepo{}, &mockOAuthAuthCodeRepo{}, &mockOAuthRefreshTokenRepo{}, &mockUserRepo{}, &mockUserIdentityRepo{findByUserIDAndClientIDFn: func(_, _ int64) (*UserIdentity, error) { return nil, nil }}, &mockAuthEventService{})

		_, oerr := svc.Exchange(ctx, OAuthTokenRequestDTO{
			GrantType:    "authorization_code",
			Code:         "code123",
			CodeVerifier: "abc",
		}, OAuthClientCredentials{ClientID: "my-client"})
		require.NotNil(t, oerr)
		assert.Contains(t, oerr.Description, "redirect_uri is required")
	})

	t.Run("authorization_code — missing code_verifier", func(t *testing.T) {
		db, _ := newMockDB(t)
		svc := newOAuthTokenSvc(db, &mockClientRepo{}, &mockOAuthAuthCodeRepo{}, &mockOAuthRefreshTokenRepo{}, &mockUserRepo{}, &mockUserIdentityRepo{findByUserIDAndClientIDFn: func(_, _ int64) (*UserIdentity, error) { return nil, nil }}, &mockAuthEventService{})

		_, oerr := svc.Exchange(ctx, OAuthTokenRequestDTO{
			GrantType:   "authorization_code",
			Code:        "code123",
			RedirectURI: "https://example.com/callback",
		}, OAuthClientCredentials{ClientID: "my-client"})
		require.NotNil(t, oerr)
		assert.Contains(t, oerr.Description, "code_verifier is required")
	})

	t.Run("authorization_code — client auth missing client_id", func(t *testing.T) {
		db, _ := newMockDB(t)
		svc := newOAuthTokenSvc(db, &mockClientRepo{}, &mockOAuthAuthCodeRepo{}, &mockOAuthRefreshTokenRepo{}, &mockUserRepo{}, &mockUserIdentityRepo{findByUserIDAndClientIDFn: func(_, _ int64) (*UserIdentity, error) { return nil, nil }}, &mockAuthEventService{})

		_, oerr := svc.Exchange(ctx, OAuthTokenRequestDTO{
			GrantType:    "authorization_code",
			Code:         "code123",
			RedirectURI:  "https://example.com/callback",
			CodeVerifier: "abc",
		}, OAuthClientCredentials{})
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_client", oerr.Code)
	})

	t.Run("authorization_code — client not found", func(t *testing.T) {
		db, mock := newMockDB(t)
		expectClientNotFound(mock)

		svc := newOAuthTokenSvc(db, &mockClientRepo{}, &mockOAuthAuthCodeRepo{}, &mockOAuthRefreshTokenRepo{}, &mockUserRepo{}, &mockUserIdentityRepo{findByUserIDAndClientIDFn: func(_, _ int64) (*UserIdentity, error) { return nil, nil }}, &mockAuthEventService{})

		_, oerr := svc.Exchange(ctx, OAuthTokenRequestDTO{
			GrantType:    "authorization_code",
			Code:         "code123",
			RedirectURI:  "https://example.com/callback",
			CodeVerifier: "abc",
		}, OAuthClientCredentials{ClientID: "unknown"})
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_client", oerr.Code)
	})

	t.Run("authorization_code — auth code not found", func(t *testing.T) {
		db, mock := newMockDB(t)
		expectClientLookup(mock, mockClientRows())

		svc := newOAuthTokenSvc(db, &mockClientRepo{},
			&mockOAuthAuthCodeRepo{
				findByCodeHashFn: func(_ string) (*OAuthAuthorizationCode, error) {
					return nil, nil
				},
			},
			&mockOAuthRefreshTokenRepo{}, &mockUserRepo{},
			&mockUserIdentityRepo{findByUserIDAndClientIDFn: func(_, _ int64) (*UserIdentity, error) { return nil, nil }},
			&mockAuthEventService{})

		_, oerr := svc.Exchange(ctx, OAuthTokenRequestDTO{
			GrantType:    "authorization_code",
			Code:         "code123",
			RedirectURI:  "https://example.com/callback",
			CodeVerifier: "abc",
		}, OAuthClientCredentials{ClientID: "my-client"})
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_grant", oerr.Code)
	})

	t.Run("authorization_code — code already used", func(t *testing.T) {
		db, mock := newMockDB(t)
		expectClientLookup(mock, mockClientRows())

		svc := newOAuthTokenSvc(db, &mockClientRepo{},
			&mockOAuthAuthCodeRepo{
				findByCodeHashFn: func(_ string) (*OAuthAuthorizationCode, error) {
					return &OAuthAuthorizationCode{
						IsUsed:   true,
						ClientID: 10,
						UserID:   1,
						TenantID: 1,
					}, nil
				},
			},
			&mockOAuthRefreshTokenRepo{}, &mockUserRepo{},
			&mockUserIdentityRepo{findByUserIDAndClientIDFn: func(_, _ int64) (*UserIdentity, error) { return nil, nil }},
			&mockAuthEventService{})

		_, oerr := svc.Exchange(ctx, OAuthTokenRequestDTO{
			GrantType:    "authorization_code",
			Code:         "code123",
			RedirectURI:  "https://example.com/callback",
			CodeVerifier: "abc",
		}, OAuthClientCredentials{ClientID: "my-client"})
		require.NotNil(t, oerr)
		assert.Contains(t, oerr.Description, "already been used")
	})

	t.Run("authorization_code — code expired", func(t *testing.T) {
		db, mock := newMockDB(t)
		expectClientLookup(mock, mockClientRows())

		svc := newOAuthTokenSvc(db, &mockClientRepo{},
			&mockOAuthAuthCodeRepo{
				findByCodeHashFn: func(_ string) (*OAuthAuthorizationCode, error) {
					return &OAuthAuthorizationCode{
						ClientID:  10,
						ExpiresAt: time.Now().Add(-1 * time.Minute),
					}, nil
				},
			},
			&mockOAuthRefreshTokenRepo{}, &mockUserRepo{},
			&mockUserIdentityRepo{findByUserIDAndClientIDFn: func(_, _ int64) (*UserIdentity, error) { return nil, nil }},
			&mockAuthEventService{})

		_, oerr := svc.Exchange(ctx, OAuthTokenRequestDTO{
			GrantType:    "authorization_code",
			Code:         "code123",
			RedirectURI:  "https://example.com/callback",
			CodeVerifier: "abc",
		}, OAuthClientCredentials{ClientID: "my-client"})
		require.NotNil(t, oerr)
		assert.Contains(t, oerr.Description, "expired")
	})

	t.Run("authorization_code — client mismatch", func(t *testing.T) {
		db, mock := newMockDB(t)
		expectClientLookup(mock, mockClientRows())

		svc := newOAuthTokenSvc(db, &mockClientRepo{},
			&mockOAuthAuthCodeRepo{
				findByCodeHashFn: func(_ string) (*OAuthAuthorizationCode, error) {
					return &OAuthAuthorizationCode{
						ClientID:  999, // different client
						ExpiresAt: time.Now().Add(10 * time.Minute),
					}, nil
				},
			},
			&mockOAuthRefreshTokenRepo{}, &mockUserRepo{},
			&mockUserIdentityRepo{findByUserIDAndClientIDFn: func(_, _ int64) (*UserIdentity, error) { return nil, nil }},
			&mockAuthEventService{})

		_, oerr := svc.Exchange(ctx, OAuthTokenRequestDTO{
			GrantType:    "authorization_code",
			Code:         "code123",
			RedirectURI:  "https://example.com/callback",
			CodeVerifier: "abc",
		}, OAuthClientCredentials{ClientID: "my-client"})
		require.NotNil(t, oerr)
		assert.Contains(t, oerr.Description, "not issued to this client")
	})

	t.Run("authorization_code — redirect URI mismatch", func(t *testing.T) {
		db, mock := newMockDB(t)
		expectClientLookup(mock, mockClientRows())

		svc := newOAuthTokenSvc(db, &mockClientRepo{},
			&mockOAuthAuthCodeRepo{
				findByCodeHashFn: func(_ string) (*OAuthAuthorizationCode, error) {
					return &OAuthAuthorizationCode{
						ClientID:    10,
						RedirectURI: "https://other.com/callback",
						ExpiresAt:   time.Now().Add(10 * time.Minute),
					}, nil
				},
			},
			&mockOAuthRefreshTokenRepo{}, &mockUserRepo{},
			&mockUserIdentityRepo{findByUserIDAndClientIDFn: func(_, _ int64) (*UserIdentity, error) { return nil, nil }},
			&mockAuthEventService{})

		_, oerr := svc.Exchange(ctx, OAuthTokenRequestDTO{
			GrantType:    "authorization_code",
			Code:         "code123",
			RedirectURI:  "https://example.com/callback",
			CodeVerifier: "abc",
		}, OAuthClientCredentials{ClientID: "my-client"})
		require.NotNil(t, oerr)
		assert.Contains(t, oerr.Description, "redirect_uri does not match")
	})

	t.Run("authorization_code — PKCE validation failed", func(t *testing.T) {
		db, mock := newMockDB(t)
		expectClientLookup(mock, mockClientRows())

		svc := newOAuthTokenSvc(db, &mockClientRepo{},
			&mockOAuthAuthCodeRepo{
				findByCodeHashFn: func(_ string) (*OAuthAuthorizationCode, error) {
					return &OAuthAuthorizationCode{
						ClientID:            10,
						RedirectURI:         "https://example.com/callback",
						CodeChallenge:       "invalidchallenge",
						CodeChallengeMethod: "S256",
						ExpiresAt:           time.Now().Add(10 * time.Minute),
					}, nil
				},
			},
			&mockOAuthRefreshTokenRepo{}, &mockUserRepo{},
			&mockUserIdentityRepo{findByUserIDAndClientIDFn: func(_, _ int64) (*UserIdentity, error) { return nil, nil }},
			&mockAuthEventService{})

		_, oerr := svc.Exchange(ctx, OAuthTokenRequestDTO{
			GrantType:    "authorization_code",
			Code:         "code123",
			RedirectURI:  "https://example.com/callback",
			CodeVerifier: "wrong-verifier",
		}, OAuthClientCredentials{ClientID: "my-client"})
		require.NotNil(t, oerr)
		assert.Contains(t, oerr.Description, "PKCE validation failed")
	})

	t.Run("authorization_code — mark used error", func(t *testing.T) {
		verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
		challenge := crypto.ComputeS256Challenge(verifier)

		db, mock := newMockDB(t)
		expectClientLookup(mock, mockClientRows())

		svc := newOAuthTokenSvc(db, &mockClientRepo{},
			&mockOAuthAuthCodeRepo{
				findByCodeHashFn: func(_ string) (*OAuthAuthorizationCode, error) {
					return &OAuthAuthorizationCode{
						ClientID:            10,
						UserID:              1,
						RedirectURI:         "https://example.com/callback",
						CodeChallenge:       challenge,
						CodeChallengeMethod: "S256",
						ExpiresAt:           time.Now().Add(10 * time.Minute),
					}, nil
				},
				markUsedFn: func(_ int64) error {
					return errors.New("mark used error")
				},
			},
			&mockOAuthRefreshTokenRepo{}, &mockUserRepo{},
			&mockUserIdentityRepo{findByUserIDAndClientIDFn: func(_, _ int64) (*UserIdentity, error) { return nil, nil }},
			&mockAuthEventService{})

		_, oerr := svc.Exchange(ctx, OAuthTokenRequestDTO{
			GrantType:    "authorization_code",
			Code:         "code123",
			RedirectURI:  "https://example.com/callback",
			CodeVerifier: verifier,
		}, OAuthClientCredentials{ClientID: "my-client"})
		require.NotNil(t, oerr)
		assert.Equal(t, "server_error", oerr.Code)
	})

	t.Run("authorization_code — resolve sub error", func(t *testing.T) {
		verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
		challenge := crypto.ComputeS256Challenge(verifier)

		db, mock := newMockDB(t)
		expectClientLookup(mock, mockClientRows())

		svc := newOAuthTokenSvc(db, &mockClientRepo{},
			&mockOAuthAuthCodeRepo{
				findByCodeHashFn: func(_ string) (*OAuthAuthorizationCode, error) {
					return &OAuthAuthorizationCode{
						ClientID:            10,
						UserID:              1,
						RedirectURI:         "https://example.com/callback",
						CodeChallenge:       challenge,
						CodeChallengeMethod: "S256",
						ExpiresAt:           time.Now().Add(10 * time.Minute),
					}, nil
				},
			},
			&mockOAuthRefreshTokenRepo{}, &mockUserRepo{},
			&mockUserIdentityRepo{
				findByUserIDAndClientIDFn: func(_, _ int64) (*UserIdentity, error) {
					return nil, errors.New("identity lookup error")
				},
			},
			&mockAuthEventService{})

		_, oerr := svc.Exchange(ctx, OAuthTokenRequestDTO{
			GrantType:    "authorization_code",
			Code:         "code123",
			RedirectURI:  "https://example.com/callback",
			CodeVerifier: verifier,
		}, OAuthClientCredentials{ClientID: "my-client"})
		require.NotNil(t, oerr)
		assert.Equal(t, "server_error", oerr.Code)
	})

	t.Run("authorization_code — user not found", func(t *testing.T) {
		verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
		challenge := crypto.ComputeS256Challenge(verifier)

		db, mock := newMockDB(t)
		expectClientLookup(mock, mockClientRows())

		svc := newOAuthTokenSvc(db, &mockClientRepo{},
			&mockOAuthAuthCodeRepo{
				findByCodeHashFn: func(_ string) (*OAuthAuthorizationCode, error) {
					return &OAuthAuthorizationCode{
						ClientID:            10,
						UserID:              1,
						RedirectURI:         "https://example.com/callback",
						CodeChallenge:       challenge,
						CodeChallengeMethod: "S256",
						ExpiresAt:           time.Now().Add(10 * time.Minute),
					}, nil
				},
			},
			&mockOAuthRefreshTokenRepo{},
			&mockUserRepo{
				findByIDFn: func(_ any, _ ...string) (*User, error) {
					return nil, nil // user not found
				},
			},
			&mockUserIdentityRepo{
				findByUserIDAndClientIDFn: func(_, _ int64) (*UserIdentity, error) {
					return &UserIdentity{Sub: "user-sub"}, nil
				},
			},
			&mockAuthEventService{})

		_, oerr := svc.Exchange(ctx, OAuthTokenRequestDTO{
			GrantType:    "authorization_code",
			Code:         "code123",
			RedirectURI:  "https://example.com/callback",
			CodeVerifier: verifier,
		}, OAuthClientCredentials{ClientID: "my-client"})
		require.NotNil(t, oerr)
		assert.Equal(t, "server_error", oerr.Code)
	})

	t.Run("authorization_code — full success", func(t *testing.T) {
		initTestJWTKeysService(t)
		verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
		challenge := crypto.ComputeS256Challenge(verifier)

		db, mock := newMockDB(t)
		expectClientLookup(mock, mockClientRows())

		svc := newOAuthTokenSvc(db, &mockClientRepo{},
			&mockOAuthAuthCodeRepo{
				findByCodeHashFn: func(_ string) (*OAuthAuthorizationCode, error) {
					return &OAuthAuthorizationCode{
						OAuthAuthorizationCodeID: 1,
						ClientID:                 10,
						UserID:                   1,
						TenantID:                 1,
						RedirectURI:              "https://example.com/callback",
						Scope:                    "openid profile offline_access",
						CodeChallenge:            challenge,
						CodeChallengeMethod:      "S256",
						ExpiresAt:                time.Now().Add(10 * time.Minute),
					}, nil
				},
			},
			&mockOAuthRefreshTokenRepo{},
			&mockUserRepo{
				findByIDFn: func(_ any, _ ...string) (*User, error) {
					return &User{UserID: 1, UserUUID: uuid.New(), Email: "test@example.com"}, nil
				},
			},
			&mockUserIdentityRepo{
				findByUserIDAndClientIDFn: func(_, _ int64) (*UserIdentity, error) {
					return &UserIdentity{Sub: "user-sub-123"}, nil
				},
			},
			&mockAuthEventService{})

		result, oerr := svc.Exchange(ctx, OAuthTokenRequestDTO{
			GrantType:    "authorization_code",
			Code:         "code123",
			RedirectURI:  "https://example.com/callback",
			CodeVerifier: verifier,
		}, OAuthClientCredentials{ClientID: "my-client"})
		require.Nil(t, oerr)
		require.NotNil(t, result)
		assert.NotEmpty(t, result.AccessToken)
		assert.NotEmpty(t, result.IDToken)
		assert.NotEmpty(t, result.RefreshToken)
		assert.Equal(t, "Bearer", result.TokenType)
		assert.Equal(t, "openid profile offline_access", result.Scope)
	})

	t.Run("authorization_code — auth code lookup error", func(t *testing.T) {
		db, mock := newMockDB(t)
		expectClientLookup(mock, mockClientRows())

		svc := newOAuthTokenSvc(db, &mockClientRepo{},
			&mockOAuthAuthCodeRepo{
				findByCodeHashFn: func(_ string) (*OAuthAuthorizationCode, error) {
					return nil, errors.New("db error")
				},
			},
			&mockOAuthRefreshTokenRepo{}, &mockUserRepo{},
			&mockUserIdentityRepo{findByUserIDAndClientIDFn: func(_, _ int64) (*UserIdentity, error) { return nil, nil }},
			&mockAuthEventService{})

		_, oerr := svc.Exchange(ctx, OAuthTokenRequestDTO{
			GrantType:    "authorization_code",
			Code:         "code123",
			RedirectURI:  "https://example.com/callback",
			CodeVerifier: "abc",
		}, OAuthClientCredentials{ClientID: "my-client"})
		require.NotNil(t, oerr)
		assert.Equal(t, "server_error", oerr.Code)
	})
}

// ── TestOAuthTokenService_Exchange_RefreshToken ─────────────────────────────

func TestOAuthTokenService_Exchange_RefreshToken(t *testing.T) {
	ctx := context.Background()

	t.Run("missing refresh_token", func(t *testing.T) {
		db, _ := newMockDB(t)
		svc := newOAuthTokenSvc(db, &mockClientRepo{}, &mockOAuthAuthCodeRepo{}, &mockOAuthRefreshTokenRepo{}, &mockUserRepo{}, &mockUserIdentityRepo{findByUserIDAndClientIDFn: func(_, _ int64) (*UserIdentity, error) { return nil, nil }}, &mockAuthEventService{})

		_, oerr := svc.Exchange(ctx, OAuthTokenRequestDTO{
			GrantType: "refresh_token",
		}, OAuthClientCredentials{ClientID: "my-client"})
		require.NotNil(t, oerr)
		assert.Contains(t, oerr.Description, "refresh_token is required")
	})

	t.Run("token not found", func(t *testing.T) {
		db, mock := newMockDB(t)
		expectClientLookup(mock, mockClientRows())

		svc := newOAuthTokenSvc(db, &mockClientRepo{}, &mockOAuthAuthCodeRepo{},
			&mockOAuthRefreshTokenRepo{
				findByTokenHashFn: func(_ string) (*OAuthRefreshToken, error) {
					return nil, nil
				},
			},
			&mockUserRepo{},
			&mockUserIdentityRepo{findByUserIDAndClientIDFn: func(_, _ int64) (*UserIdentity, error) { return nil, nil }},
			&mockAuthEventService{})

		_, oerr := svc.Exchange(ctx, OAuthTokenRequestDTO{
			GrantType:    "refresh_token",
			RefreshToken: "some-token",
		}, OAuthClientCredentials{ClientID: "my-client"})
		require.NotNil(t, oerr)
		assert.Contains(t, oerr.Description, "invalid")
	})

	t.Run("token already revoked — reuse detection", func(t *testing.T) {
		db, mock := newMockDB(t)
		expectClientLookup(mock, mockClientRows())
		familyID := uuid.New()

		svc := newOAuthTokenSvc(db, &mockClientRepo{}, &mockOAuthAuthCodeRepo{},
			&mockOAuthRefreshTokenRepo{
				findByTokenHashFn: func(_ string) (*OAuthRefreshToken, error) {
					return &OAuthRefreshToken{
						IsRevoked: true,
						FamilyID:  familyID,
						UserID:    1,
						TenantID:  1,
					}, nil
				},
			},
			&mockUserRepo{},
			&mockUserIdentityRepo{findByUserIDAndClientIDFn: func(_, _ int64) (*UserIdentity, error) { return nil, nil }},
			&mockAuthEventService{})

		_, oerr := svc.Exchange(ctx, OAuthTokenRequestDTO{
			GrantType:    "refresh_token",
			RefreshToken: "some-token",
		}, OAuthClientCredentials{ClientID: "my-client"})
		require.NotNil(t, oerr)
		assert.Contains(t, oerr.Description, "revoked")
	})

	t.Run("token expired", func(t *testing.T) {
		db, mock := newMockDB(t)
		expectClientLookup(mock, mockClientRows())

		svc := newOAuthTokenSvc(db, &mockClientRepo{}, &mockOAuthAuthCodeRepo{},
			&mockOAuthRefreshTokenRepo{
				findByTokenHashFn: func(_ string) (*OAuthRefreshToken, error) {
					return &OAuthRefreshToken{
						ClientID:  10,
						ExpiresAt: time.Now().Add(-1 * time.Minute),
					}, nil
				},
			},
			&mockUserRepo{},
			&mockUserIdentityRepo{findByUserIDAndClientIDFn: func(_, _ int64) (*UserIdentity, error) { return nil, nil }},
			&mockAuthEventService{})

		_, oerr := svc.Exchange(ctx, OAuthTokenRequestDTO{
			GrantType:    "refresh_token",
			RefreshToken: "some-token",
		}, OAuthClientCredentials{ClientID: "my-client"})
		require.NotNil(t, oerr)
		assert.Contains(t, oerr.Description, "expired")
	})

	t.Run("client mismatch", func(t *testing.T) {
		db, mock := newMockDB(t)
		expectClientLookup(mock, mockClientRows())

		svc := newOAuthTokenSvc(db, &mockClientRepo{}, &mockOAuthAuthCodeRepo{},
			&mockOAuthRefreshTokenRepo{
				findByTokenHashFn: func(_ string) (*OAuthRefreshToken, error) {
					return &OAuthRefreshToken{
						ClientID:  999,
						ExpiresAt: time.Now().Add(10 * time.Minute),
					}, nil
				},
			},
			&mockUserRepo{},
			&mockUserIdentityRepo{findByUserIDAndClientIDFn: func(_, _ int64) (*UserIdentity, error) { return nil, nil }},
			&mockAuthEventService{})

		_, oerr := svc.Exchange(ctx, OAuthTokenRequestDTO{
			GrantType:    "refresh_token",
			RefreshToken: "some-token",
		}, OAuthClientCredentials{ClientID: "my-client"})
		require.NotNil(t, oerr)
		assert.Contains(t, oerr.Description, "not issued to this client")
	})

	t.Run("transaction error", func(t *testing.T) {
		db, mock := newMockDB(t)
		expectClientLookup(mock, mockClientRows())
		mock.ExpectBegin().WillReturnError(errors.New("tx error"))

		svc := newOAuthTokenSvc(db, &mockClientRepo{}, &mockOAuthAuthCodeRepo{},
			&mockOAuthRefreshTokenRepo{
				findByTokenHashFn: func(_ string) (*OAuthRefreshToken, error) {
					return &OAuthRefreshToken{
						ClientID:  10,
						UserID:    1,
						ExpiresAt: time.Now().Add(10 * time.Minute),
					}, nil
				},
			},
			&mockUserRepo{},
			&mockUserIdentityRepo{findByUserIDAndClientIDFn: func(_, _ int64) (*UserIdentity, error) { return nil, nil }},
			&mockAuthEventService{})

		_, oerr := svc.Exchange(ctx, OAuthTokenRequestDTO{
			GrantType:    "refresh_token",
			RefreshToken: "some-token",
		}, OAuthClientCredentials{ClientID: "my-client"})
		require.NotNil(t, oerr)
		assert.Equal(t, "server_error", oerr.Code)
	})

	t.Run("full success — refresh token rotation", func(t *testing.T) {
		initTestJWTKeysService(t)
		db, mock := newMockDB(t)
		expectClientLookup(mock, mockClientRows())
		familyID := uuid.New()

		// Transaction: BEGIN + COMMIT
		mock.ExpectBegin()
		mock.ExpectCommit()

		svc := newOAuthTokenSvc(db, &mockClientRepo{}, &mockOAuthAuthCodeRepo{},
			&mockOAuthRefreshTokenRepo{
				findByTokenHashFn: func(_ string) (*OAuthRefreshToken, error) {
					return &OAuthRefreshToken{
						OAuthRefreshTokenID: 1,
						ClientID:            10,
						UserID:              1,
						TenantID:            1,
						FamilyID:            familyID,
						Scope:               "openid profile offline_access",
						ExpiresAt:           time.Now().Add(7 * 24 * time.Hour),
					}, nil
				},
				revokeByIDFn: func(_ int64) error { return nil },
			},
			&mockUserRepo{
				findByIDFn: func(_ any, _ ...string) (*User, error) {
					return &User{UserID: 1, UserUUID: uuid.New(), Email: "test@example.com"}, nil
				},
			},
			&mockUserIdentityRepo{
				findByUserIDAndClientIDFn: func(_, _ int64) (*UserIdentity, error) {
					return &UserIdentity{Sub: "user-sub-rt"}, nil
				},
			},
			&mockAuthEventService{})

		result, oerr := svc.Exchange(ctx, OAuthTokenRequestDTO{
			GrantType:    "refresh_token",
			RefreshToken: "some-token",
		}, OAuthClientCredentials{ClientID: "my-client"})
		require.Nil(t, oerr)
		require.NotNil(t, result)
		assert.NotEmpty(t, result.AccessToken)
		assert.NotEmpty(t, result.IDToken)
		assert.NotEmpty(t, result.RefreshToken)
		assert.Equal(t, "Bearer", result.TokenType)
		assert.Equal(t, "openid profile offline_access", result.Scope)
	})

	t.Run("full success — with scope narrowing", func(t *testing.T) {
		initTestJWTKeysService(t)
		db, mock := newMockDB(t)
		expectClientLookup(mock, mockClientRows())
		familyID := uuid.New()

		mock.ExpectBegin()
		mock.ExpectCommit()

		svc := newOAuthTokenSvc(db, &mockClientRepo{}, &mockOAuthAuthCodeRepo{},
			&mockOAuthRefreshTokenRepo{
				findByTokenHashFn: func(_ string) (*OAuthRefreshToken, error) {
					return &OAuthRefreshToken{
						OAuthRefreshTokenID: 1,
						ClientID:            10,
						UserID:              1,
						TenantID:            1,
						FamilyID:            familyID,
						Scope:               "openid profile email",
						ExpiresAt:           time.Now().Add(7 * 24 * time.Hour),
					}, nil
				},
				revokeByIDFn: func(_ int64) error { return nil },
			},
			&mockUserRepo{
				findByIDFn: func(_ any, _ ...string) (*User, error) {
					return &User{UserID: 1, UserUUID: uuid.New(), Email: "test@example.com"}, nil
				},
			},
			&mockUserIdentityRepo{
				findByUserIDAndClientIDFn: func(_, _ int64) (*UserIdentity, error) {
					return &UserIdentity{Sub: "user-sub-rt"}, nil
				},
			},
			&mockAuthEventService{})

		result, oerr := svc.Exchange(ctx, OAuthTokenRequestDTO{
			GrantType:    "refresh_token",
			RefreshToken: "some-token",
			Scope:        "openid email", // narrower scope
		}, OAuthClientCredentials{ClientID: "my-client"})
		require.Nil(t, oerr)
		require.NotNil(t, result)
		assert.Equal(t, "openid email", result.Scope)
	})

	t.Run("revoke by ID error in transaction", func(t *testing.T) {
		db, mock := newMockDB(t)
		expectClientLookup(mock, mockClientRows())
		mock.ExpectBegin()
		mock.ExpectRollback()

		svc := newOAuthTokenSvc(db, &mockClientRepo{}, &mockOAuthAuthCodeRepo{},
			&mockOAuthRefreshTokenRepo{
				findByTokenHashFn: func(_ string) (*OAuthRefreshToken, error) {
					return &OAuthRefreshToken{
						OAuthRefreshTokenID: 1,
						ClientID:            10,
						UserID:              1,
						ExpiresAt:           time.Now().Add(7 * 24 * time.Hour),
					}, nil
				},
				revokeByIDFn: func(_ int64) error { return errors.New("revoke error") },
			},
			&mockUserRepo{},
			&mockUserIdentityRepo{findByUserIDAndClientIDFn: func(_, _ int64) (*UserIdentity, error) { return nil, nil }},
			&mockAuthEventService{})

		_, oerr := svc.Exchange(ctx, OAuthTokenRequestDTO{
			GrantType:    "refresh_token",
			RefreshToken: "some-token",
		}, OAuthClientCredentials{ClientID: "my-client"})
		require.NotNil(t, oerr)
		assert.Equal(t, "server_error", oerr.Code)
	})

	t.Run("user not found in transaction", func(t *testing.T) {
		db, mock := newMockDB(t)
		expectClientLookup(mock, mockClientRows())
		mock.ExpectBegin()
		mock.ExpectRollback()

		svc := newOAuthTokenSvc(db, &mockClientRepo{}, &mockOAuthAuthCodeRepo{},
			&mockOAuthRefreshTokenRepo{
				findByTokenHashFn: func(_ string) (*OAuthRefreshToken, error) {
					return &OAuthRefreshToken{
						OAuthRefreshTokenID: 1,
						ClientID:            10,
						UserID:              1,
						ExpiresAt:           time.Now().Add(7 * 24 * time.Hour),
					}, nil
				},
				revokeByIDFn: func(_ int64) error { return nil },
			},
			&mockUserRepo{
				findByIDFn: func(_ any, _ ...string) (*User, error) {
					return nil, nil // user not found
				},
			},
			&mockUserIdentityRepo{
				findByUserIDAndClientIDFn: func(_, _ int64) (*UserIdentity, error) {
					return &UserIdentity{Sub: "sub"}, nil
				},
			},
			&mockAuthEventService{})

		_, oerr := svc.Exchange(ctx, OAuthTokenRequestDTO{
			GrantType:    "refresh_token",
			RefreshToken: "some-token",
		}, OAuthClientCredentials{ClientID: "my-client"})
		require.NotNil(t, oerr)
		assert.Equal(t, "server_error", oerr.Code)
	})

	t.Run("refresh token lookup error", func(t *testing.T) {
		db, mock := newMockDB(t)
		expectClientLookup(mock, mockClientRows())

		svc := newOAuthTokenSvc(db, &mockClientRepo{}, &mockOAuthAuthCodeRepo{},
			&mockOAuthRefreshTokenRepo{
				findByTokenHashFn: func(_ string) (*OAuthRefreshToken, error) {
					return nil, errors.New("db error")
				},
			},
			&mockUserRepo{},
			&mockUserIdentityRepo{findByUserIDAndClientIDFn: func(_, _ int64) (*UserIdentity, error) { return nil, nil }},
			&mockAuthEventService{})

		_, oerr := svc.Exchange(ctx, OAuthTokenRequestDTO{
			GrantType:    "refresh_token",
			RefreshToken: "some-token",
		}, OAuthClientCredentials{ClientID: "my-client"})
		require.NotNil(t, oerr)
		assert.Equal(t, "server_error", oerr.Code)
	})
}

// ── TestOAuthTokenService_Exchange_ClientCredentials ────────────────────────

func TestOAuthTokenService_Exchange_ClientCredentials(t *testing.T) {
	ctx := context.Background()

	t.Run("grant not allowed", func(t *testing.T) {
		db, mock := newMockDB(t)
		// Return a client that doesn't have client_credentials grant
		rows := sqlmock.NewRows([]string{
			"client_id", "client_uuid", "tenant_id", "identity_provider_id", "name", "display_name",
			"client_type", "domain", "identifier", "secret", "status",
			"is_default", "is_system", "token_endpoint_auth_method",
			"grant_types", "response_types", "access_token_ttl", "refresh_token_ttl",
			"require_consent", "created_at", "updated_at",
		}).AddRow(
			10, uuid.New(), 1, int64(100), "test-client", "Test Client",
			"m2m", nil, "m2m-client", nil, "active",
			false, false, "none",
			`{authorization_code}`, `{code}`, nil, nil,
			false, time.Now(), time.Now(),
		)
		expectClientLookup(mock, rows)

		svc := newOAuthTokenSvc(db, &mockClientRepo{}, &mockOAuthAuthCodeRepo{}, &mockOAuthRefreshTokenRepo{}, &mockUserRepo{},
			&mockUserIdentityRepo{findByUserIDAndClientIDFn: func(_, _ int64) (*UserIdentity, error) { return nil, nil }},
			&mockAuthEventService{})

		_, oerr := svc.Exchange(ctx, OAuthTokenRequestDTO{
			GrantType: "client_credentials",
		}, OAuthClientCredentials{ClientID: "m2m-client"})
		require.NotNil(t, oerr)
		assert.Equal(t, "unauthorized_client", oerr.Code)
	})

	t.Run("success", func(t *testing.T) {
		initTestJWTKeysService(t)
		db, mock := newMockDB(t)
		rows := sqlmock.NewRows([]string{
			"client_id", "client_uuid", "tenant_id", "identity_provider_id", "name", "display_name",
			"client_type", "domain", "identifier", "secret", "status",
			"is_default", "is_system", "token_endpoint_auth_method",
			"grant_types", "response_types", "access_token_ttl", "refresh_token_ttl",
			"require_consent", "created_at", "updated_at",
		}).AddRow(
			10, uuid.New(), 1, int64(100), "m2m-client", "M2M Client",
			"m2m", "https://auth.example.com", "m2m-client", nil, "active",
			false, false, "none",
			`{client_credentials}`, `{}`, nil, nil,
			false, time.Now(), time.Now(),
		)
		expectClientLookup(mock, rows)

		svc := newOAuthTokenSvc(db, &mockClientRepo{}, &mockOAuthAuthCodeRepo{}, &mockOAuthRefreshTokenRepo{}, &mockUserRepo{},
			&mockUserIdentityRepo{findByUserIDAndClientIDFn: func(_, _ int64) (*UserIdentity, error) { return nil, nil }},
			&mockAuthEventService{})

		result, oerr := svc.Exchange(ctx, OAuthTokenRequestDTO{
			GrantType: "client_credentials",
		}, OAuthClientCredentials{ClientID: "m2m-client"})
		require.Nil(t, oerr)
		assert.NotEmpty(t, result.AccessToken)
		assert.Equal(t, "Bearer", result.TokenType)
		assert.Empty(t, result.RefreshToken) // client_credentials grant has no refresh token
		assert.Empty(t, result.IDToken)
	})

	t.Run("success with custom access token ttl", func(t *testing.T) {
		initTestJWTKeysService(t)
		db, mock := newMockDB(t)
		rows := sqlmock.NewRows([]string{
			"client_id", "client_uuid", "tenant_id", "identity_provider_id", "name", "display_name",
			"client_type", "domain", "identifier", "secret", "status",
			"is_default", "is_system", "token_endpoint_auth_method",
			"grant_types", "response_types", "access_token_ttl", "refresh_token_ttl",
			"require_consent", "created_at", "updated_at",
		}).AddRow(
			10, uuid.New(), 1, int64(100), "m2m-client", "M2M Client",
			"m2m", "https://auth.example.com", "m2m-client", nil, "active",
			false, false, "none",
			`{client_credentials}`, `{}`, 3600, nil,
			false, time.Now(), time.Now(),
		)
		expectClientLookup(mock, rows)

		svc := newOAuthTokenSvc(db, &mockClientRepo{}, &mockOAuthAuthCodeRepo{}, &mockOAuthRefreshTokenRepo{}, &mockUserRepo{},
			&mockUserIdentityRepo{findByUserIDAndClientIDFn: func(_, _ int64) (*UserIdentity, error) { return nil, nil }},
			&mockAuthEventService{})

		result, oerr := svc.Exchange(ctx, OAuthTokenRequestDTO{
			GrantType: "client_credentials",
		}, OAuthClientCredentials{ClientID: "m2m-client"})
		require.Nil(t, oerr)
		assert.Equal(t, int64(3600), result.ExpiresIn)
	})
}

// ── TestOAuthTokenService_Revoke ────────────────────────────────────────────

func TestOAuthTokenService_Revoke(t *testing.T) {
	ctx := context.Background()

	t.Run("client auth failure", func(t *testing.T) {
		db, _ := newMockDB(t)
		svc := newOAuthTokenSvc(db, &mockClientRepo{}, &mockOAuthAuthCodeRepo{}, &mockOAuthRefreshTokenRepo{}, &mockUserRepo{},
			&mockUserIdentityRepo{findByUserIDAndClientIDFn: func(_, _ int64) (*UserIdentity, error) { return nil, nil }},
			&mockAuthEventService{})

		oerr := svc.Revoke(ctx, OAuthRevokeRequestDTO{Token: "t"}, OAuthClientCredentials{})
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_client", oerr.Code)
	})

	t.Run("revokes refresh token", func(t *testing.T) {
		db, mock := newMockDB(t)
		expectClientLookup(mock, mockClientRows())
		var revokedID int64

		svc := newOAuthTokenSvc(db, &mockClientRepo{}, &mockOAuthAuthCodeRepo{},
			&mockOAuthRefreshTokenRepo{
				findByTokenHashFn: func(_ string) (*OAuthRefreshToken, error) {
					return &OAuthRefreshToken{
						OAuthRefreshTokenID: 42,
						ClientID:            10,
						UserID:              1,
					}, nil
				},
				revokeByIDFn: func(id int64) error {
					revokedID = id
					return nil
				},
			},
			&mockUserRepo{},
			&mockUserIdentityRepo{findByUserIDAndClientIDFn: func(_, _ int64) (*UserIdentity, error) { return nil, nil }},
			&mockAuthEventService{})

		oerr := svc.Revoke(ctx, OAuthRevokeRequestDTO{Token: "t"}, OAuthClientCredentials{ClientID: "my-client"})
		require.Nil(t, oerr)
		assert.Equal(t, int64(42), revokedID)
	})

	t.Run("already revoked token — no-op", func(t *testing.T) {
		db, mock := newMockDB(t)
		expectClientLookup(mock, mockClientRows())

		svc := newOAuthTokenSvc(db, &mockClientRepo{}, &mockOAuthAuthCodeRepo{},
			&mockOAuthRefreshTokenRepo{
				findByTokenHashFn: func(_ string) (*OAuthRefreshToken, error) {
					return &OAuthRefreshToken{
						ClientID:  10,
						IsRevoked: true,
					}, nil
				},
			},
			&mockUserRepo{},
			&mockUserIdentityRepo{findByUserIDAndClientIDFn: func(_, _ int64) (*UserIdentity, error) { return nil, nil }},
			&mockAuthEventService{})

		oerr := svc.Revoke(ctx, OAuthRevokeRequestDTO{Token: "t"}, OAuthClientCredentials{ClientID: "my-client"})
		require.Nil(t, oerr)
	})

	t.Run("token not found — 200 OK per RFC 7009", func(t *testing.T) {
		db, mock := newMockDB(t)
		expectClientLookup(mock, mockClientRows())

		svc := newOAuthTokenSvc(db, &mockClientRepo{}, &mockOAuthAuthCodeRepo{},
			&mockOAuthRefreshTokenRepo{
				findByTokenHashFn: func(_ string) (*OAuthRefreshToken, error) {
					return nil, nil
				},
			},
			&mockUserRepo{},
			&mockUserIdentityRepo{findByUserIDAndClientIDFn: func(_, _ int64) (*UserIdentity, error) { return nil, nil }},
			&mockAuthEventService{})

		oerr := svc.Revoke(ctx, OAuthRevokeRequestDTO{Token: "t"}, OAuthClientCredentials{ClientID: "my-client"})
		require.Nil(t, oerr)
	})

	t.Run("client mismatch — ignore", func(t *testing.T) {
		db, mock := newMockDB(t)
		expectClientLookup(mock, mockClientRows())

		svc := newOAuthTokenSvc(db, &mockClientRepo{}, &mockOAuthAuthCodeRepo{},
			&mockOAuthRefreshTokenRepo{
				findByTokenHashFn: func(_ string) (*OAuthRefreshToken, error) {
					return &OAuthRefreshToken{
						ClientID: 999, // different client
					}, nil
				},
			},
			&mockUserRepo{},
			&mockUserIdentityRepo{findByUserIDAndClientIDFn: func(_, _ int64) (*UserIdentity, error) { return nil, nil }},
			&mockAuthEventService{})

		oerr := svc.Revoke(ctx, OAuthRevokeRequestDTO{Token: "t"}, OAuthClientCredentials{ClientID: "my-client"})
		require.Nil(t, oerr)
	})

	t.Run("token lookup error — 200 OK per RFC 7009", func(t *testing.T) {
		db, mock := newMockDB(t)
		expectClientLookup(mock, mockClientRows())

		svc := newOAuthTokenSvc(db, &mockClientRepo{}, &mockOAuthAuthCodeRepo{},
			&mockOAuthRefreshTokenRepo{
				findByTokenHashFn: func(_ string) (*OAuthRefreshToken, error) {
					return nil, errors.New("db error")
				},
			},
			&mockUserRepo{},
			&mockUserIdentityRepo{findByUserIDAndClientIDFn: func(_, _ int64) (*UserIdentity, error) { return nil, nil }},
			&mockAuthEventService{})

		oerr := svc.Revoke(ctx, OAuthRevokeRequestDTO{Token: "t"}, OAuthClientCredentials{ClientID: "my-client"})
		require.Nil(t, oerr) // always 200 OK
	})

	t.Run("revokes access token by denylisting jti", func(t *testing.T) {
		initTestJWTKeysService(t)
		origChecker := jwt.JTIChecker
		jwt.JTIChecker = nil
		t.Cleanup(func() { jwt.JTIChecker = origChecker })

		db, mock := newMockDB(t)
		expectClientLookup(mock, mockClientRows())

		token, err := jwt.GenerateAccessToken("user-sub", "openid profile", "https://auth.example.com", "my-client", "my-client", "default-provider")
		require.NoError(t, err)
		claims, err := jwt.ValidateToken(token)
		require.NoError(t, err)
		jti, _ := claims["jti"].(string)
		require.NotEmpty(t, jti)

		denylist := &recordingJTIDenylister{}
		svc := &oauthTokenService{
			db:               db,
			refreshTokenRepo: &mockOAuthRefreshTokenRepo{},
			authEventService: &mockAuthEventService{},
			jtiDenylist:      denylist,
		}

		oerr := svc.Revoke(ctx, OAuthRevokeRequestDTO{Token: token}, OAuthClientCredentials{ClientID: "my-client"})

		require.Nil(t, oerr)
		assert.Equal(t, jti, denylist.jti)
		assert.Positive(t, denylist.ttl)
		assert.LessOrEqual(t, denylist.ttl, jwt.AccessTokenTTL)
	})

	t.Run("access token denylist error returns server error", func(t *testing.T) {
		initTestJWTKeysService(t)
		origChecker := jwt.JTIChecker
		jwt.JTIChecker = nil
		t.Cleanup(func() { jwt.JTIChecker = origChecker })

		db, mock := newMockDB(t)
		expectClientLookup(mock, mockClientRows())

		token, err := jwt.GenerateAccessToken("user-sub", "openid profile", "https://auth.example.com", "my-client", "my-client", "default-provider")
		require.NoError(t, err)

		svc := &oauthTokenService{
			db:               db,
			refreshTokenRepo: &mockOAuthRefreshTokenRepo{},
			authEventService: &mockAuthEventService{},
			jtiDenylist:      &recordingJTIDenylister{err: errors.New("redis down")},
		}

		oerr := svc.Revoke(ctx, OAuthRevokeRequestDTO{Token: token}, OAuthClientCredentials{ClientID: "my-client"})

		require.NotNil(t, oerr)
		assert.Equal(t, "server_error", oerr.Code)
	})
}

// ── TestOAuthTokenService_Introspect ────────────────────────────────────────

func TestOAuthTokenService_Introspect(t *testing.T) {
	ctx := context.Background()

	t.Run("invalid token — active false", func(t *testing.T) {
		db, mock := newMockDB(t)
		expectClientLookup(mock, mockClientRows())
		svc := newOAuthTokenSvc(db, &mockClientRepo{}, &mockOAuthAuthCodeRepo{},
			&mockOAuthRefreshTokenRepo{
				findByTokenHashFn: func(_ string) (*OAuthRefreshToken, error) {
					return nil, nil
				},
			},
			&mockUserRepo{},
			&mockUserIdentityRepo{findByUserIDAndClientIDFn: func(_, _ int64) (*UserIdentity, error) { return nil, nil }},
			&mockAuthEventService{})

		result, oerr := svc.Introspect(ctx, OAuthIntrospectRequestDTO{Token: "garbage"}, OAuthClientCredentials{ClientID: "my-client"})
		require.Nil(t, oerr)
		assert.False(t, result.Active)
	})

	t.Run("valid refresh token", func(t *testing.T) {
		db, mock := newMockDB(t)
		expectClientLookup(mock, mockClientRows())
		now := time.Now()
		svc := newOAuthTokenSvc(db, &mockClientRepo{}, &mockOAuthAuthCodeRepo{},
			&mockOAuthRefreshTokenRepo{
				findByTokenHashFn: func(_ string) (*OAuthRefreshToken, error) {
					return &OAuthRefreshToken{
						UserID:    1,
						ClientID:  10,
						Scope:     "openid",
						ExpiresAt: now.Add(7 * 24 * time.Hour),
						CreatedAt: now,
					}, nil
				},
			},
			&mockUserRepo{},
			&mockUserIdentityRepo{
				findByUserIDAndClientIDFn: func(_, _ int64) (*UserIdentity, error) {
					return &UserIdentity{Sub: "user-sub"}, nil
				},
			},
			&mockAuthEventService{})

		result, oerr := svc.Introspect(ctx, OAuthIntrospectRequestDTO{Token: "rt-token"}, OAuthClientCredentials{ClientID: "my-client"})
		require.Nil(t, oerr)
		assert.True(t, result.Active)
		assert.Equal(t, "refresh_token", result.TokenType)
		assert.Equal(t, "user-sub", result.Sub)
		assert.Equal(t, "openid", result.Scope)
	})

	t.Run("revoked refresh token — active false", func(t *testing.T) {
		db, mock := newMockDB(t)
		expectClientLookup(mock, mockClientRows())
		svc := newOAuthTokenSvc(db, &mockClientRepo{}, &mockOAuthAuthCodeRepo{},
			&mockOAuthRefreshTokenRepo{
				findByTokenHashFn: func(_ string) (*OAuthRefreshToken, error) {
					return &OAuthRefreshToken{
						IsRevoked: true,
						ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
					}, nil
				},
			},
			&mockUserRepo{},
			&mockUserIdentityRepo{findByUserIDAndClientIDFn: func(_, _ int64) (*UserIdentity, error) { return nil, nil }},
			&mockAuthEventService{})

		result, oerr := svc.Introspect(ctx, OAuthIntrospectRequestDTO{Token: "rt-token"}, OAuthClientCredentials{ClientID: "my-client"})
		require.Nil(t, oerr)
		assert.False(t, result.Active)
	})

	t.Run("refresh token sub resolution error — still returns active", func(t *testing.T) {
		db, mock := newMockDB(t)
		expectClientLookup(mock, mockClientRows())
		svc := newOAuthTokenSvc(db, &mockClientRepo{}, &mockOAuthAuthCodeRepo{},
			&mockOAuthRefreshTokenRepo{
				findByTokenHashFn: func(_ string) (*OAuthRefreshToken, error) {
					return &OAuthRefreshToken{
						UserID:    1,
						ClientID:  10,
						Scope:     "openid",
						ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
						CreatedAt: time.Now(),
					}, nil
				},
			},
			&mockUserRepo{},
			&mockUserIdentityRepo{
				findByUserIDAndClientIDFn: func(_, _ int64) (*UserIdentity, error) {
					return nil, errors.New("sub error")
				},
			},
			&mockAuthEventService{})

		result, oerr := svc.Introspect(ctx, OAuthIntrospectRequestDTO{Token: "rt-token"}, OAuthClientCredentials{ClientID: "my-client"})
		require.Nil(t, oerr)
		assert.True(t, result.Active)
		assert.Empty(t, result.Sub) // sub couldn't be resolved
	})

	t.Run("valid JWT access token", func(t *testing.T) {
		initTestJWTKeysService(t)
		db, mock := newMockDB(t)
		expectClientLookup(mock, mockClientRows())

		token, err := jwt.GenerateAccessToken("user-sub", "openid profile", "https://auth.example.com", "my-client", "my-client", "default-provider")
		require.NoError(t, err)

		svc := newOAuthTokenSvc(db, &mockClientRepo{}, &mockOAuthAuthCodeRepo{},
			&mockOAuthRefreshTokenRepo{},
			&mockUserRepo{},
			&mockUserIdentityRepo{findByUserIDAndClientIDFn: func(_, _ int64) (*UserIdentity, error) { return nil, nil }},
			&mockAuthEventService{})

		result, oerr := svc.Introspect(ctx, OAuthIntrospectRequestDTO{Token: token}, OAuthClientCredentials{ClientID: "my-client"})
		require.Nil(t, oerr)
		assert.True(t, result.Active)
		assert.Equal(t, "Bearer", result.TokenType)
		assert.Equal(t, "user-sub", result.Sub)
		assert.Equal(t, "openid profile", result.Scope)
		assert.Equal(t, "my-client", result.ClientID)
		assert.Equal(t, "my-client", result.Aud)
		assert.Equal(t, "https://auth.example.com", result.Iss)
		assert.NotZero(t, result.Exp)
		assert.NotZero(t, result.Iat)
		assert.NotZero(t, result.Nbf)
		assert.NotEmpty(t, result.Jti)
	})

	t.Run("expired refresh token — active false", func(t *testing.T) {
		db, mock := newMockDB(t)
		expectClientLookup(mock, mockClientRows())
		svc := newOAuthTokenSvc(db, &mockClientRepo{}, &mockOAuthAuthCodeRepo{},
			&mockOAuthRefreshTokenRepo{
				findByTokenHashFn: func(_ string) (*OAuthRefreshToken, error) {
					return &OAuthRefreshToken{
						ExpiresAt: time.Now().Add(-1 * time.Hour),
					}, nil
				},
			},
			&mockUserRepo{},
			&mockUserIdentityRepo{findByUserIDAndClientIDFn: func(_, _ int64) (*UserIdentity, error) { return nil, nil }},
			&mockAuthEventService{})

		result, oerr := svc.Introspect(ctx, OAuthIntrospectRequestDTO{Token: "rt-token"}, OAuthClientCredentials{ClientID: "my-client"})
		require.Nil(t, oerr)
		assert.False(t, result.Active)
	})

	t.Run("refresh token lookup error — active false", func(t *testing.T) {
		db, mock := newMockDB(t)
		expectClientLookup(mock, mockClientRows())
		svc := newOAuthTokenSvc(db, &mockClientRepo{}, &mockOAuthAuthCodeRepo{},
			&mockOAuthRefreshTokenRepo{
				findByTokenHashFn: func(_ string) (*OAuthRefreshToken, error) {
					return nil, errors.New("db error")
				},
			},
			&mockUserRepo{},
			&mockUserIdentityRepo{findByUserIDAndClientIDFn: func(_, _ int64) (*UserIdentity, error) { return nil, nil }},
			&mockAuthEventService{})

		result, oerr := svc.Introspect(ctx, OAuthIntrospectRequestDTO{Token: "rt-token"}, OAuthClientCredentials{ClientID: "my-client"})
		require.Nil(t, oerr)
		assert.False(t, result.Active)
	})
}

// ── TestResolveUserSub ──────────────────────────────────────────────────────

func TestResolveUserSub(t *testing.T) {
	t.Run("returns identity sub", func(t *testing.T) {
		db, _ := newMockDB(t)
		svc := &oauthTokenService{
			db:       db,
			userRepo: &mockUserRepo{},
			userIdentityRepo: &mockUserIdentityRepo{
				findByUserIDAndClientIDFn: func(_, _ int64) (*UserIdentity, error) {
					return &UserIdentity{Sub: "id-sub"}, nil
				},
			},
		}
		sub, err := svc.resolveUserSub(1, 10)
		require.NoError(t, err)
		assert.Equal(t, "id-sub", sub)
	})

	t.Run("error when no identity exists", func(t *testing.T) {
		db, _ := newMockDB(t)
		svc := &oauthTokenService{
			db: db,
			userIdentityRepo: &mockUserIdentityRepo{
				findByUserIDAndClientIDFn: func(_, _ int64) (*UserIdentity, error) {
					return nil, nil // no identity found
				},
			},
		}
		_, err := svc.resolveUserSub(1, 10)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no identity found")
	})

	t.Run("identity lookup error", func(t *testing.T) {
		db, _ := newMockDB(t)
		svc := &oauthTokenService{
			db:       db,
			userRepo: &mockUserRepo{},
			userIdentityRepo: &mockUserIdentityRepo{
				findByUserIDAndClientIDFn: func(_, _ int64) (*UserIdentity, error) {
					return nil, errors.New("db error")
				},
			},
		}
		_, err := svc.resolveUserSub(1, 10)
		require.Error(t, err)
	})
}

// ── TestRefreshTokenTTL ─────────────────────────────────────────────────────

func TestRefreshTokenTTL(t *testing.T) {
	svc := &oauthTokenService{}

	t.Run("uses client override", func(t *testing.T) {
		ttl := 3600
		client := &Client{RefreshTokenTTL: &ttl}
		assert.Equal(t, time.Duration(3600)*time.Second, svc.refreshTokenTTL(client))
	})

	t.Run("falls back to default", func(t *testing.T) {
		client := &Client{}
		assert.Equal(t, 7*24*time.Hour, svc.refreshTokenTTL(client))
	})
}

// ── TestHasGrant ────────────────────────────────────────────────────────────

func TestClientHasGrant(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		c := &Client{GrantTypes: pq.StringArray{"authorization_code", "refresh_token"}}
		assert.True(t, clientHasGrant(c, "refresh_token"))
	})

	t.Run("not found", func(t *testing.T) {
		c := &Client{GrantTypes: pq.StringArray{"authorization_code"}}
		assert.False(t, clientHasGrant(c, "client_credentials"))
	})

	t.Run("empty", func(t *testing.T) {
		c := &Client{}
		assert.False(t, clientHasGrant(c, "authorization_code"))
	})
}

// ── TestAuthenticateClient ──────────────────────────────────────────────────

func TestAuthenticateClient(t *testing.T) {
	t.Run("empty client_id", func(t *testing.T) {
		db, _ := newMockDB(t)
		svc := &oauthTokenService{db: db, authEventService: &mockAuthEventService{}}
		_, oerr := authenticateOAuthClient(svc.db, OAuthClientCredentials{})
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_client", oerr.Code)
	})

	t.Run("client not found", func(t *testing.T) {
		db, mock := newMockDB(t)
		expectClientNotFound(mock)
		svc := &oauthTokenService{db: db, authEventService: &mockAuthEventService{}}
		_, oerr := authenticateOAuthClient(svc.db, OAuthClientCredentials{ClientID: "unknown"})
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_client", oerr.Code)
	})

	t.Run("db error", func(t *testing.T) {
		db, mock := newMockDB(t)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).WillReturnError(errors.New("connection error"))
		svc := &oauthTokenService{db: db, authEventService: &mockAuthEventService{}}
		_, oerr := authenticateOAuthClient(svc.db, OAuthClientCredentials{ClientID: "x"})
		require.NotNil(t, oerr)
		assert.Equal(t, "server_error", oerr.Code)
	})

	t.Run("public client — no secret required", func(t *testing.T) {
		db, mock := newMockDB(t)
		expectClientLookup(mock, mockClientRows()) // token_endpoint_auth_method = "none"
		svc := &oauthTokenService{db: db, authEventService: &mockAuthEventService{}}
		client, oerr := authenticateOAuthClient(svc.db, OAuthClientCredentials{ClientID: "my-client"})
		require.Nil(t, oerr)
		require.NotNil(t, client)
	})

	t.Run("secret_basic — valid secret", func(t *testing.T) {
		db, mock := newMockDB(t)
		secret := "super-secret"
		hash, hashErr := security.HashClientSecret(context.Background(), secret)
		require.NoError(t, hashErr)
		rows := sqlmock.NewRows([]string{
			"client_id", "client_uuid", "tenant_id", "identity_provider_id", "name", "display_name",
			"client_type", "domain", "identifier", "secret_hash", "status",
			"is_default", "is_system", "token_endpoint_auth_method",
			"grant_types", "response_types", "access_token_ttl", "refresh_token_ttl",
			"require_consent", "created_at", "updated_at",
		}).AddRow(
			10, uuid.New(), 1, int64(100), "test-client", "Test Client",
			"m2m", nil, "my-client", hash, "active",
			false, false, "client_secret_basic",
			`{authorization_code}`, `{code}`, nil, nil,
			true, time.Now(), time.Now(),
		)
		expectClientLookup(mock, rows)
		svc := &oauthTokenService{db: db, authEventService: &mockAuthEventService{}}
		client, oerr := authenticateOAuthClient(svc.db, OAuthClientCredentials{ClientID: "my-client", ClientSecret: secret})
		require.Nil(t, oerr)
		require.NotNil(t, client)
	})

	t.Run("secret_basic — invalid secret", func(t *testing.T) {
		db, mock := newMockDB(t)
		secret := "super-secret"
		hash, hashErr := security.HashClientSecret(context.Background(), secret)
		require.NoError(t, hashErr)
		rows := sqlmock.NewRows([]string{
			"client_id", "client_uuid", "tenant_id", "identity_provider_id", "name", "display_name",
			"client_type", "domain", "identifier", "secret_hash", "status",
			"is_default", "is_system", "token_endpoint_auth_method",
			"grant_types", "response_types", "access_token_ttl", "refresh_token_ttl",
			"require_consent", "created_at", "updated_at",
		}).AddRow(
			10, uuid.New(), 1, int64(100), "test-client", "Test Client",
			"m2m", nil, "my-client", hash, "active",
			false, false, "client_secret_basic",
			`{authorization_code}`, `{code}`, nil, nil,
			true, time.Now(), time.Now(),
		)
		expectClientLookup(mock, rows)
		svc := &oauthTokenService{db: db, authEventService: &mockAuthEventService{}}
		_, oerr := authenticateOAuthClient(svc.db, OAuthClientCredentials{ClientID: "my-client", ClientSecret: "wrong"})
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_client", oerr.Code)
	})

	t.Run("secret_post — valid secret", func(t *testing.T) {
		db, mock := newMockDB(t)
		secret := "post-secret"
		hash, hashErr := security.HashClientSecret(context.Background(), secret)
		require.NoError(t, hashErr)
		rows := sqlmock.NewRows([]string{
			"client_id", "client_uuid", "tenant_id", "identity_provider_id", "name", "display_name",
			"client_type", "domain", "identifier", "secret_hash", "status",
			"is_default", "is_system", "token_endpoint_auth_method",
			"grant_types", "response_types", "access_token_ttl", "refresh_token_ttl",
			"require_consent", "created_at", "updated_at",
		}).AddRow(
			10, uuid.New(), 1, int64(100), "test-client", "Test Client",
			"m2m", nil, "my-client", hash, "active",
			false, false, "client_secret_post",
			`{authorization_code}`, `{code}`, nil, nil,
			true, time.Now(), time.Now(),
		)
		expectClientLookup(mock, rows)
		svc := &oauthTokenService{db: db, authEventService: &mockAuthEventService{}}
		client, oerr := authenticateOAuthClient(svc.db, OAuthClientCredentials{ClientID: "my-client", ClientSecret: secret})
		require.Nil(t, oerr)
		require.NotNil(t, client)
	})

	t.Run("unsupported auth method", func(t *testing.T) {
		db, mock := newMockDB(t)
		rows := sqlmock.NewRows([]string{
			"client_id", "client_uuid", "tenant_id", "identity_provider_id", "name", "display_name",
			"client_type", "domain", "identifier", "secret", "status",
			"is_default", "is_system", "token_endpoint_auth_method",
			"grant_types", "response_types", "access_token_ttl", "refresh_token_ttl",
			"require_consent", "created_at", "updated_at",
		}).AddRow(
			10, uuid.New(), 1, int64(100), "test-client", "Test Client",
			"m2m", nil, "my-client", nil, "active",
			false, false, "private_key_jwt",
			`{authorization_code}`, `{code}`, nil, nil,
			true, time.Now(), time.Now(),
		)
		expectClientLookup(mock, rows)
		svc := &oauthTokenService{db: db, authEventService: &mockAuthEventService{}}
		_, oerr := authenticateOAuthClient(svc.db, OAuthClientCredentials{ClientID: "my-client"})
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_client", oerr.Code)
	})
}
