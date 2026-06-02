package authn

import (
	"context"
	"errors"
	"testing"

	"github.com/maintainerd/auth/internal/notifier"
	"github.com/maintainerd/auth/internal/platform/config"
	"github.com/maintainerd/auth/internal/platform/crypto"
	"github.com/maintainerd/auth/internal/platform/jwt"
	"github.com/maintainerd/auth/internal/platform/security"
	"github.com/maintainerd/auth/internal/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// ---------------------------------------------------------------------------
// Mock: notifier.SMSOtpRepository
// ---------------------------------------------------------------------------

type mockSMSOtpRepo struct {
	createFn           func(*notifier.SMSOtp) (*notifier.SMSOtp, error)
	findValidByPhoneFn func(string) (*notifier.SMSOtp, error)
	recordFailureFn    func(int64, int) error
	markUsedFn         func(int64) error
}

func (m *mockSMSOtpRepo) WithTx(_ *gorm.DB) notifier.SMSOtpRepository { return m }
func (m *mockSMSOtpRepo) Create(e *notifier.SMSOtp) (*notifier.SMSOtp, error) {
	if m.createFn != nil {
		return m.createFn(e)
	}
	return e, nil
}
func (m *mockSMSOtpRepo) CreateOrUpdate(e *notifier.SMSOtp) (*notifier.SMSOtp, error) {
	return nil, nil
}
func (m *mockSMSOtpRepo) FindAll(p ...string) ([]notifier.SMSOtp, error) { return nil, nil }
func (m *mockSMSOtpRepo) FindByUUID(id any, p ...string) (*notifier.SMSOtp, error) {
	return nil, nil
}
func (m *mockSMSOtpRepo) FindByUUIDs(ids []string, p ...string) ([]notifier.SMSOtp, error) {
	return nil, nil
}
func (m *mockSMSOtpRepo) FindByID(id any, p ...string) (*notifier.SMSOtp, error) { return nil, nil }
func (m *mockSMSOtpRepo) UpdateByUUID(id, data any) (*notifier.SMSOtp, error) {
	return nil, nil
}
func (m *mockSMSOtpRepo) UpdateByID(id, data any) (*notifier.SMSOtp, error) { return nil, nil }
func (m *mockSMSOtpRepo) DeleteByUUID(id any) error                         { return nil }
func (m *mockSMSOtpRepo) DeleteByID(id any) error                           { return nil }
func (m *mockSMSOtpRepo) Paginate(c map[string]any, pg, lim int, p ...string) (*notifier.PaginationResult[notifier.SMSOtp], error) {
	return nil, nil
}
func (m *mockSMSOtpRepo) FindValidByPhone(phone string) (*notifier.SMSOtp, error) {
	if m.findValidByPhoneFn != nil {
		return m.findValidByPhoneFn(phone)
	}
	return nil, nil
}
func (m *mockSMSOtpRepo) RecordFailure(id int64, maxAttempts int) error {
	if m.recordFailureFn != nil {
		return m.recordFailureFn(id, maxAttempts)
	}
	return nil
}
func (m *mockSMSOtpRepo) MarkUsed(id int64) error {
	if m.markUsedFn != nil {
		return m.markUsedFn(id)
	}
	return nil
}

// ---------------------------------------------------------------------------
// TestSendOTP
// ---------------------------------------------------------------------------

func TestSendOTP(t *testing.T) {
	phone := "+1234567890"
	originalBudget := config.SMSDailySendLimit
	t.Cleanup(func() {
		config.SMSDailySendLimit = originalBudget
		security.InitRateLimiter(nil)
		security.ResetSMSDailyBudgetCounters()
	})

	t.Run("user not found returns nil no error", func(t *testing.T) {
		userRepo := &mockUserRepo{
			findByPhoneFn: func(_ string) (*User, error) { return nil, nil },
		}

		svc := NewSMSLoginService(nil, userRepo, &mockSMSOtpRepo{}, &mockClientRepo{}, &mockUserIdentityRepo{}, &mockIdentityProviderRepo{}, &mockAuthEventService{})
		err := svc.SendOTP(context.Background(), SMSLoginSendDTO{Phone: phone})

		require.NoError(t, err)
	})

	t.Run("user inactive returns nil no error", func(t *testing.T) {
		userRepo := &mockUserRepo{
			findByPhoneFn: func(_ string) (*User, error) {
				return &User{UserID: 1, Phone: phone, Status: shared.StatusInactive}, nil
			},
		}

		svc := NewSMSLoginService(nil, userRepo, &mockSMSOtpRepo{}, &mockClientRepo{}, &mockUserIdentityRepo{}, &mockIdentityProviderRepo{}, &mockAuthEventService{})
		err := svc.SendOTP(context.Background(), SMSLoginSendDTO{Phone: phone})

		require.NoError(t, err)
	})

	t.Run("success rate-limit passes user found OTP stored", func(t *testing.T) {
		config.SMSDailySendLimit = 0
		userRepo := &mockUserRepo{
			findByPhoneFn: func(_ string) (*User, error) {
				return &User{UserID: 1, Phone: phone, Status: shared.StatusActive}, nil
			},
		}
		smsOtpRepo := &mockSMSOtpRepo{
			createFn: func(otp *notifier.SMSOtp) (*notifier.SMSOtp, error) {
				otp.SMSOtpID = 1
				return otp, nil
			},
		}

		svc := NewSMSLoginService(nil, userRepo, smsOtpRepo, &mockClientRepo{}, &mockUserIdentityRepo{}, &mockIdentityProviderRepo{}, &mockAuthEventService{})
		err := svc.SendOTP(context.Background(), SMSLoginSendDTO{Phone: phone})

		require.NoError(t, err)
	})

	t.Run("daily SMS budget blocks additional sends", func(t *testing.T) {
		config.SMSDailySendLimit = 1
		security.InitRateLimiter(nil)
		security.ResetSMSDailyBudgetCounters()
		userRepo := &mockUserRepo{
			findByPhoneFn: func(_ string) (*User, error) {
				return &User{UserID: 1, Phone: phone, Status: shared.StatusActive}, nil
			},
		}
		createCount := 0
		smsOtpRepo := &mockSMSOtpRepo{
			createFn: func(otp *notifier.SMSOtp) (*notifier.SMSOtp, error) {
				createCount++
				otp.SMSOtpID = int64(createCount)
				return otp, nil
			},
		}

		svc := NewSMSLoginService(nil, userRepo, smsOtpRepo, &mockClientRepo{}, &mockUserIdentityRepo{}, &mockIdentityProviderRepo{}, &mockAuthEventService{})
		require.NoError(t, svc.SendOTP(context.Background(), SMSLoginSendDTO{Phone: phone}))
		err := svc.SendOTP(context.Background(), SMSLoginSendDTO{Phone: phone})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "SMS send budget exceeded")
		assert.Equal(t, 1, createCount)
	})

	t.Run("user lookup error returns internal error", func(t *testing.T) {
		userRepo := &mockUserRepo{
			findByPhoneFn: func(_ string) (*User, error) {
				return nil, errors.New("db error")
			},
		}

		svc := NewSMSLoginService(nil, userRepo, &mockSMSOtpRepo{}, &mockClientRepo{}, &mockUserIdentityRepo{}, &mockIdentityProviderRepo{}, &mockAuthEventService{})
		err := svc.SendOTP(context.Background(), SMSLoginSendDTO{Phone: phone})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to look up user")
	})
}

// ---------------------------------------------------------------------------
// TestVerifyOTP
// ---------------------------------------------------------------------------

func TestVerifyOTP(t *testing.T) {
	phone := "+1234567890"
	otp := "123456"
	clientID := "test-client"
	providerID := "test-provider"

	t.Run("identity provider not found", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()

		idpRepo := &mockIdentityProviderRepo{
			findByIdentifierFn: func(_ string) (*IdentityProvider, error) {
				return nil, nil
			},
		}

		svc := NewSMSLoginService(gormDB, &mockUserRepo{}, &mockSMSOtpRepo{}, &mockClientRepo{}, &mockUserIdentityRepo{}, idpRepo, &mockAuthEventService{})
		resp, err := svc.VerifyOTP(context.Background(), SMSLoginVerifyDTO{Phone: phone, OTP: otp, ClientID: clientID, ProviderID: providerID})

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

		svc := NewSMSLoginService(gormDB, &mockUserRepo{}, &mockSMSOtpRepo{}, clientRepo, &mockUserIdentityRepo{}, idpRepo, &mockAuthEventService{})
		resp, err := svc.VerifyOTP(context.Background(), SMSLoginVerifyDTO{Phone: phone, OTP: otp, ClientID: clientID, ProviderID: providerID})

		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "authentication failed")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("user not found", func(t *testing.T) {
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
				return buildActiveClient(), nil
			},
		}
		userRepo := &mockUserRepo{
			findByPhoneFn: func(_ string) (*User, error) { return nil, nil },
		}

		svc := NewSMSLoginService(gormDB, userRepo, &mockSMSOtpRepo{}, clientRepo, &mockUserIdentityRepo{}, idpRepo, &mockAuthEventService{})
		resp, err := svc.VerifyOTP(context.Background(), SMSLoginVerifyDTO{Phone: phone, OTP: otp, ClientID: clientID, ProviderID: providerID})

		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "invalid phone or OTP")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("OTP not found", func(t *testing.T) {
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
				return buildActiveClient(), nil
			},
		}
		userRepo := &mockUserRepo{
			findByPhoneFn: func(_ string) (*User, error) {
				return &User{UserID: 1, Phone: phone, Status: shared.StatusActive}, nil
			},
		}
		smsOtpRepo := &mockSMSOtpRepo{
			findValidByPhoneFn: func(_ string) (*notifier.SMSOtp, error) {
				return nil, nil
			},
		}

		svc := NewSMSLoginService(gormDB, userRepo, smsOtpRepo, clientRepo, &mockUserIdentityRepo{}, idpRepo, &mockAuthEventService{})
		resp, err := svc.VerifyOTP(context.Background(), SMSLoginVerifyDTO{Phone: phone, OTP: otp, ClientID: clientID, ProviderID: providerID})

		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "invalid or expired OTP")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("OTP hash mismatch", func(t *testing.T) {
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
				return buildActiveClient(), nil
			},
		}
		userRepo := &mockUserRepo{
			findByPhoneFn: func(_ string) (*User, error) {
				return &User{UserID: 1, Phone: phone, Status: shared.StatusActive}, nil
			},
		}
		smsOtpRepo := &mockSMSOtpRepo{
			findValidByPhoneFn: func(_ string) (*notifier.SMSOtp, error) {
				return &notifier.SMSOtp{
					SMSOtpID: 1,
					OTPHash:  "wrong-hash-value-for-verification-testing",
				}, nil
			},
			recordFailureFn: func(id int64, maxAttempts int) error {
				assert.Equal(t, int64(1), id)
				assert.Equal(t, smsOTPMaxFailedAttempts, maxAttempts)
				return nil
			},
		}

		svc := NewSMSLoginService(gormDB, userRepo, smsOtpRepo, clientRepo, &mockUserIdentityRepo{}, idpRepo, &mockAuthEventService{})
		resp, err := svc.VerifyOTP(context.Background(), SMSLoginVerifyDTO{Phone: phone, OTP: otp, ClientID: clientID, ProviderID: providerID})

		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "invalid or expired OTP")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("mark used error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()

		correctHash := crypto.HashAuthorizationCode(otp)
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
			findByPhoneFn: func(_ string) (*User, error) {
				return &User{UserID: 1, Phone: phone, Status: shared.StatusActive}, nil
			},
		}
		smsOtpRepo := &mockSMSOtpRepo{
			findValidByPhoneFn: func(_ string) (*notifier.SMSOtp, error) {
				return &notifier.SMSOtp{
					SMSOtpID: 1,
					OTPHash:  correctHash,
				}, nil
			},
			markUsedFn: func(_ int64) error {
				return errors.New("db error")
			},
		}

		svc := NewSMSLoginService(gormDB, userRepo, smsOtpRepo, clientRepo, &mockUserIdentityRepo{}, idpRepo, &mockAuthEventService{})
		resp, err := svc.VerifyOTP(context.Background(), SMSLoginVerifyDTO{Phone: phone, OTP: otp, ClientID: clientID, ProviderID: providerID})

		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "failed to invalidate OTP")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("success", func(t *testing.T) {
		initTestJWTKeysService(t)
		correctHash := crypto.HashAuthorizationCode(otp)
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
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
			findByPhoneFn: func(_ string) (*User, error) {
				return &User{UserID: 1, Phone: phone, Status: shared.StatusActive, Email: "test@example.com", Username: "testuser"}, nil
			},
		}
		smsOtpRepo := &mockSMSOtpRepo{
			findValidByPhoneFn: func(_ string) (*notifier.SMSOtp, error) {
				return &notifier.SMSOtp{
					SMSOtpID: 1,
					OTPHash:  correctHash,
				}, nil
			},
		}
		userIdentityRepo := &mockUserIdentityRepo{
			findByUserIDAndClientIDFn: func(_, _ int64) (*UserIdentity, error) {
				return &UserIdentity{Sub: "sub-123"}, nil
			},
		}

		svc := NewSMSLoginService(gormDB, userRepo, smsOtpRepo, clientRepo, userIdentityRepo, idpRepo, &mockAuthEventService{})
		resp, err := svc.VerifyOTP(context.Background(), SMSLoginVerifyDTO{Phone: phone, OTP: otp, ClientID: clientID, ProviderID: providerID})

		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.NotEmpty(t, resp.AccessToken)
		assert.Equal(t, "Bearer", resp.TokenType)
		assert.Equal(t, int64(3600), resp.ExpiresIn)
		accessClaims, err := jwt.ValidateToken(resp.AccessToken)
		require.NoError(t, err)
		assert.Equal(t, jwt.ACRLevel1, accessClaims["acr"])
		assert.ElementsMatch(t, []any{jwt.AMRSMS}, accessClaims["amr"])
		idClaims, err := jwt.ValidateToken(resp.IDToken)
		require.NoError(t, err)
		assert.Equal(t, jwt.ACRLevel1, idClaims["acr"])
		assert.ElementsMatch(t, []any{jwt.AMRSMS}, idClaims["amr"])
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
