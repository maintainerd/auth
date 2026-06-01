package authn

import (
	"context"
	"errors"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/branding"
	"github.com/maintainerd/auth/internal/platform/config"
	"github.com/maintainerd/auth/internal/platform/crypto"
	"github.com/maintainerd/auth/internal/platform/email"
	"github.com/maintainerd/auth/internal/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// TestSendVerificationEmail
// ---------------------------------------------------------------------------

func TestSendVerificationEmail(t *testing.T) {
	emailAddr := "test@example.com"
	clientID := strPtr("test-client")
	providerID := strPtr("test-provider")

	t.Run("user not found returns generic success", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()

		clientRepo := &mockClientRepo{
			findByClientIDAndIdentityProviderFn: func(_, _ string) (*Client, error) {
				return &Client{
					ClientID:   1,
					Status:     shared.StatusActive,
					Domain:     strPtr("https://auth.example.com"),
					Identifier: strPtr("test-client"),
					IdentityProvider: &IdentityProvider{
						Identifier: "test-provider",
					},
				}, nil
			},
		}
		userRepo := &mockUserRepo{
			findByEmailFn: func(_ string) (*User, error) { return nil, nil },
		}

		svc := NewEmailVerificationService(gormDB, userRepo, &mockUserTokenRepo{}, clientRepo, &mockEmailTemplateRepo{})
		resp, err := svc.SendVerificationEmail(context.Background(), emailAddr, clientID, providerID)

		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.True(t, resp.Success)
		assert.NotEmpty(t, resp.Message)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("user inactive returns generic success", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()

		clientRepo := &mockClientRepo{
			findByClientIDAndIdentityProviderFn: func(_, _ string) (*Client, error) {
				return &Client{
					ClientID:   1,
					Status:     shared.StatusActive,
					Domain:     strPtr("https://auth.example.com"),
					Identifier: strPtr("test-client"),
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

		svc := NewEmailVerificationService(gormDB, userRepo, &mockUserTokenRepo{}, clientRepo, &mockEmailTemplateRepo{})
		resp, err := svc.SendVerificationEmail(context.Background(), emailAddr, clientID, providerID)

		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.True(t, resp.Success)
		assert.NotEmpty(t, resp.Message)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("already verified returns generic success", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()

		clientRepo := &mockClientRepo{
			findByClientIDAndIdentityProviderFn: func(_, _ string) (*Client, error) {
				return &Client{
					ClientID:   1,
					Status:     shared.StatusActive,
					Domain:     strPtr("https://auth.example.com"),
					Identifier: strPtr("test-client"),
					IdentityProvider: &IdentityProvider{
						Identifier: "test-provider",
					},
				}, nil
			},
		}
		userRepo := &mockUserRepo{
			findByEmailFn: func(_ string) (*User, error) {
				return &User{UserID: 1, Email: emailAddr, Status: shared.StatusActive, IsEmailVerified: true}, nil
			},
		}

		svc := NewEmailVerificationService(gormDB, userRepo, &mockUserTokenRepo{}, clientRepo, &mockEmailTemplateRepo{})
		resp, err := svc.SendVerificationEmail(context.Background(), emailAddr, clientID, providerID)

		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.True(t, resp.Success)
		assert.NotEmpty(t, resp.Message)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("success OTP created", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()

		config.AppPublicHostname = "http://localhost"

		origSendEmail := email.SendEmail
		email.SendEmail = func(_ context.Context, _ email.SendEmailParams) error { return nil }
		defer func() { email.SendEmail = origSendEmail }()

		clientRepo := &mockClientRepo{
			findByClientIDAndIdentityProviderFn: func(_, _ string) (*Client, error) {
				return &Client{
					ClientID:   1,
					Status:     shared.StatusActive,
					Domain:     strPtr("https://auth.example.com"),
					Identifier: strPtr("test-client"),
					IdentityProvider: &IdentityProvider{
						Identifier: "test-provider",
					},
				}, nil
			},
		}
		userRepo := &mockUserRepo{
			findByEmailFn: func(_ string) (*User, error) {
				return &User{UserID: 1, Email: emailAddr, Username: "testuser", UserUUID: uuid.New(), Status: shared.StatusActive, IsEmailVerified: false}, nil
			},
		}
		userTokenRepo := &mockUserTokenRepo{
			findByUserIDAndTokenTypeFn: func(_ int64, _ string) ([]UserToken, error) { return nil, nil },
			createFn:                   func(ut *UserToken) (*UserToken, error) { ut.UserTokenID = 1; return ut, nil },
		}
		emailTemplateRepo := &mockEmailTemplateRepo{
			findByNameFn: func(_ string) (*branding.EmailTemplate, error) {
				plain := "Your verification code: {{.OTP}}"
				return &branding.EmailTemplate{
					Subject:   "Email Verification",
					BodyHTML:  `<p>Your verification code: {{.OTP}}</p>`,
					BodyPlain: &plain,
				}, nil
			},
		}

		svc := NewEmailVerificationService(gormDB, userRepo, userTokenRepo, clientRepo, emailTemplateRepo)
		resp, err := svc.SendVerificationEmail(context.Background(), emailAddr, clientID, providerID)

		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.True(t, resp.Success)
		assert.NotEmpty(t, resp.Message)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// ---------------------------------------------------------------------------
// TestVerifyEmail
// ---------------------------------------------------------------------------

func TestVerifyEmail(t *testing.T) {
	emailAddr := "test@example.com"
	otp := "123456"
	otpHash := crypto.HashAuthorizationCode(otp)
	userID := int64(1)
	userUUID := uuid.New()

	t.Run("user not found", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()

		userRepo := &mockUserRepo{
			findByEmailFn: func(_ string) (*User, error) { return nil, nil },
		}

		svc := NewEmailVerificationService(gormDB, userRepo, &mockUserTokenRepo{}, &mockClientRepo{}, &mockEmailTemplateRepo{})
		resp, err := svc.VerifyEmail(context.Background(), emailAddr, otp)

		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "invalid or expired verification code")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("user lookup error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()

		userRepo := &mockUserRepo{
			findByEmailFn: func(_ string) (*User, error) {
				return nil, errors.New("db error")
			},
		}

		svc := NewEmailVerificationService(gormDB, userRepo, &mockUserTokenRepo{}, &mockClientRepo{}, &mockEmailTemplateRepo{})
		resp, err := svc.VerifyEmail(context.Background(), emailAddr, otp)

		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "failed to find user")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("inactive user", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()

		userRepo := &mockUserRepo{
			findByEmailFn: func(_ string) (*User, error) {
				return &User{UserID: userID, Email: emailAddr, Status: shared.StatusInactive}, nil
			},
		}

		svc := NewEmailVerificationService(gormDB, userRepo, &mockUserTokenRepo{}, &mockClientRepo{}, &mockEmailTemplateRepo{})
		resp, err := svc.VerifyEmail(context.Background(), emailAddr, otp)

		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "user account is not active")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("already verified idempotent success", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()

		userRepo := &mockUserRepo{
			findByEmailFn: func(_ string) (*User, error) {
				return &User{UserID: userID, UserUUID: userUUID, Email: emailAddr, Status: shared.StatusActive, IsEmailVerified: true}, nil
			},
		}

		svc := NewEmailVerificationService(gormDB, userRepo, &mockUserTokenRepo{}, &mockClientRepo{}, &mockEmailTemplateRepo{})
		resp, err := svc.VerifyEmail(context.Background(), emailAddr, otp)

		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.True(t, resp.Success)
		assert.NotEmpty(t, resp.Message)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("no match", func(t *testing.T) {
		gormDB, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT \* FROM "user_tokens" WHERE .+`).
			WithArgs(userID, "user:email:verification", otpHash).
			WillReturnRows(sqlmock.NewRows([]string{"user_token_id", "user_token_uuid", "user_id", "token_type", "token", "expires_at", "is_revoked", "ip_address", "user_agent", "last_used_at", "idle_timeout_seconds", "absolute_expires_at", "created_at", "updated_at"}))
		mock.ExpectRollback()

		userRepo := &mockUserRepo{
			findByEmailFn: func(_ string) (*User, error) {
				return &User{UserID: userID, UserUUID: userUUID, Email: emailAddr, Status: shared.StatusActive, IsEmailVerified: false}, nil
			},
		}

		svc := NewEmailVerificationService(gormDB, userRepo, &mockUserTokenRepo{}, &mockClientRepo{}, &mockEmailTemplateRepo{})
		resp, err := svc.VerifyEmail(context.Background(), emailAddr, otp)

		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "invalid or expired verification code")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("expired", func(t *testing.T) {
		pastTime := time.Now().Add(-1 * time.Hour)
		gormDB, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT \* FROM "user_tokens" WHERE .+`).
			WithArgs(userID, "user:email:verification", otpHash).
			WillReturnRows(sqlmock.NewRows([]string{"user_token_id", "user_token_uuid", "user_id", "token_type", "token", "expires_at", "is_revoked", "ip_address", "user_agent", "last_used_at", "idle_timeout_seconds", "absolute_expires_at", "created_at", "updated_at"}).
				AddRow(1, uuid.New(), userID, "user:email:verification", otpHash, pastTime, false, nil, nil, nil, nil, nil, time.Now(), time.Now()))
		mock.ExpectRollback()

		userRepo := &mockUserRepo{
			findByEmailFn: func(_ string) (*User, error) {
				return &User{UserID: userID, UserUUID: userUUID, Email: emailAddr, Status: shared.StatusActive, IsEmailVerified: false}, nil
			},
		}

		svc := NewEmailVerificationService(gormDB, userRepo, &mockUserTokenRepo{}, &mockClientRepo{}, &mockEmailTemplateRepo{})
		resp, err := svc.VerifyEmail(context.Background(), emailAddr, otp)

		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "verification code has expired")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("success", func(t *testing.T) {
		futureTime := time.Now().Add(1 * time.Hour)
		gormDB, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT \* FROM "user_tokens" WHERE .+`).
			WithArgs(userID, "user:email:verification", otpHash).
			WillReturnRows(sqlmock.NewRows([]string{"user_token_id", "user_token_uuid", "user_id", "token_type", "token", "expires_at", "is_revoked", "ip_address", "user_agent", "last_used_at", "idle_timeout_seconds", "absolute_expires_at", "created_at", "updated_at"}).
				AddRow(1, uuid.New(), userID, "user:email:verification", otpHash, futureTime, false, nil, nil, nil, nil, nil, time.Now(), time.Now()))
		mock.ExpectCommit()

		userRepo := &mockUserRepo{
			findByEmailFn: func(_ string) (*User, error) {
				return &User{UserID: userID, UserUUID: userUUID, Email: emailAddr, Username: "testuser", Status: shared.StatusActive, IsEmailVerified: false}, nil
			},
			updateByIDFn: func(_ any, _ any) (*User, error) {
				return &User{UserID: userID, UserUUID: userUUID, Email: emailAddr, IsEmailVerified: true, Status: shared.StatusActive}, nil
			},
		}
		userTokenRepo := &mockUserTokenRepo{
			findByUserIDAndTokenTypeFn: func(_ int64, _ string) ([]UserToken, error) { return nil, nil },
		}

		svc := NewEmailVerificationService(gormDB, userRepo, userTokenRepo, &mockClientRepo{}, &mockEmailTemplateRepo{})
		resp, err := svc.VerifyEmail(context.Background(), emailAddr, otp)

		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.True(t, resp.Success)
		assert.NotEmpty(t, resp.Message)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
