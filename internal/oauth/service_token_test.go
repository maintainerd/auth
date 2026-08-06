package oauth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	jwtlib "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/maintainerd/maintainerd-auth/internal/platform/cache"
	"github.com/maintainerd/maintainerd-auth/internal/platform/crypto"
	"github.com/maintainerd/maintainerd-auth/internal/platform/jwt"
	"github.com/maintainerd/maintainerd-auth/internal/platform/security"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
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

// testM2MSecret is the plaintext an m2m fixture authenticates with. An m2m client
// is confidential: token_endpoint_auth_method "none" on a non-public client is
// refused, since client_id is public and would otherwise be enough to mint tokens.
const testM2MSecret = "m2m-client-secret-for-tests"

func testM2MSecretHash(t *testing.T) string {
	t.Helper()
	hash, err := security.HashClientSecret(context.Background(), testM2MSecret)
	require.NoError(t, err)
	return hash
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

// mockConfidentialClientRows is mockClientRows as a CONFIDENTIAL client: a web
// app that authenticates with a secret. Needed because mockClientRows is an SPA
// with token_endpoint_auth_method "none", and a public client is now required to
// use PKCE — so it can no longer stand in for flows that legitimately run
// without a code_challenge.
func mockConfidentialClientRows(t *testing.T) *sqlmock.Rows {
	t.Helper()
	return sqlmock.NewRows([]string{
		"client_id", "client_uuid", "tenant_id", "identity_provider_id", "name", "display_name",
		"client_type", "domain", "identifier", "secret", "status",
		"is_default", "is_system", "token_endpoint_auth_method",
		"grant_types", "response_types", "access_token_ttl", "refresh_token_ttl",
		"require_consent", "created_at", "updated_at",
	}).AddRow(
		10, uuid.New(), 1, int64(100), "test-client", "Test Client",
		"web", "https://auth.example.com", "my-client", testM2MSecretHash(t), "active",
		false, false, "client_secret_post",
		`{authorization_code,refresh_token}`, `{code}`, nil, nil,
		true, time.Now(), time.Now(),
	)
}

// mockTenantRows returns sqlmock rows for the Client.Tenant preload.
func mockTenantRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"tenant_id", "tenant_uuid", "name", "display_name", "description",
		"identifier", "status", "is_system", "created_at", "updated_at",
	}).AddRow(
		1, uuid.New(), "Test Tenant", "Test Tenant", "",
		"test-tenant", "active", false, time.Now(), time.Now(),
	)
}

// expectClientLookup sets up sqlmock expectations for findActiveClientByIdentifier.
// Matches the main clients query + Preload("Tenant"). Preload("Service") emits no
// query because the mocked client row has no service_id (nil belongs-to FK).
func expectClientLookup(mock sqlmock.Sqlmock, rows *sqlmock.Rows) {
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).WillReturnRows(rows)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).WillReturnRows(mockTenantRows())
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

	// PKCE binds per authorization request, not globally. /authorize only demands
	// a code_challenge when the client's RequirePKCE policy is on, so a
	// confidential client legitimately running without PKCE gets a code with no
	// challenge. The token endpoint used to reject any request lacking a
	// code_verifier before it even loaded the code, so that client could obtain a
	// code and then never redeem it.
	t.Run("authorization_code — a code WITH a challenge still requires a verifier", func(t *testing.T) {
		db, mock := newMockDB(t)
		expectClientLookup(mock, mockClientRows())
		svc := newOAuthTokenSvc(db, &mockClientRepo{},
			&mockOAuthAuthCodeRepo{
				findByCodeHashFn: func(string) (*OAuthAuthorizationCode, error) {
					return &OAuthAuthorizationCode{
						ClientID:            10,
						TenantID:            1,
						RedirectURI:         "https://example.com/callback",
						ExpiresAt:           time.Now().Add(time.Minute),
						CodeChallenge:       "a-stored-challenge",
						CodeChallengeMethod: "S256",
					}, nil
				},
			},
			&mockOAuthRefreshTokenRepo{}, &mockUserRepo{},
			&mockUserIdentityRepo{findByUserIDAndClientIDFn: func(_, _ int64) (*UserIdentity, error) { return nil, nil }},
			&mockAuthEventService{})

		_, oerr := svc.Exchange(ctx, OAuthTokenRequestDTO{
			GrantType:   "authorization_code",
			Code:        "code123",
			RedirectURI: "https://example.com/callback",
		}, OAuthClientCredentials{ClientID: "my-client"})
		require.NotNil(t, oerr)
		assert.Contains(t, oerr.Description, "code_verifier is required",
			"a public client must not be able to strip PKCE by omitting the verifier")
	})

	// MOVED TO A CONFIDENTIAL CLIENT: this case used mockClientRows, which is an
	// SPA with token_endpoint_auth_method "none". A public client redeeming a
	// challenge-less code is exactly the hole being closed (see the subtest
	// below), so the "challenge-less codes are fine" contract now has to be
	// demonstrated with a client that actually authenticates.
	t.Run("authorization_code — a confidential client's code WITHOUT a challenge does not demand a verifier", func(t *testing.T) {
		db, mock := newMockDB(t)
		expectClientLookup(mock, mockConfidentialClientRows(t))
		svc := newOAuthTokenSvc(db, &mockClientRepo{},
			&mockOAuthAuthCodeRepo{
				findByCodeHashFn: func(string) (*OAuthAuthorizationCode, error) {
					return &OAuthAuthorizationCode{
						ClientID:    10,
						TenantID:    1,
						RedirectURI: "https://example.com/callback",
						ExpiresAt:   time.Now().Add(time.Minute),
						// No PKCE was used for this authorization.
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
			ClientSecret: testM2MSecret,
		}, OAuthClientCredentials{ClientID: "my-client", ClientSecret: testM2MSecret})
		// It may still fail further down on unrelated mock plumbing; it must just
		// not fail on a missing PKCE verifier.
		if oerr != nil {
			assert.NotContains(t, oerr.Description, "code_verifier")
			assert.NotContains(t, oerr.Description, "PKCE")
		}
	})

	// A public client presents no credential at this endpoint, so without PKCE
	// the code is the only secret in the flow and whoever observes it — custom
	// scheme hijack, Referer leak, proxy log — redeems it. RFC 9700 §2.1.1.
	t.Run("authorization_code — a public client's code WITHOUT a challenge is refused", func(t *testing.T) {
		db, mock := newMockDB(t)
		expectClientLookup(mock, mockClientRows()) // spa + token_endpoint_auth_method "none"
		svc := newOAuthTokenSvc(db, &mockClientRepo{},
			&mockOAuthAuthCodeRepo{
				findByCodeHashFn: func(string) (*OAuthAuthorizationCode, error) {
					return &OAuthAuthorizationCode{
						ClientID:    10,
						TenantID:    1,
						RedirectURI: "https://example.com/callback",
						ExpiresAt:   time.Now().Add(time.Minute),
					}, nil
				},
			},
			&mockOAuthRefreshTokenRepo{}, &mockUserRepo{},
			&mockUserIdentityRepo{findByUserIDAndClientIDFn: func(_, _ int64) (*UserIdentity, error) { return nil, nil }},
			&mockAuthEventService{})

		_, oerr := svc.Exchange(ctx, OAuthTokenRequestDTO{
			GrantType:   "authorization_code",
			Code:        "code123",
			RedirectURI: "https://example.com/callback",
		}, OAuthClientCredentials{ClientID: "my-client"})
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_grant", oerr.Code)
		assert.Contains(t, oerr.Description, "PKCE is required")
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
						Used:     true,
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

	t.Run("authorization_code — tenant mismatch", func(t *testing.T) {
		db, mock := newMockDB(t)
		expectClientLookup(mock, mockClientRows())
		svc := newOAuthTokenSvc(db, &mockClientRepo{},
			&mockOAuthAuthCodeRepo{findByCodeHashFn: func(string) (*OAuthAuthorizationCode, error) {
				return &OAuthAuthorizationCode{ClientID: 10, TenantID: 99, ExpiresAt: time.Now().Add(time.Minute)}, nil
			}},
			&mockOAuthRefreshTokenRepo{}, &mockUserRepo{}, &mockUserIdentityRepo{}, &mockAuthEventService{})

		_, oerr := svc.Exchange(ctx, OAuthTokenRequestDTO{
			GrantType: "authorization_code", Code: "code123",
			RedirectURI: "https://example.com/callback", CodeVerifier: "abc",
		}, OAuthClientCredentials{ClientID: "my-client"})
		require.NotNil(t, oerr)
		assert.Contains(t, oerr.Description, "not issued to this tenant")
	})

	t.Run("authorization_code — redirect URI mismatch", func(t *testing.T) {
		db, mock := newMockDB(t)
		expectClientLookup(mock, mockClientRows())

		svc := newOAuthTokenSvc(db, &mockClientRepo{},
			&mockOAuthAuthCodeRepo{
				findByCodeHashFn: func(_ string) (*OAuthAuthorizationCode, error) {
					return &OAuthAuthorizationCode{
						ClientID:    10,
						TenantID:    1,
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
						TenantID:            1,
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
						TenantID:            1,
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
						TenantID:            1,
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
						TenantID:            1,
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
						Scope:                    parseScopeFields("openid profile offline_access"),
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

	t.Run("authorization_code — token generation error", func(t *testing.T) {
		initTestJWTKeysService(t)
		orig := oauthTokenGenerateAccessTokenWithOptionsContext
		defer func() { oauthTokenGenerateAccessTokenWithOptionsContext = orig }()
		oauthTokenGenerateAccessTokenWithOptionsContext = func(context.Context, string, string, string, string, string, string, *jwt.AccessTokenOptions) (string, error) {
			return "", errors.New("token error")
		}

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
						Scope:                    parseScopeFields("openid profile"),
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

		_, oerr := svc.Exchange(ctx, OAuthTokenRequestDTO{
			GrantType:    "authorization_code",
			Code:         "code123",
			RedirectURI:  "https://example.com/callback",
			CodeVerifier: verifier,
		}, OAuthClientCredentials{ClientID: "my-client"})
		require.NotNil(t, oerr)
		assert.Equal(t, "server_error", oerr.Code)
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

	t.Run("authorization_code — client lacks authorization_code grant", func(t *testing.T) {
		db, mock := newMockDB(t)
		rows := sqlmock.NewRows([]string{
			"client_id", "client_uuid", "tenant_id", "identity_provider_id", "name", "display_name",
			"client_type", "domain", "identifier", "secret", "status",
			"is_default", "is_system", "token_endpoint_auth_method",
			"grant_types", "response_types", "access_token_ttl", "refresh_token_ttl",
			"require_consent", "created_at", "updated_at",
		}).AddRow(
			10, uuid.New(), 1, int64(100), "test-client", "Test Client",
			"spa", "https://auth.example.com", "my-client", nil, "active",
			false, false, "none",
			`{client_credentials,refresh_token}`, `{code}`, nil, nil,
			true, time.Now(), time.Now(),
		)
		expectClientLookup(mock, rows)

		svc := newOAuthTokenSvc(db, &mockClientRepo{}, &mockOAuthAuthCodeRepo{}, &mockOAuthRefreshTokenRepo{}, &mockUserRepo{},
			&mockUserIdentityRepo{findByUserIDAndClientIDFn: func(_, _ int64) (*UserIdentity, error) { return nil, nil }},
			&mockAuthEventService{})

		_, oerr := svc.Exchange(ctx, OAuthTokenRequestDTO{
			GrantType:    "authorization_code",
			Code:         "code123",
			RedirectURI:  "https://example.com/callback",
			CodeVerifier: "abc",
		}, OAuthClientCredentials{ClientID: "my-client"})
		require.NotNil(t, oerr)
		assert.Equal(t, "unauthorized_client", oerr.Code)
		assert.Contains(t, oerr.Description, "authorization_code grant")
	})

	t.Run("authorization_code — scope not in client AllowedScopes", func(t *testing.T) {
		verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
		challenge := crypto.ComputeS256Challenge(verifier)

		db, mock := newMockDB(t)
		rows := sqlmock.NewRows([]string{
			"client_id", "client_uuid", "tenant_id", "identity_provider_id", "name", "display_name",
			"client_type", "domain", "identifier", "secret", "status",
			"is_default", "is_system", "token_endpoint_auth_method",
			"grant_types", "response_types", "access_token_ttl", "refresh_token_ttl",
			"require_consent", "allowed_scopes", "created_at", "updated_at",
		}).AddRow(
			10, uuid.New(), 1, int64(100), "test-client", "Test Client",
			"spa", "https://auth.example.com", "my-client", nil, "active",
			false, false, "none",
			`{authorization_code,refresh_token}`, `{code}`, nil, nil,
			true, pq.StringArray{"openid"}, time.Now(), time.Now(),
		)
		expectClientLookup(mock, rows)

		svc := newOAuthTokenSvc(db, &mockClientRepo{},
			&mockOAuthAuthCodeRepo{
				findByCodeHashFn: func(_ string) (*OAuthAuthorizationCode, error) {
					return &OAuthAuthorizationCode{
						ClientID:            10,
						TenantID:            1,
						UserID:              1,
						RedirectURI:         "https://example.com/callback",
						Scope:               parseScopeFields("openid profile"),
						CodeChallenge:       challenge,
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
			CodeVerifier: verifier,
		}, OAuthClientCredentials{ClientID: "my-client"})
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_scope", oerr.Code)
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

	t.Run("tenant mismatch", func(t *testing.T) {
		db, mock := newMockDB(t)
		expectClientLookup(mock, mockClientRows())
		svc := newOAuthTokenSvc(db, &mockClientRepo{}, &mockOAuthAuthCodeRepo{},
			&mockOAuthRefreshTokenRepo{findByTokenHashFn: func(string) (*OAuthRefreshToken, error) {
				return &OAuthRefreshToken{ClientID: 10, TenantID: 99, ExpiresAt: time.Now().Add(time.Minute)}, nil
			}},
			&mockUserRepo{}, &mockUserIdentityRepo{}, &mockAuthEventService{})

		_, oerr := svc.Exchange(ctx, OAuthTokenRequestDTO{
			GrantType: "refresh_token", RefreshToken: "some-token",
		}, OAuthClientCredentials{ClientID: "my-client"})
		require.NotNil(t, oerr)
		assert.Contains(t, oerr.Description, "not issued to this tenant")
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
						TenantID:  1,
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
						Scope:               parseScopeFields("openid profile offline_access"),
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
						Scope:               parseScopeFields("openid profile email"),
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
						TenantID:            1,
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
						TenantID:            1,
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

	t.Run("client auth fails — client not found", func(t *testing.T) {
		db, mock := newMockDB(t)
		expectClientNotFound(mock)

		svc := newOAuthTokenSvc(db, &mockClientRepo{}, &mockOAuthAuthCodeRepo{}, &mockOAuthRefreshTokenRepo{}, &mockUserRepo{},
			&mockUserIdentityRepo{findByUserIDAndClientIDFn: func(_, _ int64) (*UserIdentity, error) { return nil, nil }},
			&mockAuthEventService{})

		_, oerr := svc.Exchange(ctx, OAuthTokenRequestDTO{
			GrantType:    "refresh_token",
			RefreshToken: "some-token",
		}, OAuthClientCredentials{ClientID: "unknown"})
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_client", oerr.Code)
	})

	t.Run("client lacks refresh_token grant", func(t *testing.T) {
		db, mock := newMockDB(t)
		rows := sqlmock.NewRows([]string{
			"client_id", "client_uuid", "tenant_id", "identity_provider_id", "name", "display_name",
			"client_type", "domain", "identifier", "secret", "status",
			"is_default", "is_system", "token_endpoint_auth_method",
			"grant_types", "response_types", "access_token_ttl", "refresh_token_ttl",
			"require_consent", "created_at", "updated_at",
		}).AddRow(
			10, uuid.New(), 1, int64(100), "test-client", "Test Client",
			"spa", "https://auth.example.com", "my-client", nil, "active",
			false, false, "none",
			`{authorization_code}`, `{code}`, nil, nil,
			true, time.Now(), time.Now(),
		)
		expectClientLookup(mock, rows)

		svc := newOAuthTokenSvc(db, &mockClientRepo{}, &mockOAuthAuthCodeRepo{}, &mockOAuthRefreshTokenRepo{}, &mockUserRepo{},
			&mockUserIdentityRepo{findByUserIDAndClientIDFn: func(_, _ int64) (*UserIdentity, error) { return nil, nil }},
			&mockAuthEventService{})

		_, oerr := svc.Exchange(ctx, OAuthTokenRequestDTO{
			GrantType:    "refresh_token",
			RefreshToken: "some-token",
		}, OAuthClientCredentials{ClientID: "my-client"})
		require.NotNil(t, oerr)
		assert.Equal(t, "unauthorized_client", oerr.Code)
		assert.Contains(t, oerr.Description, "refresh_token grant")
	})

	t.Run("resolveUserSub fails in transaction", func(t *testing.T) {
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
						TenantID:            1,
						UserID:              1,
						ExpiresAt:           time.Now().Add(7 * 24 * time.Hour),
					}, nil
				},
				revokeByIDFn: func(_ int64) error { return nil },
			},
			&mockUserRepo{},
			&mockUserIdentityRepo{
				findByUserIDAndClientIDFn: func(_, _ int64) (*UserIdentity, error) {
					return nil, errors.New("identity lookup error")
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

	t.Run("requested scope not subset of stored scope", func(t *testing.T) {
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
						TenantID:            1,
						UserID:              1,
						Scope:               parseScopeFields("openid"),
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
					return &UserIdentity{Sub: "user-sub"}, nil
				},
			},
			&mockAuthEventService{})

		_, oerr := svc.Exchange(ctx, OAuthTokenRequestDTO{
			GrantType:    "refresh_token",
			RefreshToken: "some-token",
			Scope:        "admin",
		}, OAuthClientCredentials{ClientID: "my-client"})
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_scope", oerr.Code)
		assert.Contains(t, oerr.Description, "exceeds the original grant")
	})

	t.Run("requested scope not in client AllowedScopes", func(t *testing.T) {
		db, mock := newMockDB(t)
		rows := sqlmock.NewRows([]string{
			"client_id", "client_uuid", "tenant_id", "identity_provider_id", "name", "display_name",
			"client_type", "domain", "identifier", "secret", "status",
			"is_default", "is_system", "token_endpoint_auth_method",
			"grant_types", "response_types", "access_token_ttl", "refresh_token_ttl",
			"require_consent", "allowed_scopes", "created_at", "updated_at",
		}).AddRow(
			10, uuid.New(), 1, int64(100), "test-client", "Test Client",
			"spa", "https://auth.example.com", "my-client", nil, "active",
			false, false, "none",
			`{authorization_code,refresh_token}`, `{code}`, nil, nil,
			true, pq.StringArray{"openid"}, time.Now(), time.Now(),
		)
		expectClientLookup(mock, rows)
		mock.ExpectBegin()
		mock.ExpectRollback()

		svc := newOAuthTokenSvc(db, &mockClientRepo{}, &mockOAuthAuthCodeRepo{},
			&mockOAuthRefreshTokenRepo{
				findByTokenHashFn: func(_ string) (*OAuthRefreshToken, error) {
					return &OAuthRefreshToken{
						OAuthRefreshTokenID: 1,
						ClientID:            10,
						TenantID:            1,
						UserID:              1,
						Scope:               parseScopeFields("openid profile"),
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
					return &UserIdentity{Sub: "user-sub"}, nil
				},
			},
			&mockAuthEventService{})

		_, oerr := svc.Exchange(ctx, OAuthTokenRequestDTO{
			GrantType:    "refresh_token",
			RefreshToken: "some-token",
			Scope:        "profile",
		}, OAuthClientCredentials{ClientID: "my-client"})
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_scope", oerr.Code)
		assert.Contains(t, oerr.Description, "not allowed")
	})

	t.Run("Create refresh token fails in transaction", func(t *testing.T) {
		initTestJWTKeysService(t)
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
						TenantID:            1,
						UserID:              1,
						Scope:               parseScopeFields("openid profile"),
						ExpiresAt:           time.Now().Add(7 * 24 * time.Hour),
					}, nil
				},
				revokeByIDFn: func(_ int64) error { return nil },
				createFn: func(_ *OAuthRefreshToken) (*OAuthRefreshToken, error) {
					return nil, errors.New("create error")
				},
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

		_, oerr := svc.Exchange(ctx, OAuthTokenRequestDTO{
			GrantType:    "refresh_token",
			RefreshToken: "some-token",
		}, OAuthClientCredentials{ClientID: "my-client"})
		require.NotNil(t, oerr)
		assert.Equal(t, "server_error", oerr.Code)
	})

	t.Run("GenerateRandomString fails in transaction", func(t *testing.T) {
		initTestJWTKeysService(t)
		orig := oauthTokenGenerateRandomString
		defer func() { oauthTokenGenerateRandomString = orig }()
		oauthTokenGenerateRandomString = func(int) (string, error) {
			return "", errors.New("random error")
		}

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
						TenantID:            1,
						UserID:              1,
						Scope:               parseScopeFields("openid profile"),
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

		_, oerr := svc.Exchange(ctx, OAuthTokenRequestDTO{
			GrantType:    "refresh_token",
			RefreshToken: "some-token",
		}, OAuthClientCredentials{ClientID: "my-client"})
		require.NotNil(t, oerr)
		assert.Equal(t, "server_error", oerr.Code)
	})

	t.Run("generateTokens returns OAuth error in transaction", func(t *testing.T) {
		initTestJWTKeysService(t)
		orig := oauthTokenGenerateAccessTokenWithOptionsContext
		defer func() { oauthTokenGenerateAccessTokenWithOptionsContext = orig }()
		oauthTokenGenerateAccessTokenWithOptionsContext = func(context.Context, string, string, string, string, string, string, *jwt.AccessTokenOptions) (string, error) {
			return "", errors.New("token error")
		}

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
						TenantID:            1,
						UserID:              1,
						Scope:               parseScopeFields("openid profile"),
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
			"client_type", "domain", "identifier", "secret_hash", "status",
			"is_default", "is_system", "token_endpoint_auth_method",
			"grant_types", "response_types", "access_token_ttl", "refresh_token_ttl",
			"require_consent", "created_at", "updated_at",
		}).AddRow(
			10, uuid.New(), 1, int64(100), "test-client", "Test Client",
			"m2m", nil, "m2m-client", testM2MSecretHash(t), "active",
			false, false, "client_secret_basic",
			`{authorization_code}`, `{code}`, nil, nil,
			false, time.Now(), time.Now(),
		)
		expectClientLookup(mock, rows)

		svc := newOAuthTokenSvc(db, &mockClientRepo{}, &mockOAuthAuthCodeRepo{}, &mockOAuthRefreshTokenRepo{}, &mockUserRepo{},
			&mockUserIdentityRepo{findByUserIDAndClientIDFn: func(_, _ int64) (*UserIdentity, error) { return nil, nil }},
			&mockAuthEventService{})

		_, oerr := svc.Exchange(ctx, OAuthTokenRequestDTO{
			GrantType: "client_credentials",
		}, OAuthClientCredentials{ClientID: "m2m-client", ClientSecret: testM2MSecret})
		require.NotNil(t, oerr)
		assert.Equal(t, "unauthorized_client", oerr.Code)
	})

	t.Run("success", func(t *testing.T) {
		initTestJWTKeysService(t)
		db, mock := newMockDB(t)
		rows := sqlmock.NewRows([]string{
			"client_id", "client_uuid", "tenant_id", "identity_provider_id", "name", "display_name",
			"client_type", "domain", "identifier", "secret_hash", "status",
			"is_default", "is_system", "token_endpoint_auth_method",
			"grant_types", "response_types", "access_token_ttl", "refresh_token_ttl",
			"require_consent", "created_at", "updated_at",
		}).AddRow(
			10, uuid.New(), 1, int64(100), "m2m-client", "M2M Client",
			"m2m", "https://auth.example.com", "m2m-client", testM2MSecretHash(t), "active",
			false, false, "client_secret_basic",
			`{client_credentials}`, `{}`, nil, nil,
			false, time.Now(), time.Now(),
		)
		expectClientLookup(mock, rows)

		svc := newOAuthTokenSvc(db, &mockClientRepo{}, &mockOAuthAuthCodeRepo{}, &mockOAuthRefreshTokenRepo{}, &mockUserRepo{},
			&mockUserIdentityRepo{findByUserIDAndClientIDFn: func(_, _ int64) (*UserIdentity, error) { return nil, nil }},
			&mockAuthEventService{})

		result, oerr := svc.Exchange(ctx, OAuthTokenRequestDTO{
			GrantType: "client_credentials",
		}, OAuthClientCredentials{ClientID: "m2m-client", ClientSecret: testM2MSecret})
		require.Nil(t, oerr)
		assert.NotEmpty(t, result.AccessToken)
		assert.Equal(t, "Bearer", result.TokenType)
		assert.Empty(t, result.RefreshToken) // client_credentials grant has no refresh token
		assert.Empty(t, result.IDToken)
	})

	t.Run("success with linked service principal", func(t *testing.T) {
		initTestJWTKeysService(t)
		db, mock := newMockDB(t)
		rows := sqlmock.NewRows([]string{
			"client_id", "client_uuid", "tenant_id", "service_id", "identity_provider_id", "name", "display_name",
			"client_type", "domain", "identifier", "secret_hash", "status",
			"is_default", "is_system", "token_endpoint_auth_method",
			"grant_types", "response_types", "access_token_ttl", "refresh_token_ttl",
			"require_consent", "created_at", "updated_at",
		}).AddRow(
			10, uuid.New(), 1, int64(42), int64(100), "m2m-client", "M2M Client",
			"m2m", "https://auth.example.com", "m2m-client", testM2MSecretHash(t), "active",
			false, false, "client_secret_basic",
			`{client_credentials}`, `{}`, nil, nil,
			false, time.Now(), time.Now(),
		)
		mock.MatchExpectationsInOrder(false)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).WillReturnRows(rows)
		mock.ExpectQuery(`FROM "tenants"`).WillReturnRows(mockTenantRows())
		mock.ExpectQuery(`FROM "services"`).WillReturnRows(sqlmock.NewRows([]string{"service_id", "name", "status"}).AddRow(42, "serviceA", "active"))

		svc := newOAuthTokenSvc(db, &mockClientRepo{}, &mockOAuthAuthCodeRepo{}, &mockOAuthRefreshTokenRepo{}, &mockUserRepo{},
			&mockUserIdentityRepo{findByUserIDAndClientIDFn: func(_, _ int64) (*UserIdentity, error) { return nil, nil }},
			&mockAuthEventService{})

		result, oerr := svc.Exchange(ctx, OAuthTokenRequestDTO{
			GrantType: "client_credentials",
		}, OAuthClientCredentials{ClientID: "m2m-client", ClientSecret: testM2MSecret})
		require.Nil(t, oerr)
		claims, err := jwt.ValidateToken(result.AccessToken)
		require.NoError(t, err)
		assert.Equal(t, "serviceA", claims["sub"])
		assert.Equal(t, "serviceA", claims["svc"])
		assert.Equal(t, "service", claims["sub_type"])
	})

	t.Run("success with custom access token ttl", func(t *testing.T) {
		initTestJWTKeysService(t)
		db, mock := newMockDB(t)
		rows := sqlmock.NewRows([]string{
			"client_id", "client_uuid", "tenant_id", "identity_provider_id", "name", "display_name",
			"client_type", "domain", "identifier", "secret_hash", "status",
			"is_default", "is_system", "token_endpoint_auth_method",
			"grant_types", "response_types", "access_token_ttl", "refresh_token_ttl",
			"require_consent", "created_at", "updated_at",
		}).AddRow(
			10, uuid.New(), 1, int64(100), "m2m-client", "M2M Client",
			"m2m", "https://auth.example.com", "m2m-client", testM2MSecretHash(t), "active",
			false, false, "client_secret_basic",
			`{client_credentials}`, `{}`, 3600, nil,
			false, time.Now(), time.Now(),
		)
		expectClientLookup(mock, rows)

		svc := newOAuthTokenSvc(db, &mockClientRepo{}, &mockOAuthAuthCodeRepo{}, &mockOAuthRefreshTokenRepo{}, &mockUserRepo{},
			&mockUserIdentityRepo{findByUserIDAndClientIDFn: func(_, _ int64) (*UserIdentity, error) { return nil, nil }},
			&mockAuthEventService{})

		result, oerr := svc.Exchange(ctx, OAuthTokenRequestDTO{
			GrantType: "client_credentials",
		}, OAuthClientCredentials{ClientID: "m2m-client", ClientSecret: testM2MSecret})
		require.Nil(t, oerr)
		assert.Equal(t, int64(900), result.ExpiresIn)
	})

	t.Run("client auth failure", func(t *testing.T) {
		db, mock := newMockDB(t)
		expectClientNotFound(mock)

		svc := newOAuthTokenSvc(db, &mockClientRepo{}, &mockOAuthAuthCodeRepo{}, &mockOAuthRefreshTokenRepo{}, &mockUserRepo{},
			&mockUserIdentityRepo{findByUserIDAndClientIDFn: func(_, _ int64) (*UserIdentity, error) { return nil, nil }},
			&mockAuthEventService{})

		_, oerr := svc.Exchange(ctx, OAuthTokenRequestDTO{
			GrantType: "client_credentials",
		}, OAuthClientCredentials{ClientID: "unknown"})
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_client", oerr.Code)
	})

	t.Run("access token generation error", func(t *testing.T) {
		initTestJWTKeysService(t)
		orig := oauthTokenGenerateAccessTokenWithOptionsContext
		defer func() { oauthTokenGenerateAccessTokenWithOptionsContext = orig }()
		oauthTokenGenerateAccessTokenWithOptionsContext = func(context.Context, string, string, string, string, string, string, *jwt.AccessTokenOptions) (string, error) {
			return "", errors.New("token error")
		}

		db, mock := newMockDB(t)
		rows := sqlmock.NewRows([]string{
			"client_id", "client_uuid", "tenant_id", "identity_provider_id", "name", "display_name",
			"client_type", "domain", "identifier", "secret_hash", "status",
			"is_default", "is_system", "token_endpoint_auth_method",
			"grant_types", "response_types", "access_token_ttl", "refresh_token_ttl",
			"require_consent", "created_at", "updated_at",
		}).AddRow(
			10, uuid.New(), 1, int64(100), "m2m-client", "M2M Client",
			"m2m", "https://auth.example.com", "m2m-client", testM2MSecretHash(t), "active",
			false, false, "client_secret_basic",
			`{client_credentials}`, `{}`, nil, nil,
			false, time.Now(), time.Now(),
		)
		expectClientLookup(mock, rows)

		svc := newOAuthTokenSvc(db, &mockClientRepo{}, &mockOAuthAuthCodeRepo{}, &mockOAuthRefreshTokenRepo{}, &mockUserRepo{},
			&mockUserIdentityRepo{findByUserIDAndClientIDFn: func(_, _ int64) (*UserIdentity, error) { return nil, nil }},
			&mockAuthEventService{})

		_, oerr := svc.Exchange(ctx, OAuthTokenRequestDTO{
			GrantType: "client_credentials",
		}, OAuthClientCredentials{ClientID: "m2m-client", ClientSecret: testM2MSecret})
		require.NotNil(t, oerr)
		assert.Equal(t, "server_error", oerr.Code)
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
		jwt.ResetJTIChecker()
		t.Cleanup(jwt.ResetJTIChecker)

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
		jwt.ResetJTIChecker()
		t.Cleanup(jwt.ResetJTIChecker)

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

	t.Run("nil jtiDenylist — no-op", func(t *testing.T) {
		initTestJWTKeysService(t)
		jwt.ResetJTIChecker()
		t.Cleanup(jwt.ResetJTIChecker)

		db, mock := newMockDB(t)
		expectClientLookup(mock, mockClientRows())

		token, err := jwt.GenerateAccessToken("user-sub", "openid", "https://auth.example.com", "my-client", "my-client", "default-provider")
		require.NoError(t, err)

		svc := &oauthTokenService{
			db:               db,
			refreshTokenRepo: &mockOAuthRefreshTokenRepo{},
			authEventService: &mockAuthEventService{},
			jtiDenylist:      nil,
		}

		oerr := svc.Revoke(ctx, OAuthRevokeRequestDTO{Token: token}, OAuthClientCredentials{ClientID: "my-client"})
		require.Nil(t, oerr)
	})

	t.Run("access token without jti is skipped", func(t *testing.T) {
		orig := oauthTokenValidateTokenWithContext
		defer func() { oauthTokenValidateTokenWithContext = orig }()
		oauthTokenValidateTokenWithContext = func(context.Context, string) (jwtlib.MapClaims, error) {
			return jwtlib.MapClaims{
				"token_type": "access_token",
				"client_id":  "my-client",
				"exp":        float64(time.Now().Add(time.Hour).Unix()),
			}, nil
		}

		clientID := "my-client"
		denylist := &recordingJTIDenylister{}
		svc := &oauthTokenService{
			refreshTokenRepo: &mockOAuthRefreshTokenRepo{},
			jtiDenylist:      denylist,
		}

		oerr := svc.revokeAccessToken(ctx, "token-without-jti", &Client{Identifier: &clientID})

		require.Nil(t, oerr)
		assert.Empty(t, denylist.jti)
	})

	t.Run("expired access token is skipped", func(t *testing.T) {
		orig := oauthTokenValidateTokenWithContext
		defer func() { oauthTokenValidateTokenWithContext = orig }()
		oauthTokenValidateTokenWithContext = func(context.Context, string) (jwtlib.MapClaims, error) {
			return jwtlib.MapClaims{
				"token_type": "access_token",
				"client_id":  "my-client",
				"jti":        "jti-123",
				"exp":        float64(time.Now().Add(-time.Hour).Unix()),
			}, nil
		}

		clientID := "my-client"
		denylist := &recordingJTIDenylister{}
		svc := &oauthTokenService{
			refreshTokenRepo: &mockOAuthRefreshTokenRepo{},
			jtiDenylist:      denylist,
		}

		oerr := svc.revokeAccessToken(ctx, "expired-token", &Client{Identifier: &clientID})

		require.Nil(t, oerr)
		assert.Empty(t, denylist.jti)
	})

	t.Run("id_token is skipped", func(t *testing.T) {
		initTestJWTKeysService(t)
		jwt.ResetJTIChecker()
		t.Cleanup(jwt.ResetJTIChecker)

		db, mock := newMockDB(t)
		expectClientLookup(mock, mockClientRows())

		originalGenerateIDToken := jwt.GenerateIDToken
		jwt.GenerateIDToken = func(userUUID, issuer, clientID, providerID string, profile *jwt.UserProfile, nonce string, params *jwt.IDTokenParams) (string, error) {
			return jwt.GenerateIDTokenWithContext(ctx, userUUID, issuer, clientID, providerID, profile, nonce, params)
		}
		defer func() { jwt.GenerateIDToken = originalGenerateIDToken }()

		idToken, err := jwt.GenerateIDToken("user-sub", "https://auth.example.com", "my-client", "default-provider", nil, "", nil)
		require.NoError(t, err)

		denylist := &recordingJTIDenylister{}
		svc := &oauthTokenService{
			db:               db,
			refreshTokenRepo: &mockOAuthRefreshTokenRepo{},
			authEventService: &mockAuthEventService{},
			jtiDenylist:      denylist,
		}

		oerr := svc.Revoke(ctx, OAuthRevokeRequestDTO{Token: idToken}, OAuthClientCredentials{ClientID: "my-client"})
		require.Nil(t, oerr)
		assert.Empty(t, denylist.jti)
	})

	t.Run("client ID mismatch", func(t *testing.T) {
		initTestJWTKeysService(t)
		jwt.ResetJTIChecker()
		t.Cleanup(jwt.ResetJTIChecker)

		db, mock := newMockDB(t)
		rows := sqlmock.NewRows([]string{
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
		expectClientLookup(mock, rows)

		token, err := jwt.GenerateAccessToken("user-sub", "openid", "https://auth.example.com", "other-client", "other-client", "default-provider")
		require.NoError(t, err)

		denylist := &recordingJTIDenylister{}
		svc := &oauthTokenService{
			db:               db,
			refreshTokenRepo: &mockOAuthRefreshTokenRepo{},
			authEventService: &mockAuthEventService{},
			jtiDenylist:      denylist,
		}

		oerr := svc.Revoke(ctx, OAuthRevokeRequestDTO{Token: token}, OAuthClientCredentials{ClientID: "my-client"})
		require.Nil(t, oerr)
		assert.Empty(t, denylist.jti)
	})

	t.Run("client.Identifier nil — skipped", func(t *testing.T) {
		initTestJWTKeysService(t)
		jwt.ResetJTIChecker()
		t.Cleanup(jwt.ResetJTIChecker)

		db, mock := newMockDB(t)
		rows := sqlmock.NewRows([]string{
			"client_id", "client_uuid", "tenant_id", "identity_provider_id", "name", "display_name",
			"client_type", "domain", "identifier", "secret", "status",
			"is_default", "is_system", "token_endpoint_auth_method",
			"grant_types", "response_types", "access_token_ttl", "refresh_token_ttl",
			"require_consent", "created_at", "updated_at",
		}).AddRow(
			10, uuid.New(), 1, int64(100), "test-client", "Test Client",
			"spa", "https://auth.example.com", nil, nil, "active",
			false, false, "none",
			`{authorization_code,refresh_token}`, `{code}`, nil, nil,
			true, time.Now(), time.Now(),
		)
		expectClientLookup(mock, rows)

		token, err := jwt.GenerateAccessToken("user-sub", "openid", "https://auth.example.com", "my-client", "my-client", "default-provider")
		require.NoError(t, err)

		denylist := &recordingJTIDenylister{}
		svc := &oauthTokenService{
			db:               db,
			refreshTokenRepo: &mockOAuthRefreshTokenRepo{},
			authEventService: &mockAuthEventService{},
			jtiDenylist:      denylist,
		}

		oerr := svc.Revoke(ctx, OAuthRevokeRequestDTO{Token: token}, OAuthClientCredentials{ClientID: "my-client"})
		require.Nil(t, oerr)
		assert.Empty(t, denylist.jti)
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
						UserID:   1,
						ClientID: 10,
						// Introspection is now scoped to the caller's tenant, so the
						// stored row has to name the tenant the mock client belongs to.
						TenantID:  1,
						Scope:     parseScopeFields("openid"),
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
						TenantID:  1,
						Scope:     parseScopeFields("openid"),
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

	// RFC 7662 §2.2. All tenants' tokens are signed with the same key, so a valid
	// signature says nothing about which tenant a token came from; without an
	// explicit tenant check this endpoint reported sub/scope/client_id for any
	// token this server ever issued.
	t.Run("refresh token belonging to another tenant is reported inactive", func(t *testing.T) {
		db, mock := newMockDB(t)
		expectClientLookup(mock, mockClientRows()) // caller is in tenant 1
		svc := newOAuthTokenSvc(db, &mockClientRepo{}, &mockOAuthAuthCodeRepo{},
			&mockOAuthRefreshTokenRepo{
				findByTokenHashFn: func(_ string) (*OAuthRefreshToken, error) {
					return &OAuthRefreshToken{
						UserID:    77,
						ClientID:  99,
						TenantID:  2, // a different tenant
						Scope:     parseScopeFields("openid"),
						ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
						CreatedAt: time.Now(),
					}, nil
				},
			},
			&mockUserRepo{},
			&mockUserIdentityRepo{
				findByUserIDAndClientIDFn: func(_, _ int64) (*UserIdentity, error) {
					return &UserIdentity{Sub: "other-tenant-user"}, nil
				},
			},
			&mockAuthEventService{})

		result, oerr := svc.Introspect(ctx, OAuthIntrospectRequestDTO{Token: "rt-token"}, OAuthClientCredentials{ClientID: "my-client"})
		require.Nil(t, oerr)
		assert.False(t, result.Active)
		assert.Empty(t, result.Sub, "another tenant's subject must not leak")
		assert.Empty(t, result.Scope)
	})

	t.Run("JWT issued to a client in another tenant is reported inactive", func(t *testing.T) {
		initTestJWTKeysService(t)
		db, mock := newMockDB(t)
		// 1: the caller authenticating (my-client, tenant 1).
		expectClientLookup(mock, mockClientRows())
		// 2: resolving the tenant of the token's own client — not found, so it
		// cannot be attributed to the caller's tenant.
		expectClientNotFound(mock)

		token, err := jwt.GenerateAccessToken("victim-sub", "openid admin", "https://other.example.com", "other-client", "other-client", "default-provider")
		require.NoError(t, err)

		svc := newOAuthTokenSvc(db, &mockClientRepo{}, &mockOAuthAuthCodeRepo{},
			&mockOAuthRefreshTokenRepo{},
			&mockUserRepo{},
			&mockUserIdentityRepo{findByUserIDAndClientIDFn: func(_, _ int64) (*UserIdentity, error) { return nil, nil }},
			&mockAuthEventService{})

		result, oerr := svc.Introspect(ctx, OAuthIntrospectRequestDTO{Token: token}, OAuthClientCredentials{ClientID: "my-client"})
		require.Nil(t, oerr)
		assert.False(t, result.Active)
		assert.Empty(t, result.Sub, "another tenant's subject must not leak")
		assert.Empty(t, result.Scope, "another tenant's scopes must not leak")
		assert.Empty(t, result.ClientID)
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

	t.Run("bad client auth", func(t *testing.T) {
		db, mock := newMockDB(t)
		expectClientNotFound(mock)
		svc := newOAuthTokenSvc(db, &mockClientRepo{}, &mockOAuthAuthCodeRepo{}, &mockOAuthRefreshTokenRepo{}, &mockUserRepo{},
			&mockUserIdentityRepo{findByUserIDAndClientIDFn: func(_, _ int64) (*UserIdentity, error) { return nil, nil }},
			&mockAuthEventService{})

		_, oerr := svc.Introspect(ctx, OAuthIntrospectRequestDTO{Token: "any-token"}, OAuthClientCredentials{ClientID: "unknown"})
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_client", oerr.Code)
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
		assert.Equal(t, 30*24*time.Hour, svc.refreshTokenTTL(client))
	})
}

func TestOAuthTokenService_GenerateTokens(t *testing.T) {
	domain := "https://auth.example.com"
	identifier := "my-client"
	fullClient := &Client{
		ClientID:   10,
		TenantID:   1,
		Domain:     &domain,
		Identifier: &identifier,
		IdentityProvider: &IdentityProvider{
			Identifier: "default-provider",
		},
	}
	user := &User{
		UserUUID:        uuid.New(),
		Email:           "test@example.com",
		IsEmailVerified: true,
		Fullname:        "Test User",
	}
	svc := &oauthTokenService{}

	t.Run("access token auth context", func(t *testing.T) {
		initTestJWTKeysService(t)
		result, oerr := svc.generateTokens(context.Background(), "user-sub", user, fullClient, "openid profile", nil, "", false, nil, "")
		require.Nil(t, oerr)
		require.NotNil(t, result)

		claims, err := jwt.ValidateToken(result.AccessToken)
		require.NoError(t, err)
		assert.Equal(t, jwt.ACRLevel1, claims["acr"])
		assert.ElementsMatch(t, []any{jwt.AMRPassword}, claims["amr"])
	})

	t.Run("nil Domain, nil Identifier, nil IdentityProvider", func(t *testing.T) {
		initTestJWTKeysService(t)
		nilClient := &Client{ClientID: 10, TenantID: 1}
		_, oerr := svc.generateTokens(context.Background(), "user-sub", user, nilClient, "openid profile", nil, "", false, nil, "")
		require.NotNil(t, oerr)
		assert.Equal(t, "server_error", oerr.Code)
	})

	t.Run("non-empty dpopThumbprint", func(t *testing.T) {
		initTestJWTKeysService(t)
		result, oerr := svc.generateTokens(context.Background(), "user-sub", user, fullClient, "openid profile", nil, "thumbprint123", false, nil, "")
		require.Nil(t, oerr)
		require.NotNil(t, result)
		assert.Equal(t, "DPoP", result.TokenType)
		claims, err := jwt.ValidateToken(result.AccessToken)
		require.NoError(t, err)
		assert.Contains(t, claims, "cnf")
	})

	t.Run("with nonce", func(t *testing.T) {
		initTestJWTKeysService(t)
		nonce := "nonce-abc-123"
		result, oerr := svc.generateTokens(context.Background(), "user-sub", user, fullClient, "openid profile", &nonce, "", false, nil, "")
		require.Nil(t, oerr)
		require.NotNil(t, result)
		assert.NotEmpty(t, result.IDToken)
	})

	t.Run("without offline_access scope", func(t *testing.T) {
		initTestJWTKeysService(t)
		result, oerr := svc.generateTokens(context.Background(), "user-sub", user, fullClient, "openid profile", nil, "", false, nil, "")
		require.Nil(t, oerr)
		require.NotNil(t, result)
		assert.NotEmpty(t, result.AccessToken)
		assert.NotEmpty(t, result.IDToken)
		assert.Empty(t, result.RefreshToken)
	})

	t.Run("custom AccessTokenTTL", func(t *testing.T) {
		initTestJWTKeysService(t)
		ttl := 1800
		ttlClient := &Client{
			ClientID:       10,
			TenantID:       1,
			Domain:         &domain,
			Identifier:     &identifier,
			AccessTokenTTL: &ttl,
			IdentityProvider: &IdentityProvider{
				Identifier: "default-provider",
			},
		}
		result, oerr := svc.generateTokens(context.Background(), "user-sub", user, ttlClient, "openid profile", nil, "", false, nil, "")
		require.Nil(t, oerr)
		require.NotNil(t, result)
		assert.Equal(t, int64(900), result.ExpiresIn)
	})

	t.Run("JWT generation error — keys not initialized", func(t *testing.T) {
		jwt.ResetJWTKeys()
		defer jwt.ResetJWTKeys()

		_, oerr := svc.generateTokens(context.Background(), "user-sub", user, fullClient, "openid profile", nil, "", false, nil, "")
		require.NotNil(t, oerr)
		assert.Equal(t, "server_error", oerr.Code)
	})

	t.Run("ID token generation error", func(t *testing.T) {
		initTestJWTKeysService(t)
		orig := oauthTokenGenerateIDTokenWithContext
		defer func() { oauthTokenGenerateIDTokenWithContext = orig }()
		oauthTokenGenerateIDTokenWithContext = func(context.Context, string, string, string, string, *jwt.UserProfile, string, *jwt.IDTokenParams) (string, error) {
			return "", errors.New("id token error")
		}

		_, oerr := svc.generateTokens(context.Background(), "user-sub", user, fullClient, "openid profile", nil, "", false, nil, "")
		require.NotNil(t, oerr)
		assert.Equal(t, "server_error", oerr.Code)
	})

	t.Run("refresh token random error", func(t *testing.T) {
		initTestJWTKeysService(t)
		orig := oauthTokenGenerateRandomString
		defer func() { oauthTokenGenerateRandomString = orig }()
		oauthTokenGenerateRandomString = func(int) (string, error) {
			return "", errors.New("random error")
		}

		_, oerr := svc.generateTokens(context.Background(), "user-sub", user, fullClient, "openid offline_access", nil, "", true, nil, "")
		require.NotNil(t, oerr)
		assert.Equal(t, "server_error", oerr.Code)
	})

	t.Run("refresh token create error", func(t *testing.T) {
		initTestJWTKeysService(t)
		svc := &oauthTokenService{
			refreshTokenRepo: &mockOAuthRefreshTokenRepo{
				createFn: func(_ *OAuthRefreshToken) (*OAuthRefreshToken, error) {
					return nil, errors.New("create error")
				},
			},
		}

		_, oerr := svc.generateTokens(context.Background(), "user-sub", user, fullClient, "openid offline_access", nil, "", true, nil, "")
		require.NotNil(t, oerr)
		assert.Equal(t, "server_error", oerr.Code)
	})
}

// ── TestTokenRemainingTTL ──────────────────────────────────────────────────

func TestTokenRemainingTTL(t *testing.T) {
	t.Run("float64 exp", func(t *testing.T) {
		now := time.Now().Unix()
		ttl := tokenRemainingTTL(float64(now + 3600))
		assert.Positive(t, ttl)
		assert.LessOrEqual(t, ttl, time.Duration(3600)*time.Second)
	})

	t.Run("int64 exp", func(t *testing.T) {
		now := time.Now().Unix()
		ttl := tokenRemainingTTL(int64(now + 7200))
		assert.Positive(t, ttl)
	})

	t.Run("int exp", func(t *testing.T) {
		now := time.Now().Unix()
		ttl := tokenRemainingTTL(int(now + 1800))
		assert.Positive(t, ttl)
	})

	t.Run("json.Number valid", func(t *testing.T) {
		now := time.Now().Unix()
		jn := json.Number(fmt.Sprintf("%d", now+600))
		ttl := tokenRemainingTTL(jn)
		assert.Positive(t, ttl)
	})

	t.Run("json.Number parse error", func(t *testing.T) {
		ttl := tokenRemainingTTL(json.Number("not-a-number"))
		assert.Equal(t, time.Duration(0), ttl)
	})

	t.Run("default case", func(t *testing.T) {
		ttl := tokenRemainingTTL("string-not-supported")
		assert.Equal(t, time.Duration(0), ttl)
	})

	t.Run("negative TTL for expired token", func(t *testing.T) {
		ttl := tokenRemainingTTL(float64(time.Now().Unix() - 3600))
		assert.Negative(t, ttl)
	})
}

// ── TestBuildUserProfile ────────────────────────────────────────────────────

func TestBuildUserProfile(t *testing.T) {
	t.Run("nil Profile", func(t *testing.T) {
		user := &User{
			Email:           "test@example.com",
			IsEmailVerified: true,
			Phone:           "+1234567890",
			IsPhoneVerified: true,
			Fullname:        "Fallback Name",
		}
		p := buildUserProfile(user)
		assert.Equal(t, "test@example.com", p.Email)
		assert.True(t, p.EmailVerified)
		assert.Equal(t, "+1234567890", p.Phone)
		assert.True(t, p.PhoneVerified)
		assert.Equal(t, "Fallback Name", p.Name)
	})

	t.Run("full Profile with LastName and ProfileURL", func(t *testing.T) {
		lastName := "Doe"
		profileURL := "https://example.com/avatar.jpg"
		user := &User{
			Email:           "john@example.com",
			IsEmailVerified: true,
			Fullname:        "Fallback Name",
			Profile: &Profile{
				FirstName:  "John",
				LastName:   &lastName,
				ProfileURL: &profileURL,
			},
		}
		p := buildUserProfile(user)
		assert.Equal(t, "John", p.FirstName)
		assert.Equal(t, "Doe", p.LastName)
		assert.Equal(t, "https://example.com/avatar.jpg", p.Picture)
		assert.Equal(t, "John Doe", p.Name)
	})

	t.Run("partial Profile without LastName", func(t *testing.T) {
		user := &User{
			Email:    "jane@example.com",
			Fullname: "Fallback",
			Profile: &Profile{
				FirstName: "Jane",
			},
		}
		p := buildUserProfile(user)
		assert.Equal(t, "Jane", p.FirstName)
		assert.Equal(t, "", p.LastName)
		assert.Equal(t, "Jane", p.Name)
	})

	t.Run("Profile without ProfileURL", func(t *testing.T) {
		lastName := "Smith"
		user := &User{
			Email:    "bob@example.com",
			Fullname: "Fallback",
			Profile: &Profile{
				FirstName: "Bob",
				LastName:  &lastName,
			},
		}
		p := buildUserProfile(user)
		assert.Equal(t, "", p.Picture)
		assert.Equal(t, "Bob Smith", p.Name)
	})
}

// ── TestBuildIDTokenParams ──────────────────────────────────────────────────

func TestBuildIDTokenParams(t *testing.T) {
	t.Run("empty scope", func(t *testing.T) {
		client := &Client{}
		params := buildIDTokenParams("", client)
		assert.Nil(t, params)
	})

	t.Run("populated ScopeClaimMappings and ClaimMappers", func(t *testing.T) {
		mappings := `{"openid":["sub"],"profile":["name"]}`
		rawMappings := datatypes.JSON(mappings)
		claims := `{"custom_claim":"value"}`
		rawClaims := datatypes.JSON(claims)

		client := &Client{
			ScopeClaimMappings: rawMappings,
			ClaimMappers:       rawClaims,
		}

		params := buildIDTokenParams("openid profile", client)
		require.NotNil(t, params)
		assert.Equal(t, []string{"openid", "profile"}, params.RequestedScopes)
		assert.Equal(t, []string{jwt.AMRPassword}, params.AMR)
		assert.Equal(t, jwt.ACRLevel1, params.ACR)
		require.NotNil(t, params.ScopeClaimMappings)
		assert.Equal(t, []string{"sub"}, params.ScopeClaimMappings["openid"])
		require.NotNil(t, params.ExtraClaims)
		assert.Equal(t, "value", params.ExtraClaims["custom_claim"])
	})

	t.Run("nil ScopeClaimMappings and nil ClaimMappers", func(t *testing.T) {
		client := &Client{}
		params := buildIDTokenParams("openid", client)
		require.NotNil(t, params)
		assert.Equal(t, []string{"openid"}, params.RequestedScopes)
		assert.Nil(t, params.ScopeClaimMappings)
		assert.Nil(t, params.ExtraClaims)
	})

	t.Run("invalid ScopeClaimMappings JSON", func(t *testing.T) {
		raw := datatypes.JSON("{invalid")
		client := &Client{
			ScopeClaimMappings: raw,
		}
		params := buildIDTokenParams("openid", client)
		require.NotNil(t, params)
		assert.Nil(t, params.ScopeClaimMappings)
	})

	t.Run("invalid ClaimMappers JSON", func(t *testing.T) {
		raw := datatypes.JSON("{invalid")
		client := &Client{
			ClaimMappers: raw,
		}
		params := buildIDTokenParams("openid", client)
		require.NotNil(t, params)
		assert.Nil(t, params.ExtraClaims)
	})
}

// ── TestParseScopes ─────────────────────────────────────────────────────────

func TestParseScopes(t *testing.T) {
	t.Run("empty string", func(t *testing.T) {
		assert.Nil(t, parseScopes(""))
	})

	t.Run("whitespace only", func(t *testing.T) {
		assert.Nil(t, parseScopes("   "))
	})

	t.Run("single scope", func(t *testing.T) {
		assert.Equal(t, []string{"openid"}, parseScopes("openid"))
	})

	t.Run("multiple scopes", func(t *testing.T) {
		assert.Equal(t, []string{"openid", "profile", "email"}, parseScopes("openid profile email"))
	})
}

// ── TestClientHasGrant ──────────────────────────────────────────────────────

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
