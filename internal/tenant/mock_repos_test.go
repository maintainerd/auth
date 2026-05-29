package tenant

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type mockTenantRepo struct {
	findAllFn          func(preloads ...string) ([]Tenant, error)
	findByUUIDFn       func(id any, preloads ...string) (*Tenant, error)
	findByNameFn       func(name string) (*Tenant, error)
	findByIdentifierFn func(identifier string) (*Tenant, error)
	findSystemFn       func() (*Tenant, error)
	findPaginatedFn    func(filter TenantRepositoryGetFilter) (*PaginationResult[Tenant], error)
	createFn           func(e *Tenant) (*Tenant, error)
	createOrUpdateFn   func(e *Tenant) (*Tenant, error)
	setStatusByUUIDFn  func(tenantUUID uuid.UUID, status string) error
	deleteByUUIDFn     func(id any) error
}

func (m *mockTenantRepo) WithTx(_ *gorm.DB) TenantRepository { return m }
func (m *mockTenantRepo) Create(e *Tenant) (*Tenant, error) {
	if m.createFn != nil {
		return m.createFn(e)
	}
	return e, nil
}
func (m *mockTenantRepo) FindAll(p ...string) ([]Tenant, error) {
	if m.findAllFn != nil {
		return m.findAllFn(p...)
	}
	return nil, nil
}
func (m *mockTenantRepo) FindByUUIDs(ids []string, p ...string) ([]Tenant, error) {
	return nil, nil
}
func (m *mockTenantRepo) FindByID(id any, p ...string) (*Tenant, error)   { return nil, nil }
func (m *mockTenantRepo) UpdateByUUID(id, data any) (*Tenant, error)      { return nil, nil }
func (m *mockTenantRepo) UpdateByID(id, data any) (*Tenant, error)        { return nil, nil }
func (m *mockTenantRepo) DeleteByID(id any) error                         { return nil }
func (m *mockTenantRepo) SetSystemStatusByUUID(_ uuid.UUID, _ bool) error { return nil }
func (m *mockTenantRepo) Paginate(_ map[string]any, _, _ int, _ ...string) (*PaginationResult[Tenant], error) {
	return nil, nil
}

func (m *mockTenantRepo) FindByUUID(id any, p ...string) (*Tenant, error) {
	if m.findByUUIDFn != nil {
		return m.findByUUIDFn(id, p...)
	}
	return nil, nil
}
func (m *mockTenantRepo) CreateOrUpdate(e *Tenant) (*Tenant, error) {
	if m.createOrUpdateFn != nil {
		return m.createOrUpdateFn(e)
	}
	return e, nil
}
func (m *mockTenantRepo) FindByName(name string) (*Tenant, error) {
	if m.findByNameFn != nil {
		return m.findByNameFn(name)
	}
	return nil, nil
}
func (m *mockTenantRepo) FindByIdentifier(id string) (*Tenant, error) {
	if m.findByIdentifierFn != nil {
		return m.findByIdentifierFn(id)
	}
	return nil, nil
}
func (m *mockTenantRepo) FindSystem() (*Tenant, error) {
	if m.findSystemFn != nil {
		return m.findSystemFn()
	}
	return nil, nil
}
func (m *mockTenantRepo) FindPaginated(f TenantRepositoryGetFilter) (*PaginationResult[Tenant], error) {
	if m.findPaginatedFn != nil {
		return m.findPaginatedFn(f)
	}
	return &PaginationResult[Tenant]{}, nil
}
func (m *mockTenantRepo) SetStatusByUUID(id uuid.UUID, s string) error {
	if m.setStatusByUUIDFn != nil {
		return m.setStatusByUUIDFn(id, s)
	}
	return nil
}
func (m *mockTenantRepo) DeleteByUUID(id any) error {
	if m.deleteByUUIDFn != nil {
		return m.deleteByUUIDFn(id)
	}
	return nil
}

type mockTenantMemberRepo struct {
	findByTenantMemberUUIDFn func(uuid.UUID) (*TenantMember, error)
	findByTenantAndUserFn    func(tenantID int64, userID int64) (*TenantMember, error)
	findAllByTenantFn        func(tenantID int64) ([]TenantMember, error)
	findAllByUserFn          func(userID int64) ([]TenantMember, error)
	createFn                 func(*TenantMember) (*TenantMember, error)
	createOrUpdateFn         func(*TenantMember) (*TenantMember, error)
	deleteByUUIDFn           func(any) error
}

func (m *mockTenantMemberRepo) WithTx(_ *gorm.DB) TenantMemberRepository { return m }
func (m *mockTenantMemberRepo) CreateOrUpdate(e *TenantMember) (*TenantMember, error) {
	if m.createOrUpdateFn != nil {
		return m.createOrUpdateFn(e)
	}
	return e, nil
}
func (m *mockTenantMemberRepo) FindAll(_ ...string) ([]TenantMember, error) { return nil, nil }
func (m *mockTenantMemberRepo) FindByUUID(_ any, _ ...string) (*TenantMember, error) {
	return nil, nil
}
func (m *mockTenantMemberRepo) FindByUUIDs(_ []string, _ ...string) ([]TenantMember, error) {
	return nil, nil
}
func (m *mockTenantMemberRepo) FindByID(_ any, _ ...string) (*TenantMember, error) {
	return nil, nil
}
func (m *mockTenantMemberRepo) UpdateByUUID(_, _ any) (*TenantMember, error) { return nil, nil }
func (m *mockTenantMemberRepo) UpdateByID(_, _ any) (*TenantMember, error)   { return nil, nil }
func (m *mockTenantMemberRepo) DeleteByID(_ any) error                       { return nil }
func (m *mockTenantMemberRepo) Paginate(_ map[string]any, _, _ int, _ ...string) (*PaginationResult[TenantMember], error) {
	return nil, nil
}
func (m *mockTenantMemberRepo) FindAllByUser(uID int64) ([]TenantMember, error) {
	if m.findAllByUserFn != nil {
		return m.findAllByUserFn(uID)
	}
	return nil, nil
}

func (m *mockTenantMemberRepo) FindByTenantMemberUUID(id uuid.UUID) (*TenantMember, error) {
	if m.findByTenantMemberUUIDFn != nil {
		return m.findByTenantMemberUUIDFn(id)
	}
	return nil, nil
}
func (m *mockTenantMemberRepo) FindByTenantAndUser(tID, uID int64) (*TenantMember, error) {
	if m.findByTenantAndUserFn != nil {
		return m.findByTenantAndUserFn(tID, uID)
	}
	return nil, nil
}
func (m *mockTenantMemberRepo) FindAllByTenant(tID int64) ([]TenantMember, error) {
	if m.findAllByTenantFn != nil {
		return m.findAllByTenantFn(tID)
	}
	return nil, nil
}
func (m *mockTenantMemberRepo) Create(e *TenantMember) (*TenantMember, error) {
	if m.createFn != nil {
		return m.createFn(e)
	}
	return e, nil
}
func (m *mockTenantMemberRepo) DeleteByUUID(id any) error {
	if m.deleteByUUIDFn != nil {
		return m.deleteByUUIDFn(id)
	}
	return nil
}

type mockTenantSettingRepo struct {
	findByTenantIDFn func(int64) (*TenantSetting, error)
	createFn         func(*TenantSetting) (*TenantSetting, error)
	createOrUpdateFn func(*TenantSetting) (*TenantSetting, error)
}

func (m *mockTenantSettingRepo) WithTx(_ *gorm.DB) TenantSettingRepository    { return m }
func (m *mockTenantSettingRepo) FindAll(_ ...string) ([]TenantSetting, error) { return nil, nil }
func (m *mockTenantSettingRepo) FindByUUID(_ any, _ ...string) (*TenantSetting, error) {
	return nil, nil
}
func (m *mockTenantSettingRepo) FindByUUIDs(_ []string, _ ...string) ([]TenantSetting, error) {
	return nil, nil
}
func (m *mockTenantSettingRepo) FindByID(_ any, _ ...string) (*TenantSetting, error) {
	return nil, nil
}
func (m *mockTenantSettingRepo) UpdateByUUID(_, _ any) (*TenantSetting, error) { return nil, nil }
func (m *mockTenantSettingRepo) UpdateByID(_, _ any) (*TenantSetting, error)   { return nil, nil }
func (m *mockTenantSettingRepo) DeleteByUUID(_ any) error                      { return nil }
func (m *mockTenantSettingRepo) DeleteByID(_ any) error                        { return nil }
func (m *mockTenantSettingRepo) Paginate(_ map[string]any, _, _ int, _ ...string) (*PaginationResult[TenantSetting], error) {
	return nil, nil
}
func (m *mockTenantSettingRepo) FindByTenantID(tid int64) (*TenantSetting, error) {
	if m.findByTenantIDFn != nil {
		return m.findByTenantIDFn(tid)
	}
	return nil, nil
}
func (m *mockTenantSettingRepo) Create(e *TenantSetting) (*TenantSetting, error) {
	if m.createFn != nil {
		return m.createFn(e)
	}
	return e, nil
}
func (m *mockTenantSettingRepo) CreateOrUpdate(e *TenantSetting) (*TenantSetting, error) {
	if m.createOrUpdateFn != nil {
		return m.createOrUpdateFn(e)
	}
	return e, nil
}

type mockTenantSettingService struct {
	getFn                     func(int64) (*TenantSettingServiceDataResult, error)
	getRateLimitConfigFn      func(int64) (map[string]any, error)
	getAuditConfigFn          func(int64) (map[string]any, error)
	getMaintenanceConfigFn    func(int64) (map[string]any, error)
	getFeatureFlagsFn         func(int64) (map[string]any, error)
	updateRateLimitConfigFn   func(int64, map[string]any) (*TenantSettingServiceDataResult, error)
	updateAuditConfigFn       func(int64, map[string]any) (*TenantSettingServiceDataResult, error)
	updateMaintenanceConfigFn func(int64, map[string]any) (*TenantSettingServiceDataResult, error)
	updateFeatureFlagsFn      func(int64, map[string]any) (*TenantSettingServiceDataResult, error)
}

func (m *mockTenantSettingService) Get(_ context.Context, tid int64) (*TenantSettingServiceDataResult, error) {
	if m.getFn != nil {
		return m.getFn(tid)
	}
	return nil, nil
}
func (m *mockTenantSettingService) GetRateLimitConfig(_ context.Context, tid int64) (map[string]any, error) {
	if m.getRateLimitConfigFn != nil {
		return m.getRateLimitConfigFn(tid)
	}
	return nil, nil
}
func (m *mockTenantSettingService) GetAuditConfig(_ context.Context, tid int64) (map[string]any, error) {
	if m.getAuditConfigFn != nil {
		return m.getAuditConfigFn(tid)
	}
	return nil, nil
}
func (m *mockTenantSettingService) GetMaintenanceConfig(_ context.Context, tid int64) (map[string]any, error) {
	if m.getMaintenanceConfigFn != nil {
		return m.getMaintenanceConfigFn(tid)
	}
	return nil, nil
}
func (m *mockTenantSettingService) GetFeatureFlags(_ context.Context, tid int64) (map[string]any, error) {
	if m.getFeatureFlagsFn != nil {
		return m.getFeatureFlagsFn(tid)
	}
	return nil, nil
}
func (m *mockTenantSettingService) UpdateRateLimitConfig(_ context.Context, tid int64, cfg map[string]any) (*TenantSettingServiceDataResult, error) {
	if m.updateRateLimitConfigFn != nil {
		return m.updateRateLimitConfigFn(tid, cfg)
	}
	return nil, nil
}
func (m *mockTenantSettingService) UpdateAuditConfig(_ context.Context, tid int64, cfg map[string]any) (*TenantSettingServiceDataResult, error) {
	if m.updateAuditConfigFn != nil {
		return m.updateAuditConfigFn(tid, cfg)
	}
	return nil, nil
}
func (m *mockTenantSettingService) UpdateMaintenanceConfig(_ context.Context, tid int64, cfg map[string]any) (*TenantSettingServiceDataResult, error) {
	if m.updateMaintenanceConfigFn != nil {
		return m.updateMaintenanceConfigFn(tid, cfg)
	}
	return nil, nil
}
func (m *mockTenantSettingService) UpdateFeatureFlags(_ context.Context, tid int64, cfg map[string]any) (*TenantSettingServiceDataResult, error) {
	if m.updateFeatureFlagsFn != nil {
		return m.updateFeatureFlagsFn(tid, cfg)
	}
	return nil, nil
}
