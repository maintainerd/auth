package service

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/maintainerd/auth/internal/dto"
	"github.com/maintainerd/auth/internal/model"
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
	return NewOAuthAuthorizeService(db, clientRepo, clientURIRepo, authCodeRepo, consentGrantRepo, consentChallRepo, authEventSvc)
}

func validAuthorizeRequest() dto.OAuthAuthorizeRequestDTO {
	return dto.OAuthAuthorizeRequestDTO{
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

func activeClient() *model.Client {
	return &model.Client{
		ClientID:       10,
		ClientUUID:     uuid.New(),
		TenantID:       1,
		Status:         model.StatusActive,
		GrantTypes:     pq.StringArray{model.GrantTypeAuthorizationCode},
		ResponseTypes:  pq.StringArray{model.ResponseTypeCode},
		RequireConsent: false,
		ClientURIs: &[]model.ClientURI{
			{URI: "https://example.com/callback", Type: model.ClientURITypeRedirect},
		},
	}
}

func activeClientWithConsent() *model.Client {
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

// ── TestOAuthAuthorizeService_Authorize ─────────────────────────────────────

func TestOAuthAuthorizeService_Authorize(t *testing.T) {
	ctx := context.Background()

	t.Run("issues code when consent not required", func(t *testing.T) {
		client := activeClient()
		db, _ := newMockDB(t)

		svc := newOAuthAuthorizeSvc(db,
			&mockClientRepo{
				findByClientIDAndIdentityProviderFn: func(_, _ string) (*model.Client, error) {
					return client, nil
				},
			},
			&mockClientURIRepo{},
			&mockOAuthAuthCodeRepo{},
			&mockOAuthConsentGrantRepo{},
			&mockOAuthConsentChallRepo{},
			&mockAuthEventService{},
		)

		result, oerr := svc.Authorize(ctx, validAuthorizeRequest(), 1)
		require.Nil(t, oerr)
		require.NotNil(t, result)
		assert.Contains(t, result.RedirectURI, "code=")
		assert.Contains(t, result.RedirectURI, "state=state123")
		assert.Empty(t, result.ConsentChallenge)
	})

	t.Run("returns consent challenge when consent required", func(t *testing.T) {
		client := activeClientWithConsent()
		db, _ := newMockDB(t)
		challengeUUID := uuid.New()

		svc := newOAuthAuthorizeSvc(db,
			&mockClientRepo{
				findByClientIDAndIdentityProviderFn: func(_, _ string) (*model.Client, error) {
					return client, nil
				},
			},
			&mockClientURIRepo{},
			&mockOAuthAuthCodeRepo{},
			&mockOAuthConsentGrantRepo{
				findByUserAndClientFn: func(_, _ int64) (*model.OAuthConsentGrant, error) {
					return nil, nil // no existing grant
				},
			},
			&mockOAuthConsentChallRepo{
				createFn: func(c *model.OAuthConsentChallenge) (*model.OAuthConsentChallenge, error) {
					c.OAuthConsentChallengeUUID = challengeUUID
					return c, nil
				},
			},
			&mockAuthEventService{},
		)

		result, oerr := svc.Authorize(ctx, validAuthorizeRequest(), 1)
		require.Nil(t, oerr)
		require.NotNil(t, result)
		assert.Equal(t, challengeUUID.String(), result.ConsentChallenge)
		assert.Empty(t, result.RedirectURI)
	})

	t.Run("skips consent when all scopes already granted", func(t *testing.T) {
		client := activeClientWithConsent()
		db, _ := newMockDB(t)

		svc := newOAuthAuthorizeSvc(db,
			&mockClientRepo{
				findByClientIDAndIdentityProviderFn: func(_, _ string) (*model.Client, error) {
					return client, nil
				},
			},
			&mockClientURIRepo{},
			&mockOAuthAuthCodeRepo{},
			&mockOAuthConsentGrantRepo{
				findByUserAndClientFn: func(_, _ int64) (*model.OAuthConsentGrant, error) {
					return &model.OAuthConsentGrant{Scopes: "openid profile email"}, nil
				},
			},
			&mockOAuthConsentChallRepo{},
			&mockAuthEventService{},
		)

		result, oerr := svc.Authorize(ctx, validAuthorizeRequest(), 1)
		require.Nil(t, oerr)
		assert.Contains(t, result.RedirectURI, "code=")
	})

	t.Run("requires consent when new scope requested", func(t *testing.T) {
		client := activeClientWithConsent()
		db, _ := newMockDB(t)

		svc := newOAuthAuthorizeSvc(db,
			&mockClientRepo{
				findByClientIDAndIdentityProviderFn: func(_, _ string) (*model.Client, error) {
					return client, nil
				},
			},
			&mockClientURIRepo{},
			&mockOAuthAuthCodeRepo{},
			&mockOAuthConsentGrantRepo{
				findByUserAndClientFn: func(_, _ int64) (*model.OAuthConsentGrant, error) {
					return &model.OAuthConsentGrant{Scopes: "openid"}, nil // missing "profile"
				},
			},
			&mockOAuthConsentChallRepo{},
			&mockAuthEventService{},
		)

		result, oerr := svc.Authorize(ctx, validAuthorizeRequest(), 1)
		require.Nil(t, oerr)
		assert.NotEmpty(t, result.ConsentChallenge)
	})

	t.Run("client not found", func(t *testing.T) {
		db, mock := newMockDB(t)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).WillReturnError(gorm.ErrRecordNotFound)

		svc := newOAuthAuthorizeSvc(db,
			&mockClientRepo{
				findByClientIDAndIdentityProviderFn: func(_, _ string) (*model.Client, error) {
					return nil, nil
				},
			},
			&mockClientURIRepo{},
			&mockOAuthAuthCodeRepo{},
			&mockOAuthConsentGrantRepo{},
			&mockOAuthConsentChallRepo{},
			&mockAuthEventService{},
		)

		_, oerr := svc.Authorize(ctx, validAuthorizeRequest(), 1)
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_request", oerr.Code)
	})

	t.Run("client lookup error", func(t *testing.T) {
		db, _ := newMockDB(t)

		svc := newOAuthAuthorizeSvc(db,
			&mockClientRepo{
				findByClientIDAndIdentityProviderFn: func(_, _ string) (*model.Client, error) {
					return nil, errors.New("db error")
				},
			},
			&mockClientURIRepo{},
			&mockOAuthAuthCodeRepo{},
			&mockOAuthConsentGrantRepo{},
			&mockOAuthConsentChallRepo{},
			&mockAuthEventService{},
		)

		_, oerr := svc.Authorize(ctx, validAuthorizeRequest(), 1)
		require.NotNil(t, oerr)
		assert.Equal(t, "server_error", oerr.Code)
	})

	t.Run("client inactive", func(t *testing.T) {
		db, _ := newMockDB(t)
		client := activeClient()
		client.Status = "inactive"

		svc := newOAuthAuthorizeSvc(db,
			&mockClientRepo{
				findByClientIDAndIdentityProviderFn: func(_, _ string) (*model.Client, error) {
					return client, nil
				},
			},
			&mockClientURIRepo{},
			&mockOAuthAuthCodeRepo{},
			&mockOAuthConsentGrantRepo{},
			&mockOAuthConsentChallRepo{},
			&mockAuthEventService{},
		)

		_, oerr := svc.Authorize(ctx, validAuthorizeRequest(), 1)
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_request", oerr.Code)
	})

	t.Run("grant type not allowed", func(t *testing.T) {
		db, _ := newMockDB(t)
		client := activeClient()
		client.GrantTypes = pq.StringArray{model.GrantTypeClientCredentials}

		svc := newOAuthAuthorizeSvc(db,
			&mockClientRepo{
				findByClientIDAndIdentityProviderFn: func(_, _ string) (*model.Client, error) {
					return client, nil
				},
			},
			&mockClientURIRepo{},
			&mockOAuthAuthCodeRepo{},
			&mockOAuthConsentGrantRepo{},
			&mockOAuthConsentChallRepo{},
			&mockAuthEventService{},
		)

		_, oerr := svc.Authorize(ctx, validAuthorizeRequest(), 1)
		require.NotNil(t, oerr)
		assert.Equal(t, "unauthorized_client", oerr.Code)
	})

	t.Run("response type not supported", func(t *testing.T) {
		db, _ := newMockDB(t)
		client := activeClient()
		client.ResponseTypes = pq.StringArray{}

		svc := newOAuthAuthorizeSvc(db,
			&mockClientRepo{
				findByClientIDAndIdentityProviderFn: func(_, _ string) (*model.Client, error) {
					return client, nil
				},
			},
			&mockClientURIRepo{},
			&mockOAuthAuthCodeRepo{},
			&mockOAuthConsentGrantRepo{},
			&mockOAuthConsentChallRepo{},
			&mockAuthEventService{},
		)

		_, oerr := svc.Authorize(ctx, validAuthorizeRequest(), 1)
		require.NotNil(t, oerr)
		assert.Equal(t, "unsupported_response_type", oerr.Code)
	})

	t.Run("redirect URI not registered", func(t *testing.T) {
		db, _ := newMockDB(t)
		client := activeClient()
		client.ClientURIs = &[]model.ClientURI{
			{URI: "https://other.com/callback", Type: model.ClientURITypeRedirect},
		}

		svc := newOAuthAuthorizeSvc(db,
			&mockClientRepo{
				findByClientIDAndIdentityProviderFn: func(_, _ string) (*model.Client, error) {
					return client, nil
				},
			},
			&mockClientURIRepo{},
			&mockOAuthAuthCodeRepo{},
			&mockOAuthConsentGrantRepo{},
			&mockOAuthConsentChallRepo{},
			&mockAuthEventService{},
		)

		_, oerr := svc.Authorize(ctx, validAuthorizeRequest(), 1)
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
				findByClientIDAndIdentityProviderFn: func(_, _ string) (*model.Client, error) {
					return client, nil
				},
			},
			&mockClientURIRepo{},
			&mockOAuthAuthCodeRepo{},
			&mockOAuthConsentGrantRepo{},
			&mockOAuthConsentChallRepo{},
			&mockAuthEventService{},
		)

		_, oerr := svc.Authorize(ctx, validAuthorizeRequest(), 1)
		require.NotNil(t, oerr)
		assert.Contains(t, oerr.Description, "no redirect URIs")
	})

	t.Run("consent check error", func(t *testing.T) {
		client := activeClientWithConsent()
		db, _ := newMockDB(t)

		svc := newOAuthAuthorizeSvc(db,
			&mockClientRepo{
				findByClientIDAndIdentityProviderFn: func(_, _ string) (*model.Client, error) {
					return client, nil
				},
			},
			&mockClientURIRepo{},
			&mockOAuthAuthCodeRepo{},
			&mockOAuthConsentGrantRepo{
				findByUserAndClientFn: func(_, _ int64) (*model.OAuthConsentGrant, error) {
					return nil, errors.New("db error")
				},
			},
			&mockOAuthConsentChallRepo{},
			&mockAuthEventService{},
		)

		_, oerr := svc.Authorize(ctx, validAuthorizeRequest(), 1)
		require.NotNil(t, oerr)
		assert.Equal(t, "server_error", oerr.Code)
	})

	t.Run("consent challenge creation error", func(t *testing.T) {
		client := activeClientWithConsent()
		db, _ := newMockDB(t)

		svc := newOAuthAuthorizeSvc(db,
			&mockClientRepo{
				findByClientIDAndIdentityProviderFn: func(_, _ string) (*model.Client, error) {
					return client, nil
				},
			},
			&mockClientURIRepo{},
			&mockOAuthAuthCodeRepo{},
			&mockOAuthConsentGrantRepo{
				findByUserAndClientFn: func(_, _ int64) (*model.OAuthConsentGrant, error) {
					return nil, nil
				},
			},
			&mockOAuthConsentChallRepo{
				createFn: func(_ *model.OAuthConsentChallenge) (*model.OAuthConsentChallenge, error) {
					return nil, errors.New("create error")
				},
			},
			&mockAuthEventService{},
		)

		_, oerr := svc.Authorize(ctx, validAuthorizeRequest(), 1)
		require.NotNil(t, oerr)
		assert.Equal(t, "server_error", oerr.Code)
	})

	t.Run("auth code creation error", func(t *testing.T) {
		client := activeClient()
		db, _ := newMockDB(t)

		svc := newOAuthAuthorizeSvc(db,
			&mockClientRepo{
				findByClientIDAndIdentityProviderFn: func(_, _ string) (*model.Client, error) {
					return client, nil
				},
			},
			&mockClientURIRepo{},
			&mockOAuthAuthCodeRepo{
				createFn: func(_ *model.OAuthAuthorizationCode) (*model.OAuthAuthorizationCode, error) {
					return nil, errors.New("create error")
				},
			},
			&mockOAuthConsentGrantRepo{},
			&mockOAuthConsentChallRepo{},
			&mockAuthEventService{},
		)

		_, oerr := svc.Authorize(ctx, validAuthorizeRequest(), 1)
		require.NotNil(t, oerr)
		assert.Equal(t, "server_error", oerr.Code)
	})

	t.Run("fallback to findClientByIdentifier", func(t *testing.T) {
		db, mock := newMockDB(t)
		// FindByClientIDAndIdentityProvider returns nil
		// Then findClientByIdentifier queries db directly — returns ErrRecordNotFound
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).WillReturnError(gorm.ErrRecordNotFound)

		svc := newOAuthAuthorizeSvc(db,
			&mockClientRepo{
				findByClientIDAndIdentityProviderFn: func(_, _ string) (*model.Client, error) {
					return nil, nil
				},
			},
			&mockClientURIRepo{},
			&mockOAuthAuthCodeRepo{},
			&mockOAuthConsentGrantRepo{},
			&mockOAuthConsentChallRepo{},
			&mockAuthEventService{},
		)

		_, oerr := svc.Authorize(ctx, validAuthorizeRequest(), 1)
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_request", oerr.Code)
	})

	t.Run("fallback to findClientByIdentifier db error", func(t *testing.T) {
		db, mock := newMockDB(t)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).WillReturnError(errors.New("connection error"))

		svc := newOAuthAuthorizeSvc(db,
			&mockClientRepo{
				findByClientIDAndIdentityProviderFn: func(_, _ string) (*model.Client, error) {
					return nil, nil
				},
			},
			&mockClientURIRepo{},
			&mockOAuthAuthCodeRepo{},
			&mockOAuthConsentGrantRepo{},
			&mockOAuthConsentChallRepo{},
			&mockAuthEventService{},
		)

		_, oerr := svc.Authorize(ctx, validAuthorizeRequest(), 1)
		require.NotNil(t, oerr)
		assert.Equal(t, "server_error", oerr.Code)
	})

	t.Run("authorize without state or nonce", func(t *testing.T) {
		client := activeClient()
		db, _ := newMockDB(t)

		svc := newOAuthAuthorizeSvc(db,
			&mockClientRepo{
				findByClientIDAndIdentityProviderFn: func(_, _ string) (*model.Client, error) {
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
		result, oerr := svc.Authorize(ctx, req, 1)
		require.Nil(t, oerr)
		assert.Contains(t, result.RedirectURI, "code=")
		assert.NotContains(t, result.RedirectURI, "state=")
	})

	t.Run("consent challenge with state and nonce", func(t *testing.T) {
		client := activeClientWithConsent()
		db, _ := newMockDB(t)

		var capturedChallenge *model.OAuthConsentChallenge
		svc := newOAuthAuthorizeSvc(db,
			&mockClientRepo{
				findByClientIDAndIdentityProviderFn: func(_, _ string) (*model.Client, error) {
					return client, nil
				},
			},
			&mockClientURIRepo{},
			&mockOAuthAuthCodeRepo{},
			&mockOAuthConsentGrantRepo{
				findByUserAndClientFn: func(_, _ int64) (*model.OAuthConsentGrant, error) {
					return nil, nil
				},
			},
			&mockOAuthConsentChallRepo{
				createFn: func(c *model.OAuthConsentChallenge) (*model.OAuthConsentChallenge, error) {
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
		_, oerr := svc.Authorize(ctx, req, 1)
		require.Nil(t, oerr)
		require.NotNil(t, capturedChallenge)
		require.NotNil(t, capturedChallenge.State)
		assert.Equal(t, "mystate", *capturedChallenge.State)
		require.NotNil(t, capturedChallenge.Nonce)
		assert.Equal(t, "mynonce", *capturedChallenge.Nonce)
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
				findChallengeByUUIDFn: func(_ uuid.UUID) (*model.OAuthConsentChallenge, error) {
					return &model.OAuthConsentChallenge{
						OAuthConsentChallengeUUID: challengeUUID,
						UserID:                    1,
						Scope:                     "openid profile",
						RedirectURI:               "https://example.com/callback",
						ExpiresAt:                 time.Now().Add(5 * time.Minute),
						Client: &model.Client{
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
				findChallengeByUUIDFn: func(_ uuid.UUID) (*model.OAuthConsentChallenge, error) {
					return &model.OAuthConsentChallenge{
						OAuthConsentChallengeUUID: challengeUUID,
						UserID:                    1,
						Scope:                     "openid",
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
				findChallengeByUUIDFn: func(_ uuid.UUID) (*model.OAuthConsentChallenge, error) {
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
				findChallengeByUUIDFn: func(_ uuid.UUID) (*model.OAuthConsentChallenge, error) {
					return &model.OAuthConsentChallenge{
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
				findChallengeByUUIDFn: func(_ uuid.UUID) (*model.OAuthConsentChallenge, error) {
					return &model.OAuthConsentChallenge{
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
				findChallengeByUUIDFn: func(_ uuid.UUID) (*model.OAuthConsentChallenge, error) {
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

	validDecision := func(approved bool) dto.OAuthConsentDecisionDTO {
		return dto.OAuthConsentDecisionDTO{
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
				findChallengeByUUIDFn: func(_ uuid.UUID) (*model.OAuthConsentChallenge, error) {
					return &model.OAuthConsentChallenge{
						OAuthConsentChallengeUUID: challengeUUID,
						ClientID:                  10,
						UserID:                    1,
						TenantID:                  100,
						RedirectURI:               "https://example.com/callback",
						Scope:                     "openid profile",
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

	t.Run("denied — returns error redirect", func(t *testing.T) {
		state := "mystate"
		db, _ := newMockDB(t)

		svc := newOAuthAuthorizeSvc(db,
			&mockClientRepo{}, &mockClientURIRepo{},
			&mockOAuthAuthCodeRepo{},
			&mockOAuthConsentGrantRepo{},
			&mockOAuthConsentChallRepo{
				findChallengeByUUIDFn: func(_ uuid.UUID) (*model.OAuthConsentChallenge, error) {
					return &model.OAuthConsentChallenge{
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
				findChallengeByUUIDFn: func(_ uuid.UUID) (*model.OAuthConsentChallenge, error) {
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
				findChallengeByUUIDFn: func(_ uuid.UUID) (*model.OAuthConsentChallenge, error) {
					return &model.OAuthConsentChallenge{
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
				findChallengeByUUIDFn: func(_ uuid.UUID) (*model.OAuthConsentChallenge, error) {
					return &model.OAuthConsentChallenge{
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
				findChallengeByUUIDFn: func(_ uuid.UUID) (*model.OAuthConsentChallenge, error) {
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
				findChallengeByUUIDFn: func(_ uuid.UUID) (*model.OAuthConsentChallenge, error) {
					return &model.OAuthConsentChallenge{
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
				findChallengeByUUIDFn: func(_ uuid.UUID) (*model.OAuthConsentChallenge, error) {
					return &model.OAuthConsentChallenge{
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
				findChallengeByUUIDFn: func(_ uuid.UUID) (*model.OAuthConsentChallenge, error) {
					return &model.OAuthConsentChallenge{
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
				upsertFn: func(_ *model.OAuthConsentGrant) (*model.OAuthConsentGrant, error) {
					return nil, errors.New("upsert error")
				},
			},
			&mockOAuthConsentChallRepo{
				findChallengeByUUIDFn: func(_ uuid.UUID) (*model.OAuthConsentChallenge, error) {
					return &model.OAuthConsentChallenge{
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
				createFn: func(_ *model.OAuthAuthorizationCode) (*model.OAuthAuthorizationCode, error) {
					return nil, errors.New("create error")
				},
			},
			&mockOAuthConsentGrantRepo{},
			&mockOAuthConsentChallRepo{
				findChallengeByUUIDFn: func(_ uuid.UUID) (*model.OAuthConsentChallenge, error) {
					return &model.OAuthConsentChallenge{
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
		u := buildAuthCodeRedirect("https://example.com/callback?foo=bar", "CODE123", "STATE")
		assert.Equal(t, "https://example.com/callback?foo=bar&code=CODE123&state=STATE", u)
	})

	t.Run("no state", func(t *testing.T) {
		u := buildAuthCodeRedirect("https://example.com/callback", "CODE123", "")
		assert.Equal(t, "https://example.com/callback?code=CODE123", u)
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
