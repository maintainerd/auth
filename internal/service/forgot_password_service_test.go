package service

import (
	"errors"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/config"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestForgotPasswordService_SendPasswordResetEmail(t *testing.T) {
	cases := []struct {
		name         string
		setupClient  func(*mockClientRepo)
		expectCommit bool
		wantErr      bool
	}{
		{
			name: "client findDefault error - returns error",
			setupClient: func(c *mockClientRepo) {
				c.findDefaultFn = func() (*model.Client, error) { return nil, errors.New("db error") }
			},
			expectCommit: false,
			wantErr:      true,
		},
		{
			name: "user not found - returns success (security masking)",
			setupClient: func(c *mockClientRepo) {
				c.findDefaultFn = func() (*model.Client, error) { return buildActiveClient(), nil }
			},
			expectCommit: true,
			wantErr:      false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gormDB, mock := newMockGormDB(t)
			mock.ExpectBegin()
			if tc.expectCommit {
				mock.ExpectCommit()
			} else {
				mock.ExpectRollback()
			}

			clientRepo := &mockClientRepo{}
			tc.setupClient(clientRepo)

			svc := NewForgotPasswordService(gormDB, &mockUserRepo{}, &mockUserTokenRepo{}, clientRepo, &mockEmailTemplateRepo{})
			resp, err := svc.SendPasswordResetEmail("user@example.com", nil, nil, false)

			if tc.wantErr {
				require.Error(t, err)
				assert.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
				assert.True(t, resp.Success)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestForgotPasswordService_SendPasswordResetEmail_ClientIDAndProviderID(t *testing.T) {
	gormDB, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectCommit()

	clientID := "my-client"
	providerID := "my-provider"
	clientRepo := &mockClientRepo{
		findByClientIDAndIdentityProviderFn: func(cid, pid string) (*model.Client, error) {
			assert.Equal(t, clientID, cid)
			assert.Equal(t, providerID, pid)
			return buildActiveClient(), nil
		},
	}

	svc := NewForgotPasswordService(gormDB, &mockUserRepo{}, &mockUserTokenRepo{}, clientRepo, &mockEmailTemplateRepo{})
	resp, err := svc.SendPasswordResetEmail("user@example.com", &clientID, &providerID, false)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, resp.Success)
}

func TestForgotPasswordService_SendPasswordResetEmail_FindByEmailError(t *testing.T) {
	gormDB, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectCommit()

	clientRepo := &mockClientRepo{
		findDefaultFn: func() (*model.Client, error) { return buildActiveClient(), nil },
	}
	userRepo := &mockUserRepo{
		findByEmailFn: func(_ string) (*model.User, error) { return nil, errors.New("db err") },
	}

	svc := NewForgotPasswordService(gormDB, userRepo, &mockUserTokenRepo{}, clientRepo, &mockEmailTemplateRepo{})
	resp, err := svc.SendPasswordResetEmail("user@example.com", nil, nil, false)
	// FindByEmail error returns nil (security masking), user stays nil so no email sent
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, resp.Success)
}

func TestForgotPasswordService_SendPasswordResetEmail_UserInactive(t *testing.T) {
	// Inactive user: transaction succeeds, but user var is set, so sendPasswordResetEmail is called
	os.Setenv("HMAC_SECRET_KEY", "test-secret-key-for-hmac")
	defer os.Unsetenv("HMAC_SECRET_KEY")

	origAppPublicHostname := config.AppPublicHostname
	origAuthHostname := config.AuthHostname
	origEmailLogo := config.EmailLogo
	defer func() {
		config.AppPublicHostname = origAppPublicHostname
		config.AuthHostname = origAuthHostname
		config.EmailLogo = origEmailLogo
	}()
	config.AppPublicHostname = "https://api.example.com"
	config.AuthHostname = "https://auth.example.com"
	config.EmailLogo = "https://example.com/logo.png"

	origSendEmail := util.SendEmail
	defer func() { util.SendEmail = origSendEmail }()
	util.SendEmail = func(_ util.SendEmailParams) error { return nil }

	gormDB, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectCommit()

	clientRepo := &mockClientRepo{
		findDefaultFn: func() (*model.Client, error) { return buildActiveClient(), nil },
	}
	userRepo := &mockUserRepo{
		findByEmailFn: func(_ string) (*model.User, error) {
			return &model.User{UserID: 1, UserUUID: uuid.New(), Email: "user@example.com", Status: model.StatusInactive}, nil
		},
	}
	emailTemplateRepo := &mockEmailTemplateRepo{
		findByNameFn: func(_ string) (*model.EmailTemplate, error) {
			return &model.EmailTemplate{Subject: "Reset", BodyHTML: `<a href="{{.ResetURL}}">R</a>`}, nil
		},
	}

	svc := NewForgotPasswordService(gormDB, userRepo, &mockUserTokenRepo{}, clientRepo, emailTemplateRepo)
	resp, err := svc.SendPasswordResetEmail("user@example.com", nil, nil, true)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, resp.Success)
}

func TestForgotPasswordService_SendPasswordResetEmail_FindTokensError(t *testing.T) {
	gormDB, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectRollback()

	clientRepo := &mockClientRepo{
		findDefaultFn: func() (*model.Client, error) { return buildActiveClient(), nil },
	}
	userRepo := &mockUserRepo{
		findByEmailFn: func(_ string) (*model.User, error) {
			return &model.User{UserID: 1, Email: "user@example.com", Status: model.StatusActive}, nil
		},
	}
	tokenRepo := &mockUserTokenRepo{
		findByUserIDAndTokenTypeFn: func(_ int64, _ string) ([]model.UserToken, error) {
			return nil, errors.New("token db err")
		},
	}

	svc := NewForgotPasswordService(gormDB, userRepo, tokenRepo, clientRepo, &mockEmailTemplateRepo{})
	_, err := svc.SendPasswordResetEmail("user@example.com", nil, nil, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "find existing tokens")
}

func TestForgotPasswordService_SendPasswordResetEmail_RevokeTokenError(t *testing.T) {
	gormDB, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectRollback()

	tokenUUID := uuid.New()
	clientRepo := &mockClientRepo{
		findDefaultFn: func() (*model.Client, error) { return buildActiveClient(), nil },
	}
	userRepo := &mockUserRepo{
		findByEmailFn: func(_ string) (*model.User, error) {
			return &model.User{UserID: 1, Email: "user@example.com", Status: model.StatusActive}, nil
		},
	}
	tokenRepo := &mockUserTokenRepo{
		findByUserIDAndTokenTypeFn: func(_ int64, _ string) ([]model.UserToken, error) {
			return []model.UserToken{{UserTokenUUID: tokenUUID}}, nil
		},
		revokeByUUIDFn: func(_ uuid.UUID) error { return errors.New("revoke err") },
	}

	svc := NewForgotPasswordService(gormDB, userRepo, tokenRepo, clientRepo, &mockEmailTemplateRepo{})
	_, err := svc.SendPasswordResetEmail("user@example.com", nil, nil, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "revoke existing token")
}

func TestForgotPasswordService_SendPasswordResetEmail_CreateTokenError(t *testing.T) {
	gormDB, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectRollback()

	clientRepo := &mockClientRepo{
		findDefaultFn: func() (*model.Client, error) { return buildActiveClient(), nil },
	}
	userRepo := &mockUserRepo{
		findByEmailFn: func(_ string) (*model.User, error) {
			return &model.User{UserID: 1, Email: "user@example.com", Status: model.StatusActive}, nil
		},
	}
	tokenRepo := &mockUserTokenRepo{
		findByUserIDAndTokenTypeFn: func(_ int64, _ string) ([]model.UserToken, error) {
			return nil, nil
		},
		createFn: func(_ *model.UserToken) (*model.UserToken, error) { return nil, errors.New("create err") },
	}

	svc := NewForgotPasswordService(gormDB, userRepo, tokenRepo, clientRepo, &mockEmailTemplateRepo{})
	_, err := svc.SendPasswordResetEmail("user@example.com", nil, nil, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "create reset token")
}

func TestForgotPasswordService_SendPasswordResetEmail_FullPath(t *testing.T) {
	// Full success path: user found, active, tokens revoked, new token created, email sent
	os.Setenv("HMAC_SECRET_KEY", "test-secret-key-for-hmac")
	defer os.Unsetenv("HMAC_SECRET_KEY")

	// Save original values and restore
	origAppPublicHostname := config.AppPublicHostname
	origAuthHostname := config.AuthHostname
	origAccountHostname := config.AccountHostname
	origEmailLogo := config.EmailLogo
	defer func() {
		config.AppPublicHostname = origAppPublicHostname
		config.AuthHostname = origAuthHostname
		config.AccountHostname = origAccountHostname
		config.EmailLogo = origEmailLogo
	}()
	config.AppPublicHostname = "https://api.example.com"
	config.AuthHostname = "https://auth.example.com"
	config.AccountHostname = "https://account.example.com"
	config.EmailLogo = "https://example.com/logo.png"

	// Mock SendEmail to capture the call
	origSendEmail := util.SendEmail
	defer func() { util.SendEmail = origSendEmail }()
	var emailSent bool
	util.SendEmail = func(p util.SendEmailParams) error {
		emailSent = true
		assert.Equal(t, "user@example.com", p.To)
		assert.Contains(t, p.BodyHTML, "https://auth.example.com/reset-password")
		return nil
	}

	gormDB, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectCommit()

	clientRepo := &mockClientRepo{
		findDefaultFn: func() (*model.Client, error) { return buildActiveClient(), nil },
	}
	userRepo := &mockUserRepo{
		findByEmailFn: func(_ string) (*model.User, error) {
			return &model.User{UserID: 1, UserUUID: uuid.New(), Email: "user@example.com", Status: model.StatusActive}, nil
		},
	}
	tokenRepo := &mockUserTokenRepo{
		findByUserIDAndTokenTypeFn: func(_ int64, _ string) ([]model.UserToken, error) {
			return nil, nil
		},
	}
	bodyPlain := "Reset: {{.ResetURL}}"
	emailTemplateRepo := &mockEmailTemplateRepo{
		findByNameFn: func(_ string) (*model.EmailTemplate, error) {
			return &model.EmailTemplate{
				Subject:   "Password Reset",
				BodyHTML:  `<a href="{{.ResetURL}}">Reset</a> <img src="{{.LogoURL}}"/>`,
				BodyPlain: &bodyPlain,
			}, nil
		},
	}

	svc := NewForgotPasswordService(gormDB, userRepo, tokenRepo, clientRepo, emailTemplateRepo)
	resp, err := svc.SendPasswordResetEmail("user@example.com", nil, nil, true)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, resp.Success)
	assert.True(t, emailSent)
}

func TestForgotPasswordService_SendPasswordResetEmail_ExternalURL(t *testing.T) {
	// Test the isInternal=false path → uses AccountHostname
	os.Setenv("HMAC_SECRET_KEY", "test-secret-key-for-hmac")
	defer os.Unsetenv("HMAC_SECRET_KEY")

	origAppPublicHostname := config.AppPublicHostname
	origAccountHostname := config.AccountHostname
	origEmailLogo := config.EmailLogo
	defer func() {
		config.AppPublicHostname = origAppPublicHostname
		config.AccountHostname = origAccountHostname
		config.EmailLogo = origEmailLogo
	}()
	config.AppPublicHostname = "https://api.example.com"
	config.AccountHostname = "https://account.example.com"
	config.EmailLogo = "https://example.com/logo.png"

	origSendEmail := util.SendEmail
	defer func() { util.SendEmail = origSendEmail }()
	util.SendEmail = func(p util.SendEmailParams) error {
		assert.Contains(t, p.BodyHTML, "https://account.example.com/reset-password")
		return nil
	}

	gormDB, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectCommit()

	clientRepo := &mockClientRepo{
		findDefaultFn: func() (*model.Client, error) { return buildActiveClient(), nil },
	}
	userRepo := &mockUserRepo{
		findByEmailFn: func(_ string) (*model.User, error) {
			return &model.User{UserID: 1, UserUUID: uuid.New(), Email: "user@example.com", Status: model.StatusActive}, nil
		},
	}
	tokenRepo := &mockUserTokenRepo{
		findByUserIDAndTokenTypeFn: func(_ int64, _ string) ([]model.UserToken, error) { return nil, nil },
	}
	emailTemplateRepo := &mockEmailTemplateRepo{
		findByNameFn: func(_ string) (*model.EmailTemplate, error) {
			return &model.EmailTemplate{
				Subject:  "Password Reset",
				BodyHTML: `<a href="{{.ResetURL}}">Reset</a>`,
			}, nil
		},
	}

	svc := NewForgotPasswordService(gormDB, userRepo, tokenRepo, clientRepo, emailTemplateRepo)
	resp, err := svc.SendPasswordResetEmail("user@example.com", nil, nil, false)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, resp.Success)
}

func TestForgotPasswordService_SendPasswordResetEmail_EmailSendError(t *testing.T) {
	// Email send failure is logged but doesn't cause error response
	os.Setenv("HMAC_SECRET_KEY", "test-secret-key-for-hmac")
	defer os.Unsetenv("HMAC_SECRET_KEY")

	origAppPublicHostname := config.AppPublicHostname
	origAuthHostname := config.AuthHostname
	origEmailLogo := config.EmailLogo
	defer func() {
		config.AppPublicHostname = origAppPublicHostname
		config.AuthHostname = origAuthHostname
		config.EmailLogo = origEmailLogo
	}()
	config.AppPublicHostname = "https://api.example.com"
	config.AuthHostname = "https://auth.example.com"
	config.EmailLogo = "https://example.com/logo.png"

	origSendEmail := util.SendEmail
	defer func() { util.SendEmail = origSendEmail }()
	util.SendEmail = func(_ util.SendEmailParams) error {
		return errors.New("smtp failure")
	}

	gormDB, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectCommit()

	clientRepo := &mockClientRepo{
		findDefaultFn: func() (*model.Client, error) { return buildActiveClient(), nil },
	}
	userRepo := &mockUserRepo{
		findByEmailFn: func(_ string) (*model.User, error) {
			return &model.User{UserID: 1, UserUUID: uuid.New(), Email: "user@example.com", Status: model.StatusActive}, nil
		},
	}
	tokenRepo := &mockUserTokenRepo{
		findByUserIDAndTokenTypeFn: func(_ int64, _ string) ([]model.UserToken, error) { return nil, nil },
	}
	emailTemplateRepo := &mockEmailTemplateRepo{
		findByNameFn: func(_ string) (*model.EmailTemplate, error) {
			return &model.EmailTemplate{
				Subject:  "Password Reset",
				BodyHTML: `<a href="{{.ResetURL}}">Reset</a>`,
			}, nil
		},
	}

	svc := NewForgotPasswordService(gormDB, userRepo, tokenRepo, clientRepo, emailTemplateRepo)
	resp, err := svc.SendPasswordResetEmail("user@example.com", nil, nil, true)
	require.NoError(t, err) // email failure is silently logged
	require.NotNil(t, resp)
	assert.True(t, resp.Success)
}

func TestForgotPasswordService_SendPasswordResetEmail_TemplateError(t *testing.T) {
	// Template fetch error from sendPasswordResetEmail is logged silently
	os.Setenv("HMAC_SECRET_KEY", "test-secret-key-for-hmac")
	defer os.Unsetenv("HMAC_SECRET_KEY")

	origAppPublicHostname := config.AppPublicHostname
	origAuthHostname := config.AuthHostname
	origEmailLogo := config.EmailLogo
	defer func() {
		config.AppPublicHostname = origAppPublicHostname
		config.AuthHostname = origAuthHostname
		config.EmailLogo = origEmailLogo
	}()
	config.AppPublicHostname = "https://api.example.com"
	config.AuthHostname = "https://auth.example.com"
	config.EmailLogo = "https://example.com/logo.png"

	origSendEmail := util.SendEmail
	defer func() { util.SendEmail = origSendEmail }()
	util.SendEmail = func(_ util.SendEmailParams) error { return nil }

	gormDB, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectCommit()

	clientRepo := &mockClientRepo{
		findDefaultFn: func() (*model.Client, error) { return buildActiveClient(), nil },
	}
	userRepo := &mockUserRepo{
		findByEmailFn: func(_ string) (*model.User, error) {
			return &model.User{UserID: 1, UserUUID: uuid.New(), Email: "user@example.com", Status: model.StatusActive}, nil
		},
	}
	tokenRepo := &mockUserTokenRepo{
		findByUserIDAndTokenTypeFn: func(_ int64, _ string) ([]model.UserToken, error) { return nil, nil },
	}
	emailTemplateRepo := &mockEmailTemplateRepo{
		findByNameFn: func(_ string) (*model.EmailTemplate, error) {
			return nil, errors.New("template not found")
		},
	}

	svc := NewForgotPasswordService(gormDB, userRepo, tokenRepo, clientRepo, emailTemplateRepo)
	resp, err := svc.SendPasswordResetEmail("user@example.com", nil, nil, true)
	require.NoError(t, err) // template error is logged silently
	require.NotNil(t, resp)
	assert.True(t, resp.Success)
}

func TestForgotPasswordService_SendPasswordResetEmail_HTMLParseError(t *testing.T) {
	os.Setenv("HMAC_SECRET_KEY", "test-secret-key-for-hmac")
	defer os.Unsetenv("HMAC_SECRET_KEY")

	origAppPublicHostname := config.AppPublicHostname
	origAuthHostname := config.AuthHostname
	origEmailLogo := config.EmailLogo
	defer func() {
		config.AppPublicHostname = origAppPublicHostname
		config.AuthHostname = origAuthHostname
		config.EmailLogo = origEmailLogo
	}()
	config.AppPublicHostname = "https://api.example.com"
	config.AuthHostname = "https://auth.example.com"
	config.EmailLogo = "https://example.com/logo.png"

	origSendEmail := util.SendEmail
	defer func() { util.SendEmail = origSendEmail }()

	gormDB, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectCommit()

	clientRepo := &mockClientRepo{
		findDefaultFn: func() (*model.Client, error) { return buildActiveClient(), nil },
	}
	userRepo := &mockUserRepo{
		findByEmailFn: func(_ string) (*model.User, error) {
			return &model.User{UserID: 1, UserUUID: uuid.New(), Email: "user@example.com", Status: model.StatusActive}, nil
		},
	}
	tokenRepo := &mockUserTokenRepo{
		findByUserIDAndTokenTypeFn: func(_ int64, _ string) ([]model.UserToken, error) { return nil, nil },
	}
	emailTemplateRepo := &mockEmailTemplateRepo{
		findByNameFn: func(_ string) (*model.EmailTemplate, error) {
			return &model.EmailTemplate{
				Subject:  "Reset",
				BodyHTML: `{{.InvalidSyntax`, // bad template
			}, nil
		},
	}

	svc := NewForgotPasswordService(gormDB, userRepo, tokenRepo, clientRepo, emailTemplateRepo)
	resp, err := svc.SendPasswordResetEmail("user@example.com", nil, nil, true)
	require.NoError(t, err) // error is logged silently
	require.NotNil(t, resp)
	assert.True(t, resp.Success)
}

func TestForgotPasswordService_SendPasswordResetEmail_PlainParseError(t *testing.T) {
	os.Setenv("HMAC_SECRET_KEY", "test-secret-key-for-hmac")
	defer os.Unsetenv("HMAC_SECRET_KEY")

	origAppPublicHostname := config.AppPublicHostname
	origAuthHostname := config.AuthHostname
	origEmailLogo := config.EmailLogo
	defer func() {
		config.AppPublicHostname = origAppPublicHostname
		config.AuthHostname = origAuthHostname
		config.EmailLogo = origEmailLogo
	}()
	config.AppPublicHostname = "https://api.example.com"
	config.AuthHostname = "https://auth.example.com"
	config.EmailLogo = "https://example.com/logo.png"

	origSendEmail := util.SendEmail
	defer func() { util.SendEmail = origSendEmail }()

	gormDB, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectCommit()

	clientRepo := &mockClientRepo{
		findDefaultFn: func() (*model.Client, error) { return buildActiveClient(), nil },
	}
	userRepo := &mockUserRepo{
		findByEmailFn: func(_ string) (*model.User, error) {
			return &model.User{UserID: 1, UserUUID: uuid.New(), Email: "user@example.com", Status: model.StatusActive}, nil
		},
	}
	tokenRepo := &mockUserTokenRepo{
		findByUserIDAndTokenTypeFn: func(_ int64, _ string) ([]model.UserToken, error) { return nil, nil },
	}
	badPlain := `{{.InvalidSyntax`
	emailTemplateRepo := &mockEmailTemplateRepo{
		findByNameFn: func(_ string) (*model.EmailTemplate, error) {
			return &model.EmailTemplate{
				Subject:   "Reset",
				BodyHTML:  `<a href="{{.ResetURL}}">Reset</a>`,
				BodyPlain: &badPlain,
			}, nil
		},
	}

	svc := NewForgotPasswordService(gormDB, userRepo, tokenRepo, clientRepo, emailTemplateRepo)
	resp, err := svc.SendPasswordResetEmail("user@example.com", nil, nil, true)
	require.NoError(t, err) // error is logged silently
	require.NotNil(t, resp)
	assert.True(t, resp.Success)
}

func TestForgotPasswordService_SendPasswordResetEmail_HTMLExecuteError(t *testing.T) {
	os.Setenv("HMAC_SECRET_KEY", "test-secret-key-for-hmac")
	defer os.Unsetenv("HMAC_SECRET_KEY")

	origAppPublicHostname := config.AppPublicHostname
	origAuthHostname := config.AuthHostname
	origEmailLogo := config.EmailLogo
	defer func() {
		config.AppPublicHostname = origAppPublicHostname
		config.AuthHostname = origAuthHostname
		config.EmailLogo = origEmailLogo
	}()
	config.AppPublicHostname = "https://api.example.com"
	config.AuthHostname = "https://auth.example.com"
	config.EmailLogo = "https://example.com/logo.png"

	origSendEmail := util.SendEmail
	defer func() { util.SendEmail = origSendEmail }()

	gormDB, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectCommit()

	clientRepo := &mockClientRepo{
		findDefaultFn: func() (*model.Client, error) { return buildActiveClient(), nil },
	}
	userRepo := &mockUserRepo{
		findByEmailFn: func(_ string) (*model.User, error) {
			return &model.User{UserID: 1, UserUUID: uuid.New(), Email: "user@example.com", Status: model.StatusActive}, nil
		},
	}
	tokenRepo := &mockUserTokenRepo{
		findByUserIDAndTokenTypeFn: func(_ int64, _ string) ([]model.UserToken, error) { return nil, nil },
	}
	emailTemplateRepo := &mockEmailTemplateRepo{
		findByNameFn: func(_ string) (*model.EmailTemplate, error) {
			return &model.EmailTemplate{
				Subject:  "Reset",
				BodyHTML: `{{call .ResetURL}}`, // parses ok, fails on Execute
			}, nil
		},
	}

	svc := NewForgotPasswordService(gormDB, userRepo, tokenRepo, clientRepo, emailTemplateRepo)
	resp, err := svc.SendPasswordResetEmail("user@example.com", nil, nil, true)
	require.NoError(t, err) // error is logged silently
	require.NotNil(t, resp)
	assert.True(t, resp.Success)
}

func TestForgotPasswordService_SendPasswordResetEmail_PlainExecuteError(t *testing.T) {
	os.Setenv("HMAC_SECRET_KEY", "test-secret-key-for-hmac")
	defer os.Unsetenv("HMAC_SECRET_KEY")

	origAppPublicHostname := config.AppPublicHostname
	origAuthHostname := config.AuthHostname
	origEmailLogo := config.EmailLogo
	defer func() {
		config.AppPublicHostname = origAppPublicHostname
		config.AuthHostname = origAuthHostname
		config.EmailLogo = origEmailLogo
	}()
	config.AppPublicHostname = "https://api.example.com"
	config.AuthHostname = "https://auth.example.com"
	config.EmailLogo = "https://example.com/logo.png"

	origSendEmail := util.SendEmail
	defer func() { util.SendEmail = origSendEmail }()

	gormDB, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectCommit()

	clientRepo := &mockClientRepo{
		findDefaultFn: func() (*model.Client, error) { return buildActiveClient(), nil },
	}
	userRepo := &mockUserRepo{
		findByEmailFn: func(_ string) (*model.User, error) {
			return &model.User{UserID: 1, UserUUID: uuid.New(), Email: "user@example.com", Status: model.StatusActive}, nil
		},
	}
	tokenRepo := &mockUserTokenRepo{
		findByUserIDAndTokenTypeFn: func(_ int64, _ string) ([]model.UserToken, error) { return nil, nil },
	}
	badPlain := `{{call .ResetURL}}` // parses ok, fails on Execute
	emailTemplateRepo := &mockEmailTemplateRepo{
		findByNameFn: func(_ string) (*model.EmailTemplate, error) {
			return &model.EmailTemplate{
				Subject:   "Reset",
				BodyHTML:  `<a href="{{.ResetURL}}">Reset</a>`,
				BodyPlain: &badPlain,
			}, nil
		},
	}

	svc := NewForgotPasswordService(gormDB, userRepo, tokenRepo, clientRepo, emailTemplateRepo)
	resp, err := svc.SendPasswordResetEmail("user@example.com", nil, nil, true)
	require.NoError(t, err) // error is logged silently
	require.NotNil(t, resp)
	assert.True(t, resp.Success)
}

func TestGenerateSecureToken(t *testing.T) {
	token := generateSecureToken(32)
	assert.Len(t, token, 64) // hex encoding doubles the length
	token2 := generateSecureToken(32)
	assert.NotEqual(t, token, token2)
}

func TestForgotPasswordService_SendPasswordResetEmail_WithExistingTokens(t *testing.T) {
	// Tests the path where existing tokens are found and revoked
	os.Setenv("HMAC_SECRET_KEY", "test-secret-key-for-hmac")
	defer os.Unsetenv("HMAC_SECRET_KEY")

	origAppPublicHostname := config.AppPublicHostname
	origAuthHostname := config.AuthHostname
	origEmailLogo := config.EmailLogo
	defer func() {
		config.AppPublicHostname = origAppPublicHostname
		config.AuthHostname = origAuthHostname
		config.EmailLogo = origEmailLogo
	}()
	config.AppPublicHostname = "https://api.example.com"
	config.AuthHostname = "https://auth.example.com"
	config.EmailLogo = "https://example.com/logo.png"

	origSendEmail := util.SendEmail
	defer func() { util.SendEmail = origSendEmail }()
	util.SendEmail = func(_ util.SendEmailParams) error { return nil }

	gormDB, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectCommit()

	token1UUID := uuid.New()
	token2UUID := uuid.New()
	var revokedUUIDs []uuid.UUID
	clientRepo := &mockClientRepo{
		findDefaultFn: func() (*model.Client, error) { return buildActiveClient(), nil },
	}
	userRepo := &mockUserRepo{
		findByEmailFn: func(_ string) (*model.User, error) {
			return &model.User{UserID: 1, UserUUID: uuid.New(), Email: "user@example.com", Status: model.StatusActive}, nil
		},
	}
	tokenRepo := &mockUserTokenRepo{
		findByUserIDAndTokenTypeFn: func(_ int64, _ string) ([]model.UserToken, error) {
			return []model.UserToken{
				{UserTokenUUID: token1UUID},
				{UserTokenUUID: token2UUID},
			}, nil
		},
		revokeByUUIDFn: func(id uuid.UUID) error {
			revokedUUIDs = append(revokedUUIDs, id)
			return nil
		},
	}
	emailTemplateRepo := &mockEmailTemplateRepo{
		findByNameFn: func(_ string) (*model.EmailTemplate, error) {
			return &model.EmailTemplate{
				Subject:  "Reset",
				BodyHTML: `<a href="{{.ResetURL}}">Reset</a>`,
			}, nil
		},
	}

	svc := NewForgotPasswordService(gormDB, userRepo, tokenRepo, clientRepo, emailTemplateRepo)
	resp, err := svc.SendPasswordResetEmail("user@example.com", nil, nil, true)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, resp.Success)
	assert.Equal(t, 2, len(revokedUUIDs))
	assert.Contains(t, revokedUUIDs, token1UUID)
	assert.Contains(t, revokedUUIDs, token2UUID)
}
