package setup

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/platform/ptr"
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

func TestSetupService_RegisterControlService(t *testing.T) {
	validReq := RegisterControlServiceRequestDTO{Name: "core", DisplayName: "Core"}
	tenant := &Tenant{TenantID: 1, TenantUUID: uuid.New(), Name: "maintainerd"}
	policyUUID := uuid.New()
	policy := &Policy{PolicyID: 88, PolicyUUID: policyUUID, TenantID: tenant.TenantID, Name: "auth-control", Version: "v1"}

	newService := func(db *gorm.DB, serviceRepo ServiceRepository, policyRepo PolicyRepository, servicePolicyRepo ServicePolicyRepository, tenantRepo *mockTenantRepo) SetupService {
		if tenantRepo == nil {
			tenantRepo = &mockTenantRepo{findSystemFn: func() (*Tenant, error) { return tenant, nil }}
		}
		return NewSetupService(db, &mockUserRepo{}, tenantRepo, &mockTenantMemberRepo{}, &mockClientRepo{}, &mockRoleRepo{}, &mockUserRoleRepo{}, &mockUserIdentityRepo{}, &mockProfileRepo{},
			ControlRegistrationDeps{ServiceRepo: serviceRepo, PolicyRepo: policyRepo, ServicePolicyRepo: servicePolicyRepo},
		)
	}

	t.Run("setup locked", func(t *testing.T) {
		// Locked = system tenant is marked completed.
		lockedTenantRepo := &mockTenantRepo{findSystemFn: func() (*Tenant, error) {
			return &Tenant{TenantID: tenant.TenantID, Status: "active"}, nil
		}}
		svc := newService(nil, nil, nil, nil, lockedTenantRepo)

		_, err := svc.RegisterControlService(context.Background(), validReq)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "setup is complete")
	})

	t.Run("missing dependencies", func(t *testing.T) {
		svc := newService(nil, nil, nil, nil, nil)

		_, err := svc.RegisterControlService(context.Background(), validReq)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "control registration dependencies")
	})

	t.Run("tenant must exist", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		svc := newService(db, &mockServiceRepo{}, &mockPolicyRepo{}, &mockServicePolicyRepo{}, &mockTenantRepo{findSystemFn: func() (*Tenant, error) { return nil, nil }})

		_, err := svc.RegisterControlService(context.Background(), validReq)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "tenant must be created first")
	})

	t.Run("tenant lookup error", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		svc := newService(db, &mockServiceRepo{}, &mockPolicyRepo{}, &mockServicePolicyRepo{}, &mockTenantRepo{findSystemFn: func() (*Tenant, error) { return nil, assert.AnError }})

		_, err := svc.RegisterControlService(context.Background(), validReq)

		require.ErrorIs(t, err, assert.AnError)
	})

	t.Run("control policy lookup error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := newService(db, &mockServiceRepo{}, &mockPolicyRepo{
			findByNameAndVersionFn: func(string, string, int64) (*Policy, error) { return nil, assert.AnError },
		}, &mockServicePolicyRepo{}, nil)

		_, err := svc.RegisterControlService(context.Background(), validReq)

		require.ErrorIs(t, err, assert.AnError)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("control policy must exist", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := newService(db, &mockServiceRepo{}, &mockPolicyRepo{
			findByNameAndVersionFn: func(string, string, int64) (*Policy, error) { return nil, nil },
		}, &mockServicePolicyRepo{}, nil)

		_, err := svc.RegisterControlService(context.Background(), validReq)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "control policy is not seeded")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("creates service and attaches policy", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		desc := "Maintainerd Core"
		req := validReq
		req.Description = &desc
		req.Version = "v2"
		svc := newService(db, &mockServiceRepo{
			createOrUpdateFn: func(service *Service) (*Service, error) {
				assert.Equal(t, desc, service.Description)
				assert.Equal(t, "v2", service.Version)
				service.ServiceID = 77
				return service, nil
			},
		}, &mockPolicyRepo{
			findByNameAndVersionFn: func(name, version string, tenantID int64) (*Policy, error) {
				assert.Equal(t, "auth-control", name)
				assert.Equal(t, "v1", version)
				assert.Equal(t, int64(1), tenantID)
				return policy, nil
			},
		}, &mockServicePolicyRepo{
			createFn: func(attachment *ServicePolicy) (*ServicePolicy, error) {
				assert.Equal(t, int64(77), attachment.ServiceID)
				assert.Equal(t, int64(88), attachment.PolicyID)
				return attachment, nil
			},
		}, nil)

		res, err := svc.RegisterControlService(context.Background(), req)

		require.NoError(t, err)
		assert.Equal(t, "core", res.Name)
		assert.Equal(t, policyUUID.String(), res.PolicyUUID)
		assert.True(t, res.PolicyWasAttached)
		assert.False(t, res.AlreadyExisted)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("service lookup error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := newService(db, &mockServiceRepo{
			findByNameAndTenantIDFn: func(string, int64) (*Service, error) { return nil, assert.AnError },
		}, &mockPolicyRepo{
			findByNameAndVersionFn: func(string, string, int64) (*Policy, error) { return policy, nil },
		}, &mockServicePolicyRepo{}, nil)

		_, err := svc.RegisterControlService(context.Background(), validReq)

		require.ErrorIs(t, err, assert.AnError)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("service create error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := newService(db, &mockServiceRepo{
			createOrUpdateFn: func(*Service) (*Service, error) { return nil, assert.AnError },
		}, &mockPolicyRepo{
			findByNameAndVersionFn: func(string, string, int64) (*Policy, error) { return policy, nil },
		}, &mockServicePolicyRepo{}, nil)

		_, err := svc.RegisterControlService(context.Background(), validReq)

		require.ErrorIs(t, err, assert.AnError)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("attachment lookup error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := newService(db, &mockServiceRepo{
			findByNameAndTenantIDFn: func(string, int64) (*Service, error) {
				return &Service{ServiceID: 77, ServiceUUID: uuid.New(), Name: "core", DisplayName: "Core"}, nil
			},
		}, &mockPolicyRepo{
			findByNameAndVersionFn: func(string, string, int64) (*Policy, error) { return policy, nil },
		}, &mockServicePolicyRepo{
			findByServiceAndPolicyFn: func(int64, int64) (*ServicePolicy, error) { return nil, assert.AnError },
		}, nil)

		_, err := svc.RegisterControlService(context.Background(), validReq)

		require.ErrorIs(t, err, assert.AnError)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("attachment create error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := newService(db, &mockServiceRepo{
			findByNameAndTenantIDFn: func(string, int64) (*Service, error) {
				return &Service{ServiceID: 77, ServiceUUID: uuid.New(), Name: "core", DisplayName: "Core"}, nil
			},
		}, &mockPolicyRepo{
			findByNameAndVersionFn: func(string, string, int64) (*Policy, error) { return policy, nil },
		}, &mockServicePolicyRepo{
			createFn: func(*ServicePolicy) (*ServicePolicy, error) { return nil, assert.AnError },
		}, nil)

		_, err := svc.RegisterControlService(context.Background(), validReq)

		require.ErrorIs(t, err, assert.AnError)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("idempotent when service and attachment exist", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		serviceUUID := uuid.New()
		svc := newService(db, &mockServiceRepo{
			findByNameAndTenantIDFn: func(name string, tenantID int64) (*Service, error) {
				assert.Equal(t, "core", name)
				assert.Equal(t, int64(1), tenantID)
				return &Service{ServiceID: 77, ServiceUUID: serviceUUID, Name: name, DisplayName: "Core"}, nil
			},
		}, &mockPolicyRepo{
			findByNameAndVersionFn: func(string, string, int64) (*Policy, error) { return policy, nil },
		}, &mockServicePolicyRepo{
			findByServiceAndPolicyFn: func(serviceID, policyID int64) (*ServicePolicy, error) {
				assert.Equal(t, int64(77), serviceID)
				assert.Equal(t, int64(88), policyID)
				return &ServicePolicy{ServiceID: serviceID, PolicyID: policyID}, nil
			},
		}, nil)

		res, err := svc.RegisterControlService(context.Background(), validReq)

		require.NoError(t, err)
		assert.Equal(t, serviceUUID.String(), res.ServiceUUID)
		assert.True(t, res.AlreadyExisted)
		assert.False(t, res.PolicyWasAttached)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
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

	t.Run("full setup ready but not locked", func(t *testing.T) {
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
		assert.False(t, res.IsSetupComplete)
	})

	t.Run("setup complete when system tenant is completed", func(t *testing.T) {
		svc := NewSetupService(nil,
			&mockUserRepo{},
			&mockTenantRepo{
				findAllFn:    func(...string) ([]Tenant, error) { return []Tenant{{TenantID: 1}}, nil },
				findSystemFn: func() (*Tenant, error) { return &Tenant{TenantID: 1, Status: "active"}, nil },
			},
			&mockTenantMemberRepo{}, &mockClientRepo{},
			&mockRoleRepo{}, &mockUserRoleRepo{}, &mockUserIdentityRepo{}, &mockProfileRepo{},
		)
		res, err := svc.GetSetupStatus(context.Background())
		require.NoError(t, err)
		assert.True(t, res.IsSetupComplete)
	})
}

func TestSetupService_CompleteSetup(t *testing.T) {
	t.Run("requires tenant admin and profile before locking", func(t *testing.T) {
		svc := NewSetupService(nil,
			&mockUserRepo{},
			&mockTenantRepo{findAllFn: func(...string) ([]Tenant, error) { return nil, nil }},
			&mockTenantMemberRepo{}, &mockClientRepo{},
			&mockRoleRepo{}, &mockUserRoleRepo{}, &mockUserIdentityRepo{}, &mockProfileRepo{},
		)

		_, err := svc.CompleteSetup(context.Background())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "tenant, admin, and profile setup")
	})

	t.Run("marks system tenant completed when bootstrap is ready", func(t *testing.T) {
		var saved *Tenant
		svc := NewSetupService(nil,
			&mockUserRepo{findSuperAdminFn: func() (*User, error) { return &User{UserID: 1}, nil }},
			&mockTenantRepo{
				findAllFn:    func(...string) ([]Tenant, error) { return []Tenant{{TenantID: 1}}, nil },
				findSystemFn: func() (*Tenant, error) { return &Tenant{TenantID: 1}, nil },
				createOrUpdateFn: func(t *Tenant) (*Tenant, error) {
					saved = t
					return t, nil
				},
			},
			&mockTenantMemberRepo{}, &mockClientRepo{},
			&mockRoleRepo{}, &mockUserRoleRepo{}, &mockUserIdentityRepo{},
			&mockProfileRepo{findByUserIDFn: func(_ int64) (*Profile, error) { return &Profile{ProfileID: 1}, nil }},
		)

		res, err := svc.CompleteSetup(context.Background())
		require.NoError(t, err)
		assert.True(t, res.IsSetupComplete)
		require.NotNil(t, saved)
		assert.Equal(t, "active", saved.Status)
	})

	t.Run("already complete is idempotent", func(t *testing.T) {
		svc := NewSetupService(nil,
			&mockUserRepo{}, &mockTenantRepo{
				findAllFn:    func(...string) ([]Tenant, error) { return []Tenant{{TenantID: 1}}, nil },
				findSystemFn: func() (*Tenant, error) { return &Tenant{TenantID: 1, Status: "active"}, nil },
			},
			&mockTenantMemberRepo{}, &mockClientRepo{},
			&mockRoleRepo{}, &mockUserRoleRepo{}, &mockUserIdentityRepo{}, &mockProfileRepo{},
		)

		res, err := svc.CompleteSetup(context.Background())
		require.NoError(t, err)
		assert.True(t, res.IsSetupComplete)
	})

	t.Run("status error is propagated", func(t *testing.T) {
		svc := NewSetupService(nil,
			&mockUserRepo{}, &mockTenantRepo{findAllFn: func(...string) ([]Tenant, error) { return nil, assert.AnError }},
			&mockTenantMemberRepo{}, &mockClientRepo{},
			&mockRoleRepo{}, &mockUserRoleRepo{}, &mockUserIdentityRepo{}, &mockProfileRepo{},
		)

		_, err := svc.CompleteSetup(context.Background())
		require.ErrorIs(t, err, assert.AnError)
	})

	t.Run("complete update error is propagated", func(t *testing.T) {
		svc := NewSetupService(nil,
			&mockUserRepo{findSuperAdminFn: func() (*User, error) { return &User{UserID: 1}, nil }},
			&mockTenantRepo{
				findAllFn:        func(...string) ([]Tenant, error) { return []Tenant{{TenantID: 1}}, nil },
				findSystemFn:     func() (*Tenant, error) { return &Tenant{TenantID: 1}, nil },
				createOrUpdateFn: func(*Tenant) (*Tenant, error) { return nil, assert.AnError },
			},
			&mockTenantMemberRepo{}, &mockClientRepo{},
			&mockRoleRepo{}, &mockUserRoleRepo{}, &mockUserIdentityRepo{},
			&mockProfileRepo{findByUserIDFn: func(_ int64) (*Profile, error) { return &Profile{ProfileID: 1}, nil }},
		)

		_, err := svc.CompleteSetup(context.Background())
		require.ErrorIs(t, err, assert.AnError)
	})

	t.Run("locked setup rejects tenant creation", func(t *testing.T) {
		svc := NewSetupService(nil,
			&mockUserRepo{}, &mockTenantRepo{
				findSystemFn: func() (*Tenant, error) { return &Tenant{TenantID: 1, Status: "active"}, nil },
			},
			&mockTenantMemberRepo{}, &mockClientRepo{},
			&mockRoleRepo{}, &mockUserRoleRepo{}, &mockUserIdentityRepo{}, &mockProfileRepo{},
		)

		_, err := svc.CreateTenant(context.Background(), CreateTenantRequestDTO{Name: "maintainerd", DisplayName: "Maintainerd"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "setup is complete and locked")
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
		origRunSeeders := setupRunSeeders
		defer func() { setupRunSeeders = origRunSeeders }()
		setupRunSeeders = func(_ *gorm.DB, _ string) error { return nil }

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
		origRunSeeders := setupRunSeeders
		defer func() { setupRunSeeders = origRunSeeders }()
		setupRunSeeders = func(_ *gorm.DB, _ string) error { return nil }

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
		origRunSeeders := setupRunSeeders
		defer func() { setupRunSeeders = origRunSeeders }()
		setupRunSeeders = func(_ *gorm.DB, _ string) error { return nil }

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
		origRunSeeders := setupRunSeeders
		defer func() { setupRunSeeders = origRunSeeders }()
		setupRunSeeders = func(_ *gorm.DB, _ string) error { return nil }

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
		Fullname: ptr.Ptr("Admin User"),
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

	t.Run("TX: hash password error → rollback", func(t *testing.T) {
		orig := setupHashPassword
		setupHashPassword = func(context.Context, []byte) ([]byte, error) {
			return nil, errors.New("hash error")
		}
		t.Cleanup(func() { setupHashPassword = orig })
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		tr, ur, cr, rr, uir, urr, tmr := adminRepos()
		svc := NewSetupService(db, ur, tr, tmr, cr, rr, urr, uir, &mockProfileRepo{})
		_, err := svc.CreateAdmin(context.Background(), validReq)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "hash error")
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
