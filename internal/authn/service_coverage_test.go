package authn

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/branding"
	"github.com/maintainerd/maintainerd-auth/internal/notifier"
	jwtplatform "github.com/maintainerd/maintainerd-auth/internal/platform/jwt"
	"github.com/maintainerd/maintainerd-auth/internal/platform/security"
	"github.com/maintainerd/maintainerd-auth/internal/platform/signedurl"
	"github.com/maintainerd/maintainerd-auth/internal/platform/sms"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestTokenHelper_RemainingErrorBranches(t *testing.T) {
	initTestJWTKeysService(t)
	user := buildActiveUser(t, "Password123!")
	client := buildActiveClient()

	t.Run("id token generation error", func(t *testing.T) {
		orig := jwtGenIDToken
		jwtGenIDToken = func(context.Context, string, string, string, string, *jwtplatform.UserProfile, string, *jwtplatform.IDTokenParams) (string, error) {
			return "", errors.New("id token error")
		}
		t.Cleanup(func() { jwtGenIDToken = orig })

		access, id, refresh, err := generateTokenSetWithAuthContext(context.Background(), "sub", user, client, passwordAuthContext())

		require.Error(t, err)
		assert.Empty(t, access)
		assert.Empty(t, id)
		assert.Empty(t, refresh)
	})

	t.Run("refresh token generation error", func(t *testing.T) {
		orig := jwtGenRefreshToken
		jwtGenRefreshToken = func(context.Context, string, string, string, string) (string, error) {
			return "", errors.New("refresh token error")
		}
		t.Cleanup(func() { jwtGenRefreshToken = orig })

		access, id, refresh, err := generateTokenSetWithAuthContext(context.Background(), "sub", user, client, passwordAuthContext())

		require.Error(t, err)
		assert.Empty(t, access)
		assert.Empty(t, id)
		assert.Empty(t, refresh)
	})
}

func TestEmailVerificationService_RemainingBranches(t *testing.T) {
	t.Run("find existing token error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		userRepo := &mockUserRepo{findByEmailFn: func(string) (*User, error) {
			return &User{UserID: 1, Email: "user@example.com", Status: shared.StatusActive}, nil
		}}
		tokenRepo := &mockUserTokenRepo{
			findByUserIDAndTokenTypeFn: func(int64, string) ([]UserToken, error) {
				return nil, errors.New("find tokens error")
			},
		}
		clientRepo := &mockClientRepo{findSystemFn: func() (*Client, error) { return buildActiveClient(), nil }}
		svc := NewEmailVerificationService(db, userRepo, tokenRepo, clientRepo, &mockEmailTemplateRepo{}, nil, nil)

		resp, err := svc.SendVerificationEmail(context.Background(), "user@example.com", nil, nil)

		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "existing tokens")
	})

	t.Run("revoke existing token error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		tokenID := uuid.New()
		userRepo := &mockUserRepo{findByEmailFn: func(string) (*User, error) {
			return &User{UserID: 1, Email: "user@example.com", Status: shared.StatusActive}, nil
		}}
		tokenRepo := &mockUserTokenRepo{
			findByUserIDAndTokenTypeFn: func(int64, string) ([]UserToken, error) {
				return []UserToken{{UserTokenUUID: tokenID}}, nil
			},
			revokeByUUIDFn: func(uuid.UUID) error { return errors.New("revoke error") },
		}
		clientRepo := &mockClientRepo{findSystemFn: func() (*Client, error) { return buildActiveClient(), nil }}
		svc := NewEmailVerificationService(db, userRepo, tokenRepo, clientRepo, &mockEmailTemplateRepo{}, nil, nil)

		resp, err := svc.SendVerificationEmail(context.Background(), "user@example.com", nil, nil)

		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "revoke")
	})

	t.Run("otp generation error", func(t *testing.T) {
		orig := generateEmailVerificationOTP
		generateEmailVerificationOTP = func(int) (string, error) { return "", errors.New("otp error") }
		t.Cleanup(func() { generateEmailVerificationOTP = orig })

		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		userRepo := &mockUserRepo{findByEmailFn: func(string) (*User, error) {
			return &User{UserID: 1, Email: "user@example.com", Status: shared.StatusActive}, nil
		}}
		tokenRepo := &mockUserTokenRepo{findByUserIDAndTokenTypeFn: func(int64, string) ([]UserToken, error) {
			return nil, nil
		}}
		clientRepo := &mockClientRepo{findSystemFn: func() (*Client, error) { return buildActiveClient(), nil }}
		svc := NewEmailVerificationService(db, userRepo, tokenRepo, clientRepo, &mockEmailTemplateRepo{}, nil, nil)

		resp, err := svc.SendVerificationEmail(context.Background(), "user@example.com", nil, nil)

		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "verification code")
	})
}

func TestMagicLinkService_RemainingBranches(t *testing.T) {
	t.Run("client lookup error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		clientRepo := &mockClientRepo{findSystemFn: func() (*Client, error) { return nil, errors.New("client error") }}
		svc := NewMagicLinkService(db, &mockUserRepo{}, &mockUserTokenRepo{}, clientRepo, &mockUserIdentityRepo{}, &mockIdentityProviderRepo{}, &mockEmailTemplateRepo{})

		resp, err := svc.SendMagicLink(context.Background(), "user@example.com", nil, nil, true)

		require.Error(t, err)
		assert.Nil(t, resp)
	})

	t.Run("signed url generation error", func(t *testing.T) {
		orig := signedurl.GenerateSignedURL
		signedurl.GenerateSignedURL = func(string, map[string]string, time.Duration) (string, error) {
			return "", errors.New("signed url error")
		}
		t.Cleanup(func() { signedurl.GenerateSignedURL = orig })

		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		userRepo := &mockUserRepo{findByEmailFn: func(string) (*User, error) {
			return &User{UserID: 1, Email: "user@example.com", Status: shared.StatusActive}, nil
		}}
		tokenRepo := &mockUserTokenRepo{
			findByUserIDAndTokenTypeFn: func(int64, string) ([]UserToken, error) { return nil, nil },
			createFn: func(t *UserToken) (*UserToken, error) {
				t.Token = "token"
				return t, nil
			},
		}
		clientRepo := &mockClientRepo{findSystemFn: func() (*Client, error) { return buildActiveClient(), nil }}
		templateRepo := &mockEmailTemplateRepo{findByNameFn: func(string) (*branding.EmailTemplate, error) {
			return &branding.EmailTemplate{Subject: "Magic", BodyHTML: `{{.MagicLinkURL}}`}, nil
		}}
		svc := NewMagicLinkService(db, userRepo, tokenRepo, clientRepo, &mockUserIdentityRepo{}, &mockIdentityProviderRepo{}, templateRepo)

		resp, err := svc.SendMagicLink(context.Background(), "user@example.com", nil, nil, true)

		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.True(t, resp.Success)
	})

	t.Run("frontend url conversion error", func(t *testing.T) {
		orig := signedurl.ConvertToFrontendURL
		signedurl.ConvertToFrontendURL = func(string, string) (string, error) {
			return "", errors.New("frontend url error")
		}
		t.Cleanup(func() { signedurl.ConvertToFrontendURL = orig })

		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		userRepo := &mockUserRepo{findByEmailFn: func(string) (*User, error) {
			return &User{UserID: 1, Email: "user@example.com", Status: shared.StatusActive}, nil
		}}
		tokenRepo := &mockUserTokenRepo{
			findByUserIDAndTokenTypeFn: func(int64, string) ([]UserToken, error) { return nil, nil },
			createFn: func(t *UserToken) (*UserToken, error) {
				t.Token = "token"
				return t, nil
			},
		}
		clientRepo := &mockClientRepo{findSystemFn: func() (*Client, error) { return buildActiveClient(), nil }}
		templateRepo := &mockEmailTemplateRepo{findByNameFn: func(string) (*branding.EmailTemplate, error) {
			return &branding.EmailTemplate{Subject: "Magic", BodyHTML: `{{.MagicLinkURL}}`}, nil
		}}
		svc := NewMagicLinkService(db, userRepo, tokenRepo, clientRepo, &mockUserIdentityRepo{}, &mockIdentityProviderRepo{}, templateRepo)

		resp, err := svc.SendMagicLink(context.Background(), "user@example.com", nil, nil, true)

		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.True(t, resp.Success)
	})
}

func TestSMSLoginService_RemainingBranches(t *testing.T) {
	t.Run("sms provider init error is non fatal", func(t *testing.T) {
		origFactory := newSMSProvider
		newSMSProvider = func(ctx context.Context, db *gorm.DB, tenantID int64) (sms.Provider, error) {
			return nil, errors.New("provider init error")
		}
		t.Cleanup(func() { newSMSProvider = origFactory })

		userRepo := &mockUserRepo{findByPhoneFn: func(string) (*User, error) {
			return &User{UserID: 1, Phone: "+1234567890", Status: shared.StatusActive}, nil
		}}
		otpRepo := &mockSMSOtpRepo{createFn: func(otp *notifier.UserOTP) (*notifier.UserOTP, error) {
			return otp, nil
		}}
		svc := NewSMSLoginService(nil, userRepo, otpRepo, &mockClientRepo{}, &mockUserIdentityRepo{}, &mockIdentityProviderRepo{}, nil)

		err := svc.SendOTP(context.Background(), "+1234567890", nil, nil)

		require.NoError(t, err)
	})

	t.Run("sms provider send error is non fatal", func(t *testing.T) {
		origFactory := newSMSProvider
		newSMSProvider = func(ctx context.Context, db *gorm.DB, tenantID int64) (sms.Provider, error) {
			return failingSMSProvider{}, nil
		}
		t.Cleanup(func() {
			newSMSProvider = origFactory
		})

		userRepo := &mockUserRepo{findByPhoneFn: func(string) (*User, error) {
			return &User{UserID: 1, Phone: "+1234567890", Status: shared.StatusActive}, nil
		}}
		otpRepo := &mockSMSOtpRepo{createFn: func(otp *notifier.UserOTP) (*notifier.UserOTP, error) {
			return otp, nil
		}}
		svc := NewSMSLoginService(nil, userRepo, otpRepo, &mockClientRepo{}, &mockUserIdentityRepo{}, &mockIdentityProviderRepo{}, nil)

		err := svc.SendOTP(context.Background(), "+1234567890", nil, nil)

		require.NoError(t, err)
	})

	t.Run("otp generation error", func(t *testing.T) {
		orig := generateSMSOTP
		generateSMSOTP = func(int) (string, error) { return "", errors.New("otp error") }
		t.Cleanup(func() { generateSMSOTP = orig })

		coverageClientID := "test-client"
		userRepo := &mockUserRepo{findByPhoneFn: func(string) (*User, error) {
			return &User{UserID: 1, Phone: "+1234567890", Status: shared.StatusActive}, nil
		}}
		clientRepo := &mockClientRepo{findByClientIDAndIdentityProviderFn: func(string, string) (*Client, error) {
			return &Client{TenantID: 1}, nil
		}}
		svc := NewSMSLoginService(nil, userRepo, &mockSMSOtpRepo{}, clientRepo, &mockUserIdentityRepo{}, &mockIdentityProviderRepo{}, nil)

		err := svc.SendOTP(context.Background(), "+1234567890", &coverageClientID, nil)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "OTP")
	})

	t.Run("verify otp lookup error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		domain := "example.com"
		clientID := "client"
		clientRepo := &mockClientRepo{findByClientIDAndIdentityProviderFn: func(string, string) (*Client, error) {
			return &Client{ClientID: 1, Status: shared.StatusActive, Domain: &domain}, nil
		}}
		idpRepo := &mockIdentityProviderRepo{findByIdentifierFn: func(string) (*IdentityProvider, error) {
			return &IdentityProvider{IdentityProviderID: 1}, nil
		}}
		userRepo := &mockUserRepo{findByPhoneFn: func(string) (*User, error) {
			return &User{UserID: 1, Phone: "+1234567890", Status: shared.StatusActive}, nil
		}}
		otpRepo := &mockSMSOtpRepo{findValidByPhoneFn: func(string) (*notifier.UserOTP, error) {
			return nil, errors.New("otp lookup error")
		}}
		svc := NewSMSLoginService(db, userRepo, otpRepo, clientRepo, &mockUserIdentityRepo{}, idpRepo, nil)

		resp, err := svc.VerifyOTP(context.Background(), "+1234567890", "123456", &clientID, nil)

		require.Error(t, err)
		assert.Nil(t, resp)
	})
}

type failingSMSProvider struct{}

func (failingSMSProvider) Send(context.Context, string, string) error {
	return errors.New("send error")
}

func TestResetPasswordService_RemainingBranches(t *testing.T) {
	t.Run("hash password error", func(t *testing.T) {
		orig := resetHashPasswordWithPolicy
		resetHashPasswordWithPolicy = func(context.Context, []byte, security.PasswordPolicy) ([]byte, error) {
			return nil, errors.New("hash error")
		}
		t.Cleanup(func() { resetHashPasswordWithPolicy = orig })

		tok := "some-token"
		userID := int64(42)
		tokenUUID := uuid.New()
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT \* FROM "user_tokens"`).
			WillReturnRows(validTokenRow(tok, userID, tokenUUID))
		mock.ExpectRollback()
		userRepo := &mockUserRepo{findByIDFn: func(any, ...string) (*User, error) {
			return &User{UserID: userID, Status: shared.StatusActive}, nil
		}}
		svc := NewResetPasswordService(db, userRepo, &mockUserTokenRepo{}, &mockClientRepo{
			findSystemFn: func() (*Client, error) { return &Client{ClientID: 1}, nil },
		}, nil, nil, nil)

		resp, err := svc.ResetPassword(context.Background(), tok, strongPassword, nil, nil)

		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "hash")
	})
}

func TestRegisterService_RemainingBranches(t *testing.T) {
	cid, pid := "client", "provider"
	t.Run("register public username lookup error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		m := defaultRegPublicMocks()
		m.user.findByUsernameFn = func(string) (*User, error) { return nil, errors.New("username error") }
		svc := NewRegistrationService(db, m.client, m.user, m.userRole, m.userToken, m.userIdentity, m.role, m.invite, m.idp, nil, nil, nil)

		resp, err := svc.RegisterPublic(context.Background(), "u", "User", "P@ssW0rd!", nil, nil, &cid, &pid, "")

		require.Error(t, err)
		assert.Nil(t, resp)
	})

	t.Run("register public email lookup error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		email := "user@example.com"
		m := defaultRegPublicMocks()
		m.user.findByEmailFn = func(string) (*User, error) { return nil, errors.New("email error") }
		svc := NewRegistrationService(db, m.client, m.user, m.userRole, m.userToken, m.userIdentity, m.role, m.invite, m.idp, nil, nil, nil)

		resp, err := svc.RegisterPublic(context.Background(), "u", "User", "P@ssW0rd!", &email, nil, &cid, &pid, "")

		require.Error(t, err)
		assert.Nil(t, resp)
	})

	t.Run("register public phone lookup error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		phone := "+1234567890"
		m := defaultRegPublicMocks()
		m.user.findByPhoneFn = func(string) (*User, error) { return nil, errors.New("phone error") }
		svc := NewRegistrationService(db, m.client, m.user, m.userRole, m.userToken, m.userIdentity, m.role, m.invite, m.idp, nil, nil, nil)

		resp, err := svc.RegisterPublic(context.Background(), "u", "User", "P@ssW0rd!", nil, &phone, &cid, &pid, "")

		require.Error(t, err)
		assert.Nil(t, resp)
	})

	for _, tc := range []struct {
		name string
		run  func(RegisterService) (*RegisterResponseDTO, error)
		m    func() *regMocks
	}{
		{
			name: "register public hash error",
			m:    defaultRegPublicMocks,
			run: func(s RegisterService) (*RegisterResponseDTO, error) {
				return s.RegisterPublic(context.Background(), "u", "User", "P@ssW0rd!", nil, nil, &cid, &pid, "")
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			orig := secHashPassword
			secHashPassword = func(context.Context, []byte) ([]byte, error) { return nil, errors.New("hash error") }
			t.Cleanup(func() { secHashPassword = orig })
			db, mock := newMockGormDB(t)
			mock.ExpectBegin()
			mock.ExpectRollback()
			m := tc.m()
			svc := NewRegistrationService(db, m.client, m.user, m.userRole, m.userToken, m.userIdentity, m.role, m.invite, m.idp, nil, nil, nil)

			resp, err := tc.run(svc)

			require.Error(t, err)
			assert.Nil(t, resp)
		})
	}

	// Retargeted from the deleted internal Register onto RegisterPublic: the
	// password-policy check is shared code, and this was its only coverage.
	t.Run("register password policy error", func(t *testing.T) {
		orig := secValidatePasswordPolicy
		secValidatePasswordPolicy = func(context.Context, string, security.PasswordPolicy) error {
			return errors.New("policy error")
		}
		t.Cleanup(func() { secValidatePasswordPolicy = orig })

		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		m := defaultRegPublicMocks()
		svc := NewRegistrationService(db, m.client, m.user, m.userRole, m.userToken, m.userIdentity, m.role, m.invite, m.idp, nil, nil, nil)

		cidPolicy := "client"
		pidPolicy := "provider"
		resp, err := svc.RegisterPublic(context.Background(), "u", "User", "P@ssW0rd!", nil, nil, &cidPolicy, &pidPolicy, "")

		require.Error(t, err)
		assert.Nil(t, resp)
	})

	validInvite := func() *Invite {
		future := time.Now().Add(time.Hour)
		return &Invite{
			InviteUUID:   uuid.New(),
			InvitedEmail: "invite@example.com",
			Status:       shared.StatusPending,
			ExpiresAt:    &future,
		}
	}

	for _, tc := range []struct {
		name string
		m    func() *regMocks
		run  func(RegisterService) (*RegisterResponseDTO, error)
	}{
		{
			m: defaultRegPublicMocks,
			run: func(s RegisterService) (*RegisterResponseDTO, error) {
				return s.RegisterInvitePublic(context.Background(), "u", "P@ssW0rd!", "client", "provider", "token")
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			orig := secHashPassword
			secHashPassword = func(context.Context, []byte) ([]byte, error) { return nil, errors.New("hash error") }
			t.Cleanup(func() { secHashPassword = orig })

			db, mock := newMockGormDB(t)
			mock.ExpectBegin()
			mock.ExpectRollback()
			m := tc.m()
			m.invite.findByTokenFn = func(string) (*Invite, error) { return validInvite(), nil }
			m.user.findByUsernameFn = func(string) (*User, error) { return nil, nil }
			m.user.findByEmailFn = func(string) (*User, error) { return nil, nil }
			svc := NewRegistrationService(db, m.client, m.user, m.userRole, m.userToken, m.userIdentity, m.role, m.invite, m.idp, nil, nil, nil)

			resp, err := tc.run(svc)

			require.Error(t, err)
			assert.Nil(t, resp)
		})
	}
}
