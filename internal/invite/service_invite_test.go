package invite

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/maintainerd/auth/internal/branding"
	"github.com/maintainerd/auth/internal/platform/config"
	"github.com/maintainerd/auth/internal/platform/crypto"
	"github.com/maintainerd/auth/internal/platform/email"
	"github.com/maintainerd/auth/internal/platform/signedurl"
	"github.com/maintainerd/auth/internal/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// defaultInviteClient returns a full client with domain, IDP and tenant set.
func defaultInviteClient() *Client {
	domain := "example.com"
	return &Client{
		ClientID: 1,
		Status:   shared.StatusActive,
		Domain:   &domain,
		IdentityProvider: &IdentityProvider{
			Identifier: "test-idp",
			Tenant:     &Tenant{TenantID: 10},
		},
	}
}

func TestInviteService_SendInvite(t *testing.T) {
	cases := []struct {
		name         string
		setupRepos   func(*mockClientRepo, *mockRoleRepo, *mockInviteRepo)
		expectCommit bool
		wantErr      bool
		wantErrMsg   string
	}{
		{
			name: "client findDefault error",
			setupRepos: func(c *mockClientRepo, r *mockRoleRepo, i *mockInviteRepo) {
				c.findSystemFn = func() (*Client, error) { return nil, errors.New("db error") }
			},
			expectCommit: false,
			wantErr:      true,
		},
		{
			name: "client is nil - invalid client",
			setupRepos: func(c *mockClientRepo, r *mockRoleRepo, i *mockInviteRepo) {
				c.findSystemFn = func() (*Client, error) { return nil, nil }
			},
			expectCommit: false,
			wantErr:      true,
			wantErrMsg:   "invalid client or identity provider",
		},
		{
			name: "active client with no identity provider - invalid",
			setupRepos: func(c *mockClientRepo, r *mockRoleRepo, i *mockInviteRepo) {
				// Client has no IdentityProvider set
				c.findSystemFn = func() (*Client, error) {
					return &Client{Status: shared.StatusActive}, nil
				}
			},
			expectCommit: false,
			wantErr:      true,
			wantErrMsg:   "invalid client or identity provider",
		},
		{
			name: "role findByUUIDs error",
			setupRepos: func(c *mockClientRepo, r *mockRoleRepo, i *mockInviteRepo) {
				c.findSystemFn = func() (*Client, error) { return defaultInviteClient(), nil }
				r.findByUUIDsFn = func(_ []string, _ ...string) ([]Role, error) {
					return nil, errors.New("db error")
				}
			},
			expectCommit: false,
			wantErr:      true,
		},
		{
			name: "role count mismatch - one or more roles not found",
			setupRepos: func(c *mockClientRepo, r *mockRoleRepo, i *mockInviteRepo) {
				c.findSystemFn = func() (*Client, error) { return defaultInviteClient(), nil }
				// Return fewer roles than requested
				r.findByUUIDsFn = func(_ []string, _ ...string) ([]Role, error) {
					return []Role{}, nil
				}
			},
			expectCommit: false,
			wantErr:      true,
			wantErrMsg:   "one or more roles not found",
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
			roleRepo := &mockRoleRepo{}
			inviteRepo := &mockInviteRepo{}
			tc.setupRepos(clientRepo, roleRepo, inviteRepo)

			svc := NewInviteService(gormDB, inviteRepo, clientRepo, roleRepo, &mockEmailTemplateRepo{})
			result, err := svc.SendInvite(context.Background(), 1, "user@example.com", 1, []string{"role-uuid-1"})

			if tc.wantErr {
				require.Error(t, err)
				assert.Nil(t, result)
				if tc.wantErrMsg != "" {
					assert.Contains(t, err.Error(), tc.wantErrMsg)
				}
			} else {
				require.NoError(t, err)
				assert.NotNil(t, result)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestInviteService_SendInvite_RoleTenantMismatch(t *testing.T) {
	gormDB, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectRollback()

	clientRepo := &mockClientRepo{
		findSystemFn: func() (*Client, error) { return defaultInviteClient(), nil },
	}
	roleRepo := &mockRoleRepo{
		findByUUIDsFn: func(_ []string, _ ...string) ([]Role, error) {
			return []Role{{RoleID: 1, TenantID: 999}}, nil // wrong tenant
		},
	}

	svc := NewInviteService(gormDB, &mockInviteRepo{}, clientRepo, roleRepo, &mockEmailTemplateRepo{})
	_, err := svc.SendInvite(context.Background(), 1, "user@example.com", 1, []string{"role-uuid-1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid role")
}

func TestInviteService_SendInvite_GenerateIdentifierFailure(t *testing.T) {
	orig := crypto.GenerateIdentifier
	defer func() { crypto.GenerateIdentifier = orig }()
	crypto.GenerateIdentifier = func(int) (string, error) { return "", errors.New("rand failure") }

	gormDB, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectRollback()

	clientRepo := &mockClientRepo{
		findSystemFn: func() (*Client, error) { return defaultInviteClient(), nil },
	}
	roleRepo := &mockRoleRepo{
		findByUUIDsFn: func(_ []string, _ ...string) ([]Role, error) {
			return []Role{{RoleID: 1, TenantID: 10}}, nil
		},
	}

	svc := NewInviteService(gormDB, &mockInviteRepo{}, clientRepo, roleRepo, &mockEmailTemplateRepo{})
	_, err := svc.SendInvite(context.Background(), 1, "user@example.com", 1, []string{"role-uuid-1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "rand failure")
}

func TestInviteService_SendInvite_InviteCreateError(t *testing.T) {
	gormDB, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectRollback()

	clientRepo := &mockClientRepo{
		findSystemFn: func() (*Client, error) { return defaultInviteClient(), nil },
	}
	roleRepo := &mockRoleRepo{
		findByUUIDsFn: func(_ []string, _ ...string) ([]Role, error) {
			return []Role{{RoleID: 1, TenantID: 10}}, nil
		},
	}
	inviteRepo := &mockInviteRepo{
		createFn: func(_ *Invite) (*Invite, error) { return nil, errors.New("create err") },
	}

	svc := NewInviteService(gormDB, inviteRepo, clientRepo, roleRepo, &mockEmailTemplateRepo{})
	_, err := svc.SendInvite(context.Background(), 1, "user@example.com", 1, []string{"role-uuid-1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "create err")
}

func TestInviteService_SendInvite_BulkRoleCreateError(t *testing.T) {
	gormDB, mock := newMockGormDB(t)
	mock.ExpectBegin()
	// tx.Create(&inviteRoles) will fail
	mock.ExpectQuery("INSERT INTO").WillReturnError(errors.New("bulk insert err"))
	mock.ExpectRollback()

	clientRepo := &mockClientRepo{
		findSystemFn: func() (*Client, error) { return defaultInviteClient(), nil },
	}
	roleRepo := &mockRoleRepo{
		findByUUIDsFn: func(_ []string, _ ...string) ([]Role, error) {
			return []Role{{RoleID: 1, TenantID: 10}}, nil
		},
	}

	svc := NewInviteService(gormDB, &mockInviteRepo{}, clientRepo, roleRepo, &mockEmailTemplateRepo{})
	_, err := svc.SendInvite(context.Background(), 1, "user@example.com", 1, []string{"role-uuid-1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bulk insert err")
}

func TestInviteService_SendInvite_FullSuccess(t *testing.T) {
	_ = os.Setenv("HMAC_SECRET_KEY", "test-secret-key-for-hmac")
	t.Cleanup(func() { _ = os.Unsetenv("HMAC_SECRET_KEY") })
	origAppPrivateHostname := config.AppPrivateHostname
	origAccountHostname := config.AccountHostname
	defer func() {
		config.AppPrivateHostname = origAppPrivateHostname
		config.AccountHostname = origAccountHostname
	}()
	config.AppPrivateHostname = "https://api.example.com"
	config.AccountHostname = "https://account.example.com"

	origSendEmail := email.SendEmail
	defer func() { email.SendEmail = origSendEmail }()
	var emailSent bool
	email.SendEmail = func(_ context.Context, _ *gorm.DB, p email.SendEmailParams) error {
		emailSent = true
		assert.Equal(t, "user@example.com", p.To)
		assert.Contains(t, p.BodyHTML, "https://account.example.com/register/invite")
		return nil
	}

	gormDB, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectQuery("INSERT INTO").WillReturnRows(sqlmock.NewRows([]string{"invite_role_id"}).AddRow(1))
	mock.ExpectCommit()

	clientRepo := &mockClientRepo{
		findSystemFn: func() (*Client, error) { return defaultInviteClient(), nil },
	}
	roleRepo := &mockRoleRepo{
		findByUUIDsFn: func(_ []string, _ ...string) ([]Role, error) {
			return []Role{{RoleID: 1, TenantID: 10}}, nil
		},
	}
	bodyPlain := "Join: {{.InviteURL}}"
	emailTemplateRepo := &mockEmailTemplateRepo{
		findByNameFn: func(_ string) (*branding.EmailTemplate, error) {
			return &branding.EmailTemplate{
				Subject:   "You're Invited",
				BodyHTML:  `<a href="{{.InviteURL}}">Accept</a>`,
				BodyPlain: &bodyPlain,
			}, nil
		},
	}

	svc := NewInviteService(gormDB, &mockInviteRepo{}, clientRepo, roleRepo, emailTemplateRepo)
	result, err := svc.SendInvite(context.Background(), 1, "user@example.com", 1, []string{"role-uuid-1"})
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, emailSent)
}

func TestInviteService_SendInvite_EmailSendError(t *testing.T) {
	_ = os.Setenv("HMAC_SECRET_KEY", "test-secret-key-for-hmac")
	t.Cleanup(func() { _ = os.Unsetenv("HMAC_SECRET_KEY") })
	origAppPrivateHostname := config.AppPrivateHostname
	origAccountHostname := config.AccountHostname
	defer func() {
		config.AppPrivateHostname = origAppPrivateHostname
		config.AccountHostname = origAccountHostname
	}()
	config.AppPrivateHostname = "https://api.example.com"
	config.AccountHostname = "https://account.example.com"

	origSendEmail := email.SendEmail
	defer func() { email.SendEmail = origSendEmail }()
	email.SendEmail = func(_ context.Context, _ *gorm.DB, _ email.SendEmailParams) error { return errors.New("smtp err") }

	gormDB, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectQuery("INSERT INTO").WillReturnRows(sqlmock.NewRows([]string{"invite_role_id"}).AddRow(1))
	mock.ExpectCommit()

	clientRepo := &mockClientRepo{
		findSystemFn: func() (*Client, error) { return defaultInviteClient(), nil },
	}
	roleRepo := &mockRoleRepo{
		findByUUIDsFn: func(_ []string, _ ...string) ([]Role, error) {
			return []Role{{RoleID: 1, TenantID: 10}}, nil
		},
	}
	emailTemplateRepo := &mockEmailTemplateRepo{
		findByNameFn: func(_ string) (*branding.EmailTemplate, error) {
			return &branding.EmailTemplate{
				Subject:  "Invite",
				BodyHTML: `<a href="{{.InviteURL}}">Accept</a>`,
			}, nil
		},
	}

	svc := NewInviteService(gormDB, &mockInviteRepo{}, clientRepo, roleRepo, emailTemplateRepo)
	_, err := svc.SendInvite(context.Background(), 1, "user@example.com", 1, []string{"role-uuid-1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "send invite email")
}

func TestInviteService_SendInvite_TemplateFetchError(t *testing.T) {
	_ = os.Setenv("HMAC_SECRET_KEY", "test-secret-key-for-hmac")
	t.Cleanup(func() { _ = os.Unsetenv("HMAC_SECRET_KEY") })
	origAppPrivateHostname := config.AppPrivateHostname
	origAccountHostname := config.AccountHostname
	defer func() {
		config.AppPrivateHostname = origAppPrivateHostname
		config.AccountHostname = origAccountHostname
	}()
	config.AppPrivateHostname = "https://api.example.com"
	config.AccountHostname = "https://account.example.com"

	origSendEmail := email.SendEmail
	defer func() { email.SendEmail = origSendEmail }()

	gormDB, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectQuery("INSERT INTO").WillReturnRows(sqlmock.NewRows([]string{"invite_role_id"}).AddRow(1))
	mock.ExpectCommit()

	clientRepo := &mockClientRepo{
		findSystemFn: func() (*Client, error) { return defaultInviteClient(), nil },
	}
	roleRepo := &mockRoleRepo{
		findByUUIDsFn: func(_ []string, _ ...string) ([]Role, error) {
			return []Role{{RoleID: 1, TenantID: 10}}, nil
		},
	}
	emailTemplateRepo := &mockEmailTemplateRepo{
		findByNameFn: func(_ string) (*branding.EmailTemplate, error) {
			return nil, errors.New("template not found")
		},
	}

	svc := NewInviteService(gormDB, &mockInviteRepo{}, clientRepo, roleRepo, emailTemplateRepo)
	_, err := svc.SendInvite(context.Background(), 1, "user@example.com", 1, []string{"role-uuid-1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "send invite email")
}

func TestInviteService_SendInvite_HTMLParseError(t *testing.T) {
	_ = os.Setenv("HMAC_SECRET_KEY", "test-secret-key-for-hmac")
	t.Cleanup(func() { _ = os.Unsetenv("HMAC_SECRET_KEY") })
	origAppPrivateHostname := config.AppPrivateHostname
	origAccountHostname := config.AccountHostname
	defer func() {
		config.AppPrivateHostname = origAppPrivateHostname
		config.AccountHostname = origAccountHostname
	}()
	config.AppPrivateHostname = "https://api.example.com"
	config.AccountHostname = "https://account.example.com"

	origSendEmail := email.SendEmail
	defer func() { email.SendEmail = origSendEmail }()

	gormDB, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectQuery("INSERT INTO").WillReturnRows(sqlmock.NewRows([]string{"invite_role_id"}).AddRow(1))
	mock.ExpectCommit()

	clientRepo := &mockClientRepo{
		findSystemFn: func() (*Client, error) { return defaultInviteClient(), nil },
	}
	roleRepo := &mockRoleRepo{
		findByUUIDsFn: func(_ []string, _ ...string) ([]Role, error) {
			return []Role{{RoleID: 1, TenantID: 10}}, nil
		},
	}
	emailTemplateRepo := &mockEmailTemplateRepo{
		findByNameFn: func(_ string) (*branding.EmailTemplate, error) {
			return &branding.EmailTemplate{
				Subject:  "Invite",
				BodyHTML: `{{.InvalidSyntax`, // bad template
			}, nil
		},
	}

	svc := NewInviteService(gormDB, &mockInviteRepo{}, clientRepo, roleRepo, emailTemplateRepo)
	_, err := svc.SendInvite(context.Background(), 1, "user@example.com", 1, []string{"role-uuid-1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "send invite email")
}

func TestInviteService_SendInvite_HTMLExecuteError(t *testing.T) {
	_ = os.Setenv("HMAC_SECRET_KEY", "test-secret-key-for-hmac")
	t.Cleanup(func() { _ = os.Unsetenv("HMAC_SECRET_KEY") })
	origAppPrivateHostname := config.AppPrivateHostname
	origAccountHostname := config.AccountHostname
	defer func() {
		config.AppPrivateHostname = origAppPrivateHostname
		config.AccountHostname = origAccountHostname
	}()
	config.AppPrivateHostname = "https://api.example.com"
	config.AccountHostname = "https://account.example.com"

	origSendEmail := email.SendEmail
	defer func() { email.SendEmail = origSendEmail }()

	gormDB, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectQuery("INSERT INTO").WillReturnRows(sqlmock.NewRows([]string{"invite_role_id"}).AddRow(1))
	mock.ExpectCommit()

	clientRepo := &mockClientRepo{
		findSystemFn: func() (*Client, error) { return defaultInviteClient(), nil },
	}
	roleRepo := &mockRoleRepo{
		findByUUIDsFn: func(_ []string, _ ...string) ([]Role, error) {
			return []Role{{RoleID: 1, TenantID: 10}}, nil
		},
	}
	emailTemplateRepo := &mockEmailTemplateRepo{
		findByNameFn: func(_ string) (*branding.EmailTemplate, error) {
			return &branding.EmailTemplate{
				Subject:  "Invite",
				BodyHTML: `{{call .InviteURL}}`, // parses ok, fails on Execute
			}, nil
		},
	}

	svc := NewInviteService(gormDB, &mockInviteRepo{}, clientRepo, roleRepo, emailTemplateRepo)
	_, err := svc.SendInvite(context.Background(), 1, "user@example.com", 1, []string{"role-uuid-1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "send invite email")
}

func TestInviteService_SendInvite_PlainParseError(t *testing.T) {
	_ = os.Setenv("HMAC_SECRET_KEY", "test-secret-key-for-hmac")
	t.Cleanup(func() { _ = os.Unsetenv("HMAC_SECRET_KEY") })
	origAppPrivateHostname := config.AppPrivateHostname
	origAccountHostname := config.AccountHostname
	defer func() {
		config.AppPrivateHostname = origAppPrivateHostname
		config.AccountHostname = origAccountHostname
	}()
	config.AppPrivateHostname = "https://api.example.com"
	config.AccountHostname = "https://account.example.com"

	origSendEmail := email.SendEmail
	defer func() { email.SendEmail = origSendEmail }()

	gormDB, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectQuery("INSERT INTO").WillReturnRows(sqlmock.NewRows([]string{"invite_role_id"}).AddRow(1))
	mock.ExpectCommit()

	clientRepo := &mockClientRepo{
		findSystemFn: func() (*Client, error) { return defaultInviteClient(), nil },
	}
	roleRepo := &mockRoleRepo{
		findByUUIDsFn: func(_ []string, _ ...string) ([]Role, error) {
			return []Role{{RoleID: 1, TenantID: 10}}, nil
		},
	}
	badPlain := `{{.InvalidSyntax`
	emailTemplateRepo := &mockEmailTemplateRepo{
		findByNameFn: func(_ string) (*branding.EmailTemplate, error) {
			return &branding.EmailTemplate{
				Subject:   "Invite",
				BodyHTML:  `<a href="{{.InviteURL}}">Accept</a>`,
				BodyPlain: &badPlain,
			}, nil
		},
	}

	svc := NewInviteService(gormDB, &mockInviteRepo{}, clientRepo, roleRepo, emailTemplateRepo)
	_, err := svc.SendInvite(context.Background(), 1, "user@example.com", 1, []string{"role-uuid-1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "send invite email")
}

func TestInviteService_SendInvite_PlainExecuteError(t *testing.T) {
	_ = os.Setenv("HMAC_SECRET_KEY", "test-secret-key-for-hmac")
	t.Cleanup(func() { _ = os.Unsetenv("HMAC_SECRET_KEY") })
	origAppPrivateHostname := config.AppPrivateHostname
	origAccountHostname := config.AccountHostname
	defer func() {
		config.AppPrivateHostname = origAppPrivateHostname
		config.AccountHostname = origAccountHostname
	}()
	config.AppPrivateHostname = "https://api.example.com"
	config.AccountHostname = "https://account.example.com"

	origSendEmail := email.SendEmail
	defer func() { email.SendEmail = origSendEmail }()

	gormDB, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectQuery("INSERT INTO").WillReturnRows(sqlmock.NewRows([]string{"invite_role_id"}).AddRow(1))
	mock.ExpectCommit()

	clientRepo := &mockClientRepo{
		findSystemFn: func() (*Client, error) { return defaultInviteClient(), nil },
	}
	roleRepo := &mockRoleRepo{
		findByUUIDsFn: func(_ []string, _ ...string) ([]Role, error) {
			return []Role{{RoleID: 1, TenantID: 10}}, nil
		},
	}
	badPlain := `{{call .InviteURL}}`
	emailTemplateRepo := &mockEmailTemplateRepo{
		findByNameFn: func(_ string) (*branding.EmailTemplate, error) {
			return &branding.EmailTemplate{
				Subject:   "Invite",
				BodyHTML:  `<a href="{{.InviteURL}}">Accept</a>`,
				BodyPlain: &badPlain,
			}, nil
		},
	}

	svc := NewInviteService(gormDB, &mockInviteRepo{}, clientRepo, roleRepo, emailTemplateRepo)
	_, err := svc.SendInvite(context.Background(), 1, "user@example.com", 1, []string{"role-uuid-1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "send invite email")
}

func TestInviteService_SendInvite_ClientInactive(t *testing.T) {
	gormDB, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectRollback()

	domain := "example.com"
	clientRepo := &mockClientRepo{
		findSystemFn: func() (*Client, error) {
			return &Client{
				ClientID: 1,
				Status:   shared.StatusInactive,
				Domain:   &domain,
				IdentityProvider: &IdentityProvider{
					Tenant: &Tenant{TenantID: 10},
				},
			}, nil
		},
	}

	svc := NewInviteService(gormDB, &mockInviteRepo{}, clientRepo, &mockRoleRepo{}, &mockEmailTemplateRepo{})
	_, err := svc.SendInvite(context.Background(), 1, "user@example.com", 1, []string{"role-uuid-1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid client")
}

func TestInviteService_SendInvite_ClientNoDomain(t *testing.T) {
	gormDB, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectRollback()

	clientRepo := &mockClientRepo{
		findSystemFn: func() (*Client, error) {
			return &Client{
				ClientID: 1,
				Status:   shared.StatusActive,
				// Domain is nil
				IdentityProvider: &IdentityProvider{
					Tenant: &Tenant{TenantID: 10},
				},
			}, nil
		},
	}

	svc := NewInviteService(gormDB, &mockInviteRepo{}, clientRepo, &mockRoleRepo{}, &mockEmailTemplateRepo{})
	_, err := svc.SendInvite(context.Background(), 1, "user@example.com", 1, []string{"role-uuid-1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid client")
}

func TestInviteService_SendInvite_ClientEmptyDomain(t *testing.T) {
	gormDB, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectRollback()

	empty := ""
	clientRepo := &mockClientRepo{
		findSystemFn: func() (*Client, error) {
			return &Client{
				ClientID: 1,
				Status:   shared.StatusActive,
				Domain:   &empty,
				IdentityProvider: &IdentityProvider{
					Tenant: &Tenant{TenantID: 10},
				},
			}, nil
		},
	}

	svc := NewInviteService(gormDB, &mockInviteRepo{}, clientRepo, &mockRoleRepo{}, &mockEmailTemplateRepo{})
	_, err := svc.SendInvite(context.Background(), 1, "user@example.com", 1, []string{"role-uuid-1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid client")
}

func TestInviteService_SendInvite_NoTenant(t *testing.T) {
	gormDB, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectRollback()

	domain := "example.com"
	clientRepo := &mockClientRepo{
		findSystemFn: func() (*Client, error) {
			return &Client{
				ClientID: 1,
				Status:   shared.StatusActive,
				Domain:   &domain,
				IdentityProvider: &IdentityProvider{
					Tenant: nil,
				},
			}, nil
		},
	}

	svc := NewInviteService(gormDB, &mockInviteRepo{}, clientRepo, &mockRoleRepo{}, &mockEmailTemplateRepo{})
	_, err := svc.SendInvite(context.Background(), 1, "user@example.com", 1, []string{"role-uuid-1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid client")
}

func TestInviteService_SendInvite_TenantIDZero(t *testing.T) {
	gormDB, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectRollback()

	domain := "example.com"
	clientRepo := &mockClientRepo{
		findSystemFn: func() (*Client, error) {
			return &Client{
				ClientID: 1,
				Status:   shared.StatusActive,
				Domain:   &domain,
				IdentityProvider: &IdentityProvider{
					Tenant: &Tenant{TenantID: 0},
				},
			}, nil
		},
	}

	svc := NewInviteService(gormDB, &mockInviteRepo{}, clientRepo, &mockRoleRepo{}, &mockEmailTemplateRepo{})
	_, err := svc.SendInvite(context.Background(), 1, "user@example.com", 1, []string{"role-uuid-1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid client")
}

func TestInviteService_SendInvite_GenerateSignedURLError(t *testing.T) {
	_ = os.Setenv("HMAC_SECRET_KEY", "test-secret-key-for-hmac")
	t.Cleanup(func() { _ = os.Unsetenv("HMAC_SECRET_KEY") })
	origAppPrivateHostname := config.AppPrivateHostname
	origAccountHostname := config.AccountHostname
	defer func() {
		config.AppPrivateHostname = origAppPrivateHostname
		config.AccountHostname = origAccountHostname
	}()
	config.AppPrivateHostname = "https://api.example.com"
	config.AccountHostname = "https://account.example.com"

	origGenerateSignedURL := signedurl.GenerateSignedURL
	defer func() { signedurl.GenerateSignedURL = origGenerateSignedURL }()
	signedurl.GenerateSignedURL = func(_ string, _ map[string]string, _ time.Duration) (string, error) {
		return "", errors.New("signed url failure")
	}

	gormDB, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectQuery("INSERT INTO").WillReturnRows(sqlmock.NewRows([]string{"invite_role_id"}).AddRow(1))
	mock.ExpectCommit()

	clientRepo := &mockClientRepo{
		findSystemFn: func() (*Client, error) { return defaultInviteClient(), nil },
	}
	roleRepo := &mockRoleRepo{
		findByUUIDsFn: func(_ []string, _ ...string) ([]Role, error) {
			return []Role{{RoleID: 1, TenantID: 10}}, nil
		},
	}

	svc := NewInviteService(gormDB, &mockInviteRepo{}, clientRepo, roleRepo, &mockEmailTemplateRepo{})
	_, err := svc.SendInvite(context.Background(), 1, "user@example.com", 1, []string{"role-uuid-1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to generate signed invite URL")
}

func TestInviteService_SendInvite_ConvertToFrontendURLError(t *testing.T) {
	_ = os.Setenv("HMAC_SECRET_KEY", "test-secret-key-for-hmac")
	t.Cleanup(func() { _ = os.Unsetenv("HMAC_SECRET_KEY") })
	origAppPrivateHostname := config.AppPrivateHostname
	origAccountHostname := config.AccountHostname
	defer func() {
		config.AppPrivateHostname = origAppPrivateHostname
		config.AccountHostname = origAccountHostname
	}()
	config.AppPrivateHostname = "https://api.example.com"
	config.AccountHostname = "https://account.example.com"

	origConvertToFrontendURL := signedurl.ConvertToFrontendURL
	defer func() { signedurl.ConvertToFrontendURL = origConvertToFrontendURL }()
	signedurl.ConvertToFrontendURL = func(_, _ string) (string, error) {
		return "", errors.New("frontend url failure")
	}

	gormDB, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectQuery("INSERT INTO").WillReturnRows(sqlmock.NewRows([]string{"invite_role_id"}).AddRow(1))
	mock.ExpectCommit()

	clientRepo := &mockClientRepo{
		findSystemFn: func() (*Client, error) { return defaultInviteClient(), nil },
	}
	roleRepo := &mockRoleRepo{
		findByUUIDsFn: func(_ []string, _ ...string) ([]Role, error) {
			return []Role{{RoleID: 1, TenantID: 10}}, nil
		},
	}

	svc := NewInviteService(gormDB, &mockInviteRepo{}, clientRepo, roleRepo, &mockEmailTemplateRepo{})
	_, err := svc.SendInvite(context.Background(), 1, "user@example.com", 1, []string{"role-uuid-1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to convert invite URL")
}
