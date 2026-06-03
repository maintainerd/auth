package authn

import (
	"context"
	"errors"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/notifier"
	"github.com/maintainerd/auth/internal/platform/config"
	"github.com/maintainerd/auth/internal/platform/crypto"
	"github.com/maintainerd/auth/internal/platform/jwt"
	"github.com/maintainerd/auth/internal/platform/security"
	"github.com/maintainerd/auth/internal/shared"
	"github.com/redis/go-redis/v9"
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

// ---------------------------------------------------------------------------
// Mock: SessionService for SMS login tests
// ---------------------------------------------------------------------------

type mockSMSSessionService struct {
	enforceFn func(context.Context, uuid.UUID, int64) error
	createFn  func(context.Context, int64, string, string) (*UserToken, error)
}

func (m *mockSMSSessionService) ListSessions(ctx context.Context, userID int64) ([]*SessionDataResult, error) {
	return nil, nil
}
func (m *mockSMSSessionService) RevokeSession(ctx context.Context, userID int64, sessionUUID uuid.UUID) error {
	return nil
}
func (m *mockSMSSessionService) RevokeAllSessions(ctx context.Context, userID int64) error { return nil }
func (m *mockSMSSessionService) CreateSession(ctx context.Context, userID int64, ipAddress, userAgent string) (*UserToken, error) {
	if m.createFn != nil {
		return m.createFn(ctx, userID, ipAddress, userAgent)
	}
	return &UserToken{UserTokenUUID: uuid.New(), TokenType: "session"}, nil
}
func (m *mockSMSSessionService) EnforceConcurrentLimit(ctx context.Context, userUUID uuid.UUID, userID int64) error {
	if m.enforceFn != nil {
		return m.enforceFn(ctx, userUUID, userID)
	}
	return nil
}
func (m *mockSMSSessionService) ValidateAndTouch(ctx context.Context, sessionUUID uuid.UUID, userID int64) error {
	return nil
}

// ---------------------------------------------------------------------------
// Helper: lockedRateLimiterSMS
// ---------------------------------------------------------------------------

func lockedRateLimiterSMS(t *testing.T, identifier string) func() {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	security.InitRateLimiter(rdb)
	require.NoError(t, mr.Set("rl:lock:"+identifier, "1"))
	return func() {
		security.InitRateLimiter(nil)
		rdb.Close()
		mr.Close()
	}
}

// ---------------------------------------------------------------------------
// TestNewSMSLoginService_WithSession
// ---------------------------------------------------------------------------

func TestNewSMSLoginService_WithSession(t *testing.T) {
	mockSess := &mockSMSSessionService{}
	svc := NewSMSLoginService(nil, &mockUserRepo{}, &mockSMSOtpRepo{}, &mockClientRepo{}, &mockUserIdentityRepo{}, &mockIdentityProviderRepo{}, &mockAuthEventService{}, mockSess)
	typed, ok := svc.(*smsLoginService)
	require.True(t, ok)
	assert.Same(t, mockSess, typed.sessionService)
}

// ---------------------------------------------------------------------------
// TestSendOTP – rate limit error
// ---------------------------------------------------------------------------

func TestSendOTP_RateLimited(t *testing.T) {
	phone := "+1234567890"
	cleanup := lockedRateLimiterSMS(t, "sms-otp:send:"+phone)
	defer cleanup()

	svc := NewSMSLoginService(nil, &mockUserRepo{}, &mockSMSOtpRepo{}, &mockClientRepo{}, &mockUserIdentityRepo{}, &mockIdentityProviderRepo{}, &mockAuthEventService{})
	err := svc.SendOTP(context.Background(), SMSLoginSendDTO{Phone: phone})

	require.Error(t, err)
		assert.Contains(t, err.Error(), "account is locked")
}

// ---------------------------------------------------------------------------
// TestSendOTP – storage error
// ---------------------------------------------------------------------------

func TestSendOTP_StorageError(t *testing.T) {
	phone := "+1234567890"
	originalBudget := config.SMSDailySendLimit
	t.Cleanup(func() {
		config.SMSDailySendLimit = originalBudget
		security.InitRateLimiter(nil)
		security.ResetSMSDailyBudgetCounters()
	})
	config.SMSDailySendLimit = 0

	userRepo := &mockUserRepo{
		findByPhoneFn: func(_ string) (*User, error) {
			return &User{UserID: 1, Phone: phone, Status: shared.StatusActive}, nil
		},
	}
	smsOtpRepo := &mockSMSOtpRepo{
		createFn: func(_ *notifier.SMSOtp) (*notifier.SMSOtp, error) {
			return nil, errors.New("db error")
		},
	}

	svc := NewSMSLoginService(nil, userRepo, smsOtpRepo, &mockClientRepo{}, &mockUserIdentityRepo{}, &mockIdentityProviderRepo{}, &mockAuthEventService{})
	err := svc.SendOTP(context.Background(), SMSLoginSendDTO{Phone: phone})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to store SMS OTP")
}

// ---------------------------------------------------------------------------
// TestSendOTP – SMS provider path
// ---------------------------------------------------------------------------

func TestSendOTP_SMSProviderPath(t *testing.T) {
	phone := "+1234567890"
	originalBudget := config.SMSDailySendLimit
	originalProvider := config.SMSProvider
	t.Cleanup(func() {
		config.SMSDailySendLimit = originalBudget
		config.SMSProvider = originalProvider
		security.InitRateLimiter(nil)
		security.ResetSMSDailyBudgetCounters()
	})
	config.SMSDailySendLimit = 0
	config.SMSProvider = "test-provider"

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
}

// ---------------------------------------------------------------------------
// TestVerifyOTP – rate limit error
// ---------------------------------------------------------------------------

func TestVerifyOTP_RateLimited(t *testing.T) {
	phone := "+1234567890"
	cleanup := lockedRateLimiterSMS(t, "sms-otp:verify:"+phone)
	defer cleanup()

	svc := NewSMSLoginService(nil, &mockUserRepo{}, &mockSMSOtpRepo{}, &mockClientRepo{}, &mockUserIdentityRepo{}, &mockIdentityProviderRepo{}, &mockAuthEventService{})
	resp, err := svc.VerifyOTP(context.Background(), SMSLoginVerifyDTO{Phone: phone, OTP: "123456", ClientID: "c", ProviderID: "p"})

	require.Error(t, err)
	assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "account is locked")
}


// ---------------------------------------------------------------------------
// TestVerifyOTP – user lookup error
// ---------------------------------------------------------------------------

func TestVerifyOTP_UserLookupError(t *testing.T) {
	phone := "+1234567890"
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
			return nil, errors.New("db error")
		},
	}

	svc := NewSMSLoginService(gormDB, userRepo, &mockSMSOtpRepo{}, clientRepo, &mockUserIdentityRepo{}, idpRepo, &mockAuthEventService{})
	resp, err := svc.VerifyOTP(context.Background(), SMSLoginVerifyDTO{Phone: phone, OTP: "123456", ClientID: "test-client", ProviderID: "test-provider"})

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "failed to look up user")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// TestVerifyOTP – RecordFailure error on hash mismatch
// ---------------------------------------------------------------------------

func TestVerifyOTP_RecordFailureError(t *testing.T) {
	phone := "+1234567890"
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
				OTPHash:  "wrong-hash",
			}, nil
		},
		recordFailureFn: func(_ int64, _ int) error {
			return errors.New("db error")
		},
	}

	svc := NewSMSLoginService(gormDB, userRepo, smsOtpRepo, clientRepo, &mockUserIdentityRepo{}, idpRepo, &mockAuthEventService{})
	resp, err := svc.VerifyOTP(context.Background(), SMSLoginVerifyDTO{Phone: phone, OTP: "123456", ClientID: "test-client", ProviderID: "test-provider"})

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "failed to record OTP attempt")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// TestVerifyOTP – user identity nil
// ---------------------------------------------------------------------------

func TestVerifyOTP_UserIdentityNil(t *testing.T) {
	phone := "+1234567890"
	otp := "123456"
	correctHash := crypto.HashAuthorizationCode(otp)
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
				OTPHash:  correctHash,
			}, nil
		},
	}
	userIdentityRepo := &mockUserIdentityRepo{
		findByUserIDAndClientIDFn: func(_, _ int64) (*UserIdentity, error) {
			return nil, nil
		},
	}

	svc := NewSMSLoginService(gormDB, userRepo, smsOtpRepo, clientRepo, userIdentityRepo, idpRepo, &mockAuthEventService{})
	resp, err := svc.VerifyOTP(context.Background(), SMSLoginVerifyDTO{Phone: phone, OTP: otp, ClientID: "test-client", ProviderID: "test-provider"})

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "authentication failed")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// TestVerifyOTP – success with session
// ---------------------------------------------------------------------------

func TestVerifyOTP_WithSession(t *testing.T) {
	initTestJWTKeysService(t)
	phone := "+1234567890"
	otp := "123456"
	correctHash := crypto.HashAuthorizationCode(otp)
	sessionUUID := uuid.New()
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
			return &User{UserID: 1, UserUUID: uuid.New(), Phone: phone, Status: shared.StatusActive, Email: "test@example.com", Username: "testuser"}, nil
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
	sessionSvc := &mockSMSSessionService{
		createFn: func(_ context.Context, _ int64, _, _ string) (*UserToken, error) {
			return &UserToken{UserTokenUUID: sessionUUID, TokenType: "session"}, nil
		},
	}

	svc := NewSMSLoginService(gormDB, userRepo, smsOtpRepo, clientRepo, userIdentityRepo, idpRepo, &mockAuthEventService{}, sessionSvc)
	resp, err := svc.VerifyOTP(context.Background(), SMSLoginVerifyDTO{Phone: phone, OTP: otp, ClientID: "test-client", ProviderID: "test-provider"})

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotEmpty(t, resp.AccessToken)
	assert.Equal(t, "Bearer", resp.TokenType)
	assert.Equal(t, int64(3600), resp.ExpiresIn)
	require.NotNil(t, resp.SessionID)
	assert.Equal(t, sessionUUID.String(), *resp.SessionID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// TestVerifyOTP – EnforceConcurrentLimit error
// ---------------------------------------------------------------------------

func TestVerifyOTP_EnforceConcurrentLimitError(t *testing.T) {
	initTestJWTKeysService(t)
	phone := "+1234567890"
	otp := "123456"
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
			return &User{UserID: 1, UserUUID: uuid.New(), Phone: phone, Status: shared.StatusActive, Email: "test@example.com", Username: "testuser"}, nil
		},
	}
	smsOtpRepo := &mockSMSOtpRepo{
		findValidByPhoneFn: func(_ string) (*notifier.SMSOtp, error) {
			return &notifier.SMSOtp{SMSOtpID: 1, OTPHash: correctHash}, nil
		},
	}
	userIdentityRepo := &mockUserIdentityRepo{
		findByUserIDAndClientIDFn: func(_, _ int64) (*UserIdentity, error) {
			return &UserIdentity{Sub: "sub-123"}, nil
		},
	}
	sessionSvc := &mockSMSSessionService{
		enforceFn: func(_ context.Context, _ uuid.UUID, _ int64) error {
			return errors.New("session limit exceeded")
		},
	}

	svc := NewSMSLoginService(gormDB, userRepo, smsOtpRepo, clientRepo, userIdentityRepo, idpRepo, &mockAuthEventService{}, sessionSvc)
	resp, err := svc.VerifyOTP(context.Background(), SMSLoginVerifyDTO{Phone: phone, OTP: otp, ClientID: "test-client", ProviderID: "test-provider"})

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "session limit exceeded")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// TestVerifyOTP – CreateSession error
// ---------------------------------------------------------------------------

func TestVerifyOTP_CreateSessionError(t *testing.T) {
	initTestJWTKeysService(t)
	phone := "+1234567890"
	otp := "123456"
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
			return &User{UserID: 1, UserUUID: uuid.New(), Phone: phone, Status: shared.StatusActive, Email: "test@example.com", Username: "testuser"}, nil
		},
	}
	smsOtpRepo := &mockSMSOtpRepo{
		findValidByPhoneFn: func(_ string) (*notifier.SMSOtp, error) {
			return &notifier.SMSOtp{SMSOtpID: 1, OTPHash: correctHash}, nil
		},
	}
	userIdentityRepo := &mockUserIdentityRepo{
		findByUserIDAndClientIDFn: func(_, _ int64) (*UserIdentity, error) {
			return &UserIdentity{Sub: "sub-123"}, nil
		},
	}
	sessionSvc := &mockSMSSessionService{
		createFn: func(_ context.Context, _ int64, _, _ string) (*UserToken, error) {
			return nil, errors.New("create session failed")
		},
	}

	svc := NewSMSLoginService(gormDB, userRepo, smsOtpRepo, clientRepo, userIdentityRepo, idpRepo, &mockAuthEventService{}, sessionSvc)
	resp, err := svc.VerifyOTP(context.Background(), SMSLoginVerifyDTO{Phone: phone, OTP: otp, ClientID: "test-client", ProviderID: "test-provider"})

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "create session failed")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// TestVerifyOTP – GenerateTokenSet error
// ---------------------------------------------------------------------------

func TestVerifyOTP_GenerateTokenSetError(t *testing.T) {
	jwt.ResetJWTKeys()
	defer initTestJWTKeysService(t)

	phone := "+1234567890"
	otp := "123456"
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
			return &User{UserID: 1, UserUUID: uuid.New(), Phone: phone, Status: shared.StatusActive, Email: "test@example.com", Username: "testuser"}, nil
		},
	}
	smsOtpRepo := &mockSMSOtpRepo{
		findValidByPhoneFn: func(_ string) (*notifier.SMSOtp, error) {
			return &notifier.SMSOtp{SMSOtpID: 1, OTPHash: correctHash}, nil
		},
	}
	userIdentityRepo := &mockUserIdentityRepo{
		findByUserIDAndClientIDFn: func(_, _ int64) (*UserIdentity, error) {
			return &UserIdentity{Sub: "sub-123"}, nil
		},
	}

	svc := NewSMSLoginService(gormDB, userRepo, smsOtpRepo, clientRepo, userIdentityRepo, idpRepo, &mockAuthEventService{})
	resp, err := svc.VerifyOTP(context.Background(), SMSLoginVerifyDTO{Phone: phone, OTP: otp, ClientID: "test-client", ProviderID: "test-provider"})

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.NoError(t, mock.ExpectationsWereMet())
}
