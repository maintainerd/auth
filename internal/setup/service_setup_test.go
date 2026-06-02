package setup

import (
	"context"
	"errors"
	"math"
	"testing"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/platform/crypto"
	"github.com/maintainerd/auth/internal/platform/runner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func buildSetupService(t *testing.T,
	tenantRepo *mockTenantRepo,
	userRepo *mockUserRepo,
	profileRepo *mockProfileRepo,
	clientRepo *mockClientRepo,
	roleRepo *mockRoleRepo,
	userRoleRepo *mockUserRoleRepo,
	userIdentityRepo *mockUserIdentityRepo,
	tenantMemberRepo *mockTenantMemberRepo,
) SetupService {
	t.Helper()
	db, _ := newMockGormDB(t)
	return NewSetupService(db, userRepo, tenantRepo, tenantMemberRepo,
		clientRepo, roleRepo, userRoleRepo, userIdentityRepo, profileRepo)
}

// ---------------------------------------------------------------------------
// GetSetupStatus
// ---------------------------------------------------------------------------

func TestSetupService_GetSetupStatus(t *testing.T) {
	t.Run("tenant repo error", func(t *testing.T) {
		svc := buildSetupService(t,
			&mockTenantRepo{findAllFn: func(...string) ([]Tenant, error) { return nil, errors.New("db error") }},
			&mockUserRepo{}, &mockProfileRepo{}, &mockClientRepo{},
			&mockRoleRepo{}, &mockUserRoleRepo{}, &mockUserIdentityRepo{}, &mockTenantMemberRepo{},
		)
		_, err := svc.GetSetupStatus(context.Background())
		require.Error(t, err)
	})

	t.Run("no tenants setup", func(t *testing.T) {
		svc := buildSetupService(t,
			&mockTenantRepo{findAllFn: func(...string) ([]Tenant, error) { return []Tenant{}, nil }},
			&mockUserRepo{}, &mockProfileRepo{}, &mockClientRepo{},
			&mockRoleRepo{}, &mockUserRoleRepo{}, &mockUserIdentityRepo{}, &mockTenantMemberRepo{},
		)
		res, err := svc.GetSetupStatus(context.Background())
		require.NoError(t, err)
		assert.False(t, res.IsTenantSetup)
		assert.False(t, res.IsAdminSetup)
		assert.False(t, res.IsProfileSetup)
		assert.False(t, res.IsSetupComplete)
	})

	t.Run("tenant exists, FindDefault error", func(t *testing.T) {
		svc := buildSetupService(t,
			&mockTenantRepo{
				findAllFn:    func(...string) ([]Tenant, error) { return []Tenant{{Name: "main"}}, nil },
				findSystemFn: func() (*Tenant, error) { return nil, errors.New("db err") },
			},
			&mockUserRepo{}, &mockProfileRepo{}, &mockClientRepo{},
			&mockRoleRepo{}, &mockUserRoleRepo{}, &mockUserIdentityRepo{}, &mockTenantMemberRepo{},
		)
		res, err := svc.GetSetupStatus(context.Background())
		require.NoError(t, err)
		assert.True(t, res.IsTenantSetup)
		assert.False(t, res.IsAdminSetup)
	})

	t.Run("tenant exists, FindDefault nil", func(t *testing.T) {
		svc := buildSetupService(t,
			&mockTenantRepo{
				findAllFn:    func(...string) ([]Tenant, error) { return []Tenant{{Name: "main"}}, nil },
				findSystemFn: func() (*Tenant, error) { return nil, nil },
			},
			&mockUserRepo{}, &mockProfileRepo{}, &mockClientRepo{},
			&mockRoleRepo{}, &mockUserRoleRepo{}, &mockUserIdentityRepo{}, &mockTenantMemberRepo{},
		)
		res, err := svc.GetSetupStatus(context.Background())
		require.NoError(t, err)
		assert.True(t, res.IsTenantSetup)
		assert.False(t, res.IsAdminSetup)
	})

	t.Run("tenant exists, FindSuperAdmin error", func(t *testing.T) {
		svc := buildSetupService(t,
			&mockTenantRepo{
				findAllFn:    func(...string) ([]Tenant, error) { return []Tenant{{Name: "main"}}, nil },
				findSystemFn: func() (*Tenant, error) { return &Tenant{TenantID: 1}, nil },
			},
			&mockUserRepo{findSuperAdminFn: func() (*User, error) { return nil, errors.New("err") }},
			&mockProfileRepo{}, &mockClientRepo{},
			&mockRoleRepo{}, &mockUserRoleRepo{}, &mockUserIdentityRepo{}, &mockTenantMemberRepo{},
		)
		res, err := svc.GetSetupStatus(context.Background())
		require.NoError(t, err)
		assert.True(t, res.IsTenantSetup)
		assert.False(t, res.IsAdminSetup)
	})

	t.Run("tenant exists, no admin", func(t *testing.T) {
		svc := buildSetupService(t,
			&mockTenantRepo{
				findAllFn:    func(...string) ([]Tenant, error) { return []Tenant{{Name: "main"}}, nil },
				findSystemFn: func() (*Tenant, error) { return &Tenant{TenantID: 1}, nil },
			},
			&mockUserRepo{findSuperAdminFn: func() (*User, error) { return nil, nil }},
			&mockProfileRepo{}, &mockClientRepo{},
			&mockRoleRepo{}, &mockUserRoleRepo{}, &mockUserIdentityRepo{}, &mockTenantMemberRepo{},
		)
		res, err := svc.GetSetupStatus(context.Background())
		require.NoError(t, err)
		assert.True(t, res.IsTenantSetup)
		assert.False(t, res.IsAdminSetup)
	})

	t.Run("admin exists, FindByUserID error", func(t *testing.T) {
		svc := buildSetupService(t,
			&mockTenantRepo{
				findAllFn:    func(...string) ([]Tenant, error) { return []Tenant{{Name: "main"}}, nil },
				findSystemFn: func() (*Tenant, error) { return &Tenant{TenantID: 1}, nil },
			},
			&mockUserRepo{findSuperAdminFn: func() (*User, error) { return &User{UserID: 1}, nil }},
			&mockProfileRepo{findByUserIDFn: func(_ int64) (*Profile, error) { return nil, errors.New("err") }},
			&mockClientRepo{},
			&mockRoleRepo{}, &mockUserRoleRepo{}, &mockUserIdentityRepo{}, &mockTenantMemberRepo{},
		)
		res, err := svc.GetSetupStatus(context.Background())
		require.NoError(t, err)
		assert.True(t, res.IsAdminSetup)
		assert.False(t, res.IsProfileSetup)
	})

	t.Run("admin exists, no profile", func(t *testing.T) {
		svc := buildSetupService(t,
			&mockTenantRepo{
				findAllFn:    func(...string) ([]Tenant, error) { return []Tenant{{Name: "main"}}, nil },
				findSystemFn: func() (*Tenant, error) { return &Tenant{TenantID: 1}, nil },
			},
			&mockUserRepo{findSuperAdminFn: func() (*User, error) { return &User{UserID: 1}, nil }},
			&mockProfileRepo{findByUserIDFn: func(_ int64) (*Profile, error) { return nil, nil }},
			&mockClientRepo{},
			&mockRoleRepo{}, &mockUserRoleRepo{}, &mockUserIdentityRepo{}, &mockTenantMemberRepo{},
		)
		res, err := svc.GetSetupStatus(context.Background())
		require.NoError(t, err)
		assert.True(t, res.IsAdminSetup)
		assert.False(t, res.IsProfileSetup)
	})

	t.Run("full setup complete", func(t *testing.T) {
		svc := buildSetupService(t,
			&mockTenantRepo{
				findAllFn:    func(...string) ([]Tenant, error) { return []Tenant{{Name: "main"}}, nil },
				findSystemFn: func() (*Tenant, error) { return &Tenant{TenantID: 1}, nil },
			},
			&mockUserRepo{findSuperAdminFn: func() (*User, error) { return &User{UserID: 1}, nil }},
			&mockProfileRepo{findByUserIDFn: func(_ int64) (*Profile, error) { return &Profile{ProfileID: 1}, nil }},
			&mockClientRepo{},
			&mockRoleRepo{}, &mockUserRoleRepo{}, &mockUserIdentityRepo{}, &mockTenantMemberRepo{},
		)
		res, err := svc.GetSetupStatus(context.Background())
		require.NoError(t, err)
		assert.True(t, res.IsTenantSetup)
		assert.True(t, res.IsAdminSetup)
		assert.True(t, res.IsProfileSetup)
		assert.True(t, res.IsSetupComplete)
	})
}

// ---------------------------------------------------------------------------
// CreateTenant
// ---------------------------------------------------------------------------

func TestSetupService_CreateTenant(t *testing.T) {
	desc := "Test description"
	validReq := CreateTenantRequestDTO{Name: "maintainerd", DisplayName: "Maintainerd"}
	reqWithDesc := CreateTenantRequestDTO{Name: "maintainerd", DisplayName: "Maintainerd", Description: &desc}

	t.Run("findAll error", func(t *testing.T) {
		svc := buildSetupService(t,
			&mockTenantRepo{findAllFn: func(...string) ([]Tenant, error) { return nil, errors.New("db err") }},
			&mockUserRepo{}, &mockProfileRepo{}, &mockClientRepo{},
			&mockRoleRepo{}, &mockUserRoleRepo{}, &mockUserIdentityRepo{}, &mockTenantMemberRepo{},
		)
		_, err := svc.CreateTenant(context.Background(), validReq)
		require.Error(t, err)
	})

	t.Run("tenant already exists", func(t *testing.T) {
		svc := buildSetupService(t,
			&mockTenantRepo{findAllFn: func(...string) ([]Tenant, error) { return []Tenant{{Name: "main"}}, nil }},
			&mockUserRepo{}, &mockProfileRepo{}, &mockClientRepo{},
			&mockRoleRepo{}, &mockUserRoleRepo{}, &mockUserIdentityRepo{}, &mockTenantMemberRepo{},
		)
		_, err := svc.CreateTenant(context.Background(), validReq)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "tenant already exists")
	})

	t.Run("GenerateIdentifier failure", func(t *testing.T) {
		orig := crypto.GenerateIdentifier
		defer func() { crypto.GenerateIdentifier = orig }()
		crypto.GenerateIdentifier = func(int) (string, error) { return "", errors.New("rand failure") }

		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewSetupService(db, &mockUserRepo{},
			&mockTenantRepo{},
			&mockTenantMemberRepo{}, &mockClientRepo{},
			&mockRoleRepo{}, &mockUserRoleRepo{},
			&mockUserIdentityRepo{}, &mockProfileRepo{},
		)
		_, err := svc.CreateTenant(context.Background(), validReq)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "rand failure")
	})

	t.Run("tenant Create error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewSetupService(db, &mockUserRepo{},
			&mockTenantRepo{
				createFn: func(_ *Tenant) (*Tenant, error) { return nil, errors.New("create failed") },
			},
			&mockTenantMemberRepo{}, &mockClientRepo{},
			&mockRoleRepo{}, &mockUserRoleRepo{},
			&mockUserIdentityRepo{}, &mockProfileRepo{},
		)
		_, err := svc.CreateTenant(context.Background(), validReq)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "create failed")
	})

	t.Run("RunSeeders error → rollback (with description)", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewSetupService(db, &mockUserRepo{},
			&mockTenantRepo{},
			&mockTenantMemberRepo{}, &mockClientRepo{},
			&mockRoleRepo{}, &mockUserRoleRepo{},
			&mockUserIdentityRepo{}, &mockProfileRepo{},
		)
		// RunSeeders will fail because sqlmock has no matching SQL expectations
		_, err := svc.CreateTenant(context.Background(), reqWithDesc)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to initialize tenant structure")
	})

	t.Run("RunSeeders error → rollback (nil description, nil metadata)", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewSetupService(db, &mockUserRepo{},
			&mockTenantRepo{},
			&mockTenantMemberRepo{}, &mockClientRepo{},
			&mockRoleRepo{}, &mockUserRoleRepo{},
			&mockUserIdentityRepo{}, &mockProfileRepo{},
		)
		_, err := svc.CreateTenant(context.Background(), validReq)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to initialize tenant structure")
	})

	t.Run("RunSeeders error → rollback (with metadata)", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		meta := &TenantMetadataDTO{}
		req := CreateTenantRequestDTO{Name: "maintainerd", DisplayName: "Maintainerd", Metadata: meta}
		svc := NewSetupService(db, &mockUserRepo{},
			&mockTenantRepo{},
			&mockTenantMemberRepo{}, &mockClientRepo{},
			&mockRoleRepo{}, &mockUserRoleRepo{},
			&mockUserIdentityRepo{}, &mockProfileRepo{},
		)
		_, err := svc.CreateTenant(context.Background(), req)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to initialize tenant structure")
	})

	t.Run("success without metadata", func(t *testing.T) {
		origRunSeeders := runner.RunSeeders
		defer func() { runner.RunSeeders = origRunSeeders }()
		runner.RunSeeders = func(_ *gorm.DB, _ string) error { return nil }

		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		clientIdent := "default-client"
		svc := NewSetupService(db, &mockUserRepo{},
			&mockTenantRepo{
				createFn: func(tenant *Tenant) (*Tenant, error) {
					tenant.TenantID = 1
					tenant.TenantUUID = uuid.New()
					return tenant, nil
				},
			},
			&mockTenantMemberRepo{},
			&mockClientRepo{
				findSystemFn: func() (*Client, error) {
					return &Client{
						Identifier: &clientIdent,
						IdentityProvider: &IdentityProvider{
							Identifier: "default-provider",
						},
					}, nil
				},
			},
			&mockRoleRepo{}, &mockUserRoleRepo{},
			&mockUserIdentityRepo{}, &mockProfileRepo{},
		)
		res, err := svc.CreateTenant(context.Background(), validReq)
		require.NoError(t, err)
		assert.Equal(t, "maintainerd", res.Tenant.Name)
		assert.Equal(t, "default-client", res.DefaultClientID)
		assert.Equal(t, "default-provider", res.DefaultProviderID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("success with metadata and description", func(t *testing.T) {
		origRunSeeders := runner.RunSeeders
		defer func() { runner.RunSeeders = origRunSeeders }()
		runner.RunSeeders = func(_ *gorm.DB, _ string) error { return nil }

		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		lang := "en"
		meta := &TenantMetadataDTO{Language: &lang}
		req := CreateTenantRequestDTO{
			Name:        "maintainerd",
			DisplayName: "Maintainerd",
			Description: &desc,
			Metadata:    meta,
		}
		svc := NewSetupService(db, &mockUserRepo{},
			&mockTenantRepo{
				createFn: func(tenant *Tenant) (*Tenant, error) {
					tenant.TenantID = 1
					tenant.TenantUUID = uuid.New()
					return tenant, nil
				},
			},
			&mockTenantMemberRepo{},
			&mockClientRepo{
				findSystemFn: func() (*Client, error) { return nil, nil },
			},
			&mockRoleRepo{}, &mockUserRoleRepo{},
			&mockUserIdentityRepo{}, &mockProfileRepo{},
		)
		res, err := svc.CreateTenant(context.Background(), req)
		require.NoError(t, err)
		assert.Equal(t, "maintainerd", res.Tenant.Name)
		assert.Empty(t, res.DefaultClientID)
		assert.Empty(t, res.DefaultProviderID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("success but FindDefault error", func(t *testing.T) {
		origRunSeeders := runner.RunSeeders
		defer func() { runner.RunSeeders = origRunSeeders }()
		runner.RunSeeders = func(_ *gorm.DB, _ string) error { return nil }

		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		svc := NewSetupService(db, &mockUserRepo{},
			&mockTenantRepo{
				createFn: func(tenant *Tenant) (*Tenant, error) {
					tenant.TenantID = 1
					tenant.TenantUUID = uuid.New()
					return tenant, nil
				},
			},
			&mockTenantMemberRepo{},
			&mockClientRepo{
				findSystemFn: func() (*Client, error) { return nil, errors.New("find err") },
			},
			&mockRoleRepo{}, &mockUserRoleRepo{},
			&mockUserIdentityRepo{}, &mockProfileRepo{},
		)
		_, err := svc.CreateTenant(context.Background(), validReq)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "find err")
	})

	t.Run("success with invalid metadata JSON in tenant", func(t *testing.T) {
		origRunSeeders := runner.RunSeeders
		defer func() { runner.RunSeeders = origRunSeeders }()
		runner.RunSeeders = func(_ *gorm.DB, _ string) error { return nil }

		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		svc := NewSetupService(db, &mockUserRepo{},
			&mockTenantRepo{
				createFn: func(tenant *Tenant) (*Tenant, error) {
					tenant.TenantID = 1
					tenant.TenantUUID = uuid.New()
					tenant.Metadata = []byte(`{invalid-json}`)
					return tenant, nil
				},
			},
			&mockTenantMemberRepo{},
			&mockClientRepo{
				findSystemFn: func() (*Client, error) { return nil, nil },
			},
			&mockRoleRepo{}, &mockUserRoleRepo{},
			&mockUserIdentityRepo{}, &mockProfileRepo{},
		)
		res, err := svc.CreateTenant(context.Background(), validReq)
		require.NoError(t, err)
		require.NotNil(t, res)
	})
}

// ---------------------------------------------------------------------------
// CreateAdmin
// ---------------------------------------------------------------------------

func TestSetupService_CreateAdmin(t *testing.T) {
	validReq := CreateAdminRequestDTO{
		Username: "admin",
		Fullname: "Admin User",
		Email:    "admin@test.com",
		Password: "password123",
	}

	defaultTenant := &Tenant{TenantID: 1, TenantUUID: uuid.New()}
	clientID := "default-client"
	defaultClient := &Client{ClientID: 1, Identifier: &clientID}

	t.Run("FindAll error", func(t *testing.T) {
		svc := buildSetupService(t,
			&mockTenantRepo{findAllFn: func(...string) ([]Tenant, error) { return nil, errors.New("db err") }},
			&mockUserRepo{}, &mockProfileRepo{}, &mockClientRepo{},
			&mockRoleRepo{}, &mockUserRoleRepo{}, &mockUserIdentityRepo{}, &mockTenantMemberRepo{},
		)
		_, err := svc.CreateAdmin(context.Background(), validReq)
		require.Error(t, err)
	})

	t.Run("no tenants", func(t *testing.T) {
		svc := buildSetupService(t,
			&mockTenantRepo{findAllFn: func(...string) ([]Tenant, error) { return []Tenant{}, nil }},
			&mockUserRepo{}, &mockProfileRepo{}, &mockClientRepo{},
			&mockRoleRepo{}, &mockUserRoleRepo{}, &mockUserIdentityRepo{}, &mockTenantMemberRepo{},
		)
		_, err := svc.CreateAdmin(context.Background(), validReq)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "tenant must be created first")
	})

	t.Run("FindSuperAdmin error", func(t *testing.T) {
		svc := buildSetupService(t,
			&mockTenantRepo{findAllFn: func(...string) ([]Tenant, error) { return []Tenant{{Name: "t"}}, nil }},
			&mockUserRepo{findSuperAdminFn: func() (*User, error) { return nil, errors.New("db err") }},
			&mockProfileRepo{}, &mockClientRepo{},
			&mockRoleRepo{}, &mockUserRoleRepo{}, &mockUserIdentityRepo{}, &mockTenantMemberRepo{},
		)
		_, err := svc.CreateAdmin(context.Background(), validReq)
		require.Error(t, err)
	})

	t.Run("admin already exists", func(t *testing.T) {
		svc := buildSetupService(t,
			&mockTenantRepo{findAllFn: func(...string) ([]Tenant, error) { return []Tenant{{Name: "t"}}, nil }},
			&mockUserRepo{findSuperAdminFn: func() (*User, error) { return &User{UserID: 1}, nil }},
			&mockProfileRepo{}, &mockClientRepo{},
			&mockRoleRepo{}, &mockUserRoleRepo{}, &mockUserIdentityRepo{}, &mockTenantMemberRepo{},
		)
		_, err := svc.CreateAdmin(context.Background(), validReq)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "admin user already exists")
	})

	t.Run("FindDefault tenant error", func(t *testing.T) {
		svc := buildSetupService(t,
			&mockTenantRepo{
				findAllFn:    func(...string) ([]Tenant, error) { return []Tenant{{Name: "t"}}, nil },
				findSystemFn: func() (*Tenant, error) { return nil, errors.New("db err") },
			},
			&mockUserRepo{}, &mockProfileRepo{}, &mockClientRepo{},
			&mockRoleRepo{}, &mockUserRoleRepo{}, &mockUserIdentityRepo{}, &mockTenantMemberRepo{},
		)
		_, err := svc.CreateAdmin(context.Background(), validReq)
		require.Error(t, err)
	})

	t.Run("default tenant nil", func(t *testing.T) {
		svc := buildSetupService(t,
			&mockTenantRepo{
				findAllFn:    func(...string) ([]Tenant, error) { return []Tenant{{Name: "t"}}, nil },
				findSystemFn: func() (*Tenant, error) { return nil, nil },
			},
			&mockUserRepo{}, &mockProfileRepo{}, &mockClientRepo{},
			&mockRoleRepo{}, &mockUserRoleRepo{}, &mockUserIdentityRepo{}, &mockTenantMemberRepo{},
		)
		_, err := svc.CreateAdmin(context.Background(), validReq)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "default tenant not found")
	})

	t.Run("FindDefault client error", func(t *testing.T) {
		svc := buildSetupService(t,
			&mockTenantRepo{
				findAllFn:    func(...string) ([]Tenant, error) { return []Tenant{{Name: "t"}}, nil },
				findSystemFn: func() (*Tenant, error) { return defaultTenant, nil },
			},
			&mockUserRepo{}, &mockProfileRepo{},
			&mockClientRepo{findByNameAndTenantIDFn: func(string, int64) (*Client, error) { return nil, errors.New("db err") }},

			&mockRoleRepo{}, &mockUserRoleRepo{}, &mockUserIdentityRepo{}, &mockTenantMemberRepo{},
		)
		_, err := svc.CreateAdmin(context.Background(), validReq)
		require.Error(t, err)
	})

	t.Run("default client nil", func(t *testing.T) {
		svc := buildSetupService(t,
			&mockTenantRepo{
				findAllFn:    func(...string) ([]Tenant, error) { return []Tenant{{Name: "t"}}, nil },
				findSystemFn: func() (*Tenant, error) { return defaultTenant, nil },
			},
			&mockUserRepo{}, &mockProfileRepo{},
			&mockClientRepo{findByNameAndTenantIDFn: func(string, int64) (*Client, error) { return nil, nil }},

			&mockRoleRepo{}, &mockUserRoleRepo{}, &mockUserIdentityRepo{}, &mockTenantMemberRepo{},
		)
		_, err := svc.CreateAdmin(context.Background(), validReq)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "auth-console system client not found")
	})

	// --- Transaction tests ---

	adminRepos := func(overrides ...func(*mockUserRepo, *mockRoleRepo, *mockUserIdentityRepo, *mockUserRoleRepo, *mockTenantMemberRepo)) (
		*mockTenantRepo, *mockUserRepo, *mockClientRepo, *mockRoleRepo, *mockUserIdentityRepo, *mockUserRoleRepo, *mockTenantMemberRepo,
	) {
		tr := &mockTenantRepo{
			findAllFn:    func(...string) ([]Tenant, error) { return []Tenant{{Name: "t"}}, nil },
			findSystemFn: func() (*Tenant, error) { return defaultTenant, nil },
		}
		ur := &mockUserRepo{}
		cr := &mockClientRepo{findByNameAndTenantIDFn: func(string, int64) (*Client, error) { return defaultClient, nil }}
		rr := &mockRoleRepo{
			findRegisteredRoleForSetupFn: func(_ int64) (*Role, error) { return &Role{RoleID: 10}, nil },
			findSuperAdminRoleForSetupFn: func(_ int64) (*Role, error) { return &Role{RoleID: 20}, nil },
		}
		uir := &mockUserIdentityRepo{}
		urr := &mockUserRoleRepo{}
		tmr := &mockTenantMemberRepo{}
		for _, o := range overrides {
			o(ur, rr, uir, urr, tmr)
		}
		return tr, ur, cr, rr, uir, urr, tmr
	}

	t.Run("TX: FindByEmail error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		tr, ur, cr, rr, uir, urr, tmr := adminRepos(func(u *mockUserRepo, _ *mockRoleRepo, _ *mockUserIdentityRepo, _ *mockUserRoleRepo, _ *mockTenantMemberRepo) {
			u.findByEmailFn = func(_ string) (*User, error) { return nil, errors.New("db err") }
		})
		svc := NewSetupService(db, ur, tr, tmr, cr, rr, urr, uir, &mockProfileRepo{})
		_, err := svc.CreateAdmin(context.Background(), validReq)
		require.Error(t, err)
	})

	t.Run("TX: user with email exists → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		tr, ur, cr, rr, uir, urr, tmr := adminRepos(func(u *mockUserRepo, _ *mockRoleRepo, _ *mockUserIdentityRepo, _ *mockUserRoleRepo, _ *mockTenantMemberRepo) {
			u.findByEmailFn = func(_ string) (*User, error) { return &User{Email: "admin@test.com"}, nil }
		})
		svc := NewSetupService(db, ur, tr, tmr, cr, rr, urr, uir, &mockProfileRepo{})
		_, err := svc.CreateAdmin(context.Background(), validReq)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user with this email already exists")
	})

	t.Run("TX: Create user error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		tr, ur, cr, rr, uir, urr, tmr := adminRepos(func(u *mockUserRepo, _ *mockRoleRepo, _ *mockUserIdentityRepo, _ *mockUserRoleRepo, _ *mockTenantMemberRepo) {
			u.createFn = func(_ *User) (*User, error) { return nil, errors.New("create failed") }
		})
		svc := NewSetupService(db, ur, tr, tmr, cr, rr, urr, uir, &mockProfileRepo{})
		_, err := svc.CreateAdmin(context.Background(), validReq)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "create failed")
	})

	t.Run("TX: Create user identity error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		tr, ur, cr, rr, uir, urr, tmr := adminRepos(func(_ *mockUserRepo, _ *mockRoleRepo, ui *mockUserIdentityRepo, _ *mockUserRoleRepo, _ *mockTenantMemberRepo) {
			ui.createFn = func(_ *UserIdentity) (*UserIdentity, error) { return nil, errors.New("identity failed") }
		})
		svc := NewSetupService(db, ur, tr, tmr, cr, rr, urr, uir, &mockProfileRepo{})
		_, err := svc.CreateAdmin(context.Background(), validReq)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "identity failed")
	})

	t.Run("TX: FindRegisteredRoleForSetup error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		tr, ur, cr, rr, uir, urr, tmr := adminRepos(func(_ *mockUserRepo, r *mockRoleRepo, _ *mockUserIdentityRepo, _ *mockUserRoleRepo, _ *mockTenantMemberRepo) {
			r.findRegisteredRoleForSetupFn = func(_ int64) (*Role, error) { return nil, errors.New("db err") }
		})
		svc := NewSetupService(db, ur, tr, tmr, cr, rr, urr, uir, &mockProfileRepo{})
		_, err := svc.CreateAdmin(context.Background(), validReq)
		require.Error(t, err)
	})

	t.Run("TX: registered role nil → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		tr, ur, cr, rr, uir, urr, tmr := adminRepos(func(_ *mockUserRepo, r *mockRoleRepo, _ *mockUserIdentityRepo, _ *mockUserRoleRepo, _ *mockTenantMemberRepo) {
			r.findRegisteredRoleForSetupFn = func(_ int64) (*Role, error) { return nil, nil }
		})
		svc := NewSetupService(db, ur, tr, tmr, cr, rr, urr, uir, &mockProfileRepo{})
		_, err := svc.CreateAdmin(context.Background(), validReq)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "registered role not found")
	})

	t.Run("TX: Create registered user role error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		callCount := 0
		tr, ur, cr, rr, uir, urr, tmr := adminRepos(func(_ *mockUserRepo, _ *mockRoleRepo, _ *mockUserIdentityRepo, ur2 *mockUserRoleRepo, _ *mockTenantMemberRepo) {
			ur2.createFn = func(_ *UserRole) (*UserRole, error) {
				callCount++
				if callCount == 1 {
					return nil, errors.New("user role failed")
				}
				return &UserRole{}, nil
			}
		})
		svc := NewSetupService(db, ur, tr, tmr, cr, rr, urr, uir, &mockProfileRepo{})
		_, err := svc.CreateAdmin(context.Background(), validReq)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user role failed")
	})

	t.Run("TX: FindSuperAdminRoleForSetup error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		tr, ur, cr, rr, uir, urr, tmr := adminRepos(func(_ *mockUserRepo, r *mockRoleRepo, _ *mockUserIdentityRepo, _ *mockUserRoleRepo, _ *mockTenantMemberRepo) {
			r.findSuperAdminRoleForSetupFn = func(_ int64) (*Role, error) { return nil, errors.New("db err") }
		})
		svc := NewSetupService(db, ur, tr, tmr, cr, rr, urr, uir, &mockProfileRepo{})
		_, err := svc.CreateAdmin(context.Background(), validReq)
		require.Error(t, err)
	})

	t.Run("TX: super-admin role nil → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		tr, ur, cr, rr, uir, urr, tmr := adminRepos(func(_ *mockUserRepo, r *mockRoleRepo, _ *mockUserIdentityRepo, _ *mockUserRoleRepo, _ *mockTenantMemberRepo) {
			r.findSuperAdminRoleForSetupFn = func(_ int64) (*Role, error) { return nil, nil }
		})
		svc := NewSetupService(db, ur, tr, tmr, cr, rr, urr, uir, &mockProfileRepo{})
		_, err := svc.CreateAdmin(context.Background(), validReq)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "super-admin role not found")
	})

	t.Run("TX: Create super-admin user role error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		callCount := 0
		tr, ur, cr, rr, uir, urr, tmr := adminRepos(func(_ *mockUserRepo, _ *mockRoleRepo, _ *mockUserIdentityRepo, ur2 *mockUserRoleRepo, _ *mockTenantMemberRepo) {
			ur2.createFn = func(_ *UserRole) (*UserRole, error) {
				callCount++
				if callCount == 2 {
					return nil, errors.New("super role failed")
				}
				return &UserRole{}, nil
			}
		})
		svc := NewSetupService(db, ur, tr, tmr, cr, rr, urr, uir, &mockProfileRepo{})
		_, err := svc.CreateAdmin(context.Background(), validReq)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "super role failed")
	})

	t.Run("TX: Create tenant member error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		tr, ur, cr, rr, uir, urr, tmr := adminRepos(func(_ *mockUserRepo, _ *mockRoleRepo, _ *mockUserIdentityRepo, _ *mockUserRoleRepo, tm *mockTenantMemberRepo) {
			tm.createFn = func(_ *TenantMember) (*TenantMember, error) { return nil, errors.New("member failed") }
		})
		svc := NewSetupService(db, ur, tr, tmr, cr, rr, urr, uir, &mockProfileRepo{})
		_, err := svc.CreateAdmin(context.Background(), validReq)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "member failed")
	})
	t.Run("success → commit", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		tr, ur, cr, rr, uir, urr, tmr := adminRepos()
		svc := NewSetupService(db, ur, tr, tmr, cr, rr, urr, uir, &mockProfileRepo{})
		res, err := svc.CreateAdmin(context.Background(), validReq)
		require.NoError(t, err)
		assert.Equal(t, "admin@test.com", res.User.Email)
		assert.Equal(t, "admin", res.User.Username)
		assert.Equal(t, "Admin User", res.User.Fullname)
	})

}

// ---------------------------------------------------------------------------
// CreateProfile
// ---------------------------------------------------------------------------

func TestSetupService_CreateProfile(t *testing.T) {
	superAdmin := &User{UserID: 1, UserUUID: uuid.New()}
	validReq := CreateProfileRequestDTO{FirstName: "John"}

	t.Run("FindSuperAdmin error", func(t *testing.T) {
		svc := buildSetupService(t,
			&mockTenantRepo{}, &mockUserRepo{findSuperAdminFn: func() (*User, error) { return nil, errors.New("db err") }},
			&mockProfileRepo{}, &mockClientRepo{},
			&mockRoleRepo{}, &mockUserRoleRepo{}, &mockUserIdentityRepo{}, &mockTenantMemberRepo{},
		)
		_, err := svc.CreateProfile(context.Background(), validReq)
		require.Error(t, err)
	})

	t.Run("no admin user", func(t *testing.T) {
		svc := buildSetupService(t,
			&mockTenantRepo{}, &mockUserRepo{findSuperAdminFn: func() (*User, error) { return nil, nil }},
			&mockProfileRepo{}, &mockClientRepo{},
			&mockRoleRepo{}, &mockUserRoleRepo{}, &mockUserIdentityRepo{}, &mockTenantMemberRepo{},
		)
		_, err := svc.CreateProfile(context.Background(), validReq)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no admin user found")
	})

	t.Run("FindByUserID error", func(t *testing.T) {
		svc := buildSetupService(t,
			&mockTenantRepo{},
			&mockUserRepo{findSuperAdminFn: func() (*User, error) { return superAdmin, nil }},
			&mockProfileRepo{findByUserIDFn: func(_ int64) (*Profile, error) { return nil, errors.New("db err") }},
			&mockClientRepo{},
			&mockRoleRepo{}, &mockUserRoleRepo{}, &mockUserIdentityRepo{}, &mockTenantMemberRepo{},
		)
		_, err := svc.CreateProfile(context.Background(), validReq)
		require.Error(t, err)
	})

	t.Run("profile already exists", func(t *testing.T) {
		svc := buildSetupService(t,
			&mockTenantRepo{},
			&mockUserRepo{findSuperAdminFn: func() (*User, error) { return superAdmin, nil }},
			&mockProfileRepo{findByUserIDFn: func(_ int64) (*Profile, error) { return &Profile{ProfileID: 1}, nil }},
			&mockClientRepo{},
			&mockRoleRepo{}, &mockUserRoleRepo{}, &mockUserIdentityRepo{}, &mockTenantMemberRepo{},
		)
		_, err := svc.CreateProfile(context.Background(), validReq)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "profile already exists")
	})

	t.Run("invalid birthdate format", func(t *testing.T) {
		bd := "not-a-date"
		req := CreateProfileRequestDTO{FirstName: "John", Birthdate: &bd}
		svc := buildSetupService(t,
			&mockTenantRepo{},
			&mockUserRepo{findSuperAdminFn: func() (*User, error) { return superAdmin, nil }},
			&mockProfileRepo{}, &mockClientRepo{},
			&mockRoleRepo{}, &mockUserRoleRepo{}, &mockUserIdentityRepo{}, &mockTenantMemberRepo{},
		)
		_, err := svc.CreateProfile(context.Background(), req)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid birthdate format")
	})

	t.Run("empty birthdate string → treated as no birthdate", func(t *testing.T) {
		bd := ""
		req := CreateProfileRequestDTO{FirstName: "John", Birthdate: &bd}
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		svc := NewSetupService(db,
			&mockUserRepo{findSuperAdminFn: func() (*User, error) { return superAdmin, nil }},
			&mockTenantRepo{}, &mockTenantMemberRepo{}, &mockClientRepo{},
			&mockRoleRepo{}, &mockUserRoleRepo{},
			&mockUserIdentityRepo{}, &mockProfileRepo{},
		)
		res, err := svc.CreateProfile(context.Background(), req)
		require.NoError(t, err)
		assert.Equal(t, "John", res.Profile.FirstName)
	})

	t.Run("valid birthdate", func(t *testing.T) {
		bd := "1990-01-15"
		req := CreateProfileRequestDTO{FirstName: "John", Birthdate: &bd}
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		svc := NewSetupService(db,
			&mockUserRepo{findSuperAdminFn: func() (*User, error) { return superAdmin, nil }},
			&mockTenantRepo{}, &mockTenantMemberRepo{}, &mockClientRepo{},
			&mockRoleRepo{}, &mockUserRoleRepo{},
			&mockUserIdentityRepo{}, &mockProfileRepo{},
		)
		res, err := svc.CreateProfile(context.Background(), req)
		require.NoError(t, err)
		assert.Equal(t, "John", res.Profile.FirstName)
	})

	t.Run("metadata marshal error → rollback", func(t *testing.T) {
		req := CreateProfileRequestDTO{FirstName: "John", Metadata: map[string]any{"bad": math.Inf(1)}}
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewSetupService(db,
			&mockUserRepo{findSuperAdminFn: func() (*User, error) { return superAdmin, nil }},
			&mockTenantRepo{}, &mockTenantMemberRepo{}, &mockClientRepo{},
			&mockRoleRepo{}, &mockUserRoleRepo{},
			&mockUserIdentityRepo{}, &mockProfileRepo{},
		)
		_, err := svc.CreateProfile(context.Background(), req)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid metadata format")
	})

	t.Run("with metadata → success", func(t *testing.T) {
		req := CreateProfileRequestDTO{FirstName: "John", Metadata: map[string]any{"key": "val"}}
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		svc := NewSetupService(db,
			&mockUserRepo{findSuperAdminFn: func() (*User, error) { return superAdmin, nil }},
			&mockTenantRepo{}, &mockTenantMemberRepo{}, &mockClientRepo{},
			&mockRoleRepo{}, &mockUserRoleRepo{},
			&mockUserIdentityRepo{}, &mockProfileRepo{},
		)
		res, err := svc.CreateProfile(context.Background(), req)
		require.NoError(t, err)
		assert.Equal(t, "John", res.Profile.FirstName)
	})

	t.Run("TX: Create profile error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewSetupService(db,
			&mockUserRepo{findSuperAdminFn: func() (*User, error) { return superAdmin, nil }},
			&mockTenantRepo{}, &mockTenantMemberRepo{}, &mockClientRepo{},
			&mockRoleRepo{}, &mockUserRoleRepo{},
			&mockUserIdentityRepo{},
			&mockProfileRepo{createFn: func(_ *Profile) (*Profile, error) { return nil, errors.New("create failed") }},
		)
		_, err := svc.CreateProfile(context.Background(), validReq)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "create failed")
	})

	t.Run("TX: UpdateByUUID error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewSetupService(db,
			&mockUserRepo{
				findSuperAdminFn: func() (*User, error) { return superAdmin, nil },
				updateByUUIDFn:   func(_, _ any) (*User, error) { return nil, errors.New("update failed") },
			},
			&mockTenantRepo{}, &mockTenantMemberRepo{}, &mockClientRepo{},
			&mockRoleRepo{}, &mockUserRoleRepo{},
			&mockUserIdentityRepo{}, &mockProfileRepo{},
		)
		_, err := svc.CreateProfile(context.Background(), validReq)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "update failed")
	})

	t.Run("success with nil birthdate and nil metadata → commit", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		svc := NewSetupService(db,
			&mockUserRepo{findSuperAdminFn: func() (*User, error) { return superAdmin, nil }},
			&mockTenantRepo{}, &mockTenantMemberRepo{}, &mockClientRepo{},
			&mockRoleRepo{}, &mockUserRoleRepo{},
			&mockUserIdentityRepo{}, &mockProfileRepo{},
		)
		res, err := svc.CreateProfile(context.Background(), validReq)
		require.NoError(t, err)
		assert.Equal(t, "John", res.Profile.FirstName)
	})
}
