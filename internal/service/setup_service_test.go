package service

import (
	"errors"
	"testing"

	"github.com/maintainerd/auth/internal/dto"
	"github.com/maintainerd/auth/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func buildSetupService(t *testing.T,
	tenantRepo *mockTenantRepo,
	userRepo *mockUserRepo,
	profileRepo *mockProfileRepo,
	clientRepo *mockClientRepo,
	idpRepo *mockIdentityProviderRepo,
	roleRepo *mockRoleRepo,
	userRoleRepo *mockUserRoleRepo,
	userIdentityRepo *mockUserIdentityRepo,
	tenantMemberRepo *mockTenantMemberRepo,
	tenantUserRepo *mockTenantUserRepo,
) SetupService {
	t.Helper()
	db, _ := newMockGormDB(t)
	return NewSetupService(db, userRepo, tenantRepo, tenantMemberRepo, tenantUserRepo,
		clientRepo, idpRepo, roleRepo, userRoleRepo, nil, userIdentityRepo, profileRepo)
}

func TestSetupService_GetSetupStatus(t *testing.T) {
	tests := []struct {
		name            string
		tenantsFn       func(...string) ([]model.Tenant, error)
		defaultTenantFn func() (*model.Tenant, error)
		superAdminFn    func() (*model.User, error)
		profileFn       func(int64) (*model.Profile, error)
		wantErr         bool
		wantTenant      bool
		wantAdmin       bool
		wantProfile     bool
	}{
		{
			name:      "tenant repo error",
			tenantsFn: func(...string) ([]model.Tenant, error) { return nil, errors.New("db error") },
			wantErr:   true,
		},
		{
			name:       "no tenants setup",
			tenantsFn:  func(...string) ([]model.Tenant, error) { return []model.Tenant{}, nil },
			wantTenant: false,
		},
		{
			name:            "tenant exists, no admin",
			tenantsFn:       func(...string) ([]model.Tenant, error) { return []model.Tenant{{Name: "main"}}, nil },
			defaultTenantFn: func() (*model.Tenant, error) { return &model.Tenant{TenantID: 1}, nil },
			superAdminFn:    func() (*model.User, error) { return nil, nil },
			wantTenant:      true,
			wantAdmin:       false,
		},
		{
			name:            "full setup complete",
			tenantsFn:       func(...string) ([]model.Tenant, error) { return []model.Tenant{{Name: "main"}}, nil },
			defaultTenantFn: func() (*model.Tenant, error) { return &model.Tenant{TenantID: 1}, nil },
			superAdminFn:    func() (*model.User, error) { return &model.User{UserID: 1}, nil },
			profileFn:       func(int64) (*model.Profile, error) { return &model.Profile{ProfileID: 1}, nil },
			wantTenant:      true,
			wantAdmin:       true,
			wantProfile:     true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := buildSetupService(t,
				&mockTenantRepo{findAllFn: tc.tenantsFn, findDefaultFn: tc.defaultTenantFn},
				&mockUserRepo{findSuperAdminFn: tc.superAdminFn},
				&mockProfileRepo{findByUserIDFn: tc.profileFn},
				&mockClientRepo{}, &mockIdentityProviderRepo{}, &mockRoleRepo{},
				&mockUserRoleRepo{}, &mockUserIdentityRepo{}, &mockTenantMemberRepo{}, &mockTenantUserRepo{},
			)
			res, err := svc.GetSetupStatus()
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.wantTenant, res.IsTenantSetup)
			assert.Equal(t, tc.wantAdmin, res.IsAdminSetup)
			assert.Equal(t, tc.wantProfile, res.IsProfileSetup)
		})
	}
}

func TestSetupService_CreateTenant(t *testing.T) {
	validReq := dto.CreateTenantRequestDto{Name: "maintainerd", DisplayName: "Maintainerd"}

	t.Run("tenant already exists", func(t *testing.T) {
		svc := buildSetupService(t,
			&mockTenantRepo{findAllFn: func(...string) ([]model.Tenant, error) { return []model.Tenant{{Name: "main"}}, nil }},
			&mockUserRepo{}, &mockProfileRepo{}, &mockClientRepo{}, &mockIdentityProviderRepo{},
			&mockRoleRepo{}, &mockUserRoleRepo{}, &mockUserIdentityRepo{}, &mockTenantMemberRepo{}, &mockTenantUserRepo{},
		)
		_, err := svc.CreateTenant(validReq)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "tenant already exists")
	})

	t.Run("findAll error", func(t *testing.T) {
		svc := buildSetupService(t,
			&mockTenantRepo{findAllFn: func(...string) ([]model.Tenant, error) { return nil, errors.New("db err") }},
			&mockUserRepo{}, &mockProfileRepo{}, &mockClientRepo{}, &mockIdentityProviderRepo{},
			&mockRoleRepo{}, &mockUserRoleRepo{}, &mockUserIdentityRepo{}, &mockTenantMemberRepo{}, &mockTenantUserRepo{},
		)
		_, err := svc.CreateTenant(validReq)
		require.Error(t, err)
	})
}
