package authn

import (
	"context"
	"errors"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/branding"
	"github.com/maintainerd/maintainerd-auth/internal/platform/config"
	"github.com/maintainerd/maintainerd-auth/internal/platform/email"
	"github.com/maintainerd/maintainerd-auth/internal/platform/jwt"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newMockGormDBRegex(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	sqlDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	return gormDB, mock
}

func validEmailTemplate() *branding.EmailTemplate {
	plain := "Click here to login: {{.MagicLinkURL}}"
	return &branding.EmailTemplate{
		Subject:   "Magic Link",
		BodyHTML:  `<a href="{{.MagicLinkURL}}">Click here to login</a>`,
		BodyPlain: &plain,
	}
}

type mockMagicLinkLoginCoordinator struct {
	challengeFn func(*User, int64) (*LoginResponseDTO, error)
	issueFn     func(string, *User, *Client) (*LoginResponseDTO, error)
	enforceFn   func(*User, int64) error
}

func (m *mockMagicLinkLoginCoordinator) EnforcePhoneVerification(_ context.Context, user *User, tenantID int64) error {
	if m.enforceFn != nil {
		return m.enforceFn(user, tenantID)
	}
	return nil
}

func (m *mockMagicLinkLoginCoordinator) MagicLinkMFAChallenge(_ context.Context, user *User, tenantID int64) (*LoginResponseDTO, error) {
	if m.challengeFn != nil {
		return m.challengeFn(user, tenantID)
	}
	return nil, nil
}

func (m *mockMagicLinkLoginCoordinator) IssueMagicLinkSession(_ context.Context, sub string, user *User, client *Client) (*LoginResponseDTO, error) {
	if m.issueFn != nil {
		return m.issueFn(sub, user, client)
	}
	return nil, nil
}

// ---------------------------------------------------------------------------
// TestSendMagicLink
// ---------------------------------------------------------------------------

func TestSendMagicLink(t *testing.T) {
	emailAddr := "test@example.com"
	clientID := strPtr("test-client")
	var providerID *string

	t.Run("success rate-limit skip happy path", func(t *testing.T) {
		initTestJWTKeysService(t)
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()

		config.AppPublicHostname = "http://localhost"
		config.AppFrontendIdentityHostname = "http://localhost"
		config.AppFrontendConsoleHostname = "http://localhost"

		origSendEmail := email.SendEmail
		email.SendEmail = func(_ context.Context, _ *gorm.DB, _ email.SendEmailParams) error { return nil }
		defer func() { email.SendEmail = origSendEmail }()

		userRepo := &mockUserRepo{
			findByEmailFn: func(_ string) (*User, error) {
				return &User{UserID: 1, UserUUID: uuid.New(), Email: emailAddr, Username: "testuser", Status: shared.StatusActive}, nil
			},
		}
		userTokenRepo := &mockUserTokenRepo{
			findByUserIDAndTokenTypeFn: func(_ int64, _ string) ([]UserToken, error) { return nil, nil },
			createFn:                   func(ut *UserToken) (*UserToken, error) { ut.UserTokenID = 1; return ut, nil },
		}
		clientRepo := &mockClientRepo{
			findByClientIDAndIdentityProviderFn: func(_, _ string) (*Client, error) {
				return &Client{
					ClientID: 1,
					Status:   shared.StatusActive,
					// Magic link is opt-in per client; these cases exercise the send flow.
					AllowMagicLink: true,
					Domain:         strPtr("https://auth.example.com"),
					Identifier:     strPtr("test-client"),
					IdentityProvider: &IdentityProvider{
						Identifier: "test-provider",
					},
				}, nil
			},
		}
		emailTemplateRepo := &mockEmailTemplateRepo{
			findByNameFn: func(_ string) (*branding.EmailTemplate, error) {
				return validEmailTemplate(), nil
			},
		}

		svc := NewMagicLinkService(gormDB, userRepo, userTokenRepo, clientRepo, &mockUserIdentityRepo{}, &mockIdentityProviderRepo{}, emailTemplateRepo)
		resp, err := svc.SendMagicLink(context.Background(), emailAddr, clientID, providerID, false)

		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.True(t, resp.Success)
		assert.NotEmpty(t, resp.Message)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("client not found", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()

		clientRepo := &mockClientRepo{
			findByClientIDAndIdentityProviderFn: func(_, _ string) (*Client, error) {
				return nil, nil
			},
		}

		svc := NewMagicLinkService(gormDB, &mockUserRepo{}, &mockUserTokenRepo{}, clientRepo, &mockUserIdentityRepo{}, &mockIdentityProviderRepo{}, &mockEmailTemplateRepo{})
		resp, err := svc.SendMagicLink(context.Background(), emailAddr, clientID, providerID, false)

		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "auth client not found")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("user not found returns an explicit error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()

		clientRepo := &mockClientRepo{
			findByClientIDAndIdentityProviderFn: func(_, _ string) (*Client, error) {
				return &Client{
					ClientID: 1,
					Status:   shared.StatusActive,
					// Magic link is opt-in per client; these cases exercise the send flow.
					AllowMagicLink: true,
					Domain:         strPtr("https://auth.example.com"),
					Identifier:     strPtr("test-client"),
					IdentityProvider: &IdentityProvider{
						Identifier: "test-provider",
					},
				}, nil
			},
		}
		userRepo := &mockUserRepo{
			findByEmailFn: func(_ string) (*User, error) { return nil, nil },
		}

		svc := NewMagicLinkService(gormDB, userRepo, &mockUserTokenRepo{}, clientRepo, &mockUserIdentityRepo{}, &mockIdentityProviderRepo{}, &mockEmailTemplateRepo{})
		resp, err := svc.SendMagicLink(context.Background(), emailAddr, clientID, providerID, false)

		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "no account found with that email address")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("user inactive returns generic success", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()

		clientRepo := &mockClientRepo{
			findByClientIDAndIdentityProviderFn: func(_, _ string) (*Client, error) {
				return &Client{
					ClientID: 1,
					Status:   shared.StatusActive,
					// Magic link is opt-in per client; these cases exercise the send flow.
					AllowMagicLink: true,
					Domain:         strPtr("https://auth.example.com"),
					Identifier:     strPtr("test-client"),
					IdentityProvider: &IdentityProvider{
						Identifier: "test-provider",
					},
				}, nil
			},
		}
		userRepo := &mockUserRepo{
			findByEmailFn: func(_ string) (*User, error) {
				return &User{UserID: 1, Email: emailAddr, Status: shared.StatusInactive}, nil
			},
		}

		svc := NewMagicLinkService(gormDB, userRepo, &mockUserTokenRepo{}, clientRepo, &mockUserIdentityRepo{}, &mockIdentityProviderRepo{}, &mockEmailTemplateRepo{})
		resp, err := svc.SendMagicLink(context.Background(), emailAddr, clientID, providerID, false)

		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.True(t, resp.Success)
		assert.NotEmpty(t, resp.Message)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// ---------------------------------------------------------------------------
// TestLoginWithMagicLink
// ---------------------------------------------------------------------------

func TestLoginWithMagicLink(t *testing.T) {
	token := "valid-magic-link-token-0000000000000000000000000000000000000000000000000000"
	tokenHash := hashUserBearerToken(token)
	clientID := "test-client"

	t.Run("identity provider not found", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()

		idpRepo := &mockIdentityProviderRepo{
			findByIdentifierFn: func(_ string) (*IdentityProvider, error) {
				return nil, nil
			},
		}

		svc := NewMagicLinkService(gormDB, &mockUserRepo{}, &mockUserTokenRepo{}, &mockClientRepo{}, &mockUserIdentityRepo{}, idpRepo, &mockEmailTemplateRepo{})
		resp, err := svc.LoginWithMagicLink(context.Background(), token, &clientID, nil)

		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "authentication failed")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("identity provider lookup error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()

		idpRepo := &mockIdentityProviderRepo{
			findByIdentifierFn: func(_ string) (*IdentityProvider, error) {
				return nil, errors.New("db error")
			},
		}

		svc := NewMagicLinkService(gormDB, &mockUserRepo{}, &mockUserTokenRepo{}, &mockClientRepo{}, &mockUserIdentityRepo{}, idpRepo, &mockEmailTemplateRepo{})
		resp, err := svc.LoginWithMagicLink(context.Background(), token, &clientID, nil)

		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "authentication failed")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("client not found", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()

		idpRepo := &mockIdentityProviderRepo{
			findByIdentifierFn: func(_ string) (*IdentityProvider, error) {
				return buildActiveIdentityProvider(), nil
			},
		}
		clientRepo := &mockClientRepo{
			findByClientIDAndIdentityProviderFn: func(_, _ string) (*Client, error) {
				return nil, nil
			},
		}

		svc := NewMagicLinkService(gormDB, &mockUserRepo{}, &mockUserTokenRepo{}, clientRepo, &mockUserIdentityRepo{}, idpRepo, &mockEmailTemplateRepo{})
		resp, err := svc.LoginWithMagicLink(context.Background(), token, &clientID, nil)

		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "authentication failed")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("client inactive", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()

		idpRepo := &mockIdentityProviderRepo{
			findByIdentifierFn: func(_ string) (*IdentityProvider, error) {
				return buildActiveIdentityProvider(), nil
			},
		}
		clientRepo := &mockClientRepo{
			findByClientIDAndIdentityProviderFn: func(_, _ string) (*Client, error) {
				c := buildActiveClient()
				c.Status = shared.StatusInactive
				return c, nil
			},
		}

		svc := NewMagicLinkService(gormDB, &mockUserRepo{}, &mockUserTokenRepo{}, clientRepo, &mockUserIdentityRepo{}, idpRepo, &mockEmailTemplateRepo{})
		resp, err := svc.LoginWithMagicLink(context.Background(), token, &clientID, nil)

		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "authentication failed")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("token not found", func(t *testing.T) {
		gormDB, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT \* FROM "user_tokens" WHERE .+`).
			WithArgs("user:magic_link", tokenHash).
			WillReturnRows(sqlmock.NewRows([]string{"user_token_id", "user_token_uuid", "user_id", "token_type", "token", "expires_at", "is_revoked", "ip_address", "user_agent", "last_used_at", "idle_timeout_seconds", "absolute_expires_at", "created_at", "updated_at"}))
		mock.ExpectRollback()

		idpRepo := &mockIdentityProviderRepo{
			findByIdentifierFn: func(_ string) (*IdentityProvider, error) {
				return buildActiveIdentityProvider(), nil
			},
		}
		clientRepo := &mockClientRepo{
			findByClientIDAndIdentityProviderFn: func(_, _ string) (*Client, error) {
				return buildActiveClient(), nil
			},
		}

		svc := NewMagicLinkService(gormDB, &mockUserRepo{}, &mockUserTokenRepo{}, clientRepo, &mockUserIdentityRepo{}, idpRepo, &mockEmailTemplateRepo{})
		resp, err := svc.LoginWithMagicLink(context.Background(), token, &clientID, nil)

		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "invalid or expired sign-in link")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("token expired", func(t *testing.T) {
		gormDB, mock := newMockGormDBRegex(t)
		pastTime := time.Now().Add(-1 * time.Hour)
		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT \* FROM "user_tokens" WHERE .+`).
			WithArgs("user:magic_link", tokenHash).
			WillReturnRows(sqlmock.NewRows([]string{"user_token_id", "user_token_uuid", "user_id", "token_type", "token", "expires_at", "is_revoked", "ip_address", "user_agent", "last_used_at", "idle_timeout_seconds", "absolute_expires_at", "created_at", "updated_at"}).
				AddRow(1, uuid.New(), 1, "user:magic_link", tokenHash, pastTime, false, nil, nil, nil, nil, nil, time.Now(), time.Now()))
		mock.ExpectRollback()

		idpRepo := &mockIdentityProviderRepo{
			findByIdentifierFn: func(_ string) (*IdentityProvider, error) {
				return buildActiveIdentityProvider(), nil
			},
		}
		clientRepo := &mockClientRepo{
			findByClientIDAndIdentityProviderFn: func(_, _ string) (*Client, error) {
				return buildActiveClient(), nil
			},
		}

		svc := NewMagicLinkService(gormDB, &mockUserRepo{}, &mockUserTokenRepo{}, clientRepo, &mockUserIdentityRepo{}, idpRepo, &mockEmailTemplateRepo{})
		resp, err := svc.LoginWithMagicLink(context.Background(), token, &clientID, nil)

		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "sign-in link has expired")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("user not found", func(t *testing.T) {
		userID := int64(1)
		gormDB, mock := newMockGormDBRegex(t)
		futureTime := time.Now().Add(1 * time.Hour)
		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT \* FROM "user_tokens" WHERE .+`).
			WithArgs("user:magic_link", tokenHash).
			WillReturnRows(sqlmock.NewRows([]string{"user_token_id", "user_token_uuid", "user_id", "token_type", "token", "expires_at", "is_revoked", "ip_address", "user_agent", "last_used_at", "idle_timeout_seconds", "absolute_expires_at", "created_at", "updated_at"}).
				AddRow(1, uuid.New(), userID, "user:magic_link", tokenHash, futureTime, false, nil, nil, nil, nil, nil, time.Now(), time.Now()))
		mock.ExpectRollback()

		idpRepo := &mockIdentityProviderRepo{
			findByIdentifierFn: func(_ string) (*IdentityProvider, error) {
				return buildActiveIdentityProvider(), nil
			},
		}
		clientRepo := &mockClientRepo{
			findByClientIDAndIdentityProviderFn: func(_, _ string) (*Client, error) {
				return buildActiveClient(), nil
			},
		}
		userRepo := &mockUserRepo{
			findByIDFn: func(_ any, _ ...string) (*User, error) {
				return nil, nil
			},
		}

		svc := NewMagicLinkService(gormDB, userRepo, &mockUserTokenRepo{}, clientRepo, &mockUserIdentityRepo{}, idpRepo, &mockEmailTemplateRepo{})
		resp, err := svc.LoginWithMagicLink(context.Background(), token, &clientID, nil)

		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "authentication failed")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("user inactive", func(t *testing.T) {
		userID := int64(1)
		gormDB, mock := newMockGormDBRegex(t)
		futureTime := time.Now().Add(1 * time.Hour)
		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT \* FROM "user_tokens" WHERE .+`).
			WithArgs("user:magic_link", tokenHash).
			WillReturnRows(sqlmock.NewRows([]string{"user_token_id", "user_token_uuid", "user_id", "token_type", "token", "expires_at", "is_revoked", "ip_address", "user_agent", "last_used_at", "idle_timeout_seconds", "absolute_expires_at", "created_at", "updated_at"}).
				AddRow(1, uuid.New(), userID, "user:magic_link", tokenHash, futureTime, false, nil, nil, nil, nil, nil, time.Now(), time.Now()))
		mock.ExpectRollback()

		idpRepo := &mockIdentityProviderRepo{
			findByIdentifierFn: func(_ string) (*IdentityProvider, error) {
				return buildActiveIdentityProvider(), nil
			},
		}
		clientRepo := &mockClientRepo{
			findByClientIDAndIdentityProviderFn: func(_, _ string) (*Client, error) {
				return buildActiveClient(), nil
			},
		}
		userRepo := &mockUserRepo{
			findByIDFn: func(_ any, _ ...string) (*User, error) {
				return &User{UserID: userID, TenantID: 1, UserUUID: uuid.New(), Email: "test@example.com", Username: "testuser", Status: shared.StatusInactive}, nil
			},
		}
		userIdentityRepo := &mockUserIdentityRepo{
			findByUserIDAndClientIDFn: func(_, _ int64) (*UserIdentity, error) {
				return &UserIdentity{Sub: "sub-123"}, nil
			},
		}

		svc := NewMagicLinkService(gormDB, userRepo, &mockUserTokenRepo{}, clientRepo, userIdentityRepo, idpRepo, &mockEmailTemplateRepo{})
		resp, err := svc.LoginWithMagicLink(context.Background(), token, &clientID, nil)

		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "account is not active")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("success", func(t *testing.T) {
		userID := int64(1)
		userUUID := uuid.New()
		emailAddr := "test@example.com"
		gormDB, mock := newMockGormDBRegex(t)
		futureTime := time.Now().Add(1 * time.Hour)
		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT \* FROM "user_tokens" WHERE .+`).
			WithArgs("user:magic_link", tokenHash).
			WillReturnRows(sqlmock.NewRows([]string{"user_token_id", "user_token_uuid", "user_id", "token_type", "token", "expires_at", "is_revoked", "ip_address", "user_agent", "last_used_at", "idle_timeout_seconds", "absolute_expires_at", "created_at", "updated_at"}).
				AddRow(1, uuid.New(), userID, "user:magic_link", tokenHash, futureTime, false, nil, nil, nil, nil, nil, time.Now(), time.Now()))
		mock.ExpectCommit()

		idpRepo := &mockIdentityProviderRepo{
			findByIdentifierFn: func(_ string) (*IdentityProvider, error) {
				return buildActiveIdentityProvider(), nil
			},
		}
		clientRepo := &mockClientRepo{
			findByClientIDAndIdentityProviderFn: func(_, _ string) (*Client, error) {
				return buildActiveClient(), nil
			},
		}
		userRepo := &mockUserRepo{
			findByIDFn: func(_ any, _ ...string) (*User, error) {
				return &User{UserID: userID, TenantID: 1, UserUUID: userUUID, Email: emailAddr, Username: "testuser", Status: shared.StatusActive}, nil
			},
			updateByIDFn: func(_ any, _ any) (*User, error) {
				return &User{UserID: userID, TenantID: 1, UserUUID: userUUID, Email: emailAddr, IsEmailVerified: true, Status: shared.StatusActive}, nil
			},
		}
		userTokenRepo := &mockUserTokenRepo{
			findByUserIDAndTokenTypeFn: func(_ int64, _ string) ([]UserToken, error) { return nil, nil },
		}
		userIdentityRepo := &mockUserIdentityRepo{
			findByUserIDAndClientIDFn: func(_, _ int64) (*UserIdentity, error) {
				return &UserIdentity{Sub: "sub-123"}, nil
			},
		}

		svc := NewMagicLinkService(gormDB, userRepo, userTokenRepo, clientRepo, userIdentityRepo, idpRepo, &mockEmailTemplateRepo{})
		svc.SetLoginCoordinator(&mockMagicLinkLoginCoordinator{
			challengeFn: func(gotUser *User, tenantID int64) (*LoginResponseDTO, error) {
				assert.Equal(t, userID, gotUser.UserID)
				return nil, nil
			},
			issueFn: func(sub string, gotUser *User, client *Client) (*LoginResponseDTO, error) {
				assert.Equal(t, "sub-123", sub)
				assert.Equal(t, userID, gotUser.UserID)
				assert.Equal(t, "test-client", *client.Identifier)
				return &LoginResponseDTO{AccessToken: "coordinated-token", TokenType: "Bearer", ExpiresIn: 3600}, nil
			},
		})
		resp, err := svc.LoginWithMagicLink(context.Background(), token, &clientID, nil)

		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, "coordinated-token", resp.AccessToken)
		assert.Equal(t, "Bearer", resp.TokenType)
		assert.Equal(t, int64(3600), resp.ExpiresIn)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// ---------------------------------------------------------------------------
// TestSendMagicLink_DefaultClient
// ---------------------------------------------------------------------------

// Passwordless email sign-in is opt-in per client. The endpoint is public, so
// hiding the button is not enough — a client that has not enabled it must be
// refused server-side.
func TestSendMagicLink_DisabledForClient(t *testing.T) {
	gormDB, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectRollback()
	clientRepo := &mockClientRepo{
		findByClientIDAndIdentityProviderFn: func(_, _ string) (*Client, error) {
			return &Client{
				ClientID:       1,
				Status:         shared.StatusActive,
				AllowMagicLink: false,
				Domain:         strPtr("https://auth.example.com"),
				Identifier:     strPtr("test-client"),
			}, nil
		},
	}

	svc := NewMagicLinkService(gormDB, &mockUserRepo{}, &mockUserTokenRepo{}, clientRepo, &mockUserIdentityRepo{}, &mockIdentityProviderRepo{}, &mockEmailTemplateRepo{})
	_, err := svc.SendMagicLink(context.Background(), "user@example.com", strPtr("test-client"), nil, false)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not enabled")
}

// Internal/system callers are not the hosted login page and keep working.
func TestSendMagicLink_DisabledStillAllowsInternalCaller(t *testing.T) {
	gormDB, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectRollback()
	clientRepo := &mockClientRepo{
		findByClientIDAndIdentityProviderFn: func(_, _ string) (*Client, error) {
			return &Client{
				ClientID:       1,
				Status:         shared.StatusActive,
				AllowMagicLink: false,
				Domain:         strPtr("https://auth.example.com"),
				Identifier:     strPtr("test-client"),
			}, nil
		},
	}
	userRepo := &mockUserRepo{
		findByEmailAndTenantIDFn: func(_ string, _ int64) (*User, error) { return nil, nil },
	}

	svc := NewMagicLinkService(gormDB, userRepo, &mockUserTokenRepo{}, clientRepo, &mockUserIdentityRepo{}, &mockIdentityProviderRepo{}, &mockEmailTemplateRepo{})
	_, err := svc.SendMagicLink(context.Background(), "user@example.com", strPtr("test-client"), nil, true)

	// Passes the gate and fails later on the missing user, not on the toggle.
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "not enabled")
}

func TestSendMagicLink_DefaultClient(t *testing.T) {
	emailAddr := "test@example.com"

	gormDB, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectCommit()

	config.AppPublicHostname = "http://localhost"
	config.AppFrontendIdentityHostname = "http://localhost"

	origSendEmail := email.SendEmail
	email.SendEmail = func(_ context.Context, _ *gorm.DB, _ email.SendEmailParams) error { return nil }
	defer func() { email.SendEmail = origSendEmail }()

	clientRepo := &mockClientRepo{
		findSystemFn: func() (*Client, error) {
			return &Client{
				ClientID: 1,
				Status:   shared.StatusActive,
				// Magic link is opt-in per client; these cases exercise the send flow.
				AllowMagicLink: true,
				Domain:         strPtr("https://auth.example.com"),
				Identifier:     strPtr("system"),
				IdentityProvider: &IdentityProvider{
					Identifier: "default",
				},
			}, nil
		},
	}
	userRepo := &mockUserRepo{
		findByEmailFn: func(_ string) (*User, error) {
			return &User{UserID: 1, UserUUID: uuid.New(), Email: emailAddr, Username: "testuser", Status: shared.StatusActive}, nil
		},
	}
	userTokenRepo := &mockUserTokenRepo{
		findByUserIDAndTokenTypeFn: func(_ int64, _ string) ([]UserToken, error) { return nil, nil },
		createFn:                   func(ut *UserToken) (*UserToken, error) { ut.UserTokenID = 1; return ut, nil },
	}
	emailTemplateRepo := &mockEmailTemplateRepo{
		findByNameFn: func(_ string) (*branding.EmailTemplate, error) {
			return validEmailTemplate(), nil
		},
	}

	svc := NewMagicLinkService(gormDB, userRepo, userTokenRepo, clientRepo, &mockUserIdentityRepo{}, &mockIdentityProviderRepo{}, emailTemplateRepo)
	resp, err := svc.SendMagicLink(context.Background(), emailAddr, nil, nil, false)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, resp.Success)
	assert.NotEmpty(t, resp.Message)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// TestSendMagicLink_ExistingTokensError
// ---------------------------------------------------------------------------

func TestSendMagicLink_ExistingTokensError(t *testing.T) {
	emailAddr := "test@example.com"
	clientID := strPtr("test-client")
	var providerID *string

	gormDB, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectRollback()

	clientRepo := &mockClientRepo{
		findByClientIDAndIdentityProviderFn: func(_, _ string) (*Client, error) {
			return &Client{
				ClientID: 1,
				Status:   shared.StatusActive,
				// Magic link is opt-in per client; these cases exercise the send flow.
				AllowMagicLink: true,
				Domain:         strPtr("https://auth.example.com"),
				Identifier:     strPtr("test-client"),
				IdentityProvider: &IdentityProvider{
					Identifier: "test-provider",
				},
			}, nil
		},
	}
	userRepo := &mockUserRepo{
		findByEmailFn: func(_ string) (*User, error) {
			return &User{UserID: 1, UserUUID: uuid.New(), Email: emailAddr, Username: "testuser", Status: shared.StatusActive}, nil
		},
	}
	userTokenRepo := &mockUserTokenRepo{
		findByUserIDAndTokenTypeFn: func(_ int64, _ string) ([]UserToken, error) {
			return nil, errors.New("db error")
		},
	}

	svc := NewMagicLinkService(gormDB, userRepo, userTokenRepo, clientRepo, &mockUserIdentityRepo{}, &mockIdentityProviderRepo{}, &mockEmailTemplateRepo{})
	resp, err := svc.SendMagicLink(context.Background(), emailAddr, clientID, providerID, false)

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "failed to find existing tokens")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// TestSendMagicLink_RevokeError
// ---------------------------------------------------------------------------

func TestSendMagicLink_RevokeError(t *testing.T) {
	emailAddr := "test@example.com"
	clientID := strPtr("test-client")
	var providerID *string

	gormDB, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectRollback()

	clientRepo := &mockClientRepo{
		findByClientIDAndIdentityProviderFn: func(_, _ string) (*Client, error) {
			return &Client{
				ClientID: 1,
				Status:   shared.StatusActive,
				// Magic link is opt-in per client; these cases exercise the send flow.
				AllowMagicLink: true,
				Domain:         strPtr("https://auth.example.com"),
				Identifier:     strPtr("test-client"),
				IdentityProvider: &IdentityProvider{
					Identifier: "test-provider",
				},
			}, nil
		},
	}
	userRepo := &mockUserRepo{
		findByEmailFn: func(_ string) (*User, error) {
			return &User{UserID: 1, UserUUID: uuid.New(), Email: emailAddr, Username: "testuser", Status: shared.StatusActive}, nil
		},
	}
	userTokenRepo := &mockUserTokenRepo{
		findByUserIDAndTokenTypeFn: func(_ int64, _ string) ([]UserToken, error) {
			return []UserToken{{UserTokenUUID: uuid.New()}}, nil
		},
		revokeByUUIDFn: func(_ uuid.UUID) error {
			return errors.New("revoke error")
		},
	}

	svc := NewMagicLinkService(gormDB, userRepo, userTokenRepo, clientRepo, &mockUserIdentityRepo{}, &mockIdentityProviderRepo{}, &mockEmailTemplateRepo{})
	resp, err := svc.SendMagicLink(context.Background(), emailAddr, clientID, providerID, false)

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "failed to revoke existing token")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// TestSendMagicLink_TokenCreateError
// ---------------------------------------------------------------------------

func TestSendMagicLink_TokenCreateError(t *testing.T) {
	emailAddr := "test@example.com"
	clientID := strPtr("test-client")
	var providerID *string

	gormDB, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectRollback()

	clientRepo := &mockClientRepo{
		findByClientIDAndIdentityProviderFn: func(_, _ string) (*Client, error) {
			return &Client{
				ClientID: 1,
				Status:   shared.StatusActive,
				// Magic link is opt-in per client; these cases exercise the send flow.
				AllowMagicLink: true,
				Domain:         strPtr("https://auth.example.com"),
				Identifier:     strPtr("test-client"),
				IdentityProvider: &IdentityProvider{
					Identifier: "test-provider",
				},
			}, nil
		},
	}
	userRepo := &mockUserRepo{
		findByEmailFn: func(_ string) (*User, error) {
			return &User{UserID: 1, UserUUID: uuid.New(), Email: emailAddr, Username: "testuser", Status: shared.StatusActive}, nil
		},
	}
	userTokenRepo := &mockUserTokenRepo{
		findByUserIDAndTokenTypeFn: func(_ int64, _ string) ([]UserToken, error) { return nil, nil },
		createFn:                   func(_ *UserToken) (*UserToken, error) { return nil, errors.New("create error") },
	}

	svc := NewMagicLinkService(gormDB, userRepo, userTokenRepo, clientRepo, &mockUserIdentityRepo{}, &mockIdentityProviderRepo{}, &mockEmailTemplateRepo{})
	resp, err := svc.SendMagicLink(context.Background(), emailAddr, clientID, providerID, false)

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "failed to create magic link token")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// TestSendMagicLink_Internal
// ---------------------------------------------------------------------------

func TestSendMagicLink_Internal(t *testing.T) {
	emailAddr := "test@example.com"
	clientID := strPtr("test-client")
	var providerID *string

	gormDB, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectCommit()

	config.AppPublicHostname = "http://localhost"
	config.AppFrontendIdentityHostname = "http://localhost"
	config.AppFrontendConsoleHostname = "http://localhost"

	origSendEmail := email.SendEmail
	email.SendEmail = func(_ context.Context, _ *gorm.DB, _ email.SendEmailParams) error { return nil }
	defer func() { email.SendEmail = origSendEmail }()

	clientRepo := &mockClientRepo{
		findByClientIDAndIdentityProviderFn: func(_, _ string) (*Client, error) {
			return &Client{
				ClientID: 1,
				Status:   shared.StatusActive,
				// Magic link is opt-in per client; these cases exercise the send flow.
				AllowMagicLink: true,
				Domain:         strPtr("https://auth.example.com"),
				Identifier:     strPtr("test-client"),
				IdentityProvider: &IdentityProvider{
					Identifier: "test-provider",
				},
			}, nil
		},
	}
	userRepo := &mockUserRepo{
		findByEmailFn: func(_ string) (*User, error) {
			return &User{UserID: 1, UserUUID: uuid.New(), Email: emailAddr, Username: "testuser", Status: shared.StatusActive}, nil
		},
	}
	userTokenRepo := &mockUserTokenRepo{
		findByUserIDAndTokenTypeFn: func(_ int64, _ string) ([]UserToken, error) { return nil, nil },
		createFn:                   func(ut *UserToken) (*UserToken, error) { ut.UserTokenID = 1; return ut, nil },
	}
	emailTemplateRepo := &mockEmailTemplateRepo{
		findByNameFn: func(_ string) (*branding.EmailTemplate, error) {
			return validEmailTemplate(), nil
		},
	}

	svc := NewMagicLinkService(gormDB, userRepo, userTokenRepo, clientRepo, &mockUserIdentityRepo{}, &mockIdentityProviderRepo{}, emailTemplateRepo)
	resp, err := svc.SendMagicLink(context.Background(), emailAddr, clientID, providerID, true)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, resp.Success)
	assert.NotEmpty(t, resp.Message)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// TestSendMagicLink_TemplateFindByNameError
// ---------------------------------------------------------------------------

func TestSendMagicLink_TemplateFindByNameError(t *testing.T) {
	emailAddr := "test@example.com"
	clientID := strPtr("test-client")
	var providerID *string

	gormDB, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectCommit()

	origSendEmail := email.SendEmail
	email.SendEmail = func(_ context.Context, _ *gorm.DB, _ email.SendEmailParams) error { return nil }
	defer func() { email.SendEmail = origSendEmail }()

	clientRepo := &mockClientRepo{
		findByClientIDAndIdentityProviderFn: func(_, _ string) (*Client, error) {
			return &Client{
				ClientID: 1,
				Status:   shared.StatusActive,
				// Magic link is opt-in per client; these cases exercise the send flow.
				AllowMagicLink: true,
				Domain:         strPtr("https://auth.example.com"),
				Identifier:     strPtr("test-client"),
				IdentityProvider: &IdentityProvider{
					Identifier: "test-provider",
				},
			}, nil
		},
	}
	userRepo := &mockUserRepo{
		findByEmailFn: func(_ string) (*User, error) {
			return &User{UserID: 1, UserUUID: uuid.New(), Email: emailAddr, Username: "testuser", Status: shared.StatusActive}, nil
		},
	}
	userTokenRepo := &mockUserTokenRepo{
		findByUserIDAndTokenTypeFn: func(_ int64, _ string) ([]UserToken, error) { return nil, nil },
		createFn:                   func(ut *UserToken) (*UserToken, error) { ut.UserTokenID = 1; return ut, nil },
	}
	emailTemplateRepo := &mockEmailTemplateRepo{
		findByNameFn: func(_ string) (*branding.EmailTemplate, error) {
			return nil, errors.New("template not found")
		},
	}

	svc := NewMagicLinkService(gormDB, userRepo, userTokenRepo, clientRepo, &mockUserIdentityRepo{}, &mockIdentityProviderRepo{}, emailTemplateRepo)
	resp, err := svc.SendMagicLink(context.Background(), emailAddr, clientID, providerID, false)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, resp.Success)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// TestSendMagicLink_TemplateParseError
// ---------------------------------------------------------------------------

func TestSendMagicLink_TemplateParseError(t *testing.T) {
	emailAddr := "test@example.com"
	clientID := strPtr("test-client")
	var providerID *string

	gormDB, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectCommit()

	origSendEmail := email.SendEmail
	email.SendEmail = func(_ context.Context, _ *gorm.DB, _ email.SendEmailParams) error { return nil }
	defer func() { email.SendEmail = origSendEmail }()

	clientRepo := &mockClientRepo{
		findByClientIDAndIdentityProviderFn: func(_, _ string) (*Client, error) {
			return &Client{
				ClientID: 1,
				Status:   shared.StatusActive,
				// Magic link is opt-in per client; these cases exercise the send flow.
				AllowMagicLink: true,
				Domain:         strPtr("https://auth.example.com"),
				Identifier:     strPtr("test-client"),
				IdentityProvider: &IdentityProvider{
					Identifier: "test-provider",
				},
			}, nil
		},
	}
	userRepo := &mockUserRepo{
		findByEmailFn: func(_ string) (*User, error) {
			return &User{UserID: 1, UserUUID: uuid.New(), Email: emailAddr, Username: "testuser", Status: shared.StatusActive}, nil
		},
	}
	userTokenRepo := &mockUserTokenRepo{
		findByUserIDAndTokenTypeFn: func(_ int64, _ string) ([]UserToken, error) { return nil, nil },
		createFn:                   func(ut *UserToken) (*UserToken, error) { ut.UserTokenID = 1; return ut, nil },
	}
	emailTemplateRepo := &mockEmailTemplateRepo{
		findByNameFn: func(_ string) (*branding.EmailTemplate, error) {
			return &branding.EmailTemplate{
				Subject:  "Magic Link",
				BodyHTML: `{{if}}`,
			}, nil
		},
	}

	svc := NewMagicLinkService(gormDB, userRepo, userTokenRepo, clientRepo, &mockUserIdentityRepo{}, &mockIdentityProviderRepo{}, emailTemplateRepo)
	resp, err := svc.SendMagicLink(context.Background(), emailAddr, clientID, providerID, false)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, resp.Success)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// TestSendMagicLink_TemplateExecuteError
// ---------------------------------------------------------------------------

func TestSendMagicLink_TemplateExecuteError(t *testing.T) {
	emailAddr := "test@example.com"
	clientID := strPtr("test-client")
	var providerID *string

	gormDB, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectCommit()

	origSendEmail := email.SendEmail
	email.SendEmail = func(_ context.Context, _ *gorm.DB, _ email.SendEmailParams) error { return nil }
	defer func() { email.SendEmail = origSendEmail }()

	clientRepo := &mockClientRepo{
		findByClientIDAndIdentityProviderFn: func(_, _ string) (*Client, error) {
			return &Client{
				ClientID: 1,
				Status:   shared.StatusActive,
				// Magic link is opt-in per client; these cases exercise the send flow.
				AllowMagicLink: true,
				Domain:         strPtr("https://auth.example.com"),
				Identifier:     strPtr("test-client"),
				IdentityProvider: &IdentityProvider{
					Identifier: "test-provider",
				},
			}, nil
		},
	}
	userRepo := &mockUserRepo{
		findByEmailFn: func(_ string) (*User, error) {
			return &User{UserID: 1, UserUUID: uuid.New(), Email: emailAddr, Username: "testuser", Status: shared.StatusActive}, nil
		},
	}
	userTokenRepo := &mockUserTokenRepo{
		findByUserIDAndTokenTypeFn: func(_ int64, _ string) ([]UserToken, error) { return nil, nil },
		createFn:                   func(ut *UserToken) (*UserToken, error) { ut.UserTokenID = 1; return ut, nil },
	}
	emailTemplateRepo := &mockEmailTemplateRepo{
		findByNameFn: func(_ string) (*branding.EmailTemplate, error) {
			return &branding.EmailTemplate{
				Subject:  "Magic Link",
				BodyHTML: `{{index . 0}}`,
			}, nil
		},
	}

	svc := NewMagicLinkService(gormDB, userRepo, userTokenRepo, clientRepo, &mockUserIdentityRepo{}, &mockIdentityProviderRepo{}, emailTemplateRepo)
	resp, err := svc.SendMagicLink(context.Background(), emailAddr, clientID, providerID, false)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, resp.Success)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// TestSendMagicLink_PlaintextParseError
// ---------------------------------------------------------------------------

func TestSendMagicLink_PlaintextParseError(t *testing.T) {
	emailAddr := "test@example.com"
	clientID := strPtr("test-client")
	var providerID *string

	gormDB, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectCommit()

	origSendEmail := email.SendEmail
	email.SendEmail = func(_ context.Context, _ *gorm.DB, _ email.SendEmailParams) error { return nil }
	defer func() { email.SendEmail = origSendEmail }()

	clientRepo := &mockClientRepo{
		findByClientIDAndIdentityProviderFn: func(_, _ string) (*Client, error) {
			return &Client{
				ClientID: 1,
				Status:   shared.StatusActive,
				// Magic link is opt-in per client; these cases exercise the send flow.
				AllowMagicLink: true,
				Domain:         strPtr("https://auth.example.com"),
				Identifier:     strPtr("test-client"),
				IdentityProvider: &IdentityProvider{
					Identifier: "test-provider",
				},
			}, nil
		},
	}
	userRepo := &mockUserRepo{
		findByEmailFn: func(_ string) (*User, error) {
			return &User{UserID: 1, UserUUID: uuid.New(), Email: emailAddr, Username: "testuser", Status: shared.StatusActive}, nil
		},
	}
	userTokenRepo := &mockUserTokenRepo{
		findByUserIDAndTokenTypeFn: func(_ int64, _ string) ([]UserToken, error) { return nil, nil },
		createFn:                   func(ut *UserToken) (*UserToken, error) { ut.UserTokenID = 1; return ut, nil },
	}
	invalidPlain := "{{if}}"
	emailTemplateRepo := &mockEmailTemplateRepo{
		findByNameFn: func(_ string) (*branding.EmailTemplate, error) {
			return &branding.EmailTemplate{
				Subject:   "Magic Link",
				BodyHTML:  `<a href="{{.MagicLinkURL}}">Click here to login</a>`,
				BodyPlain: &invalidPlain,
			}, nil
		},
	}

	svc := NewMagicLinkService(gormDB, userRepo, userTokenRepo, clientRepo, &mockUserIdentityRepo{}, &mockIdentityProviderRepo{}, emailTemplateRepo)
	resp, err := svc.SendMagicLink(context.Background(), emailAddr, clientID, providerID, false)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, resp.Success)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// TestSendMagicLink_PlaintextExecuteError
// ---------------------------------------------------------------------------

func TestSendMagicLink_PlaintextExecuteError(t *testing.T) {
	emailAddr := "test@example.com"
	clientID := strPtr("test-client")
	var providerID *string

	gormDB, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectCommit()

	origSendEmail := email.SendEmail
	email.SendEmail = func(_ context.Context, _ *gorm.DB, _ email.SendEmailParams) error { return nil }
	defer func() { email.SendEmail = origSendEmail }()

	clientRepo := &mockClientRepo{
		findByClientIDAndIdentityProviderFn: func(_, _ string) (*Client, error) {
			return &Client{
				ClientID: 1,
				Status:   shared.StatusActive,
				// Magic link is opt-in per client; these cases exercise the send flow.
				AllowMagicLink: true,
				Domain:         strPtr("https://auth.example.com"),
				Identifier:     strPtr("test-client"),
				IdentityProvider: &IdentityProvider{
					Identifier: "test-provider",
				},
			}, nil
		},
	}
	userRepo := &mockUserRepo{
		findByEmailFn: func(_ string) (*User, error) {
			return &User{UserID: 1, UserUUID: uuid.New(), Email: emailAddr, Username: "testuser", Status: shared.StatusActive}, nil
		},
	}
	userTokenRepo := &mockUserTokenRepo{
		findByUserIDAndTokenTypeFn: func(_ int64, _ string) ([]UserToken, error) { return nil, nil },
		createFn:                   func(ut *UserToken) (*UserToken, error) { ut.UserTokenID = 1; return ut, nil },
	}
	invalidPlain := "{{index . 0}}"
	emailTemplateRepo := &mockEmailTemplateRepo{
		findByNameFn: func(_ string) (*branding.EmailTemplate, error) {
			return &branding.EmailTemplate{
				Subject:   "Magic Link",
				BodyHTML:  `<a href="{{.MagicLinkURL}}">Click here to login</a>`,
				BodyPlain: &invalidPlain,
			}, nil
		},
	}

	svc := NewMagicLinkService(gormDB, userRepo, userTokenRepo, clientRepo, &mockUserIdentityRepo{}, &mockIdentityProviderRepo{}, emailTemplateRepo)
	resp, err := svc.SendMagicLink(context.Background(), emailAddr, clientID, providerID, false)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, resp.Success)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// TestLoginWithMagicLink_TokenQueryError
// ---------------------------------------------------------------------------

func TestLoginWithMagicLink_TokenQueryError(t *testing.T) {
	token := "valid-magic-link-token-0000000000000000000000000000000000000000000000000000"
	tokenHash := hashUserBearerToken(token)
	clientID := "test-client"

	gormDB, mock := newMockGormDBRegex(t)
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT \* FROM "user_tokens" WHERE .+`).
		WithArgs("user:magic_link", tokenHash).
		WillReturnError(errors.New("db error"))
	mock.ExpectRollback()

	idpRepo := &mockIdentityProviderRepo{
		findByIdentifierFn: func(_ string) (*IdentityProvider, error) {
			return buildActiveIdentityProvider(), nil
		},
	}
	clientRepo := &mockClientRepo{
		findByClientIDAndIdentityProviderFn: func(_, _ string) (*Client, error) {
			return buildActiveClient(), nil
		},
	}

	svc := NewMagicLinkService(gormDB, &mockUserRepo{}, &mockUserTokenRepo{}, clientRepo, &mockUserIdentityRepo{}, idpRepo, &mockEmailTemplateRepo{})
	resp, err := svc.LoginWithMagicLink(context.Background(), token, &clientID, nil)

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "failed to find magic link token")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// TestLoginWithMagicLink_UserIdentityNotFound
// ---------------------------------------------------------------------------

func TestLoginWithMagicLink_UserIdentityNotFound(t *testing.T) {
	token := "valid-magic-link-token-0000000000000000000000000000000000000000000000000000"
	tokenHash := hashUserBearerToken(token)
	userID := int64(1)
	clientID := "test-client"

	gormDB, mock := newMockGormDBRegex(t)
	futureTime := time.Now().Add(1 * time.Hour)
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT \* FROM "user_tokens" WHERE .+`).
		WithArgs("user:magic_link", tokenHash).
		WillReturnRows(sqlmock.NewRows([]string{"user_token_id", "user_token_uuid", "user_id", "token_type", "token", "expires_at", "is_revoked", "ip_address", "user_agent", "last_used_at", "idle_timeout_seconds", "absolute_expires_at", "created_at", "updated_at"}).
			AddRow(1, uuid.New(), userID, "user:magic_link", tokenHash, futureTime, false, nil, nil, nil, nil, nil, time.Now(), time.Now()))
	mock.ExpectRollback()

	idpRepo := &mockIdentityProviderRepo{
		findByIdentifierFn: func(_ string) (*IdentityProvider, error) {
			return buildActiveIdentityProvider(), nil
		},
	}
	clientRepo := &mockClientRepo{
		findByClientIDAndIdentityProviderFn: func(_, _ string) (*Client, error) {
			return buildActiveClient(), nil
		},
	}
	userRepo := &mockUserRepo{
		findByIDFn: func(_ any, _ ...string) (*User, error) {
			return &User{UserID: userID, TenantID: 1, UserUUID: uuid.New(), Email: "test@example.com", Username: "testuser", Status: shared.StatusActive}, nil
		},
	}
	userIdentityRepo := &mockUserIdentityRepo{
		findByUserIDAndClientIDFn: func(_, _ int64) (*UserIdentity, error) {
			return nil, nil
		},
	}

	svc := NewMagicLinkService(gormDB, userRepo, &mockUserTokenRepo{}, clientRepo, userIdentityRepo, idpRepo, &mockEmailTemplateRepo{})
	resp, err := svc.LoginWithMagicLink(context.Background(), token, &clientID, nil)

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "authentication failed")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// TestLoginWithMagicLink_ExistingTokensError
// ---------------------------------------------------------------------------

func TestLoginWithMagicLink_ExistingTokensError(t *testing.T) {
	token := "valid-magic-link-token-0000000000000000000000000000000000000000000000000000"
	tokenHash := hashUserBearerToken(token)
	userID := int64(1)
	clientID := "test-client"

	gormDB, mock := newMockGormDBRegex(t)
	futureTime := time.Now().Add(1 * time.Hour)
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT \* FROM "user_tokens" WHERE .+`).
		WithArgs("user:magic_link", tokenHash).
		WillReturnRows(sqlmock.NewRows([]string{"user_token_id", "user_token_uuid", "user_id", "token_type", "token", "expires_at", "is_revoked", "ip_address", "user_agent", "last_used_at", "idle_timeout_seconds", "absolute_expires_at", "created_at", "updated_at"}).
			AddRow(1, uuid.New(), userID, "user:magic_link", tokenHash, futureTime, false, nil, nil, nil, nil, nil, time.Now(), time.Now()))
	mock.ExpectRollback()

	idpRepo := &mockIdentityProviderRepo{
		findByIdentifierFn: func(_ string) (*IdentityProvider, error) {
			return buildActiveIdentityProvider(), nil
		},
	}
	clientRepo := &mockClientRepo{
		findByClientIDAndIdentityProviderFn: func(_, _ string) (*Client, error) {
			return buildActiveClient(), nil
		},
	}
	userRepo := &mockUserRepo{
		findByIDFn: func(_ any, _ ...string) (*User, error) {
			return &User{UserID: userID, TenantID: 1, UserUUID: uuid.New(), Email: "test@example.com", Username: "testuser", Status: shared.StatusActive}, nil
		},
	}
	userIdentityRepo := &mockUserIdentityRepo{
		findByUserIDAndClientIDFn: func(_, _ int64) (*UserIdentity, error) {
			return &UserIdentity{Sub: "sub-123"}, nil
		},
	}
	userTokenRepo := &mockUserTokenRepo{
		findByUserIDAndTokenTypeFn: func(_ int64, _ string) ([]UserToken, error) {
			return nil, errors.New("db error")
		},
	}

	svc := NewMagicLinkService(gormDB, userRepo, userTokenRepo, clientRepo, userIdentityRepo, idpRepo, &mockEmailTemplateRepo{})
	resp, err := svc.LoginWithMagicLink(context.Background(), token, &clientID, nil)

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "failed to find existing tokens")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// TestLoginWithMagicLink_RevokeByUUIDError
// ---------------------------------------------------------------------------

func TestLoginWithMagicLink_RevokeByUUIDError(t *testing.T) {
	token := "valid-magic-link-token-0000000000000000000000000000000000000000000000000000"
	tokenHash := hashUserBearerToken(token)
	userID := int64(1)
	clientID := "test-client"

	gormDB, mock := newMockGormDBRegex(t)
	futureTime := time.Now().Add(1 * time.Hour)
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT \* FROM "user_tokens" WHERE .+`).
		WithArgs("user:magic_link", tokenHash).
		WillReturnRows(sqlmock.NewRows([]string{"user_token_id", "user_token_uuid", "user_id", "token_type", "token", "expires_at", "is_revoked", "ip_address", "user_agent", "last_used_at", "idle_timeout_seconds", "absolute_expires_at", "created_at", "updated_at"}).
			AddRow(1, uuid.New(), userID, "user:magic_link", tokenHash, futureTime, false, nil, nil, nil, nil, nil, time.Now(), time.Now()))
	mock.ExpectRollback()

	idpRepo := &mockIdentityProviderRepo{
		findByIdentifierFn: func(_ string) (*IdentityProvider, error) {
			return buildActiveIdentityProvider(), nil
		},
	}
	clientRepo := &mockClientRepo{
		findByClientIDAndIdentityProviderFn: func(_, _ string) (*Client, error) {
			return buildActiveClient(), nil
		},
	}
	userRepo := &mockUserRepo{
		findByIDFn: func(_ any, _ ...string) (*User, error) {
			return &User{UserID: userID, TenantID: 1, UserUUID: uuid.New(), Email: "test@example.com", Username: "testuser", Status: shared.StatusActive}, nil
		},
	}
	userIdentityRepo := &mockUserIdentityRepo{
		findByUserIDAndClientIDFn: func(_, _ int64) (*UserIdentity, error) {
			return &UserIdentity{Sub: "sub-123"}, nil
		},
	}
	userTokenRepo := &mockUserTokenRepo{
		findByUserIDAndTokenTypeFn: func(_ int64, _ string) ([]UserToken, error) {
			return []UserToken{{UserTokenUUID: uuid.New()}}, nil
		},
		revokeByUUIDFn: func(_ uuid.UUID) error {
			return errors.New("revoke error")
		},
	}

	svc := NewMagicLinkService(gormDB, userRepo, userTokenRepo, clientRepo, userIdentityRepo, idpRepo, &mockEmailTemplateRepo{})
	resp, err := svc.LoginWithMagicLink(context.Background(), token, &clientID, nil)

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "failed to revoke magic link token")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// TestLoginWithMagicLink_UpdateByIDError
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// TestLoginWithMagicLink_GenerateTokenResponseError
// ---------------------------------------------------------------------------

func TestLoginWithMagicLink_GenerateTokenResponseError(t *testing.T) {
	token := "valid-magic-link-token-0000000000000000000000000000000000000000000000000000"
	tokenHash := hashUserBearerToken(token)
	userID := int64(1)
	clientID := "test-client"

	jwt.ResetJWTKeys()
	defer initTestJWTKeysService(t)

	gormDB, mock := newMockGormDBRegex(t)
	futureTime := time.Now().Add(1 * time.Hour)
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT \* FROM "user_tokens" WHERE .+`).
		WithArgs("user:magic_link", tokenHash).
		WillReturnRows(sqlmock.NewRows([]string{"user_token_id", "user_token_uuid", "user_id", "token_type", "token", "expires_at", "is_revoked", "ip_address", "user_agent", "last_used_at", "idle_timeout_seconds", "absolute_expires_at", "created_at", "updated_at"}).
			AddRow(1, uuid.New(), userID, "user:magic_link", tokenHash, futureTime, false, nil, nil, nil, nil, nil, time.Now(), time.Now()))
	mock.ExpectCommit()

	idpRepo := &mockIdentityProviderRepo{
		findByIdentifierFn: func(_ string) (*IdentityProvider, error) {
			return buildActiveIdentityProvider(), nil
		},
	}
	clientRepo := &mockClientRepo{
		findByClientIDAndIdentityProviderFn: func(_, _ string) (*Client, error) {
			return buildActiveClient(), nil
		},
	}
	userRepo := &mockUserRepo{
		findByIDFn: func(_ any, _ ...string) (*User, error) {
			return &User{UserID: userID, TenantID: 1, UserUUID: uuid.New(), Email: "test@example.com", Username: "testuser", Status: shared.StatusActive, IsEmailVerified: true}, nil
		},
	}
	userIdentityRepo := &mockUserIdentityRepo{
		findByUserIDAndClientIDFn: func(_, _ int64) (*UserIdentity, error) {
			return &UserIdentity{Sub: "sub-123"}, nil
		},
	}
	userTokenRepo := &mockUserTokenRepo{
		findByUserIDAndTokenTypeFn: func(_ int64, _ string) ([]UserToken, error) { return nil, nil },
	}

	svc := NewMagicLinkService(gormDB, userRepo, userTokenRepo, clientRepo, userIdentityRepo, idpRepo, &mockEmailTemplateRepo{})
	resp, err := svc.LoginWithMagicLink(context.Background(), token, &clientID, nil)

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// TestSendMagicLink_SignedURLError
// ---------------------------------------------------------------------------

func TestSendMagicLink_SignedURLError(t *testing.T) {
	emailAddr := "test@example.com"
	clientID := strPtr("test-client")
	var providerID *string

	initTestJWTKeysService(t)
	gormDB, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectCommit()

	origHostname := config.AppPublicHostname
	config.AppPublicHostname = ""
	defer func() { config.AppPublicHostname = origHostname }()

	origSendEmail := email.SendEmail
	email.SendEmail = func(_ context.Context, _ *gorm.DB, _ email.SendEmailParams) error { return nil }
	defer func() { email.SendEmail = origSendEmail }()

	clientRepo := &mockClientRepo{
		findByClientIDAndIdentityProviderFn: func(_, _ string) (*Client, error) {
			return &Client{
				ClientID: 1,
				Status:   shared.StatusActive,
				// Magic link is opt-in per client; these cases exercise the send flow.
				AllowMagicLink: true,
				Domain:         strPtr("https://auth.example.com"),
				Identifier:     strPtr("test-client"),
				IdentityProvider: &IdentityProvider{
					Identifier: "test-provider",
				},
			}, nil
		},
	}
	userRepo := &mockUserRepo{
		findByEmailFn: func(_ string) (*User, error) {
			return &User{UserID: 1, UserUUID: uuid.New(), Email: emailAddr, Username: "testuser", Status: shared.StatusActive}, nil
		},
	}
	userTokenRepo := &mockUserTokenRepo{
		findByUserIDAndTokenTypeFn: func(_ int64, _ string) ([]UserToken, error) { return nil, nil },
		createFn:                   func(ut *UserToken) (*UserToken, error) { ut.UserTokenID = 1; return ut, nil },
	}
	emailTemplateRepo := &mockEmailTemplateRepo{
		findByNameFn: func(_ string) (*branding.EmailTemplate, error) {
			return validEmailTemplate(), nil
		},
	}

	svc := NewMagicLinkService(gormDB, userRepo, userTokenRepo, clientRepo, &mockUserIdentityRepo{}, &mockIdentityProviderRepo{}, emailTemplateRepo)
	resp, err := svc.SendMagicLink(context.Background(), emailAddr, clientID, providerID, false)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, resp.Success)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// TestSendMagicLink_ConvertToFrontendURLError
// ---------------------------------------------------------------------------

func TestSendMagicLink_ConvertToFrontendURLError(t *testing.T) {
	emailAddr := "test@example.com"
	clientID := strPtr("test-client")
	var providerID *string

	initTestJWTKeysService(t)
	gormDB, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectCommit()

	config.AppPublicHostname = "http://localhost"
	origAccount := config.AppFrontendIdentityHostname
	config.AppFrontendIdentityHostname = "://invalid-url"
	defer func() { config.AppFrontendIdentityHostname = origAccount }()

	origSendEmail := email.SendEmail
	email.SendEmail = func(_ context.Context, _ *gorm.DB, _ email.SendEmailParams) error { return nil }
	defer func() { email.SendEmail = origSendEmail }()

	clientRepo := &mockClientRepo{
		findByClientIDAndIdentityProviderFn: func(_, _ string) (*Client, error) {
			return &Client{
				ClientID: 1,
				Status:   shared.StatusActive,
				// Magic link is opt-in per client; these cases exercise the send flow.
				AllowMagicLink: true,
				Domain:         strPtr("https://auth.example.com"),
				Identifier:     strPtr("test-client"),
				IdentityProvider: &IdentityProvider{
					Identifier: "test-provider",
				},
			}, nil
		},
	}
	userRepo := &mockUserRepo{
		findByEmailFn: func(_ string) (*User, error) {
			return &User{UserID: 1, UserUUID: uuid.New(), Email: emailAddr, Username: "testuser", Status: shared.StatusActive}, nil
		},
	}
	userTokenRepo := &mockUserTokenRepo{
		findByUserIDAndTokenTypeFn: func(_ int64, _ string) ([]UserToken, error) { return nil, nil },
		createFn:                   func(ut *UserToken) (*UserToken, error) { ut.UserTokenID = 1; return ut, nil },
	}
	emailTemplateRepo := &mockEmailTemplateRepo{
		findByNameFn: func(_ string) (*branding.EmailTemplate, error) {
			return validEmailTemplate(), nil
		},
	}

	svc := NewMagicLinkService(gormDB, userRepo, userTokenRepo, clientRepo, &mockUserIdentityRepo{}, &mockIdentityProviderRepo{}, emailTemplateRepo)
	resp, err := svc.SendMagicLink(context.Background(), emailAddr, clientID, providerID, false)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, resp.Success)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestLoginWithMagicLink_UpdateByIDError(t *testing.T) {
	token := "valid-magic-link-token-0000000000000000000000000000000000000000000000000000"
	tokenHash := hashUserBearerToken(token)
	userID := int64(1)
	clientID := "test-client"

	gormDB, mock := newMockGormDBRegex(t)
	futureTime := time.Now().Add(1 * time.Hour)
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT \* FROM "user_tokens" WHERE .+`).
		WithArgs("user:magic_link", tokenHash).
		WillReturnRows(sqlmock.NewRows([]string{"user_token_id", "user_token_uuid", "user_id", "token_type", "token", "expires_at", "is_revoked", "ip_address", "user_agent", "last_used_at", "idle_timeout_seconds", "absolute_expires_at", "created_at", "updated_at"}).
			AddRow(1, uuid.New(), userID, "user:magic_link", tokenHash, futureTime, false, nil, nil, nil, nil, nil, time.Now(), time.Now()))
	mock.ExpectRollback()

	idpRepo := &mockIdentityProviderRepo{
		findByIdentifierFn: func(_ string) (*IdentityProvider, error) {
			return buildActiveIdentityProvider(), nil
		},
	}
	clientRepo := &mockClientRepo{
		findByClientIDAndIdentityProviderFn: func(_, _ string) (*Client, error) {
			return buildActiveClient(), nil
		},
	}
	userRepo := &mockUserRepo{
		findByIDFn: func(_ any, _ ...string) (*User, error) {
			return &User{UserID: userID, TenantID: 1, UserUUID: uuid.New(), Email: "test@example.com", Username: "testuser", Status: shared.StatusActive, IsEmailVerified: false}, nil
		},
		updateByIDFn: func(_ any, _ any) (*User, error) {
			return nil, errors.New("update error")
		},
	}
	userIdentityRepo := &mockUserIdentityRepo{
		findByUserIDAndClientIDFn: func(_, _ int64) (*UserIdentity, error) {
			return &UserIdentity{Sub: "sub-123"}, nil
		},
	}
	userTokenRepo := &mockUserTokenRepo{
		findByUserIDAndTokenTypeFn: func(_ int64, _ string) ([]UserToken, error) { return nil, nil },
	}

	svc := NewMagicLinkService(gormDB, userRepo, userTokenRepo, clientRepo, userIdentityRepo, idpRepo, &mockEmailTemplateRepo{})
	resp, err := svc.LoginWithMagicLink(context.Background(), token, &clientID, nil)

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "failed to update user verification status")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSendMagicLink_FindByEmailError(t *testing.T) {
	emailAddr := "test@example.com"
	clientID := strPtr("test-client")
	var providerID *string

	gormDB, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectRollback()

	clientRepo := &mockClientRepo{
		findByClientIDAndIdentityProviderFn: func(_, _ string) (*Client, error) {
			return &Client{
				ClientID: 1,
				Status:   shared.StatusActive,
				// Magic link is opt-in per client; these cases exercise the send flow.
				AllowMagicLink: true,
				Domain:         strPtr("https://auth.example.com"),
				Identifier:     strPtr("test-client"),
				IdentityProvider: &IdentityProvider{
					Identifier: "test-provider",
				},
			}, nil
		},
	}
	userRepo := &mockUserRepo{
		findByEmailFn: func(_ string) (*User, error) {
			return nil, errors.New("db error")
		},
	}

	svc := NewMagicLinkService(gormDB, userRepo, &mockUserTokenRepo{}, clientRepo, &mockUserIdentityRepo{}, &mockIdentityProviderRepo{}, &mockEmailTemplateRepo{})
	resp, err := svc.SendMagicLink(context.Background(), emailAddr, clientID, providerID, false)

	require.Error(t, err)
	require.Nil(t, resp)
	assert.NoError(t, mock.ExpectationsWereMet())
}
