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
	"gorm.io/gorm"
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

		svc := NewEmailVerificationService(gormDB, userRepo, &mockUserTokenRepo{}, clientRepo, &mockEmailTemplateRepo{}, nil, nil)
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

		svc := NewEmailVerificationService(gormDB, userRepo, &mockUserTokenRepo{}, clientRepo, &mockEmailTemplateRepo{}, nil, nil)
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

		svc := NewEmailVerificationService(gormDB, userRepo, &mockUserTokenRepo{}, clientRepo, &mockEmailTemplateRepo{}, nil, nil)
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
		email.SendEmail = func(_ context.Context, _ *gorm.DB, _ email.SendEmailParams) error { return nil }
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

		svc := NewEmailVerificationService(gormDB, userRepo, userTokenRepo, clientRepo, emailTemplateRepo, nil, nil)
		resp, err := svc.SendVerificationEmail(context.Background(), emailAddr, clientID, providerID)

		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.True(t, resp.Success)
		assert.NotEmpty(t, resp.Message)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// ---------------------------------------------------------------------------
// TestSendVerificationEmail_SystemClientLookupError
// ---------------------------------------------------------------------------

func TestSendVerificationEmail_SystemClientLookupError(t *testing.T) {
	emailAddr := "test@example.com"

	gormDB, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectRollback()

	clientRepo := &mockClientRepo{
		findSystemFn: func() (*Client, error) {
			return nil, errors.New("db error")
		},
	}

	svc := NewEmailVerificationService(gormDB, &mockUserRepo{}, &mockUserTokenRepo{}, clientRepo, &mockEmailTemplateRepo{}, nil, nil)
	resp, err := svc.SendVerificationEmail(context.Background(), emailAddr, nil, nil)

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "failed to find auth client")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// TestSendVerificationEmail_UserLookupError
// ---------------------------------------------------------------------------

func TestSendVerificationEmail_UserLookupError(t *testing.T) {
	emailAddr := "test@example.com"
	clientID := strPtr("test-client")
	providerID := strPtr("test-provider")

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
			return nil, errors.New("db error")
		},
	}

	svc := NewEmailVerificationService(gormDB, userRepo, &mockUserTokenRepo{}, clientRepo, &mockEmailTemplateRepo{}, nil, nil)
	resp, err := svc.SendVerificationEmail(context.Background(), emailAddr, clientID, providerID)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, resp.Success)
	assert.NotEmpty(t, resp.Message)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// TestVerifyEmail_TokenQueryError
// ---------------------------------------------------------------------------

func TestVerifyEmail_TokenQueryError(t *testing.T) {
	emailAddr := "test@example.com"
	otp := "123456"
	userID := int64(1)
	userUUID := uuid.New()

	gormDB, mock := newMockGormDBRegex(t)
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT \* FROM "user_tokens"`).
		WillReturnError(errors.New("db error"))
	mock.ExpectRollback()

	userRepo := &mockUserRepo{
		findByEmailFn: func(_ string) (*User, error) {
			return &User{UserID: userID, UserUUID: userUUID, Email: emailAddr, Status: shared.StatusActive, IsEmailVerified: false}, nil
		},
	}

	svc := NewEmailVerificationService(gormDB, userRepo, &mockUserTokenRepo{}, &mockClientRepo{}, &mockEmailTemplateRepo{}, nil, nil)
	resp, err := svc.VerifyEmail(context.Background(), emailAddr, otp)

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "failed to find verification token")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// TestVerifyEmail_UpdateByIDError
// ---------------------------------------------------------------------------

func TestVerifyEmail_UpdateByIDError(t *testing.T) {
	emailAddr := "test@example.com"
	otp := "123456"
	otpHash := crypto.HashAuthorizationCode(otp)
	userID := int64(1)
	userUUID := uuid.New()

	futureTime := time.Now().Add(1 * time.Hour)
	gormDB, mock := newMockGormDBRegex(t)
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT \* FROM "user_tokens"`).
		WithArgs(userID, "user:email:verification", otpHash).
		WillReturnRows(sqlmock.NewRows([]string{"user_token_id", "user_token_uuid", "user_id", "token_type", "token", "expires_at", "is_revoked", "ip_address", "user_agent", "last_used_at", "idle_timeout_seconds", "absolute_expires_at", "created_at", "updated_at"}).
			AddRow(1, uuid.New(), userID, "user:email:verification", otpHash, futureTime, false, nil, nil, nil, nil, nil, time.Now(), time.Now()))
	mock.ExpectRollback()

	userRepo := &mockUserRepo{
		findByEmailFn: func(_ string) (*User, error) {
			return &User{UserID: userID, UserUUID: userUUID, Email: emailAddr, Status: shared.StatusActive, IsEmailVerified: false}, nil
		},
		updateByIDFn: func(_ any, _ any) (*User, error) {
			return nil, errors.New("update error")
		},
	}

	svc := NewEmailVerificationService(gormDB, userRepo, &mockUserTokenRepo{}, &mockClientRepo{}, &mockEmailTemplateRepo{}, nil, nil)
	resp, err := svc.VerifyEmail(context.Background(), emailAddr, otp)

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "failed to update user verification status")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// TestVerifyEmail_FindExistingTokensError
// ---------------------------------------------------------------------------

func TestVerifyEmail_FindExistingTokensError(t *testing.T) {
	emailAddr := "test@example.com"
	otp := "123456"
	otpHash := crypto.HashAuthorizationCode(otp)
	userID := int64(1)
	userUUID := uuid.New()

	futureTime := time.Now().Add(1 * time.Hour)
	gormDB, mock := newMockGormDBRegex(t)
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT \* FROM "user_tokens"`).
		WithArgs(userID, "user:email:verification", otpHash).
		WillReturnRows(sqlmock.NewRows([]string{"user_token_id", "user_token_uuid", "user_id", "token_type", "token", "expires_at", "is_revoked", "ip_address", "user_agent", "last_used_at", "idle_timeout_seconds", "absolute_expires_at", "created_at", "updated_at"}).
			AddRow(1, uuid.New(), userID, "user:email:verification", otpHash, futureTime, false, nil, nil, nil, nil, nil, time.Now(), time.Now()))
	mock.ExpectRollback()

	userRepo := &mockUserRepo{
		findByEmailFn: func(_ string) (*User, error) {
			return &User{UserID: userID, UserUUID: userUUID, Email: emailAddr, Status: shared.StatusActive, IsEmailVerified: false}, nil
		},
	}
	userTokenRepo := &mockUserTokenRepo{
		findByUserIDAndTokenTypeFn: func(_ int64, _ string) ([]UserToken, error) {
			return nil, errors.New("db error")
		},
	}

	svc := NewEmailVerificationService(gormDB, userRepo, userTokenRepo, &mockClientRepo{}, &mockEmailTemplateRepo{}, nil, nil)
	resp, err := svc.VerifyEmail(context.Background(), emailAddr, otp)

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "failed to find existing tokens")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// TestVerifyEmail_RevokeByUUIDIterationError
// ---------------------------------------------------------------------------

func TestVerifyEmail_RevokeByUUIDIterationError(t *testing.T) {
	emailAddr := "test@example.com"
	otp := "123456"
	otpHash := crypto.HashAuthorizationCode(otp)
	userID := int64(1)
	userUUID := uuid.New()

	futureTime := time.Now().Add(1 * time.Hour)
	otherUUID := uuid.New()
	gormDB, mock := newMockGormDBRegex(t)
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT \* FROM "user_tokens"`).
		WithArgs(userID, "user:email:verification", otpHash).
		WillReturnRows(sqlmock.NewRows([]string{"user_token_id", "user_token_uuid", "user_id", "token_type", "token", "expires_at", "is_revoked", "ip_address", "user_agent", "last_used_at", "idle_timeout_seconds", "absolute_expires_at", "created_at", "updated_at"}).
			AddRow(1, uuid.New(), userID, "user:email:verification", otpHash, futureTime, false, nil, nil, nil, nil, nil, time.Now(), time.Now()))
	mock.ExpectRollback()

	userRepo := &mockUserRepo{
		findByEmailFn: func(_ string) (*User, error) {
			return &User{UserID: userID, UserUUID: userUUID, Email: emailAddr, Status: shared.StatusActive, IsEmailVerified: false}, nil
		},
	}
	revokeCount := 0
	userTokenRepo := &mockUserTokenRepo{
		findByUserIDAndTokenTypeFn: func(_ int64, _ string) ([]UserToken, error) {
			return []UserToken{
				{UserTokenUUID: otherUUID}, // triggers RevokeByUUID
			}, nil
		},
		revokeByUUIDFn: func(_ uuid.UUID) error {
			revokeCount++
			return errors.New("revoke error")
		},
	}

	svc := NewEmailVerificationService(gormDB, userRepo, userTokenRepo, &mockClientRepo{}, &mockEmailTemplateRepo{}, nil, nil)
	resp, err := svc.VerifyEmail(context.Background(), emailAddr, otp)

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "failed to revoke verification token")
	assert.Equal(t, 1, revokeCount)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// TestSendVerificationEmail_DefaultClient
// ---------------------------------------------------------------------------

func TestSendVerificationEmail_DefaultClient(t *testing.T) {
	emailAddr := "test@example.com"

	gormDB, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectCommit()

	config.AppPublicHostname = "http://localhost"

	origSendEmail := email.SendEmail
	email.SendEmail = func(_ context.Context, _ *gorm.DB, _ email.SendEmailParams) error { return nil }
	defer func() { email.SendEmail = origSendEmail }()

	clientRepo := &mockClientRepo{
		findSystemFn: func() (*Client, error) {
			return &Client{
				ClientID: 1,
				Status:   shared.StatusActive,
				Domain:   strPtr("https://auth.example.com"),
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

	svc := NewEmailVerificationService(gormDB, userRepo, userTokenRepo, clientRepo, emailTemplateRepo, nil, nil)
	resp, err := svc.SendVerificationEmail(context.Background(), emailAddr, nil, nil)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, resp.Success)
	assert.NotEmpty(t, resp.Message)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// TestSendVerificationEmail_ClientIDProviderLookupError
// ---------------------------------------------------------------------------

func TestSendVerificationEmail_ClientIDProviderLookupError(t *testing.T) {
	emailAddr := "test@example.com"
	clientID := strPtr("test-client")
	providerID := strPtr("test-provider")

	gormDB, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectRollback()

	clientRepo := &mockClientRepo{
		findByClientIDAndIdentityProviderFn: func(_, _ string) (*Client, error) {
			return nil, errors.New("db error")
		},
	}

	svc := NewEmailVerificationService(gormDB, &mockUserRepo{}, &mockUserTokenRepo{}, clientRepo, &mockEmailTemplateRepo{}, nil, nil)
	resp, err := svc.SendVerificationEmail(context.Background(), emailAddr, clientID, providerID)

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// TestSendVerificationEmail_OTPError
// ---------------------------------------------------------------------------

func TestSendVerificationEmail_OTPError(t *testing.T) {
	t.Skip("crypto/rand.Reader replacement causes panic on this Go version")
}

// ---------------------------------------------------------------------------
// TestSendVerificationEmail_RevokeError
// ---------------------------------------------------------------------------

func TestSendVerificationEmail_RevokeError(t *testing.T) {
	emailAddr := "test@example.com"
	clientID := strPtr("test-client")
	providerID := strPtr("test-provider")

	gormDB, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectRollback()

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
		findByUserIDAndTokenTypeFn: func(_ int64, _ string) ([]UserToken, error) {
			return []UserToken{{UserTokenUUID: uuid.New()}}, nil
		},
		revokeByUUIDFn: func(_ uuid.UUID) error {
			return errors.New("revoke error")
		},
	}

	svc := NewEmailVerificationService(gormDB, userRepo, userTokenRepo, clientRepo, &mockEmailTemplateRepo{}, nil, nil)
	resp, err := svc.SendVerificationEmail(context.Background(), emailAddr, clientID, providerID)

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "failed to revoke existing token")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// TestSendVerificationEmail_TokenCreateError
// ---------------------------------------------------------------------------

func TestSendVerificationEmail_TokenCreateError(t *testing.T) {
	emailAddr := "test@example.com"
	clientID := strPtr("test-client")
	providerID := strPtr("test-provider")

	gormDB, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectRollback()

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
		createFn:                   func(_ *UserToken) (*UserToken, error) { return nil, errors.New("create error") },
	}

	svc := NewEmailVerificationService(gormDB, userRepo, userTokenRepo, clientRepo, &mockEmailTemplateRepo{}, nil, nil)
	resp, err := svc.SendVerificationEmail(context.Background(), emailAddr, clientID, providerID)

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "failed to create verification token")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// TestSendVerificationEmail_TemplateFindByNameError
// ---------------------------------------------------------------------------

func TestSendVerificationEmail_TemplateFindByNameError(t *testing.T) {
	emailAddr := "test@example.com"
	clientID := strPtr("test-client")
	providerID := strPtr("test-provider")

	gormDB, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectCommit()

	origSendEmail := email.SendEmail
	email.SendEmail = func(_ context.Context, _ *gorm.DB, _ email.SendEmailParams) error { return nil }
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
			return nil, errors.New("template not found")
		},
	}

	svc := NewEmailVerificationService(gormDB, userRepo, userTokenRepo, clientRepo, emailTemplateRepo, nil, nil)
	resp, err := svc.SendVerificationEmail(context.Background(), emailAddr, clientID, providerID)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, resp.Success)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// TestSendVerificationEmail_TemplateParseError
// ---------------------------------------------------------------------------

func TestSendVerificationEmail_TemplateParseError(t *testing.T) {
	emailAddr := "test@example.com"
	clientID := strPtr("test-client")
	providerID := strPtr("test-provider")

	gormDB, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectCommit()

	origSendEmail := email.SendEmail
	email.SendEmail = func(_ context.Context, _ *gorm.DB, _ email.SendEmailParams) error { return nil }
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
			return &branding.EmailTemplate{
				Subject:  "Email Verification",
				BodyHTML: `{{if}}`,
			}, nil
		},
	}

	svc := NewEmailVerificationService(gormDB, userRepo, userTokenRepo, clientRepo, emailTemplateRepo, nil, nil)
	resp, err := svc.SendVerificationEmail(context.Background(), emailAddr, clientID, providerID)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, resp.Success)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// TestSendVerificationEmail_TemplateExecuteError
// ---------------------------------------------------------------------------

func TestSendVerificationEmail_TemplateExecuteError(t *testing.T) {
	emailAddr := "test@example.com"
	clientID := strPtr("test-client")
	providerID := strPtr("test-provider")

	gormDB, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectCommit()

	origSendEmail := email.SendEmail
	email.SendEmail = func(_ context.Context, _ *gorm.DB, _ email.SendEmailParams) error { return nil }
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
			return &branding.EmailTemplate{
				Subject:  "Email Verification",
				BodyHTML: `{{index . 0}}`,
			}, nil
		},
	}

	svc := NewEmailVerificationService(gormDB, userRepo, userTokenRepo, clientRepo, emailTemplateRepo, nil, nil)
	resp, err := svc.SendVerificationEmail(context.Background(), emailAddr, clientID, providerID)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, resp.Success)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// TestSendVerificationEmail_PlaintextParseError
// ---------------------------------------------------------------------------

func TestSendVerificationEmail_PlaintextParseError(t *testing.T) {
	emailAddr := "test@example.com"
	clientID := strPtr("test-client")
	providerID := strPtr("test-provider")

	gormDB, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectCommit()

	origSendEmail := email.SendEmail
	email.SendEmail = func(_ context.Context, _ *gorm.DB, _ email.SendEmailParams) error { return nil }
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
	invalidPlain := "{{if}}"
	emailTemplateRepo := &mockEmailTemplateRepo{
		findByNameFn: func(_ string) (*branding.EmailTemplate, error) {
			return &branding.EmailTemplate{
				Subject:   "Email Verification",
				BodyHTML:  `<p>Your verification code: {{.OTP}}</p>`,
				BodyPlain: &invalidPlain,
			}, nil
		},
	}

	svc := NewEmailVerificationService(gormDB, userRepo, userTokenRepo, clientRepo, emailTemplateRepo, nil, nil)
	resp, err := svc.SendVerificationEmail(context.Background(), emailAddr, clientID, providerID)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, resp.Success)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// TestSendVerificationEmail_PlaintextExecuteError
// ---------------------------------------------------------------------------

func TestSendVerificationEmail_PlaintextExecuteError(t *testing.T) {
	emailAddr := "test@example.com"
	clientID := strPtr("test-client")
	providerID := strPtr("test-provider")

	gormDB, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectCommit()

	origSendEmail := email.SendEmail
	email.SendEmail = func(_ context.Context, _ *gorm.DB, _ email.SendEmailParams) error { return nil }
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
	invalidPlain := "{{index . 0}}"
	emailTemplateRepo := &mockEmailTemplateRepo{
		findByNameFn: func(_ string) (*branding.EmailTemplate, error) {
			return &branding.EmailTemplate{
				Subject:   "Email Verification",
				BodyHTML:  `<p>Your verification code: {{.OTP}}</p>`,
				BodyPlain: &invalidPlain,
			}, nil
		},
	}

	svc := NewEmailVerificationService(gormDB, userRepo, userTokenRepo, clientRepo, emailTemplateRepo, nil, nil)
	resp, err := svc.SendVerificationEmail(context.Background(), emailAddr, clientID, providerID)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, resp.Success)
	assert.NoError(t, mock.ExpectationsWereMet())
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

		svc := NewEmailVerificationService(gormDB, userRepo, &mockUserTokenRepo{}, &mockClientRepo{}, &mockEmailTemplateRepo{}, nil, nil)
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

		svc := NewEmailVerificationService(gormDB, userRepo, &mockUserTokenRepo{}, &mockClientRepo{}, &mockEmailTemplateRepo{}, nil, nil)
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

		svc := NewEmailVerificationService(gormDB, userRepo, &mockUserTokenRepo{}, &mockClientRepo{}, &mockEmailTemplateRepo{}, nil, nil)
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

		svc := NewEmailVerificationService(gormDB, userRepo, &mockUserTokenRepo{}, &mockClientRepo{}, &mockEmailTemplateRepo{}, nil, nil)
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

		svc := NewEmailVerificationService(gormDB, userRepo, &mockUserTokenRepo{}, &mockClientRepo{}, &mockEmailTemplateRepo{}, nil, nil)
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

		svc := NewEmailVerificationService(gormDB, userRepo, &mockUserTokenRepo{}, &mockClientRepo{}, &mockEmailTemplateRepo{}, nil, nil)
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

		svc := NewEmailVerificationService(gormDB, userRepo, userTokenRepo, &mockClientRepo{}, &mockEmailTemplateRepo{}, nil, nil)
		resp, err := svc.VerifyEmail(context.Background(), emailAddr, otp)

		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.True(t, resp.Success)
		assert.NotEmpty(t, resp.Message)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("success invalidates the user-context cache", func(t *testing.T) {
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
		identityRepo := &mockUserIdentityRepo{
			findByUserIDFn: func(_ int64) ([]UserIdentity, error) {
				return []UserIdentity{{Sub: "sub-abc"}}, nil
			},
		}
		inv := &recordingInvalidator{}

		svc := NewEmailVerificationService(gormDB, userRepo, userTokenRepo, &mockClientRepo{}, &mockEmailTemplateRepo{}, identityRepo, inv)
		resp, err := svc.VerifyEmail(context.Background(), emailAddr, otp)

		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.True(t, resp.Success)
		// The freshly-verified user's cached context must be cleared so /account
		// no longer returns the stale email_verified=false captured at register.
		assert.Equal(t, []string{"sub-abc"}, inv.subs)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// recordingInvalidator is a cache.Invalidator test double that records the subs
// passed to InvalidateUserAll.
type recordingInvalidator struct {
	subs []string
}

func (r *recordingInvalidator) InvalidateUser(_ context.Context, _, _ string) {}
func (r *recordingInvalidator) InvalidateUserAll(_ context.Context, sub string) {
	r.subs = append(r.subs, sub)
}
func (r *recordingInvalidator) InvalidateAllUsers(_ context.Context) {}
