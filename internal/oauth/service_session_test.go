package oauth

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	jwtlib "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/platform/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newOAuthSessionSvc(
	db *gorm.DB,
	userRepo *mockUserRepo,
	refreshTokenRepo *mockOAuthRefreshTokenRepo,
	authEventSvc *mockAuthEventService,
) OAuthSessionService {
	return &oauthSessionService{
		db:               db,
		clientRepo:       &mockClientRepo{},
		userRepo:         userRepo,
		refreshTokenRepo: refreshTokenRepo,
		authEventService: authEventSvc,
	}
}

func expectSessionClientURILookup(mock sqlmock.Sqlmock, uri, clientID string) {
	rows := sqlmock.NewRows([]string{
		"client_id", "client_uuid", "tenant_id", "identity_provider_id", "name", "display_name",
		"client_type", "domain", "identifier", "secret", "status",
		"is_default", "is_system", "token_endpoint_auth_method",
		"grant_types", "response_types", "access_token_ttl", "refresh_token_ttl",
		"require_consent", "created_at", "updated_at",
	}).AddRow(
		10, uuid.New(), 1, int64(100), "test-client", "Test Client",
		"spa", nil, clientID, nil, "active",
		false, false, "none",
		`{}`, `{}`, nil, nil,
		false, time.Now(), time.Now(),
	)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).WillReturnRows(rows)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).WillReturnRows(
		sqlmock.NewRows([]string{"client_uri_id", "client_uri_uuid", "tenant_id", "client_id", "uri", "type", "created_at", "updated_at"}).
			AddRow(1, uuid.New(), 1, 10, uri, "redirect", time.Now(), time.Now()),
	)
}

// ── TestOAuthSessionService_EndSession ──────────────────────────────────────

func TestOAuthSessionService_EndSession(t *testing.T) {
	ctx := context.Background()

	t.Run("no id_token_hint returns empty redirect", func(t *testing.T) {
		svc := newOAuthSessionSvc(nil, &mockUserRepo{}, &mockOAuthRefreshTokenRepo{}, &mockAuthEventService{})

		redirect, oerr := svc.EndSession(ctx, OAuthEndSessionRequestDTO{})
		require.Nil(t, oerr)
		assert.Empty(t, redirect)
	})

	t.Run("id_token_hint valid finds user revokes sessions", func(t *testing.T) {
		initTestJWTKeysService(t)
		db, mock := newMockDB(t)
		expectSessionClientURILookup(mock, "https://example.com/logout", "my-client")

		token, err := jwt.GenerateAccessToken("user-sub-123", "openid", "https://auth.example.com", "my-client", "my-client", "default-provider")
		require.NoError(t, err)

		svc := newOAuthSessionSvc(db,
			&mockUserRepo{
				findBySubAndClientIDFn: func(sub, clientID string) (*User, error) {
					return &User{UserID: 42, UserUUID: uuid.New(), Email: "a@b.com"}, nil
				},
			},
			&mockOAuthRefreshTokenRepo{},
			&mockAuthEventService{})

		redirect, oerr := svc.EndSession(ctx, OAuthEndSessionRequestDTO{
			IDTokenHint:           token,
			ClientID:              "my-client",
			PostLogoutRedirectURI: "https://example.com/logout",
			State:                 "state123",
		})
		require.Nil(t, oerr)
		assert.Equal(t, "https://example.com/logout?state=state123", redirect)
	})

	t.Run("post_logout_redirect with no state", func(t *testing.T) {
		initTestJWTKeysService(t)
		db, mock := newMockDB(t)
		expectSessionClientURILookup(mock, "https://example.com/logout", "my-client")

		token, err := jwt.GenerateAccessToken("user-sub-123", "openid", "https://auth.example.com", "my-client", "my-client", "default-provider")
		require.NoError(t, err)

		svc := newOAuthSessionSvc(db,
			&mockUserRepo{
				findBySubAndClientIDFn: func(sub, clientID string) (*User, error) {
					return &User{UserID: 42, UserUUID: uuid.New(), Email: "a@b.com"}, nil
				},
			},
			&mockOAuthRefreshTokenRepo{},
			&mockAuthEventService{})

		redirect, oerr := svc.EndSession(ctx, OAuthEndSessionRequestDTO{
			IDTokenHint:           token,
			ClientID:              "my-client",
			PostLogoutRedirectURI: "https://example.com/logout",
		})
		require.Nil(t, oerr)
		assert.Equal(t, "https://example.com/logout", redirect)
	})

	t.Run("post_logout_redirect invalid URL", func(t *testing.T) {
		svc := newOAuthSessionSvc(nil, &mockUserRepo{}, &mockOAuthRefreshTokenRepo{}, &mockAuthEventService{})

		redirect, oerr := svc.EndSession(ctx, OAuthEndSessionRequestDTO{
			PostLogoutRedirectURI: "://invalid-url",
		})
		require.Nil(t, oerr)
		assert.Empty(t, redirect)
	})

	t.Run("id_token_hint valid no client match no revoke", func(t *testing.T) {
		initTestJWTKeysService(t)

		token, err := jwt.GenerateAccessToken("user-sub-456", "openid", "https://auth.example.com", "other-client", "other-client", "default-provider")
		require.NoError(t, err)

		svc := newOAuthSessionSvc(nil,
			&mockUserRepo{
				findBySubAndClientIDFn: func(sub, clientID string) (*User, error) {
					return nil, nil
				},
			},
			&mockOAuthRefreshTokenRepo{},
			&mockAuthEventService{})

		redirect, oerr := svc.EndSession(ctx, OAuthEndSessionRequestDTO{
			IDTokenHint: token,
			ClientID:    "my-client",
		})
		require.Nil(t, oerr)
		assert.Empty(t, redirect)
	})
}

// ── TestOAuthSessionService_BackchannelLogout ───────────────────────────────

func TestOAuthSessionService_BackchannelLogout(t *testing.T) {
	ctx := context.Background()

	t.Run("invalid token", func(t *testing.T) {
		svc := newOAuthSessionSvc(nil, &mockUserRepo{}, &mockOAuthRefreshTokenRepo{}, &mockAuthEventService{})

		oerr := svc.BackchannelLogout(ctx, OAuthBackchannelLogoutRequestDTO{LogoutToken: "garbage-token"})
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_request", oerr.Code)
		assert.Contains(t, oerr.Description, "logout_token is invalid or expired")
	})

	t.Run("expired token", func(t *testing.T) {
		svc := newOAuthSessionSvc(nil, &mockUserRepo{}, &mockOAuthRefreshTokenRepo{}, &mockAuthEventService{})

		oerr := svc.BackchannelLogout(ctx, OAuthBackchannelLogoutRequestDTO{LogoutToken: "expired-token"})
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_request", oerr.Code)
		assert.Contains(t, oerr.Description, "logout_token is invalid or expired")
	})

	t.Run("missing sub claim", func(t *testing.T) {
		orig := oauthSessionValidateTokenWithContext
		defer func() { oauthSessionValidateTokenWithContext = orig }()
		oauthSessionValidateTokenWithContext = func(context.Context, string) (jwtlib.MapClaims, error) {
			return jwtlib.MapClaims{"client_id": "my-client"}, nil
		}

		svc := newOAuthSessionSvc(nil, &mockUserRepo{}, &mockOAuthRefreshTokenRepo{}, &mockAuthEventService{})

		oerr := svc.BackchannelLogout(ctx, OAuthBackchannelLogoutRequestDTO{LogoutToken: "valid-no-sub"})
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_request", oerr.Code)
		assert.Contains(t, oerr.Description, "sub claim")
	})

	t.Run("valid logout_token revokes sessions", func(t *testing.T) {
		initTestJWTKeysService(t)

		token, err := jwt.GenerateAccessToken("user-sub-789", "openid", "https://auth.example.com", "my-client", "my-client", "default-provider")
		require.NoError(t, err)

		svc := newOAuthSessionSvc(nil,
			&mockUserRepo{
				findBySubAndClientIDFn: func(sub, clientID string) (*User, error) {
					return &User{UserID: 99, UserUUID: uuid.New(), Email: "a@b.com"}, nil
				},
			},
			&mockOAuthRefreshTokenRepo{},
			&mockAuthEventService{})

		oerr := svc.BackchannelLogout(ctx, OAuthBackchannelLogoutRequestDTO{LogoutToken: token})
		require.Nil(t, oerr)
	})

	t.Run("valid logout_token user not found no revoke", func(t *testing.T) {
		initTestJWTKeysService(t)

		token, err := jwt.GenerateAccessToken("user-sub-none", "openid", "https://auth.example.com", "my-client", "my-client", "default-provider")
		require.NoError(t, err)

		svc := newOAuthSessionSvc(nil,
			&mockUserRepo{
				findBySubAndClientIDFn: func(sub, clientID string) (*User, error) {
					return nil, nil
				},
			},
			&mockOAuthRefreshTokenRepo{},
			&mockAuthEventService{})

		oerr := svc.BackchannelLogout(ctx, OAuthBackchannelLogoutRequestDTO{LogoutToken: token})
		require.Nil(t, oerr)
	})

	t.Run("valid logout_token user lookup error", func(t *testing.T) {
		initTestJWTKeysService(t)

		token, err := jwt.GenerateAccessToken("user-sub-err", "openid", "https://auth.example.com", "my-client", "my-client", "default-provider")
		require.NoError(t, err)

		svc := newOAuthSessionSvc(nil,
			&mockUserRepo{
				findBySubAndClientIDFn: func(sub, clientID string) (*User, error) {
					return nil, errors.New("db error")
				},
			},
			&mockOAuthRefreshTokenRepo{},
			&mockAuthEventService{})

		oerr := svc.BackchannelLogout(ctx, OAuthBackchannelLogoutRequestDTO{LogoutToken: token})
		require.NotNil(t, oerr)
		assert.Equal(t, "server_error", oerr.Code)
	})
}

// ── TestOAuthSessionService_validateClientPostLogoutRedirect ────────────────

func TestOAuthSessionService_validateClientPostLogoutRedirect(t *testing.T) {
	t.Run("empty clientID returns false", func(t *testing.T) {
		svc := &oauthSessionService{db: nil}
		assert.False(t, svc.validateClientPostLogoutRedirect("", "https://x.com"))
	})

	t.Run("client not found returns false", func(t *testing.T) {
		db, mock := newMockDB(t)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).WillReturnError(gorm.ErrRecordNotFound)

		svc := &oauthSessionService{db: db}
		assert.False(t, svc.validateClientPostLogoutRedirect("unknown", "https://x.com"))
	})

	t.Run("no matching URI returns false", func(t *testing.T) {
		db, mock := newMockDB(t)
		rows := sqlmock.NewRows([]string{
			"client_id", "client_uuid", "tenant_id", "identity_provider_id", "name", "display_name",
			"client_type", "domain", "identifier", "secret", "status",
			"is_default", "is_system", "token_endpoint_auth_method",
			"grant_types", "response_types", "access_token_ttl", "refresh_token_ttl",
			"require_consent", "created_at", "updated_at",
		}).AddRow(
			10, uuid.New(), 1, int64(100), "test-client", "Test Client",
			"spa", nil, "my-client", nil, "active",
			false, false, "none",
			`{}`, `{}`, nil, nil,
			false, time.Now(), time.Now(),
		)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).WillReturnRows(rows)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).WillReturnRows(
			sqlmock.NewRows([]string{"client_uri_id", "client_uri_uuid", "tenant_id", "client_id", "uri", "type", "created_at", "updated_at"}).
				AddRow(1, uuid.New(), 1, 10, "https://other.com/cb", "redirect", time.Now(), time.Now()),
		)

		svc := &oauthSessionService{db: db}
		assert.False(t, svc.validateClientPostLogoutRedirect("my-client", "https://x.com"))
	})

	t.Run("matching URI returns true", func(t *testing.T) {
		db, mock := newMockDB(t)
		rows := sqlmock.NewRows([]string{
			"client_id", "client_uuid", "tenant_id", "identity_provider_id", "name", "display_name",
			"client_type", "domain", "identifier", "secret", "status",
			"is_default", "is_system", "token_endpoint_auth_method",
			"grant_types", "response_types", "access_token_ttl", "refresh_token_ttl",
			"require_consent", "created_at", "updated_at",
		}).AddRow(
			10, uuid.New(), 1, int64(100), "test-client", "Test Client",
			"spa", nil, "my-client", nil, "active",
			false, false, "none",
			`{}`, `{}`, nil, nil,
			false, time.Now(), time.Now(),
		)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).WillReturnRows(rows)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).WillReturnRows(
			sqlmock.NewRows([]string{"client_uri_id", "client_uri_uuid", "tenant_id", "client_id", "uri", "type", "created_at", "updated_at"}).
				AddRow(1, uuid.New(), 1, 10, "https://example.com/logout", "redirect", time.Now(), time.Now()),
		)

		svc := &oauthSessionService{db: db}
		assert.True(t, svc.validateClientPostLogoutRedirect("my-client", "https://example.com/logout"))
	})

	t.Run("nil ClientURIs returns false", func(t *testing.T) {
		db, mock := newMockDB(t)
		rows := sqlmock.NewRows([]string{
			"client_id", "client_uuid", "tenant_id", "identity_provider_id", "name", "display_name",
			"client_type", "domain", "identifier", "secret", "status",
			"is_default", "is_system", "token_endpoint_auth_method",
			"grant_types", "response_types", "access_token_ttl", "refresh_token_ttl",
			"require_consent", "created_at", "updated_at",
		}).AddRow(
			10, uuid.New(), 1, int64(100), "test-client", "Test Client",
			"spa", nil, "my-client", nil, "active",
			false, false, "none",
			`{}`, `{}`, nil, nil,
			false, time.Now(), time.Now(),
		)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).WillReturnRows(rows)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).WillReturnRows(sqlmock.NewRows(nil))

		svc := &oauthSessionService{db: db}
		assert.False(t, svc.validateClientPostLogoutRedirect("my-client", "https://x.com"))
	})
}

// ── TestOAuthSessionService_EndSession_Additional ───────────────────────────

func TestOAuthSessionService_EndSession_Additional(t *testing.T) {
	ctx := context.Background()

	t.Run("invalid id_token_hint silently ignored", func(t *testing.T) {
		svc := newOAuthSessionSvc(nil, &mockUserRepo{}, &mockOAuthRefreshTokenRepo{}, &mockAuthEventService{})

		redirect, oerr := svc.EndSession(ctx, OAuthEndSessionRequestDTO{
			IDTokenHint: "garbage",
		})
		require.Nil(t, oerr)
		assert.Empty(t, redirect)
	})

	t.Run("dangerous post_logout_redirect_uri ignored", func(t *testing.T) {
		svc := newOAuthSessionSvc(nil, &mockUserRepo{}, &mockOAuthRefreshTokenRepo{}, &mockAuthEventService{})

		redirect, oerr := svc.EndSession(ctx, OAuthEndSessionRequestDTO{
			PostLogoutRedirectURI: "javascript:alert(1)",
			ClientID:              "my-client",
		})
		require.Nil(t, oerr)
		assert.Empty(t, redirect)
	})
}
