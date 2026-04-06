package service

import (
	"errors"
	"os"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/maintainerd/auth/internal/config"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// defaultInviteClient returns a full client with domain, IDP and tenant set.
func defaultInviteClient() *model.Client {
	domain := "example.com"
	return &model.Client{
		ClientID: 1,
		Status:   model.StatusActive,
		Domain:   &domain,
		IdentityProvider: &model.IdentityProvider{
			Identifier: "test-idp",
			Tenant:     &model.Tenant{TenantID: 10},
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
				c.findDefaultFn = func() (*model.Client, error) { return nil, errors.New("db error") }
			},
			expectCommit: false,
			wantErr:      true,
		},
		{
			name: "client is nil - invalid client",
			setupRepos: func(c *mockClientRepo, r *mockRoleRepo, i *mockInviteRepo) {
				c.findDefaultFn = func() (*model.Client, error) { return nil, nil }
			},
			expectCommit: false,
			wantErr:      true,
			wantErrMsg:   "invalid client or identity provider",
		},
		{
			name: "active client with no identity provider - invalid",
			setupRepos: func(c *mockClientRepo, r *mockRoleRepo, i *mockInviteRepo) {
				// Client has no IdentityProvider set
				c.findDefaultFn = func() (*model.Client, error) {
					return &model.Client{Status: model.StatusActive}, nil
				}
			},
			expectCommit: false,
			wantErr:      true,
			wantErrMsg:   "invalid client or identity provider",
		},
		{
			name: "role findByUUIDs error",
			setupRepos: func(c *mockClientRepo, r *mockRoleRepo, i *mockInviteRepo) {
				c.findDefaultFn = func() (*model.Client, error) { return defaultInviteClient(), nil }
				r.findByUUIDsFn = func(_ []string, _ ...string) ([]model.Role, error) {
					return nil, errors.New("db error")
				}
			},
			expectCommit: false,
			wantErr:      true,
		},
		{
			name: "role count mismatch - one or more roles not found",
			setupRepos: func(c *mockClientRepo, r *mockRoleRepo, i *mockInviteRepo) {
				c.findDefaultFn = func() (*model.Client, error) { return defaultInviteClient(), nil }
				// Return fewer roles than requested
				r.findByUUIDsFn = func(_ []string, _ ...string) ([]model.Role, error) {
					return []model.Role{}, nil
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
			result, err := svc.SendInvite(1, "user@example.com", 1, []string{"role-uuid-1"})

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
		findDefaultFn: func() (*model.Client, error) { return defaultInviteClient(), nil },
	}
	roleRepo := &mockRoleRepo{
		findByUUIDsFn: func(_ []string, _ ...string) ([]model.Role, error) {
			return []model.Role{{RoleID: 1, TenantID: 999}}, nil // wrong tenant
		},
	}

	svc := NewInviteService(gormDB, &mockInviteRepo{}, clientRepo, roleRepo, &mockEmailTemplateRepo{})
	_, err := svc.SendInvite(1, "user@example.com", 1, []string{"role-uuid-1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid role")
}

func TestInviteService_SendInvite_InviteCreateError(t *testing.T) {
	gormDB, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectRollback()

	clientRepo := &mockClientRepo{
		findDefaultFn: func() (*model.Client, error) { return defaultInviteClient(), nil },
	}
	roleRepo := &mockRoleRepo{
		findByUUIDsFn: func(_ []string, _ ...string) ([]model.Role, error) {
			return []model.Role{{RoleID: 1, TenantID: 10}}, nil
		},
	}
	inviteRepo := &mockInviteRepo{
		createFn: func(_ *model.Invite) (*model.Invite, error) { return nil, errors.New("create err") },
	}

	svc := NewInviteService(gormDB, inviteRepo, clientRepo, roleRepo, &mockEmailTemplateRepo{})
	_, err := svc.SendInvite(1, "user@example.com", 1, []string{"role-uuid-1"})
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
		findDefaultFn: func() (*model.Client, error) { return defaultInviteClient(), nil },
	}
	roleRepo := &mockRoleRepo{
		findByUUIDsFn: func(_ []string, _ ...string) ([]model.Role, error) {
			return []model.Role{{RoleID: 1, TenantID: 10}}, nil
		},
	}

	svc := NewInviteService(gormDB, &mockInviteRepo{}, clientRepo, roleRepo, &mockEmailTemplateRepo{})
	_, err := svc.SendInvite(1, "user@example.com", 1, []string{"role-uuid-1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bulk insert err")
}

func TestInviteService_SendInvite_FullSuccess(t *testing.T) {
	os.Setenv("HMAC_SECRET_KEY", "test-secret-key-for-hmac")
	defer os.Unsetenv("HMAC_SECRET_KEY")

	origAppPrivateHostname := config.AppPrivateHostname
	origAccountHostname := config.AccountHostname
	defer func() {
		config.AppPrivateHostname = origAppPrivateHostname
		config.AccountHostname = origAccountHostname
	}()
	config.AppPrivateHostname = "https://api.example.com"
	config.AccountHostname = "https://account.example.com"

	origSendEmail := util.SendEmail
	defer func() { util.SendEmail = origSendEmail }()
	var emailSent bool
	util.SendEmail = func(p util.SendEmailParams) error {
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
		findDefaultFn: func() (*model.Client, error) { return defaultInviteClient(), nil },
	}
	roleRepo := &mockRoleRepo{
		findByUUIDsFn: func(_ []string, _ ...string) ([]model.Role, error) {
			return []model.Role{{RoleID: 1, TenantID: 10}}, nil
		},
	}
	bodyPlain := "Join: {{.InviteURL}}"
	emailTemplateRepo := &mockEmailTemplateRepo{
		findByNameFn: func(_ string) (*model.EmailTemplate, error) {
			return &model.EmailTemplate{
				Subject:   "You're Invited",
				BodyHTML:  `<a href="{{.InviteURL}}">Accept</a>`,
				BodyPlain: &bodyPlain,
			}, nil
		},
	}

	svc := NewInviteService(gormDB, &mockInviteRepo{}, clientRepo, roleRepo, emailTemplateRepo)
	result, err := svc.SendInvite(1, "user@example.com", 1, []string{"role-uuid-1"})
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, emailSent)
}

func TestInviteService_SendInvite_EmailSendError(t *testing.T) {
	os.Setenv("HMAC_SECRET_KEY", "test-secret-key-for-hmac")
	defer os.Unsetenv("HMAC_SECRET_KEY")

	origAppPrivateHostname := config.AppPrivateHostname
	origAccountHostname := config.AccountHostname
	defer func() {
		config.AppPrivateHostname = origAppPrivateHostname
		config.AccountHostname = origAccountHostname
	}()
	config.AppPrivateHostname = "https://api.example.com"
	config.AccountHostname = "https://account.example.com"

	origSendEmail := util.SendEmail
	defer func() { util.SendEmail = origSendEmail }()
	util.SendEmail = func(_ util.SendEmailParams) error { return errors.New("smtp err") }

	gormDB, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectQuery("INSERT INTO").WillReturnRows(sqlmock.NewRows([]string{"invite_role_id"}).AddRow(1))
	mock.ExpectCommit()

	clientRepo := &mockClientRepo{
		findDefaultFn: func() (*model.Client, error) { return defaultInviteClient(), nil },
	}
	roleRepo := &mockRoleRepo{
		findByUUIDsFn: func(_ []string, _ ...string) ([]model.Role, error) {
			return []model.Role{{RoleID: 1, TenantID: 10}}, nil
		},
	}
	emailTemplateRepo := &mockEmailTemplateRepo{
		findByNameFn: func(_ string) (*model.EmailTemplate, error) {
			return &model.EmailTemplate{
				Subject:  "Invite",
				BodyHTML: `<a href="{{.InviteURL}}">Accept</a>`,
			}, nil
		},
	}

	svc := NewInviteService(gormDB, &mockInviteRepo{}, clientRepo, roleRepo, emailTemplateRepo)
	_, err := svc.SendInvite(1, "user@example.com", 1, []string{"role-uuid-1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "send invite email")
}

func TestInviteService_SendInvite_TemplateFetchError(t *testing.T) {
	os.Setenv("HMAC_SECRET_KEY", "test-secret-key-for-hmac")
	defer os.Unsetenv("HMAC_SECRET_KEY")

	origAppPrivateHostname := config.AppPrivateHostname
	origAccountHostname := config.AccountHostname
	defer func() {
		config.AppPrivateHostname = origAppPrivateHostname
		config.AccountHostname = origAccountHostname
	}()
	config.AppPrivateHostname = "https://api.example.com"
	config.AccountHostname = "https://account.example.com"

	origSendEmail := util.SendEmail
	defer func() { util.SendEmail = origSendEmail }()

	gormDB, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectQuery("INSERT INTO").WillReturnRows(sqlmock.NewRows([]string{"invite_role_id"}).AddRow(1))
	mock.ExpectCommit()

	clientRepo := &mockClientRepo{
		findDefaultFn: func() (*model.Client, error) { return defaultInviteClient(), nil },
	}
	roleRepo := &mockRoleRepo{
		findByUUIDsFn: func(_ []string, _ ...string) ([]model.Role, error) {
			return []model.Role{{RoleID: 1, TenantID: 10}}, nil
		},
	}
	emailTemplateRepo := &mockEmailTemplateRepo{
		findByNameFn: func(_ string) (*model.EmailTemplate, error) {
			return nil, errors.New("template not found")
		},
	}

	svc := NewInviteService(gormDB, &mockInviteRepo{}, clientRepo, roleRepo, emailTemplateRepo)
	_, err := svc.SendInvite(1, "user@example.com", 1, []string{"role-uuid-1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "send invite email")
}

func TestInviteService_SendInvite_HTMLParseError(t *testing.T) {
	os.Setenv("HMAC_SECRET_KEY", "test-secret-key-for-hmac")
	defer os.Unsetenv("HMAC_SECRET_KEY")

	origAppPrivateHostname := config.AppPrivateHostname
	origAccountHostname := config.AccountHostname
	defer func() {
		config.AppPrivateHostname = origAppPrivateHostname
		config.AccountHostname = origAccountHostname
	}()
	config.AppPrivateHostname = "https://api.example.com"
	config.AccountHostname = "https://account.example.com"

	origSendEmail := util.SendEmail
	defer func() { util.SendEmail = origSendEmail }()

	gormDB, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectQuery("INSERT INTO").WillReturnRows(sqlmock.NewRows([]string{"invite_role_id"}).AddRow(1))
	mock.ExpectCommit()

	clientRepo := &mockClientRepo{
		findDefaultFn: func() (*model.Client, error) { return defaultInviteClient(), nil },
	}
	roleRepo := &mockRoleRepo{
		findByUUIDsFn: func(_ []string, _ ...string) ([]model.Role, error) {
			return []model.Role{{RoleID: 1, TenantID: 10}}, nil
		},
	}
	emailTemplateRepo := &mockEmailTemplateRepo{
		findByNameFn: func(_ string) (*model.EmailTemplate, error) {
			return &model.EmailTemplate{
				Subject:  "Invite",
				BodyHTML: `{{.InvalidSyntax`, // bad template
			}, nil
		},
	}

	svc := NewInviteService(gormDB, &mockInviteRepo{}, clientRepo, roleRepo, emailTemplateRepo)
	_, err := svc.SendInvite(1, "user@example.com", 1, []string{"role-uuid-1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "send invite email")
}

func TestInviteService_SendInvite_HTMLExecuteError(t *testing.T) {
	os.Setenv("HMAC_SECRET_KEY", "test-secret-key-for-hmac")
	defer os.Unsetenv("HMAC_SECRET_KEY")

	origAppPrivateHostname := config.AppPrivateHostname
	origAccountHostname := config.AccountHostname
	defer func() {
		config.AppPrivateHostname = origAppPrivateHostname
		config.AccountHostname = origAccountHostname
	}()
	config.AppPrivateHostname = "https://api.example.com"
	config.AccountHostname = "https://account.example.com"

	origSendEmail := util.SendEmail
	defer func() { util.SendEmail = origSendEmail }()

	gormDB, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectQuery("INSERT INTO").WillReturnRows(sqlmock.NewRows([]string{"invite_role_id"}).AddRow(1))
	mock.ExpectCommit()

	clientRepo := &mockClientRepo{
		findDefaultFn: func() (*model.Client, error) { return defaultInviteClient(), nil },
	}
	roleRepo := &mockRoleRepo{
		findByUUIDsFn: func(_ []string, _ ...string) ([]model.Role, error) {
			return []model.Role{{RoleID: 1, TenantID: 10}}, nil
		},
	}
	emailTemplateRepo := &mockEmailTemplateRepo{
		findByNameFn: func(_ string) (*model.EmailTemplate, error) {
			return &model.EmailTemplate{
				Subject:  "Invite",
				BodyHTML: `{{call .InviteURL}}`, // parses ok, fails on Execute
			}, nil
		},
	}

	svc := NewInviteService(gormDB, &mockInviteRepo{}, clientRepo, roleRepo, emailTemplateRepo)
	_, err := svc.SendInvite(1, "user@example.com", 1, []string{"role-uuid-1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "send invite email")
}

func TestInviteService_SendInvite_PlainParseError(t *testing.T) {
	os.Setenv("HMAC_SECRET_KEY", "test-secret-key-for-hmac")
	defer os.Unsetenv("HMAC_SECRET_KEY")

	origAppPrivateHostname := config.AppPrivateHostname
	origAccountHostname := config.AccountHostname
	defer func() {
		config.AppPrivateHostname = origAppPrivateHostname
		config.AccountHostname = origAccountHostname
	}()
	config.AppPrivateHostname = "https://api.example.com"
	config.AccountHostname = "https://account.example.com"

	origSendEmail := util.SendEmail
	defer func() { util.SendEmail = origSendEmail }()

	gormDB, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectQuery("INSERT INTO").WillReturnRows(sqlmock.NewRows([]string{"invite_role_id"}).AddRow(1))
	mock.ExpectCommit()

	clientRepo := &mockClientRepo{
		findDefaultFn: func() (*model.Client, error) { return defaultInviteClient(), nil },
	}
	roleRepo := &mockRoleRepo{
		findByUUIDsFn: func(_ []string, _ ...string) ([]model.Role, error) {
			return []model.Role{{RoleID: 1, TenantID: 10}}, nil
		},
	}
	badPlain := `{{.InvalidSyntax`
	emailTemplateRepo := &mockEmailTemplateRepo{
		findByNameFn: func(_ string) (*model.EmailTemplate, error) {
			return &model.EmailTemplate{
				Subject:   "Invite",
				BodyHTML:  `<a href="{{.InviteURL}}">Accept</a>`,
				BodyPlain: &badPlain,
			}, nil
		},
	}

	svc := NewInviteService(gormDB, &mockInviteRepo{}, clientRepo, roleRepo, emailTemplateRepo)
	_, err := svc.SendInvite(1, "user@example.com", 1, []string{"role-uuid-1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "send invite email")
}

func TestInviteService_SendInvite_PlainExecuteError(t *testing.T) {
	os.Setenv("HMAC_SECRET_KEY", "test-secret-key-for-hmac")
	defer os.Unsetenv("HMAC_SECRET_KEY")

	origAppPrivateHostname := config.AppPrivateHostname
	origAccountHostname := config.AccountHostname
	defer func() {
		config.AppPrivateHostname = origAppPrivateHostname
		config.AccountHostname = origAccountHostname
	}()
	config.AppPrivateHostname = "https://api.example.com"
	config.AccountHostname = "https://account.example.com"

	origSendEmail := util.SendEmail
	defer func() { util.SendEmail = origSendEmail }()

	gormDB, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectQuery("INSERT INTO").WillReturnRows(sqlmock.NewRows([]string{"invite_role_id"}).AddRow(1))
	mock.ExpectCommit()

	clientRepo := &mockClientRepo{
		findDefaultFn: func() (*model.Client, error) { return defaultInviteClient(), nil },
	}
	roleRepo := &mockRoleRepo{
		findByUUIDsFn: func(_ []string, _ ...string) ([]model.Role, error) {
			return []model.Role{{RoleID: 1, TenantID: 10}}, nil
		},
	}
	badPlain := `{{call .InviteURL}}`
	emailTemplateRepo := &mockEmailTemplateRepo{
		findByNameFn: func(_ string) (*model.EmailTemplate, error) {
			return &model.EmailTemplate{
				Subject:   "Invite",
				BodyHTML:  `<a href="{{.InviteURL}}">Accept</a>`,
				BodyPlain: &badPlain,
			}, nil
		},
	}

	svc := NewInviteService(gormDB, &mockInviteRepo{}, clientRepo, roleRepo, emailTemplateRepo)
	_, err := svc.SendInvite(1, "user@example.com", 1, []string{"role-uuid-1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "send invite email")
}

func TestInviteService_SendInvite_ClientInactive(t *testing.T) {
	gormDB, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectRollback()

	domain := "example.com"
	clientRepo := &mockClientRepo{
		findDefaultFn: func() (*model.Client, error) {
			return &model.Client{
				ClientID: 1,
				Status:   model.StatusInactive,
				Domain:   &domain,
				IdentityProvider: &model.IdentityProvider{
					Tenant: &model.Tenant{TenantID: 10},
				},
			}, nil
		},
	}

	svc := NewInviteService(gormDB, &mockInviteRepo{}, clientRepo, &mockRoleRepo{}, &mockEmailTemplateRepo{})
	_, err := svc.SendInvite(1, "user@example.com", 1, []string{"role-uuid-1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid client")
}

func TestInviteService_SendInvite_ClientNoDomain(t *testing.T) {
	gormDB, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectRollback()

	clientRepo := &mockClientRepo{
		findDefaultFn: func() (*model.Client, error) {
			return &model.Client{
				ClientID: 1,
				Status:   model.StatusActive,
				// Domain is nil
				IdentityProvider: &model.IdentityProvider{
					Tenant: &model.Tenant{TenantID: 10},
				},
			}, nil
		},
	}

	svc := NewInviteService(gormDB, &mockInviteRepo{}, clientRepo, &mockRoleRepo{}, &mockEmailTemplateRepo{})
	_, err := svc.SendInvite(1, "user@example.com", 1, []string{"role-uuid-1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid client")
}

func TestInviteService_SendInvite_ClientEmptyDomain(t *testing.T) {
	gormDB, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectRollback()

	empty := ""
	clientRepo := &mockClientRepo{
		findDefaultFn: func() (*model.Client, error) {
			return &model.Client{
				ClientID: 1,
				Status:   model.StatusActive,
				Domain:   &empty,
				IdentityProvider: &model.IdentityProvider{
					Tenant: &model.Tenant{TenantID: 10},
				},
			}, nil
		},
	}

	svc := NewInviteService(gormDB, &mockInviteRepo{}, clientRepo, &mockRoleRepo{}, &mockEmailTemplateRepo{})
	_, err := svc.SendInvite(1, "user@example.com", 1, []string{"role-uuid-1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid client")
}

func TestInviteService_SendInvite_NoTenant(t *testing.T) {
	gormDB, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectRollback()

	domain := "example.com"
	clientRepo := &mockClientRepo{
		findDefaultFn: func() (*model.Client, error) {
			return &model.Client{
				ClientID: 1,
				Status:   model.StatusActive,
				Domain:   &domain,
				IdentityProvider: &model.IdentityProvider{
					Tenant: nil,
				},
			}, nil
		},
	}

	svc := NewInviteService(gormDB, &mockInviteRepo{}, clientRepo, &mockRoleRepo{}, &mockEmailTemplateRepo{})
	_, err := svc.SendInvite(1, "user@example.com", 1, []string{"role-uuid-1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid client")
}

func TestInviteService_SendInvite_TenantIDZero(t *testing.T) {
	gormDB, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectRollback()

	domain := "example.com"
	clientRepo := &mockClientRepo{
		findDefaultFn: func() (*model.Client, error) {
			return &model.Client{
				ClientID: 1,
				Status:   model.StatusActive,
				Domain:   &domain,
				IdentityProvider: &model.IdentityProvider{
					Tenant: &model.Tenant{TenantID: 0},
				},
			}, nil
		},
	}

	svc := NewInviteService(gormDB, &mockInviteRepo{}, clientRepo, &mockRoleRepo{}, &mockEmailTemplateRepo{})
	_, err := svc.SendInvite(1, "user@example.com", 1, []string{"role-uuid-1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid client")
}
