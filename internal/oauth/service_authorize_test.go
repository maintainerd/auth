package oauth

import (
	"context"
	"errors"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/maintainerd/maintainerd-auth/internal/platform/config"
	"github.com/maintainerd/maintainerd-auth/internal/platform/crypto"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ── helpers ─────────────────────────────────────────────────────────────────

func newOAuthAuthorizeSvc(
	db *gorm.DB,
	clientRepo *mockClientRepo,
	clientURIRepo *mockClientURIRepo,
	authCodeRepo *mockOAuthAuthCodeRepo,
	consentGrantRepo *mockOAuthConsentGrantRepo,
	consentChallRepo *mockOAuthConsentChallRepo,
	authEventSvc *mockAuthEventService,
) OAuthAuthorizeService {
	return NewOAuthAuthorizeService(db, clientRepo, clientURIRepo, authCodeRepo, consentGrantRepo, consentChallRepo, authEventSvc, NewOAuthAuthorizeRequestRepository(db))
}

func validAuthorizeRequest() OAuthAuthorizeRequestDTO {
	return OAuthAuthorizeRequestDTO{
		ResponseType:        "code",
		ClientID:            "my-client",
		RedirectURI:         "https://example.com/callback",
		Scope:               "openid profile",
		State:               "state123",
		CodeChallenge:       strings.Repeat("A", 43),
		CodeChallengeMethod: "S256",
		Nonce:               "nonce123",
	}
}

func activeClient() *Client {
	return &Client{
		ClientID:       10,
		ClientUUID:     uuid.New(),
		TenantID:       1,
		Status:         shared.StatusActive,
		GrantTypes:     pq.StringArray{GrantTypeAuthorizationCode},
		ResponseTypes:  pq.StringArray{ResponseTypeCode},
		RequireConsent: false,
		ClientURIs: &[]ClientURI{
			{URI: "https://example.com/callback", Type: shared.ClientURITypeRedirect},
		},
	}
}

func activeClientWithConsent() *Client {
	c := activeClient()
	c.RequireConsent = true
	return c
}

// newMockDB creates a *gorm.DB backed by sqlmock for transaction tests.
func newMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	return gormDB, mock
}

func TestOAuthAuthorizeService_PrepareAuthorize(t *testing.T) {
	ctx := context.Background()

	build := func(clientRepo *mockClientRepo) OAuthAuthorizeService {
		db, _ := newMockDB(t)
		return newOAuthAuthorizeSvc(db, clientRepo, &mockClientURIRepo{}, &mockOAuthAuthCodeRepo{},
			&mockOAuthConsentGrantRepo{}, &mockOAuthConsentChallRepo{}, &mockAuthEventService{})
	}

	t.Run("valid request returns nil", func(t *testing.T) {
		svc := build(&mockClientRepo{findByClientIDAndIdentityProviderFn: func(_, _ string) (*Client, error) { return activeClient(), nil }})
		assert.Nil(t, svc.PrepareAuthorize(ctx, validAuthorizeRequest()))
	})

	t.Run("client lookup error returns server_error", func(t *testing.T) {
		svc := build(&mockClientRepo{findByClientIDAndIdentityProviderFn: func(_, _ string) (*Client, error) { return nil, errors.New("db error") }})
		oerr := svc.PrepareAuthorize(ctx, validAuthorizeRequest())
		require.NotNil(t, oerr)
		assert.Equal(t, "server_error", oerr.Code)
	})

	t.Run("unknown client returns invalid_request", func(t *testing.T) {
		svc := build(&mockClientRepo{
			findByClientIDAndIdentityProviderFn: func(_, _ string) (*Client, error) { return nil, nil },
			findByIdentifierFn:                  func(_ string) (*Client, error) { return nil, nil },
		})
		oerr := svc.PrepareAuthorize(ctx, validAuthorizeRequest())
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_request", oerr.Code)
	})

	t.Run("inactive client returns invalid_request", func(t *testing.T) {
		client := activeClient()
		client.Status = shared.StatusInactive
		svc := build(&mockClientRepo{findByClientIDAndIdentityProviderFn: func(_, _ string) (*Client, error) { return client, nil }})
		oerr := svc.PrepareAuthorize(ctx, validAuthorizeRequest())
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_request", oerr.Code)
	})

	t.Run("unregistered redirect_uri is rejected", func(t *testing.T) {
		svc := build(&mockClientRepo{findByClientIDAndIdentityProviderFn: func(_, _ string) (*Client, error) { return activeClient(), nil }})
		req := validAuthorizeRequest()
		req.RedirectURI = "https://evil.example.com/cb"
		require.NotNil(t, svc.PrepareAuthorize(ctx, req))
	})
}

// TestOAuthAuthorizeService_ContinueAuthorize_Binding covers the browser-binding
// gate: a persisted authorize request may only be continued by the browser that
// initiated it (whose httpOnly cookie carries the secret whose hash was stored at
// prepare time). This blocks a different authenticated session in the same tenant
// from consuming another user's pending request with a leaked request_id.
func TestOAuthAuthorizeService_ContinueAuthorize_Binding(t *testing.T) {
	ctx := context.Background()
	reqUUID := uuid.New()

	build := func() (OAuthAuthorizeService, sqlmock.Sqlmock) {
		db, mock := newMockDB(t)
		svc := newOAuthAuthorizeSvc(db, &mockClientRepo{}, &mockClientURIRepo{}, &mockOAuthAuthCodeRepo{},
			&mockOAuthConsentGrantRepo{}, &mockOAuthConsentChallRepo{}, &mockAuthEventService{})
		return svc, mock
	}

	// savedRow builds the FindByUUID result for a live (unexpired, pending)
	// authorize request with the given binding_hash (nil = no binding stored).
	savedRow := func(bindingHash interface{}) *sqlmock.Rows {
		return sqlmock.NewRows([]string{
			"oauth_authorize_request_id", "oauth_authorize_request_uuid", "client_id", "tenant_id",
			"redirect_uri", "response_type", "status", "binding_hash", "expires_at",
		}).AddRow(int64(1), reqUUID, int64(10), int64(1),
			"https://example.com/callback", "code", "pending", bindingHash, time.Now().Add(5*time.Minute))
	}

	t.Run("missing session binding is rejected", func(t *testing.T) {
		svc, mock := build()
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).WillReturnRows(savedRow(nil))
		_, oerr := svc.ContinueAuthorize(ctx, reqUUID.String(), "any-secret", 5, 1)
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_request", oerr.Code)
		assert.Contains(t, oerr.Description, "not bound")
	})

	t.Run("binding secret mismatch is rejected", func(t *testing.T) {
		svc, mock := build()
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).WillReturnRows(savedRow(crypto.HashOAuthBindingToken("correct-secret")))
		_, oerr := svc.ContinueAuthorize(ctx, reqUUID.String(), "wrong-secret", 5, 1)
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_request", oerr.Code)
		assert.Contains(t, oerr.Description, "binding mismatch")
	})

	t.Run("invalid request_id is rejected before any lookup", func(t *testing.T) {
		svc, _ := build()
		_, oerr := svc.ContinueAuthorize(ctx, "not-a-uuid", "any-secret", 5, 1)
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_request", oerr.Code)
	})
}

func TestOAuthAuthorizeService_StartBroker(t *testing.T) {
	ctx := context.Background()
	req := validAuthorizeRequest()
	req.IdpHint = "google"

	// Resolver returns a provider with explicit endpoints.
	origResolver := brokerProviderResolver
	brokerProviderResolver = &mockBrokerProviderResolver{
		resolveFn: func(_ context.Context, idpHint string) (*BrokerProvider, error) {
			assert.Equal(t, "google", idpHint)
			return &BrokerProvider{
				AuthorizationEndpoint: "https://accounts.google.com/o/oauth2/v2/auth",
				ClientID:              "upstream-client",
				Scopes:                []string{"openid", "email"},
			}, nil
		},
	}
	t.Cleanup(func() { brokerProviderResolver = origResolver })

	origHost := config.AppPublicHostname
	config.AppPublicHostname = "https://auth.id.app"
	t.Cleanup(func() { config.AppPublicHostname = origHost })

	t.Run("creates broker session and returns provider authorize URL", func(t *testing.T) {
		db, mock := newMockDB(t)
		// Enabled connection for the client + identity_provider preload.
		mock.ExpectQuery(`SELECT \* FROM "client_identity_providers" WHERE.*client_id = \$1.*enabled = \$2`).
			WithArgs(int64(10), true).
			WillReturnRows(sqlmock.NewRows([]string{
				"client_identity_provider_id", "client_identity_provider_uuid", "client_id", "tenant_id",
				"identity_provider_id", "is_default", "enabled", "display_order",
			}).AddRow(1, uuid.New(), 10, 1, 100, false, true, 0))
		mock.ExpectQuery(`SELECT \* FROM "identity_providers"`).
			WillReturnRows(sqlmock.NewRows([]string{
				"identity_provider_id", "identity_provider_uuid", "tenant_id", "name", "display_name",
				"provider", "provider_type", "identifier", "status", "is_default", "is_system",
			}).AddRow(100, uuid.New(), 1, "google", "Google", "google", "social", "google", shared.StatusActive, false, false))
		// Broker session INSERT — auto-transaction.
		mock.ExpectBegin()
		mock.ExpectQuery(`INSERT INTO "oauth_broker_sessions"`).
			WillReturnRows(sqlmock.NewRows([]string{"oauth_broker_session_id"}).AddRow(int64(1)))
		mock.ExpectCommit()

		svc := newOAuthAuthorizeSvc(db,
			&mockClientRepo{findByClientIDAndIdentityProviderFn: func(_, _ string) (*Client, error) { return activeClient(), nil }},
			&mockClientURIRepo{},
			&mockOAuthAuthCodeRepo{}, &mockOAuthConsentGrantRepo{}, &mockOAuthConsentChallRepo{}, &mockAuthEventService{})

		result, oerr := svc.StartBroker(ctx, req)
		require.Nil(t, oerr)
		require.NotNil(t, result)
		assert.Contains(t, result.RedirectURI, "https://accounts.google.com/o/oauth2/v2/auth")
		assert.Contains(t, result.RedirectURI, "client_id=upstream-client")
		assert.Contains(t, result.RedirectURI, "scope=openid+email")
		assert.Contains(t, result.RedirectURI, "redirect_uri=https%3A%2F%2Fauth.id.app%2Fapi%2Fv1%2Foauth%2Fcallback%2Fgoogle")
		assert.Contains(t, result.RedirectURI, "code_challenge=")
		assert.Contains(t, result.RedirectURI, "code_challenge_method=S256")
		assert.Contains(t, result.RedirectURI, "state=")
		assert.Contains(t, result.RedirectURI, "nonce=")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestOAuthAuthorizeService_HandleCallback(t *testing.T) {
	ctx := context.Background()
	origHost := config.AppPublicHostname
	config.AppPublicHostname = "https://auth.id.app"
	t.Cleanup(func() { config.AppPublicHostname = origHost })

	sessionRows := func() *sqlmock.Rows {
		return sqlmock.NewRows([]string{
			"oauth_broker_session_id", "oauth_broker_session_uuid", "tenant_id", "client_id",
			"identity_provider_id", "identity_provider_identifier", "app_redirect_uri", "app_state", "app_scope", "app_nonce",
			"app_code_challenge", "app_code_challenge_method", "idp_state", "idp_pkce_verifier",
			"idp_nonce", "expires_at", "consumed_at", "created_at",
		}).AddRow(
			int64(1), uuid.New(), int64(1), int64(10),
			int64(100), "google", "https://example.com/callback", "app-state", pq.StringArray{"openid", "profile"}, "app-nonce",
			strings.Repeat("A", 43), "S256", "state-1", "pkce-verifier",
			"idp-nonce", time.Now().Add(time.Minute), nil, time.Now(),
		)
	}

	t.Run("consumes session and issues downstream code", func(t *testing.T) {
		initTestJWTKeysService(t)
		db, mock := newMockDB(t)
		mock.ExpectQuery(`SELECT \* FROM "oauth_broker_sessions" WHERE idp_state = .*consumed_at IS NULL`).
			WillReturnRows(sessionRows())
		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE "oauth_broker_sessions" SET "consumed_at"=.*WHERE oauth_broker_session_id = .* AND consumed_at IS NULL`).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		federatedSessionUUID := uuid.New()
		origResolver := brokerCallbackResolver
		brokerCallbackResolver = &mockBrokerCallbackResolver{
			resolveFn: func(_ context.Context, idpID int64, code, pkceVerifier, nonce, redirectURI string, clientID int64) (*BrokerResolvedUser, error) {
				assert.Equal(t, int64(100), idpID)
				assert.Equal(t, "provider-code", code)
				assert.Equal(t, "pkce-verifier", pkceVerifier)
				assert.Equal(t, "idp-nonce", nonce)
				assert.Equal(t, "https://auth.id.app/api/v1/oauth/callback/google", redirectURI)
				assert.Equal(t, int64(10), clientID)
				return &BrokerResolvedUser{UserID: 50, UserUUID: uuid.New(), IdentitySub: "internal-sub", SessionID: federatedSessionUUID.String()}, nil
			},
		}
		t.Cleanup(func() { brokerCallbackResolver = origResolver })

		identifier := "my-client"
		appClient := activeClient()
		appClient.Identifier = &identifier
		var createdCode *OAuthAuthorizationCode
		svc := newOAuthAuthorizeSvc(db,
			&mockClientRepo{
				findByIDFn: func(id any, _ ...string) (*Client, error) {
					assert.Equal(t, int64(10), id)
					return appClient, nil
				},
			},
			&mockClientURIRepo{},
			&mockOAuthAuthCodeRepo{
				createFn: func(code *OAuthAuthorizationCode) (*OAuthAuthorizationCode, error) {
					createdCode = code
					return code, nil
				},
			},
			&mockOAuthConsentGrantRepo{}, &mockOAuthConsentChallRepo{}, &mockAuthEventService{})

		redirectURL, accessToken, oerr := svc.HandleCallback(ctx, "google", "provider-code", "state-1")
		require.Nil(t, oerr)
		assert.Contains(t, redirectURL, "https://example.com/callback?code=")
		assert.Contains(t, redirectURL, "state=app-state")
		require.NotNil(t, createdCode)
		assert.Equal(t, int64(10), createdCode.ClientID)
		assert.Equal(t, int64(50), createdCode.UserID)
		assert.Equal(t, pq.StringArray{"openid", "profile"}, createdCode.Scope)
		assert.Equal(t, "S256", createdCode.CodeChallengeMethod)
		// The code must be bound to the federated session the broker created, so
		// tokens minted from it (at the downstream token exchange) carry a `sid`
		// and survive session validation.
		require.NotNil(t, createdCode.UserSessionUUID)
		assert.Equal(t, federatedSessionUUID, *createdCode.UserSessionUUID)
		// No SSO cookie token is minted at the broker callback anymore: it would
		// land on the issuer host the identity app never reads. The identity
		// session is established same-origin by the identity /callback page.
		assert.Empty(t, accessToken)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("third-party downstream client: SSO cookie binds to the hosted-login client", func(t *testing.T) {
		initTestJWTKeysService(t)
		db, mock := newMockDB(t)
		mock.ExpectQuery(`SELECT \* FROM "oauth_broker_sessions" WHERE idp_state = .*consumed_at IS NULL`).
			WillReturnRows(sessionRows())
		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE "oauth_broker_sessions" SET "consumed_at"=.*WHERE oauth_broker_session_id = .* AND consumed_at IS NULL`).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		federatedSessionUUID := uuid.New()
		origResolver := brokerCallbackResolver
		brokerCallbackResolver = &mockBrokerCallbackResolver{
			resolveFn: func(_ context.Context, _ int64, _, _, _, _ string, _ int64) (*BrokerResolvedUser, error) {
				return &BrokerResolvedUser{UserID: 50, UserUUID: uuid.New(), IdentitySub: "cognito-sub", SessionID: federatedSessionUUID.String()}, nil
			},
		}
		t.Cleanup(func() { brokerCallbackResolver = origResolver })

		externalID := "external-app"
		externalClient := activeClient()
		externalClient.Identifier = &externalID
		var createdCode *OAuthAuthorizationCode

		svc := newOAuthAuthorizeSvc(db,
			&mockClientRepo{
				findByIDFn: func(any, ...string) (*Client, error) { return externalClient, nil },
			},
			&mockClientURIRepo{},
			&mockOAuthAuthCodeRepo{
				createFn: func(code *OAuthAuthorizationCode) (*OAuthAuthorizationCode, error) {
					createdCode = code
					return code, nil
				},
			},
			&mockOAuthConsentGrantRepo{}, &mockOAuthConsentChallRepo{}, &mockAuthEventService{})

		redirectURL, accessToken, oerr := svc.HandleCallback(ctx, "google", "provider-code", "state-1")
		require.Nil(t, oerr)
		assert.Contains(t, redirectURL, "https://example.com/callback?code=")
		// The downstream (third-party) client's code is bound to the federated
		// session so its tokens carry a sid...
		require.NotNil(t, createdCode)
		require.NotNil(t, createdCode.UserSessionUUID)
		assert.Equal(t, federatedSessionUUID, *createdCode.UserSessionUUID)
		// ...and no SSO token/cookie is minted at the callback (dead issuer-host
		// surface removed; the identity app logs in same-origin instead).
		assert.Empty(t, accessToken)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("already consumed during transaction rejects replay", func(t *testing.T) {
		db, mock := newMockDB(t)
		mock.ExpectQuery(`SELECT \* FROM "oauth_broker_sessions" WHERE idp_state = .*consumed_at IS NULL`).
			WillReturnRows(sessionRows())
		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE "oauth_broker_sessions" SET "consumed_at"=.*WHERE oauth_broker_session_id = .* AND consumed_at IS NULL`).
			WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectRollback()

		origResolver := brokerCallbackResolver
		brokerCallbackResolver = &mockBrokerCallbackResolver{}
		t.Cleanup(func() { brokerCallbackResolver = origResolver })

		identifier := "my-client"
		appClient := activeClient()
		appClient.Identifier = &identifier
		codeCreated := false
		svc := newOAuthAuthorizeSvc(db,
			&mockClientRepo{findByIDFn: func(any, ...string) (*Client, error) { return appClient, nil }},
			&mockClientURIRepo{},
			&mockOAuthAuthCodeRepo{createFn: func(code *OAuthAuthorizationCode) (*OAuthAuthorizationCode, error) {
				codeCreated = true
				return code, nil
			}},
			&mockOAuthConsentGrantRepo{}, &mockOAuthConsentChallRepo{}, &mockAuthEventService{})

		redirectURL, accessToken, oerr := svc.HandleCallback(ctx, "google", "provider-code", "state-1")
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_request", oerr.Code)
		assert.Empty(t, redirectURL)
		assert.Empty(t, accessToken)
		assert.False(t, codeCreated)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("rejects callback when provider path does not match the broker session", func(t *testing.T) {
		db, mock := newMockDB(t)
		mock.ExpectQuery(`SELECT \* FROM "oauth_broker_sessions" WHERE idp_state = .*consumed_at IS NULL`).
			WillReturnRows(sessionRows())

		svc := newOAuthAuthorizeSvc(db,
			&mockClientRepo{}, &mockClientURIRepo{}, &mockOAuthAuthCodeRepo{},
			&mockOAuthConsentGrantRepo{}, &mockOAuthConsentChallRepo{}, &mockAuthEventService{})

		redirectURL, accessToken, oerr := svc.HandleCallback(ctx, "github", "provider-code", "state-1")
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_request", oerr.Code)
		assert.Empty(t, redirectURL)
		assert.Empty(t, accessToken)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// ── TestOAuthAuthorizeService_Authorize ─────────────────────────────────────

func TestOAuthAuthorizeService_Authorize(t *testing.T) {
	ctx := context.Background()

	t.Run("issues code when consent not required", func(t *testing.T) {
		client := activeClient()
		db, _ := newMockDB(t)

		svc := newOAuthAuthorizeSvc(db,
			&mockClientRepo{
				findByClientIDAndIdentityProviderFn: func(_, _ string) (*Client, error) {
					return client, nil
				},
			},
			&mockClientURIRepo{},
			&mockOAuthAuthCodeRepo{},
			&mockOAuthConsentGrantRepo{},
			&mockOAuthConsentChallRepo{},
			&mockAuthEventService{},
		)

		result, oerr := svc.Authorize(ctx, validAuthorizeRequest(), 1, 1)
		require.Nil(t, oerr)
		require.NotNil(t, result)
		assert.Contains(t, result.RedirectURI, "code=")
		assert.Contains(t, result.RedirectURI, "state=state123")
		assert.Empty(t, result.ConsentChallenge)
	})

	// Tenant binding (SECURITY-CRITICAL): a client_id must belong to the tenant
	// context the user is acting in. Matching → ok; mismatch → hard reject with no
	// fallback to the system tenant.
	t.Run("tenant binding: matching client tenant and context is accepted", func(t *testing.T) {
		client := activeClient() // TenantID = 1
		db, _ := newMockDB(t)

		svc := newOAuthAuthorizeSvc(db,
			&mockClientRepo{
				findByClientIDAndIdentityProviderFn: func(_, _ string) (*Client, error) {
					return client, nil
				},
			},
			&mockClientURIRepo{},
			&mockOAuthAuthCodeRepo{},
			&mockOAuthConsentGrantRepo{},
			&mockOAuthConsentChallRepo{},
			&mockAuthEventService{},
		)

		result, oerr := svc.Authorize(ctx, validAuthorizeRequest(), 1, client.TenantID)
		require.Nil(t, oerr)
		require.NotNil(t, result)
		assert.Contains(t, result.RedirectURI, "code=")
	})

	t.Run("tenant binding: mismatched client tenant and context is rejected", func(t *testing.T) {
		client := activeClient() // TenantID = 1
		db, _ := newMockDB(t)

		svc := newOAuthAuthorizeSvc(db,
			&mockClientRepo{
				findByClientIDAndIdentityProviderFn: func(_, _ string) (*Client, error) {
					return client, nil
				},
			},
			&mockClientURIRepo{},
			&mockOAuthAuthCodeRepo{},
			&mockOAuthConsentGrantRepo{},
			&mockOAuthConsentChallRepo{},
			&mockAuthEventService{},
		)

		// Act as tenant 2 with a client that belongs to tenant 1 → cross-tenant.
		result, oerr := svc.Authorize(ctx, validAuthorizeRequest(), 1, 2)
		require.Nil(t, result)
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_request", oerr.Code)
	})

	t.Run("tenant binding: a seeded system client is not a valid public client_id", func(t *testing.T) {
		sysClient := activeClient()
		sysClient.IsSystem = true
		sysClient.Name = "internal-system" // not the allowed hosted-login client
		db, _ := newMockDB(t)

		svc := newOAuthAuthorizeSvc(db,
			&mockClientRepo{
				findByClientIDAndIdentityProviderFn: func(_, _ string) (*Client, error) {
					return sysClient, nil
				},
				findByIdentifierFn: func(_ string) (*Client, error) {
					return sysClient, nil
				},
			},
			&mockClientURIRepo{},
			&mockOAuthAuthCodeRepo{},
			&mockOAuthConsentGrantRepo{},
			&mockOAuthConsentChallRepo{},
			&mockAuthEventService{},
		)

		result, oerr := svc.Authorize(ctx, validAuthorizeRequest(), 1, sysClient.TenantID)
		require.Nil(t, result)
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_request", oerr.Code)
	})

	t.Run("returns consent challenge when consent required", func(t *testing.T) {
		client := activeClientWithConsent()
		db, _ := newMockDB(t)
		challengeUUID := uuid.New()

		svc := newOAuthAuthorizeSvc(db,
			&mockClientRepo{
				findByClientIDAndIdentityProviderFn: func(_, _ string) (*Client, error) {
					return client, nil
				},
			},
			&mockClientURIRepo{},
			&mockOAuthAuthCodeRepo{},
			&mockOAuthConsentGrantRepo{
				findByUserAndClientFn: func(_, _ int64) (*OAuthConsentGrant, error) {
					return nil, nil // no existing grant
				},
			},
			&mockOAuthConsentChallRepo{
				createFn: func(c *OAuthConsentChallenge) (*OAuthConsentChallenge, error) {
					c.OAuthConsentChallengeUUID = challengeUUID
					return c, nil
				},
			},
			&mockAuthEventService{},
		)

		result, oerr := svc.Authorize(ctx, validAuthorizeRequest(), 1, 1)
		require.Nil(t, oerr)
		require.NotNil(t, result)
		assert.Equal(t, challengeUUID.String(), result.ConsentChallenge)
		assert.Empty(t, result.RedirectURI)
	})

	t.Run("prompt none returns consent_required instead of creating UI challenge", func(t *testing.T) {
		client := activeClientWithConsent()
		db, _ := newMockDB(t)
		challengeCreated := false

		svc := newOAuthAuthorizeSvc(db,
			&mockClientRepo{
				findByClientIDAndIdentityProviderFn: func(_, _ string) (*Client, error) {
					return client, nil
				},
			},
			&mockClientURIRepo{},
			&mockOAuthAuthCodeRepo{},
			&mockOAuthConsentGrantRepo{
				findByUserAndClientFn: func(_, _ int64) (*OAuthConsentGrant, error) {
					return nil, nil
				},
			},
			&mockOAuthConsentChallRepo{
				createFn: func(c *OAuthConsentChallenge) (*OAuthConsentChallenge, error) {
					challengeCreated = true
					return c, nil
				},
			},
			&mockAuthEventService{},
		)

		req := validAuthorizeRequest()
		req.Prompt = "none"
		result, oerr := svc.Authorize(ctx, req, 1, 1)
		require.Nil(t, result)
		require.NotNil(t, oerr)
		assert.Equal(t, "consent_required", oerr.Code)
		assert.False(t, challengeCreated)
	})

	t.Run("skips consent when all scopes already granted", func(t *testing.T) {
		client := activeClientWithConsent()
		db, _ := newMockDB(t)

		svc := newOAuthAuthorizeSvc(db,
			&mockClientRepo{
				findByClientIDAndIdentityProviderFn: func(_, _ string) (*Client, error) {
					return client, nil
				},
			},
			&mockClientURIRepo{},
			&mockOAuthAuthCodeRepo{},
			&mockOAuthConsentGrantRepo{
				findByUserAndClientFn: func(_, _ int64) (*OAuthConsentGrant, error) {
					return &OAuthConsentGrant{Scopes: parseScopeFields("openid profile email")}, nil
				},
			},
			&mockOAuthConsentChallRepo{},
			&mockAuthEventService{},
		)

		result, oerr := svc.Authorize(ctx, validAuthorizeRequest(), 1, 1)
		require.Nil(t, oerr)
		assert.Contains(t, result.RedirectURI, "code=")
	})

	t.Run("requires consent when new scope requested", func(t *testing.T) {
		client := activeClientWithConsent()
		db, _ := newMockDB(t)

		svc := newOAuthAuthorizeSvc(db,
			&mockClientRepo{
				findByClientIDAndIdentityProviderFn: func(_, _ string) (*Client, error) {
					return client, nil
				},
			},
			&mockClientURIRepo{},
			&mockOAuthAuthCodeRepo{},
			&mockOAuthConsentGrantRepo{
				findByUserAndClientFn: func(_, _ int64) (*OAuthConsentGrant, error) {
					return &OAuthConsentGrant{Scopes: parseScopeFields("openid")}, nil // missing "profile"
				},
			},
			&mockOAuthConsentChallRepo{},
			&mockAuthEventService{},
		)

		result, oerr := svc.Authorize(ctx, validAuthorizeRequest(), 1, 1)
		require.Nil(t, oerr)
		assert.NotEmpty(t, result.ConsentChallenge)
	})

	t.Run("client not found", func(t *testing.T) {
		db, mock := newMockDB(t)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).WillReturnError(gorm.ErrRecordNotFound)

		svc := newOAuthAuthorizeSvc(db,
			&mockClientRepo{
				findByClientIDAndIdentityProviderFn: func(_, _ string) (*Client, error) {
					return nil, nil
				},
			},
			&mockClientURIRepo{},
			&mockOAuthAuthCodeRepo{},
			&mockOAuthConsentGrantRepo{},
			&mockOAuthConsentChallRepo{},
			&mockAuthEventService{},
		)

		_, oerr := svc.Authorize(ctx, validAuthorizeRequest(), 1, 1)
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_request", oerr.Code)
	})

	t.Run("client lookup error", func(t *testing.T) {
		db, _ := newMockDB(t)

		svc := newOAuthAuthorizeSvc(db,
			&mockClientRepo{
				findByClientIDAndIdentityProviderFn: func(_, _ string) (*Client, error) {
					return nil, errors.New("db error")
				},
			},
			&mockClientURIRepo{},
			&mockOAuthAuthCodeRepo{},
			&mockOAuthConsentGrantRepo{},
			&mockOAuthConsentChallRepo{},
			&mockAuthEventService{},
		)

		_, oerr := svc.Authorize(ctx, validAuthorizeRequest(), 1, 1)
		require.NotNil(t, oerr)
		assert.Equal(t, "server_error", oerr.Code)
	})

	t.Run("client inactive", func(t *testing.T) {
		db, _ := newMockDB(t)
		client := activeClient()
		client.Status = "inactive"

		svc := newOAuthAuthorizeSvc(db,
			&mockClientRepo{
				findByClientIDAndIdentityProviderFn: func(_, _ string) (*Client, error) {
					return client, nil
				},
			},
			&mockClientURIRepo{},
			&mockOAuthAuthCodeRepo{},
			&mockOAuthConsentGrantRepo{},
			&mockOAuthConsentChallRepo{},
			&mockAuthEventService{},
		)

		_, oerr := svc.Authorize(ctx, validAuthorizeRequest(), 1, 1)
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_request", oerr.Code)
	})

	t.Run("grant type not allowed", func(t *testing.T) {
		db, _ := newMockDB(t)
		client := activeClient()
		client.GrantTypes = pq.StringArray{GrantTypeClientCredentials}

		svc := newOAuthAuthorizeSvc(db,
			&mockClientRepo{
				findByClientIDAndIdentityProviderFn: func(_, _ string) (*Client, error) {
					return client, nil
				},
			},
			&mockClientURIRepo{},
			&mockOAuthAuthCodeRepo{},
			&mockOAuthConsentGrantRepo{},
			&mockOAuthConsentChallRepo{},
			&mockAuthEventService{},
		)

		_, oerr := svc.Authorize(ctx, validAuthorizeRequest(), 1, 1)
		require.NotNil(t, oerr)
		assert.Equal(t, "unauthorized_client", oerr.Code)
	})

	t.Run("response type not supported", func(t *testing.T) {
		db, _ := newMockDB(t)
		client := activeClient()
		client.ResponseTypes = pq.StringArray{}

		svc := newOAuthAuthorizeSvc(db,
			&mockClientRepo{
				findByClientIDAndIdentityProviderFn: func(_, _ string) (*Client, error) {
					return client, nil
				},
			},
			&mockClientURIRepo{},
			&mockOAuthAuthCodeRepo{},
			&mockOAuthConsentGrantRepo{},
			&mockOAuthConsentChallRepo{},
			&mockAuthEventService{},
		)

		_, oerr := svc.Authorize(ctx, validAuthorizeRequest(), 1, 1)
		require.NotNil(t, oerr)
		assert.Equal(t, "unsupported_response_type", oerr.Code)
	})

	t.Run("redirect URI not registered", func(t *testing.T) {
		db, _ := newMockDB(t)
		client := activeClient()
		client.ClientURIs = &[]ClientURI{
			{URI: "https://other.com/callback", Type: shared.ClientURITypeRedirect},
		}

		svc := newOAuthAuthorizeSvc(db,
			&mockClientRepo{
				findByClientIDAndIdentityProviderFn: func(_, _ string) (*Client, error) {
					return client, nil
				},
			},
			&mockClientURIRepo{},
			&mockOAuthAuthCodeRepo{},
			&mockOAuthConsentGrantRepo{},
			&mockOAuthConsentChallRepo{},
			&mockAuthEventService{},
		)

		_, oerr := svc.Authorize(ctx, validAuthorizeRequest(), 1, 1)
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_request", oerr.Code)
		assert.Contains(t, oerr.Description, "redirect_uri")
	})

	t.Run("no redirect URIs registered", func(t *testing.T) {
		db, _ := newMockDB(t)
		client := activeClient()
		client.ClientURIs = nil

		svc := newOAuthAuthorizeSvc(db,
			&mockClientRepo{
				findByClientIDAndIdentityProviderFn: func(_, _ string) (*Client, error) {
					return client, nil
				},
			},
			&mockClientURIRepo{},
			&mockOAuthAuthCodeRepo{},
			&mockOAuthConsentGrantRepo{},
			&mockOAuthConsentChallRepo{},
			&mockAuthEventService{},
		)

		_, oerr := svc.Authorize(ctx, validAuthorizeRequest(), 1, 1)
		require.NotNil(t, oerr)
		assert.Contains(t, oerr.Description, "no redirect URIs")
	})

	t.Run("consent check error", func(t *testing.T) {
		client := activeClientWithConsent()
		db, _ := newMockDB(t)

		svc := newOAuthAuthorizeSvc(db,
			&mockClientRepo{
				findByClientIDAndIdentityProviderFn: func(_, _ string) (*Client, error) {
					return client, nil
				},
			},
			&mockClientURIRepo{},
			&mockOAuthAuthCodeRepo{},
			&mockOAuthConsentGrantRepo{
				findByUserAndClientFn: func(_, _ int64) (*OAuthConsentGrant, error) {
					return nil, errors.New("db error")
				},
			},
			&mockOAuthConsentChallRepo{},
			&mockAuthEventService{},
		)

		_, oerr := svc.Authorize(ctx, validAuthorizeRequest(), 1, 1)
		require.NotNil(t, oerr)
		assert.Equal(t, "server_error", oerr.Code)
	})

	t.Run("consent challenge creation error", func(t *testing.T) {
		client := activeClientWithConsent()
		db, _ := newMockDB(t)

		svc := newOAuthAuthorizeSvc(db,
			&mockClientRepo{
				findByClientIDAndIdentityProviderFn: func(_, _ string) (*Client, error) {
					return client, nil
				},
			},
			&mockClientURIRepo{},
			&mockOAuthAuthCodeRepo{},
			&mockOAuthConsentGrantRepo{
				findByUserAndClientFn: func(_, _ int64) (*OAuthConsentGrant, error) {
					return nil, nil
				},
			},
			&mockOAuthConsentChallRepo{
				createFn: func(_ *OAuthConsentChallenge) (*OAuthConsentChallenge, error) {
					return nil, errors.New("create error")
				},
			},
			&mockAuthEventService{},
		)

		_, oerr := svc.Authorize(ctx, validAuthorizeRequest(), 1, 1)
		require.NotNil(t, oerr)
		assert.Equal(t, "server_error", oerr.Code)
	})

	t.Run("auth code creation error", func(t *testing.T) {
		client := activeClient()
		db, _ := newMockDB(t)

		svc := newOAuthAuthorizeSvc(db,
			&mockClientRepo{
				findByClientIDAndIdentityProviderFn: func(_, _ string) (*Client, error) {
					return client, nil
				},
			},
			&mockClientURIRepo{},
			&mockOAuthAuthCodeRepo{
				createFn: func(_ *OAuthAuthorizationCode) (*OAuthAuthorizationCode, error) {
					return nil, errors.New("create error")
				},
			},
			&mockOAuthConsentGrantRepo{},
			&mockOAuthConsentChallRepo{},
			&mockAuthEventService{},
		)

		_, oerr := svc.Authorize(ctx, validAuthorizeRequest(), 1, 1)
		require.NotNil(t, oerr)
		assert.Equal(t, "server_error", oerr.Code)
	})

	t.Run("fallback to findClientByIdentifier", func(t *testing.T) {
		db, _ := newMockDB(t)

		svc := newOAuthAuthorizeSvc(db,
			&mockClientRepo{
				findByClientIDAndIdentityProviderFn: func(_, _ string) (*Client, error) {
					return nil, nil
				},
				findByIdentifierFn: func(_ string) (*Client, error) {
					return nil, nil
				},
			},
			&mockClientURIRepo{},
			&mockOAuthAuthCodeRepo{},
			&mockOAuthConsentGrantRepo{},
			&mockOAuthConsentChallRepo{},
			&mockAuthEventService{},
		)

		_, oerr := svc.Authorize(ctx, validAuthorizeRequest(), 1, 1)
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_request", oerr.Code)
	})

	t.Run("fallback to findClientByIdentifier db error", func(t *testing.T) {
		db, _ := newMockDB(t)

		svc := newOAuthAuthorizeSvc(db,
			&mockClientRepo{
				findByClientIDAndIdentityProviderFn: func(_, _ string) (*Client, error) {
					return nil, nil
				},
				findByIdentifierFn: func(_ string) (*Client, error) {
					return nil, errors.New("connection error")
				},
			},
			&mockClientURIRepo{},
			&mockOAuthAuthCodeRepo{},
			&mockOAuthConsentGrantRepo{},
			&mockOAuthConsentChallRepo{},
			&mockAuthEventService{},
		)

		_, oerr := svc.Authorize(ctx, validAuthorizeRequest(), 1, 1)
		require.NotNil(t, oerr)
		assert.Equal(t, "server_error", oerr.Code)
	})

	t.Run("non-console system client_id is rejected on public authorize", func(t *testing.T) {
		client := activeClient()
		client.Name = "some-system-client"
		client.IsSystem = true
		db, _ := newMockDB(t)

		svc := newOAuthAuthorizeSvc(db,
			&mockClientRepo{
				findByClientIDAndIdentityProviderFn: func(_, _ string) (*Client, error) {
					return client, nil
				},
			},
			&mockClientURIRepo{},
			&mockOAuthAuthCodeRepo{},
			&mockOAuthConsentGrantRepo{},
			&mockOAuthConsentChallRepo{},
			&mockAuthEventService{},
		)

		_, oerr := svc.Authorize(ctx, validAuthorizeRequest(), 1, 1)

		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_request", oerr.Code)
	})

	t.Run("auth-console system client_id is allowed on public authorize", func(t *testing.T) {
		client := activeClient()
		client.Name = shared.SystemClientNameAuthConsole
		client.IsSystem = true
		db, _ := newMockDB(t)

		svc := newOAuthAuthorizeSvc(db,
			&mockClientRepo{
				findByClientIDAndIdentityProviderFn: func(_, _ string) (*Client, error) {
					return client, nil
				},
			},
			&mockClientURIRepo{},
			&mockOAuthAuthCodeRepo{},
			&mockOAuthConsentGrantRepo{},
			&mockOAuthConsentChallRepo{},
			&mockAuthEventService{},
		)

		result, oerr := svc.Authorize(ctx, validAuthorizeRequest(), 1, 1)

		require.Nil(t, oerr)
		require.NotNil(t, result)
		assert.Contains(t, result.RedirectURI, "https://example.com/callback?code=")
	})

	t.Run("authorize without state or nonce", func(t *testing.T) {
		client := activeClient()
		db, _ := newMockDB(t)

		svc := newOAuthAuthorizeSvc(db,
			&mockClientRepo{
				findByClientIDAndIdentityProviderFn: func(_, _ string) (*Client, error) {
					return client, nil
				},
			},
			&mockClientURIRepo{},
			&mockOAuthAuthCodeRepo{},
			&mockOAuthConsentGrantRepo{},
			&mockOAuthConsentChallRepo{},
			&mockAuthEventService{},
		)

		req := validAuthorizeRequest()
		req.State = ""
		req.Nonce = ""
		_, oerr := svc.Authorize(ctx, req, 1, 1)
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_request", oerr.Code)
		assert.Contains(t, oerr.Description, "state")
	})

	t.Run("consent challenge with state and nonce", func(t *testing.T) {
		client := activeClientWithConsent()
		db, _ := newMockDB(t)

		var capturedChallenge *OAuthConsentChallenge
		svc := newOAuthAuthorizeSvc(db,
			&mockClientRepo{
				findByClientIDAndIdentityProviderFn: func(_, _ string) (*Client, error) {
					return client, nil
				},
			},
			&mockClientURIRepo{},
			&mockOAuthAuthCodeRepo{},
			&mockOAuthConsentGrantRepo{
				findByUserAndClientFn: func(_, _ int64) (*OAuthConsentGrant, error) {
					return nil, nil
				},
			},
			&mockOAuthConsentChallRepo{
				createFn: func(c *OAuthConsentChallenge) (*OAuthConsentChallenge, error) {
					capturedChallenge = c
					c.OAuthConsentChallengeUUID = uuid.New()
					return c, nil
				},
			},
			&mockAuthEventService{},
		)

		req := validAuthorizeRequest()
		req.State = "mystate"
		req.Nonce = "mynonce"
		_, oerr := svc.Authorize(ctx, req, 1, 1)
		require.Nil(t, oerr)
		require.NotNil(t, capturedChallenge)
		require.NotNil(t, capturedChallenge.State)
		assert.Equal(t, "mystate", *capturedChallenge.State)
		require.NotNil(t, capturedChallenge.Nonce)
		assert.Equal(t, "mynonce", *capturedChallenge.Nonce)
	})
}

// ── Request-host tenant binding (SECURITY-CRITICAL) ─────────────────────────

// stubAuthorizeTenantResolver is a function-field test double for TenantResolver.
type stubAuthorizeTenantResolver struct {
	fn       func(ctx context.Context, name string) (int64, bool, error)
	systemFn func(ctx context.Context) (int64, error)
}

func (s stubAuthorizeTenantResolver) ResolveTenantIDByName(ctx context.Context, name string) (int64, bool, error) {
	if s.fn != nil {
		return s.fn(ctx, name)
	}
	return 0, false, nil
}

func (s stubAuthorizeTenantResolver) ResolveSystemTenantID(ctx context.Context) (int64, error) {
	if s.systemFn != nil {
		return s.systemFn(ctx)
	}
	return 0, nil
}

// withAuthorizeTenantResolver installs r as the package-level authorize tenant
// resolver for the duration of the test. oauth tests do not run in parallel, so
// the shared global is safe here.
func withAuthorizeTenantResolver(t *testing.T, r TenantResolver) {
	t.Helper()
	prev := authorizeTenantResolver
	authorizeTenantResolver = r
	t.Cleanup(func() { authorizeTenantResolver = prev })
}

// ctxWithRequestTenantSlug returns a context carrying a resolved request tenant
// with the given subdomain slug, as RequestTenantMiddleware would set it.
func ctxWithRequestTenantSlug(slug string) context.Context {
	return middleware.WithRequestTenant(context.Background(), middleware.RequestTenant{
		Surface: shared.FrontendSurfaceIdentity,
		Slug:    slug,
		OK:      true,
	})
}

// ctxWithRequestTenantSystem returns a context carrying a resolved request tenant
// for the bare system host (empty slug, IsSystem true), as
// RequestTenantMiddleware would set it.
func ctxWithRequestTenantSystem() context.Context {
	return middleware.WithRequestTenant(context.Background(), middleware.RequestTenant{
		Surface:  shared.FrontendSurfaceIdentity,
		IsSystem: true,
		OK:       true,
	})
}

// TestOAuthAuthorizeService_RequestHostTenantBinding exercises the STRICT
// enforcement that an authorize client_id belongs to the tenant identified by
// the REQUEST's own host, resolved server-side — not from a passed tenant_id.
func TestOAuthAuthorizeService_RequestHostTenantBinding(t *testing.T) {
	build := func(client *Client) OAuthAuthorizeService {
		db, _ := newMockDB(t)
		return newOAuthAuthorizeSvc(db,
			&mockClientRepo{
				findByClientIDAndIdentityProviderFn: func(_, _ string) (*Client, error) { return client, nil },
			},
			&mockClientURIRepo{},
			&mockOAuthAuthCodeRepo{},
			&mockOAuthConsentGrantRepo{},
			&mockOAuthConsentChallRepo{},
			&mockAuthEventService{},
		)
	}

	// (a) request host = tenant acme + client of acme → accepted. The request host
	// is authoritative even though the session tenant is a DIFFERENT value.
	t.Run("request host acme with acme client is accepted", func(t *testing.T) {
		const acmeTenantID int64 = 42
		withAuthorizeTenantResolver(t, stubAuthorizeTenantResolver{
			fn: func(_ context.Context, name string) (int64, bool, error) {
				assert.Equal(t, "acme", name)
				return acmeTenantID, false, nil
			},
		})
		client := activeClient()
		client.TenantID = acmeTenantID

		// Session tenant deliberately different (999) — proves the request host,
		// not the session tenant, drives the binding.
		result, oerr := build(client).Authorize(ctxWithRequestTenantSlug("acme"), validAuthorizeRequest(), 1, 999)
		require.Nil(t, oerr)
		require.NotNil(t, result)
		assert.Contains(t, result.RedirectURI, "code=")
	})

	// (b) request host = acme + client of another tenant → HARD reject. This is the
	// exact hole: the session tenant and the client agree (both tenant 1), so the
	// legacy session check would PASS — but the browser is on acme, so it must be
	// rejected with NO fallback to system.
	t.Run("request host acme with cross-tenant client is hard rejected", func(t *testing.T) {
		const acmeTenantID int64 = 42
		withAuthorizeTenantResolver(t, stubAuthorizeTenantResolver{
			fn: func(_ context.Context, _ string) (int64, bool, error) { return acmeTenantID, false, nil },
		})
		client := activeClient() // TenantID = 1 (system / other tenant)

		result, oerr := build(client).Authorize(ctxWithRequestTenantSlug("acme"), validAuthorizeRequest(), 1, 1)
		require.Nil(t, result)
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_request", oerr.Code)
	})

	// (b') the same hole at the PRE-AUTH PrepareAuthorize path — a cross-tenant
	// client must be rejected BEFORE any login page is shown.
	t.Run("PrepareAuthorize rejects cross-tenant client before login page", func(t *testing.T) {
		const acmeTenantID int64 = 42
		withAuthorizeTenantResolver(t, stubAuthorizeTenantResolver{
			fn: func(_ context.Context, _ string) (int64, bool, error) { return acmeTenantID, false, nil },
		})
		client := activeClient() // TenantID = 1

		oerr := build(client).PrepareAuthorize(ctxWithRequestTenantSlug("acme"), validAuthorizeRequest())
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_request", oerr.Code)
	})

	t.Run("PrepareAuthorize accepts a matching-tenant client", func(t *testing.T) {
		const acmeTenantID int64 = 42
		withAuthorizeTenantResolver(t, stubAuthorizeTenantResolver{
			fn: func(_ context.Context, _ string) (int64, bool, error) { return acmeTenantID, false, nil },
		})
		client := activeClient()
		client.TenantID = acmeTenantID

		oerr := build(client).PrepareAuthorize(ctxWithRequestTenantSlug("acme"), validAuthorizeRequest())
		assert.Nil(t, oerr)
	})

	// (c) unresolvable request host + valid session → the existing session-tenant
	// binding still applies (no weakening).
	t.Run("unresolvable host falls back to session binding: mismatch rejected", func(t *testing.T) {
		withAuthorizeTenantResolver(t, stubAuthorizeTenantResolver{
			fn: func(_ context.Context, _ string) (int64, bool, error) {
				t.Fatalf("resolver must not be consulted when the request host is unresolvable")
				return 0, false, nil
			},
		})
		client := activeClient() // TenantID = 1

		// No request tenant in context → falls back to session check; session
		// tenant 2 != client tenant 1 → rejected exactly as before.
		result, oerr := build(client).Authorize(context.Background(), validAuthorizeRequest(), 1, 2)
		require.Nil(t, result)
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_request", oerr.Code)
	})

	t.Run("unresolvable host falls back to session binding: match accepted", func(t *testing.T) {
		withAuthorizeTenantResolver(t, stubAuthorizeTenantResolver{})
		client := activeClient() // TenantID = 1

		result, oerr := build(client).Authorize(context.Background(), validAuthorizeRequest(), 1, 1)
		require.Nil(t, oerr)
		require.NotNil(t, result)
		assert.Contains(t, result.RedirectURI, "code=")
	})

	// A wired request-tenant slug that does not resolve (unknown tenant) falls
	// back to the session binding rather than failing open.
	t.Run("unknown request-tenant slug falls back to session binding", func(t *testing.T) {
		withAuthorizeTenantResolver(t, stubAuthorizeTenantResolver{
			fn: func(_ context.Context, _ string) (int64, bool, error) { return 0, false, nil },
		})
		client := activeClient() // TenantID = 1

		// Session tenant matches the client → accepted via fallback.
		result, oerr := build(client).Authorize(ctxWithRequestTenantSlug("ghost"), validAuthorizeRequest(), 1, 1)
		require.Nil(t, oerr)
		require.NotNil(t, result)
		assert.Contains(t, result.RedirectURI, "code=")
	})

	// (a) system client on the bare system host → accepted. The system host binds
	// strictly to the system tenant's ID (resolved via ResolveSystemTenantID), and
	// the client belongs to it.
	t.Run("system host with system-tenant client is accepted", func(t *testing.T) {
		const systemTenantID int64 = 1
		resolverCalled := false
		withAuthorizeTenantResolver(t, stubAuthorizeTenantResolver{
			fn: func(_ context.Context, _ string) (int64, bool, error) {
				t.Fatalf("ResolveTenantIDByName must not be called on the bare system host")
				return 0, false, nil
			},
			systemFn: func(_ context.Context) (int64, error) {
				resolverCalled = true
				return systemTenantID, nil
			},
		})
		client := activeClient() // TenantID = 1 = system tenant

		// Session tenant deliberately different (999) — the request host is
		// authoritative.
		result, oerr := build(client).Authorize(ctxWithRequestTenantSystem(), validAuthorizeRequest(), 1, 999)
		require.Nil(t, oerr)
		require.NotNil(t, result)
		assert.Contains(t, result.RedirectURI, "code=")
		assert.True(t, resolverCalled, "system-tenant resolver must be consulted on the bare system host")
	})

	// (b) a regular tenant's client on the bare system host → HARD reject. This is
	// the reverse hole: the session tenant and client agree (both tenant 42), so
	// the legacy session check would PASS — but the browser is on the system host,
	// so it must be rejected with NO fallback.
	t.Run("system host with regular-tenant client is hard rejected", func(t *testing.T) {
		const systemTenantID int64 = 1
		withAuthorizeTenantResolver(t, stubAuthorizeTenantResolver{
			systemFn: func(_ context.Context) (int64, error) { return systemTenantID, nil },
		})
		client := activeClient()
		client.TenantID = 42 // regular tenant, not the system tenant

		result, oerr := build(client).Authorize(ctxWithRequestTenantSystem(), validAuthorizeRequest(), 1, 42)
		require.Nil(t, result)
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_request", oerr.Code)
	})

	// PrepareAuthorize enforces the system-host binding before any login page too.
	t.Run("PrepareAuthorize rejects regular-tenant client on the system host", func(t *testing.T) {
		const systemTenantID int64 = 1
		withAuthorizeTenantResolver(t, stubAuthorizeTenantResolver{
			systemFn: func(_ context.Context) (int64, error) { return systemTenantID, nil },
		})
		client := activeClient()
		client.TenantID = 42

		oerr := build(client).PrepareAuthorize(ctxWithRequestTenantSystem(), validAuthorizeRequest())
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_request", oerr.Code)
	})
}

// ── TestOAuthAuthorizeService_GetConsentChallenge ───────────────────────────

func TestOAuthAuthorizeService_GetConsentChallenge(t *testing.T) {
	ctx := context.Background()
	challengeUUID := uuid.New()
	clientUUID := uuid.New()

	t.Run("success", func(t *testing.T) {
		db, _ := newMockDB(t)
		svc := newOAuthAuthorizeSvc(db,
			&mockClientRepo{}, &mockClientURIRepo{}, &mockOAuthAuthCodeRepo{},
			&mockOAuthConsentGrantRepo{},
			&mockOAuthConsentChallRepo{
				findChallengeByUUIDFn: func(_ uuid.UUID) (*OAuthConsentChallenge, error) {
					return &OAuthConsentChallenge{
						OAuthConsentChallengeUUID: challengeUUID,
						UserID:                    1,
						Scope:                     parseScopeFields("openid profile"),
						RedirectURI:               "https://example.com/callback",
						ExpiresAt:                 time.Now().Add(5 * time.Minute),
						Client: &Client{
							ClientUUID:  clientUUID,
							DisplayName: "Test App",
						},
					}, nil
				},
			},
			&mockAuthEventService{},
		)

		result, err := svc.GetConsentChallenge(ctx, challengeUUID, 1)
		require.NoError(t, err)
		assert.Equal(t, challengeUUID.String(), result.ChallengeID)
		assert.Equal(t, "Test App", result.ClientName)
		assert.Equal(t, clientUUID.String(), result.ClientUUID)
		assert.Equal(t, []string{"openid", "profile"}, result.Scopes)
	})

	t.Run("nil client", func(t *testing.T) {
		db, _ := newMockDB(t)
		svc := newOAuthAuthorizeSvc(db,
			&mockClientRepo{}, &mockClientURIRepo{}, &mockOAuthAuthCodeRepo{},
			&mockOAuthConsentGrantRepo{},
			&mockOAuthConsentChallRepo{
				findChallengeByUUIDFn: func(_ uuid.UUID) (*OAuthConsentChallenge, error) {
					return &OAuthConsentChallenge{
						OAuthConsentChallengeUUID: challengeUUID,
						UserID:                    1,
						Scope:                     parseScopeFields("openid"),
						RedirectURI:               "https://example.com/callback",
						ExpiresAt:                 time.Now().Add(5 * time.Minute),
					}, nil
				},
			},
			&mockAuthEventService{},
		)

		result, err := svc.GetConsentChallenge(ctx, challengeUUID, 1)
		require.NoError(t, err)
		assert.Empty(t, result.ClientName)
		assert.Empty(t, result.ClientUUID)
	})

	t.Run("not found", func(t *testing.T) {
		db, _ := newMockDB(t)
		svc := newOAuthAuthorizeSvc(db,
			&mockClientRepo{}, &mockClientURIRepo{}, &mockOAuthAuthCodeRepo{},
			&mockOAuthConsentGrantRepo{},
			&mockOAuthConsentChallRepo{
				findChallengeByUUIDFn: func(_ uuid.UUID) (*OAuthConsentChallenge, error) {
					return nil, nil
				},
			},
			&mockAuthEventService{},
		)

		_, err := svc.GetConsentChallenge(ctx, uuid.New(), 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "consent challenge not found")
	})

	t.Run("user mismatch", func(t *testing.T) {
		db, _ := newMockDB(t)
		svc := newOAuthAuthorizeSvc(db,
			&mockClientRepo{}, &mockClientURIRepo{}, &mockOAuthAuthCodeRepo{},
			&mockOAuthConsentGrantRepo{},
			&mockOAuthConsentChallRepo{
				findChallengeByUUIDFn: func(_ uuid.UUID) (*OAuthConsentChallenge, error) {
					return &OAuthConsentChallenge{
						UserID:    999,
						ExpiresAt: time.Now().Add(5 * time.Minute),
					}, nil
				},
			},
			&mockAuthEventService{},
		)

		_, err := svc.GetConsentChallenge(ctx, uuid.New(), 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "does not belong")
	})

	t.Run("expired", func(t *testing.T) {
		db, _ := newMockDB(t)
		svc := newOAuthAuthorizeSvc(db,
			&mockClientRepo{}, &mockClientURIRepo{}, &mockOAuthAuthCodeRepo{},
			&mockOAuthConsentGrantRepo{},
			&mockOAuthConsentChallRepo{
				findChallengeByUUIDFn: func(_ uuid.UUID) (*OAuthConsentChallenge, error) {
					return &OAuthConsentChallenge{
						UserID:    1,
						ExpiresAt: time.Now().Add(-1 * time.Minute),
					}, nil
				},
			},
			&mockAuthEventService{},
		)

		_, err := svc.GetConsentChallenge(ctx, uuid.New(), 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "expired")
	})

	t.Run("repo error", func(t *testing.T) {
		db, _ := newMockDB(t)
		svc := newOAuthAuthorizeSvc(db,
			&mockClientRepo{}, &mockClientURIRepo{}, &mockOAuthAuthCodeRepo{},
			&mockOAuthConsentGrantRepo{},
			&mockOAuthConsentChallRepo{
				findChallengeByUUIDFn: func(_ uuid.UUID) (*OAuthConsentChallenge, error) {
					return nil, errors.New("db error")
				},
			},
			&mockAuthEventService{},
		)

		_, err := svc.GetConsentChallenge(ctx, uuid.New(), 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to retrieve")
	})
}

// ── TestOAuthAuthorizeService_HandleConsent ──────────────────────────────────

func TestOAuthAuthorizeService_HandleConsent(t *testing.T) {
	ctx := context.Background()
	challengeUUID := uuid.New()

	validDecision := func(approved bool) OAuthConsentDecisionDTO {
		return OAuthConsentDecisionDTO{
			ChallengeID: challengeUUID.String(),
			Approved:    approved,
		}
	}

	t.Run("approved — issues code", func(t *testing.T) {
		state := "mystate"
		db, mock := newMockDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()

		svc := newOAuthAuthorizeSvc(db,
			&mockClientRepo{}, &mockClientURIRepo{},
			&mockOAuthAuthCodeRepo{},
			&mockOAuthConsentGrantRepo{},
			&mockOAuthConsentChallRepo{
				findChallengeByUUIDFn: func(_ uuid.UUID) (*OAuthConsentChallenge, error) {
					return &OAuthConsentChallenge{
						OAuthConsentChallengeUUID: challengeUUID,
						ClientID:                  10,
						UserID:                    1,
						TenantID:                  100,
						RedirectURI:               "https://example.com/callback",
						Scope:                     parseScopeFields("openid profile"),
						CodeChallenge:             strings.Repeat("A", 43),
						CodeChallengeMethod:       "S256",
						State:                     &state,
						ExpiresAt:                 time.Now().Add(5 * time.Minute),
					}, nil
				},
			},
			&mockAuthEventService{},
		)

		result, oerr := svc.HandleConsent(ctx, validDecision(true), 1)
		require.Nil(t, oerr)
		require.NotNil(t, result)
		assert.Contains(t, result.RedirectURI, "code=")
		assert.Contains(t, result.RedirectURI, "state=mystate")
	})

	t.Run("approved — GenerateRandomString error in transaction", func(t *testing.T) {
		db, mock := newMockDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()

		orig := crypto.GenerateRandomString
		defer func() { crypto.GenerateRandomString = orig }()
		crypto.GenerateRandomString = func(int) (string, error) { return "", errors.New("rand failure") }

		svc := newOAuthAuthorizeSvc(db,
			&mockClientRepo{}, &mockClientURIRepo{},
			&mockOAuthAuthCodeRepo{},
			&mockOAuthConsentGrantRepo{},
			&mockOAuthConsentChallRepo{
				findChallengeByUUIDFn: func(_ uuid.UUID) (*OAuthConsentChallenge, error) {
					return &OAuthConsentChallenge{
						OAuthConsentChallengeUUID: challengeUUID,
						ClientID:                  10,
						UserID:                    1,
						TenantID:                  100,
						RedirectURI:               "https://example.com/callback",
						ExpiresAt:                 time.Now().Add(5 * time.Minute),
					}, nil
				},
			},
			&mockAuthEventService{},
		)

		_, oerr := svc.HandleConsent(ctx, validDecision(true), 1)
		require.NotNil(t, oerr)
		assert.Equal(t, "server_error", oerr.Code)
	})

	t.Run("denied — returns error redirect", func(t *testing.T) {
		state := "mystate"
		db, _ := newMockDB(t)

		svc := newOAuthAuthorizeSvc(db,
			&mockClientRepo{}, &mockClientURIRepo{},
			&mockOAuthAuthCodeRepo{},
			&mockOAuthConsentGrantRepo{},
			&mockOAuthConsentChallRepo{
				findChallengeByUUIDFn: func(_ uuid.UUID) (*OAuthConsentChallenge, error) {
					return &OAuthConsentChallenge{
						OAuthConsentChallengeUUID: challengeUUID,
						ClientID:                  10,
						UserID:                    1,
						TenantID:                  100,
						RedirectURI:               "https://example.com/callback",
						State:                     &state,
						ExpiresAt:                 time.Now().Add(5 * time.Minute),
					}, nil
				},
			},
			&mockAuthEventService{},
		)

		result, oerr := svc.HandleConsent(ctx, validDecision(false), 1)
		// Denial is not an OAuth error — it returns a redirect with error param
		require.Nil(t, oerr)
		require.NotNil(t, result)
		assert.Contains(t, result.RedirectURI, "error=access_denied")
		assert.Contains(t, result.RedirectURI, "state=mystate")
	})

	t.Run("challenge not found", func(t *testing.T) {
		db, _ := newMockDB(t)

		svc := newOAuthAuthorizeSvc(db,
			&mockClientRepo{}, &mockClientURIRepo{},
			&mockOAuthAuthCodeRepo{},
			&mockOAuthConsentGrantRepo{},
			&mockOAuthConsentChallRepo{
				findChallengeByUUIDFn: func(_ uuid.UUID) (*OAuthConsentChallenge, error) {
					return nil, nil
				},
			},
			&mockAuthEventService{},
		)

		_, oerr := svc.HandleConsent(ctx, validDecision(true), 1)
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_request", oerr.Code)
	})

	t.Run("challenge expired", func(t *testing.T) {
		db, _ := newMockDB(t)

		svc := newOAuthAuthorizeSvc(db,
			&mockClientRepo{}, &mockClientURIRepo{},
			&mockOAuthAuthCodeRepo{},
			&mockOAuthConsentGrantRepo{},
			&mockOAuthConsentChallRepo{
				findChallengeByUUIDFn: func(_ uuid.UUID) (*OAuthConsentChallenge, error) {
					return &OAuthConsentChallenge{
						OAuthConsentChallengeUUID: challengeUUID,
						UserID:                    1,
						ExpiresAt:                 time.Now().Add(-1 * time.Minute),
					}, nil
				},
			},
			&mockAuthEventService{},
		)

		_, oerr := svc.HandleConsent(ctx, validDecision(true), 1)
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_request", oerr.Code)
	})

	t.Run("challenge user mismatch", func(t *testing.T) {
		db, _ := newMockDB(t)

		svc := newOAuthAuthorizeSvc(db,
			&mockClientRepo{}, &mockClientURIRepo{},
			&mockOAuthAuthCodeRepo{},
			&mockOAuthConsentGrantRepo{},
			&mockOAuthConsentChallRepo{
				findChallengeByUUIDFn: func(_ uuid.UUID) (*OAuthConsentChallenge, error) {
					return &OAuthConsentChallenge{
						OAuthConsentChallengeUUID: challengeUUID,
						UserID:                    999,
						ExpiresAt:                 time.Now().Add(5 * time.Minute),
					}, nil
				},
			},
			&mockAuthEventService{},
		)

		_, oerr := svc.HandleConsent(ctx, validDecision(true), 1)
		require.NotNil(t, oerr)
		assert.Equal(t, "access_denied", oerr.Code)
	})

	t.Run("challenge lookup error", func(t *testing.T) {
		db, _ := newMockDB(t)

		svc := newOAuthAuthorizeSvc(db,
			&mockClientRepo{}, &mockClientURIRepo{},
			&mockOAuthAuthCodeRepo{},
			&mockOAuthConsentGrantRepo{},
			&mockOAuthConsentChallRepo{
				findChallengeByUUIDFn: func(_ uuid.UUID) (*OAuthConsentChallenge, error) {
					return nil, errors.New("db error")
				},
			},
			&mockAuthEventService{},
		)

		_, oerr := svc.HandleConsent(ctx, validDecision(true), 1)
		require.NotNil(t, oerr)
		assert.Equal(t, "server_error", oerr.Code)
	})

	t.Run("transaction error on approve", func(t *testing.T) {
		db, mock := newMockDB(t)
		mock.ExpectBegin().WillReturnError(errors.New("tx error"))

		svc := newOAuthAuthorizeSvc(db,
			&mockClientRepo{}, &mockClientURIRepo{},
			&mockOAuthAuthCodeRepo{},
			&mockOAuthConsentGrantRepo{},
			&mockOAuthConsentChallRepo{
				findChallengeByUUIDFn: func(_ uuid.UUID) (*OAuthConsentChallenge, error) {
					return &OAuthConsentChallenge{
						OAuthConsentChallengeUUID: challengeUUID,
						ClientID:                  10,
						UserID:                    1,
						TenantID:                  100,
						RedirectURI:               "https://example.com/callback",
						ExpiresAt:                 time.Now().Add(5 * time.Minute),
					}, nil
				},
			},
			&mockAuthEventService{},
		)

		_, oerr := svc.HandleConsent(ctx, validDecision(true), 1)
		require.NotNil(t, oerr)
		assert.Equal(t, "server_error", oerr.Code)
	})

	t.Run("denied with nil state", func(t *testing.T) {
		db, _ := newMockDB(t)

		svc := newOAuthAuthorizeSvc(db,
			&mockClientRepo{}, &mockClientURIRepo{},
			&mockOAuthAuthCodeRepo{},
			&mockOAuthConsentGrantRepo{},
			&mockOAuthConsentChallRepo{
				findChallengeByUUIDFn: func(_ uuid.UUID) (*OAuthConsentChallenge, error) {
					return &OAuthConsentChallenge{
						OAuthConsentChallengeUUID: challengeUUID,
						ClientID:                  10,
						UserID:                    1,
						TenantID:                  100,
						RedirectURI:               "https://example.com/callback",
						ExpiresAt:                 time.Now().Add(5 * time.Minute),
					}, nil
				},
			},
			&mockAuthEventService{},
		)

		result, oerr := svc.HandleConsent(ctx, validDecision(false), 1)
		require.Nil(t, oerr)
		assert.Contains(t, result.RedirectURI, "error=access_denied")
		assert.NotContains(t, result.RedirectURI, "state=")
	})

	t.Run("denied with delete error still returns redirect", func(t *testing.T) {
		db, _ := newMockDB(t)

		svc := newOAuthAuthorizeSvc(db,
			&mockClientRepo{}, &mockClientURIRepo{},
			&mockOAuthAuthCodeRepo{},
			&mockOAuthConsentGrantRepo{},
			&mockOAuthConsentChallRepo{
				findChallengeByUUIDFn: func(_ uuid.UUID) (*OAuthConsentChallenge, error) {
					return &OAuthConsentChallenge{
						OAuthConsentChallengeUUID: challengeUUID,
						ClientID:                  10,
						UserID:                    1,
						TenantID:                  100,
						RedirectURI:               "https://example.com/callback",
						ExpiresAt:                 time.Now().Add(5 * time.Minute),
					}, nil
				},
				deleteChallengeByUUIDFn: func(_ uuid.UUID) error {
					return errors.New("delete failed")
				},
			},
			&mockAuthEventService{},
		)

		result, oerr := svc.HandleConsent(ctx, validDecision(false), 1)
		require.Nil(t, oerr)
		assert.Contains(t, result.RedirectURI, "error=access_denied")
	})

	t.Run("upsert error in transaction", func(t *testing.T) {
		db, mock := newMockDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()

		svc := newOAuthAuthorizeSvc(db,
			&mockClientRepo{}, &mockClientURIRepo{},
			&mockOAuthAuthCodeRepo{},
			&mockOAuthConsentGrantRepo{
				upsertFn: func(_ *OAuthConsentGrant) (*OAuthConsentGrant, error) {
					return nil, errors.New("upsert error")
				},
			},
			&mockOAuthConsentChallRepo{
				findChallengeByUUIDFn: func(_ uuid.UUID) (*OAuthConsentChallenge, error) {
					return &OAuthConsentChallenge{
						OAuthConsentChallengeUUID: challengeUUID,
						ClientID:                  10,
						UserID:                    1,
						TenantID:                  100,
						RedirectURI:               "https://example.com/callback",
						ExpiresAt:                 time.Now().Add(5 * time.Minute),
					}, nil
				},
			},
			&mockAuthEventService{},
		)

		_, oerr := svc.HandleConsent(ctx, validDecision(true), 1)
		require.NotNil(t, oerr)
		assert.Equal(t, "server_error", oerr.Code)
	})

	t.Run("auth code create error in transaction", func(t *testing.T) {
		db, mock := newMockDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()

		svc := newOAuthAuthorizeSvc(db,
			&mockClientRepo{}, &mockClientURIRepo{},
			&mockOAuthAuthCodeRepo{
				createFn: func(_ *OAuthAuthorizationCode) (*OAuthAuthorizationCode, error) {
					return nil, errors.New("create error")
				},
			},
			&mockOAuthConsentGrantRepo{},
			&mockOAuthConsentChallRepo{
				findChallengeByUUIDFn: func(_ uuid.UUID) (*OAuthConsentChallenge, error) {
					return &OAuthConsentChallenge{
						OAuthConsentChallengeUUID: challengeUUID,
						ClientID:                  10,
						UserID:                    1,
						TenantID:                  100,
						RedirectURI:               "https://example.com/callback",
						ExpiresAt:                 time.Now().Add(5 * time.Minute),
					}, nil
				},
			},
			&mockAuthEventService{},
		)

		_, oerr := svc.HandleConsent(ctx, validDecision(true), 1)
		require.NotNil(t, oerr)
		assert.Equal(t, "server_error", oerr.Code)
	})
}

// ── TestBuildAuthCodeRedirect ───────────────────────────────────────────────

func TestBuildAuthCodeRedirect(t *testing.T) {
	t.Run("simple URI", func(t *testing.T) {
		u := buildAuthCodeRedirect("https://example.com/callback", "CODE123", "STATE")
		assert.Equal(t, "https://example.com/callback?code=CODE123&state=STATE", u)
	})

	t.Run("URI with existing query params", func(t *testing.T) {
		// Parameter ORDER changed when this moved to url.Values, which encodes
		// keys sorted. The registered URI's own parameters are still preserved,
		// which is the part the test is about.
		u := buildAuthCodeRedirect("https://example.com/callback?foo=bar", "CODE123", "STATE")
		assert.Equal(t, "https://example.com/callback?code=CODE123&foo=bar&state=STATE", u)
	})

	t.Run("no state", func(t *testing.T) {
		u := buildAuthCodeRedirect("https://example.com/callback", "CODE123", "")
		assert.Equal(t, "https://example.com/callback?code=CODE123", u)
	})

	// state is client-controlled and reaches this function already URL-decoded,
	// so raw concatenation let it re-partition the callback query.
	t.Run("state cannot inject extra query parameters", func(t *testing.T) {
		u := buildAuthCodeRedirect("https://example.com/callback", "CODE123", "x&code=attacker")
		parsed, err := url.Parse(u)
		require.NoError(t, err)
		assert.Equal(t, []string{"CODE123"}, parsed.Query()["code"],
			"the injected code must not appear as a second code parameter")
		assert.Equal(t, "x&code=attacker", parsed.Query().Get("state"))
	})

	t.Run("state cannot truncate the query into a fragment", func(t *testing.T) {
		u := buildAuthCodeRedirect("https://example.com/callback", "CODE123", "x#frag")
		parsed, err := url.Parse(u)
		require.NoError(t, err)
		assert.Empty(t, parsed.Fragment)
		assert.Equal(t, "x#frag", parsed.Query().Get("state"))
		assert.Equal(t, "CODE123", parsed.Query().Get("code"))
	})
}

// ── TestSplitScopes ─────────────────────────────────────────────────────────

func TestSplitScopes(t *testing.T) {
	t.Run("multiple scopes", func(t *testing.T) {
		assert.Equal(t, []string{"openid", "profile", "email"}, splitScopes("openid profile email"))
	})

	t.Run("single scope", func(t *testing.T) {
		assert.Equal(t, []string{"openid"}, splitScopes("openid"))
	})

	t.Run("empty string", func(t *testing.T) {
		assert.Nil(t, splitScopes(""))
	})

	t.Run("extra whitespace", func(t *testing.T) {
		assert.Equal(t, []string{"a", "b"}, splitScopes("  a   b  "))
	})
}

// ── TestOAuthAuthorizeService_findClientByIdentifier ────────────────────────

func TestOAuthAuthorizeService_findClientByIdentifier(t *testing.T) {
	t.Run("successful lookup", func(t *testing.T) {
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
			pq.StringArray{GrantTypeAuthorizationCode}, pq.StringArray{ResponseTypeCode}, nil, nil,
			false, time.Now(), time.Now(),
		)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).WillReturnRows(rows)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).WillReturnRows(sqlmock.NewRows(nil))
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).WillReturnRows(sqlmock.NewRows(nil))
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).WillReturnRows(sqlmock.NewRows(nil))

		svc := &oauthAuthorizeService{db: db}
		client, err := svc.findClientByIdentifier("my-client")
		require.NoError(t, err)
		require.NotNil(t, client)
		assert.Equal(t, int64(10), client.ClientID)
	})
}

// ── TestOAuthAuthorizeService_validateRedirectURI ────────────────────────────

func TestOAuthAuthorizeService_validateRedirectURI(t *testing.T) {
	t.Run("dangerous scheme", func(t *testing.T) {
		svc := &oauthAuthorizeService{}
		client := &Client{}
		oerr := svc.validateRedirectURI(client, "javascript:alert(1)")
		require.NotNil(t, oerr)
		assert.Contains(t, oerr.Description, "forbidden scheme")
	})
}

// ── TestOAuthAuthorizeService_Authorize_Additional ──────────────────────────

func TestOAuthAuthorizeService_Authorize_Additional(t *testing.T) {
	ctx := context.Background()

	t.Run("scope not allowed", func(t *testing.T) {
		client := activeClient()
		client.AllowedScopes = pq.StringArray{"openid"}
		db, _ := newMockDB(t)

		svc := newOAuthAuthorizeSvc(db,
			&mockClientRepo{
				findByClientIDAndIdentityProviderFn: func(_, _ string) (*Client, error) {
					return client, nil
				},
			},
			&mockClientURIRepo{},
			&mockOAuthAuthCodeRepo{},
			&mockOAuthConsentGrantRepo{},
			&mockOAuthConsentChallRepo{},
			&mockAuthEventService{},
		)

		req := validAuthorizeRequest()
		req.Scope = "openid admin"
		_, oerr := svc.Authorize(ctx, req, 1, 1)
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_scope", oerr.Code)
	})

	t.Run("id_token response_type with nonce", func(t *testing.T) {
		client := activeClient()
		client.ResponseTypes = pq.StringArray{"code id_token"}
		db, _ := newMockDB(t)

		svc := newOAuthAuthorizeSvc(db,
			&mockClientRepo{
				findByClientIDAndIdentityProviderFn: func(_, _ string) (*Client, error) {
					return client, nil
				},
			},
			&mockClientURIRepo{},
			&mockOAuthAuthCodeRepo{},
			&mockOAuthConsentGrantRepo{},
			&mockOAuthConsentChallRepo{},
			&mockAuthEventService{},
		)

		req := validAuthorizeRequest()
		req.ResponseType = "code id_token"
		req.Nonce = "nonce123"
		result, oerr := svc.Authorize(ctx, req, 1, 1)
		require.Nil(t, oerr)
		require.NotNil(t, result)
		assert.Contains(t, result.RedirectURI, "code=")
	})

	t.Run("id_token response_type without nonce", func(t *testing.T) {
		client := activeClient()
		client.ResponseTypes = pq.StringArray{"code id_token"}
		db, _ := newMockDB(t)

		svc := newOAuthAuthorizeSvc(db,
			&mockClientRepo{
				findByClientIDAndIdentityProviderFn: func(_, _ string) (*Client, error) {
					return client, nil
				},
			},
			&mockClientURIRepo{},
			&mockOAuthAuthCodeRepo{},
			&mockOAuthConsentGrantRepo{},
			&mockOAuthConsentChallRepo{},
			&mockAuthEventService{},
		)

		req := validAuthorizeRequest()
		req.ResponseType = "code id_token"
		req.Nonce = ""
		_, oerr := svc.Authorize(ctx, req, 1, 1)
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_request", oerr.Code)
		assert.Contains(t, oerr.Description, "nonce")
	})

	t.Run("issueAuthorizationCode with empty nonce", func(t *testing.T) {
		client := activeClient()
		db, _ := newMockDB(t)

		svc := newOAuthAuthorizeSvc(db,
			&mockClientRepo{
				findByClientIDAndIdentityProviderFn: func(_, _ string) (*Client, error) {
					return client, nil
				},
			},
			&mockClientURIRepo{},
			&mockOAuthAuthCodeRepo{},
			&mockOAuthConsentGrantRepo{},
			&mockOAuthConsentChallRepo{},
			&mockAuthEventService{},
		)

		req := validAuthorizeRequest()
		req.Nonce = ""
		result, oerr := svc.Authorize(ctx, req, 1, 1)
		require.Nil(t, oerr)
		require.NotNil(t, result)
		assert.Contains(t, result.RedirectURI, "code=")
	})

	t.Run("issueAuthorizationCode GenerateRandomString error", func(t *testing.T) {
		client := activeClient()
		db, _ := newMockDB(t)

		orig := crypto.GenerateRandomString
		defer func() { crypto.GenerateRandomString = orig }()
		crypto.GenerateRandomString = func(int) (string, error) { return "", errors.New("rand failure") }

		svc := newOAuthAuthorizeSvc(db,
			&mockClientRepo{
				findByClientIDAndIdentityProviderFn: func(_, _ string) (*Client, error) {
					return client, nil
				},
			},
			&mockClientURIRepo{},
			&mockOAuthAuthCodeRepo{},
			&mockOAuthConsentGrantRepo{},
			&mockOAuthConsentChallRepo{},
			&mockAuthEventService{},
		)

		_, oerr := svc.Authorize(ctx, validAuthorizeRequest(), 1, 1)
		require.NotNil(t, oerr)
		assert.Equal(t, "server_error", oerr.Code)
	})
}

// ── TestOAuthAuthorizeService_HandleConsent_Additional ──────────────────────

func TestOAuthAuthorizeService_HandleConsent_Additional(t *testing.T) {
	ctx := context.Background()
	challengeUUID := uuid.New()

	t.Run("approved with nil state", func(t *testing.T) {
		db, mock := newMockDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()

		svc := newOAuthAuthorizeSvc(db,
			&mockClientRepo{}, &mockClientURIRepo{},
			&mockOAuthAuthCodeRepo{},
			&mockOAuthConsentGrantRepo{},
			&mockOAuthConsentChallRepo{
				findChallengeByUUIDFn: func(_ uuid.UUID) (*OAuthConsentChallenge, error) {
					return &OAuthConsentChallenge{
						OAuthConsentChallengeUUID: challengeUUID,
						ClientID:                  10,
						UserID:                    1,
						TenantID:                  100,
						RedirectURI:               "https://example.com/callback",
						Scope:                     parseScopeFields("openid profile"),
						CodeChallenge:             strings.Repeat("A", 43),
						CodeChallengeMethod:       "S256",
						ExpiresAt:                 time.Now().Add(5 * time.Minute),
					}, nil
				},
			},
			&mockAuthEventService{},
		)

		decision := OAuthConsentDecisionDTO{
			ChallengeID: challengeUUID.String(),
			Approved:    true,
		}
		result, oerr := svc.HandleConsent(ctx, decision, 1)
		require.Nil(t, oerr)
		require.NotNil(t, result)
		assert.Contains(t, result.RedirectURI, "code=")
		assert.NotContains(t, result.RedirectURI, "state=")
	})
}
