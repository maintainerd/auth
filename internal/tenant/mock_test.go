package tenant

import (
	"context"

	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// mockTenantService
// ---------------------------------------------------------------------------

type mockTenantService struct {
	getFn             func(TenantServiceGetFilter) (*TenantServiceGetResult, error)
	getByUUIDFn       func(uuid.UUID) (*TenantServiceDataResult, error)
	getSystemFn       func() (*TenantServiceDataResult, error)
	getByNameFn       func(string) (*TenantServiceDataResult, error)
	createFn          func(string, string, string, string) (*TenantServiceDataResult, error)
	updateFn          func(uuid.UUID, string, string, string, string) (*TenantServiceDataResult, error)
	setStatusByUUIDFn func(uuid.UUID, string) (*TenantServiceDataResult, error)
	deleteByUUIDFn    func(uuid.UUID) (*TenantServiceDataResult, error)
}

func (m *mockTenantService) Get(_ context.Context, f TenantServiceGetFilter) (*TenantServiceGetResult, error) {
	if m.getFn != nil {
		return m.getFn(f)
	}
	return &TenantServiceGetResult{}, nil
}
func (m *mockTenantService) GetByUUID(_ context.Context, id uuid.UUID) (*TenantServiceDataResult, error) {
	if m.getByUUIDFn != nil {
		return m.getByUUIDFn(id)
	}
	return nil, nil
}
func (m *mockTenantService) GetSystem(_ context.Context) (*TenantServiceDataResult, error) {
	if m.getSystemFn != nil {
		return m.getSystemFn()
	}
	return nil, nil
}
func (m *mockTenantService) GetByName(_ context.Context, name string) (*TenantServiceDataResult, error) {
	if m.getByNameFn != nil {
		return m.getByNameFn(name)
	}
	return nil, nil
}
func (m *mockTenantService) Create(_ context.Context, n, dn, desc, s string) (*TenantServiceDataResult, error) {
	if m.createFn != nil {
		return m.createFn(n, dn, desc, s)
	}
	return nil, nil
}
func (m *mockTenantService) Update(_ context.Context, id uuid.UUID, n, dn, desc, s string) (*TenantServiceDataResult, error) {
	if m.updateFn != nil {
		return m.updateFn(id, n, dn, desc, s)
	}
	return nil, nil
}
func (m *mockTenantService) SetStatusByUUID(_ context.Context, id uuid.UUID, s string) (*TenantServiceDataResult, error) {
	if m.setStatusByUUIDFn != nil {
		return m.setStatusByUUIDFn(id, s)
	}
	return nil, nil
}
func (m *mockTenantService) DeleteByUUID(_ context.Context, id uuid.UUID, _ int64) (*TenantServiceDataResult, error) {
	if m.deleteByUUIDFn != nil {
		return m.deleteByUUIDFn(id)
	}
	return nil, nil
}

// ---------------------------------------------------------------------------
// mockTenantMemberService
// ---------------------------------------------------------------------------

type mockTenantMemberService struct {
	createFn             func(int64, int64, string) (*TenantMemberServiceDataResult, error)
	createByUserUUIDFn   func(int64, uuid.UUID, string) (*TenantMemberServiceDataResult, error)
	getByUUIDFn          func(uuid.UUID) (*TenantMemberServiceDataResult, error)
	getByTenantAndUserFn func(int64, int64) (*TenantMemberServiceDataResult, error)
	listByTenantFn       func(TenantMemberServiceListFilter) (*TenantMemberServiceListResult, error)
	listByUserFn         func(int64) ([]TenantMemberServiceDataResult, error)
	updateRoleFn         func(int64, uuid.UUID, string) (*TenantMemberServiceDataResult, error)
	deleteByUUIDFn       func(int64, uuid.UUID) error
	isUserInTenantFn     func(int64, uuid.UUID) (bool, error)
	canManageTenantFn    func(int64, uuid.UUID) (bool, error)
}

func (m *mockTenantMemberService) Create(_ context.Context, tenantID, userID int64, role string, _ int64) (*TenantMemberServiceDataResult, error) {
	if m.createFn != nil {
		return m.createFn(tenantID, userID, role)
	}
	return nil, nil
}
func (m *mockTenantMemberService) CreateByUserUUID(_ context.Context, tenantID int64, userUUID uuid.UUID, role string, _ int64) (*TenantMemberServiceDataResult, error) {
	if m.createByUserUUIDFn != nil {
		return m.createByUserUUIDFn(tenantID, userUUID, role)
	}
	return nil, nil
}
func (m *mockTenantMemberService) GetByUUID(_ context.Context, id uuid.UUID) (*TenantMemberServiceDataResult, error) {
	if m.getByUUIDFn != nil {
		return m.getByUUIDFn(id)
	}
	return nil, nil
}
func (m *mockTenantMemberService) GetByTenantAndUser(_ context.Context, tenantID, userID int64) (*TenantMemberServiceDataResult, error) {
	if m.getByTenantAndUserFn != nil {
		return m.getByTenantAndUserFn(tenantID, userID)
	}
	return nil, nil
}
func (m *mockTenantMemberService) ListByTenant(_ context.Context, filter TenantMemberServiceListFilter) (*TenantMemberServiceListResult, error) {
	if m.listByTenantFn != nil {
		return m.listByTenantFn(filter)
	}
	return nil, nil
}
func (m *mockTenantMemberService) ListByUser(_ context.Context, userID int64) ([]TenantMemberServiceDataResult, error) {
	if m.listByUserFn != nil {
		return m.listByUserFn(userID)
	}
	return nil, nil
}
func (m *mockTenantMemberService) UpdateRole(_ context.Context, tenantID int64, id uuid.UUID, role string, _ int64) (*TenantMemberServiceDataResult, error) {
	if m.updateRoleFn != nil {
		return m.updateRoleFn(tenantID, id, role)
	}
	return nil, nil
}
func (m *mockTenantMemberService) DeleteByUUID(_ context.Context, tenantID int64, id uuid.UUID, _ int64) error {
	if m.deleteByUUIDFn != nil {
		return m.deleteByUUIDFn(tenantID, id)
	}
	return nil
}

func (m *mockTenantMemberService) ResolveUserID(_ context.Context, _ uuid.UUID) (int64, error) {
	return 1, nil
}
func (m *mockTenantMemberService) IsUserInTenant(_ context.Context, userID int64, tenantUUID uuid.UUID) (bool, error) {
	if m.isUserInTenantFn != nil {
		return m.isUserInTenantFn(userID, tenantUUID)
	}
	return false, nil
}

// CanManageTenant defaults to allow so handler tests exercise the handler logic;
// set canManageTenantFn to test the access-denied path explicitly.
func (m *mockTenantMemberService) CanManageTenant(_ context.Context, userID int64, tenantUUID uuid.UUID) (bool, error) {
	if m.canManageTenantFn != nil {
		return m.canManageTenantFn(userID, tenantUUID)
	}
	return true, nil
}
