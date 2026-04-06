package service

import (
	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/repository"
	"gorm.io/gorm"
)

// ---------------------------------------------------------------------------
// Mock: TenantRepository
// ---------------------------------------------------------------------------

type mockTenantRepo struct {
	findAllFn                func(preloads ...string) ([]model.Tenant, error)
	findByUUIDFn             func(id any, preloads ...string) (*model.Tenant, error)
	findByNameFn             func(name string) (*model.Tenant, error)
	findByIdentifierFn       func(identifier string) (*model.Tenant, error)
	findDefaultFn            func() (*model.Tenant, error)
	findPaginatedFn          func(filter repository.TenantRepositoryGetFilter) (*repository.PaginationResult[model.Tenant], error)
	createFn                 func(e *model.Tenant) (*model.Tenant, error)
	createOrUpdateFn         func(e *model.Tenant) (*model.Tenant, error)
	setStatusByUUIDFn        func(tenantUUID uuid.UUID, status string) error
	setDefaultStatusByUUIDFn func(tenantUUID uuid.UUID, isDefault bool) error
	deleteByUUIDFn           func(id any) error
}

func (m *mockTenantRepo) WithTx(_ *gorm.DB) repository.TenantRepository { return m }
func (m *mockTenantRepo) Create(e *model.Tenant) (*model.Tenant, error) {
	if m.createFn != nil {
		return m.createFn(e)
	}
	return e, nil
}
func (m *mockTenantRepo) FindAll(p ...string) ([]model.Tenant, error) {
	if m.findAllFn != nil {
		return m.findAllFn(p...)
	}
	return nil, nil
}
func (m *mockTenantRepo) FindByUUIDs(ids []string, p ...string) ([]model.Tenant, error) {
	return nil, nil
}
func (m *mockTenantRepo) FindByID(id any, p ...string) (*model.Tenant, error) { return nil, nil }
func (m *mockTenantRepo) UpdateByUUID(id, data any) (*model.Tenant, error)    { return nil, nil }
func (m *mockTenantRepo) UpdateByID(id, data any) (*model.Tenant, error)      { return nil, nil }
func (m *mockTenantRepo) DeleteByID(id any) error                             { return nil }
func (m *mockTenantRepo) SetSystemStatusByUUID(_ uuid.UUID, _ bool) error     { return nil }
func (m *mockTenantRepo) Paginate(_ map[string]any, _, _ int, _ ...string) (*repository.PaginationResult[model.Tenant], error) {
	return nil, nil
}

func (m *mockTenantRepo) FindByUUID(id any, p ...string) (*model.Tenant, error) {
	if m.findByUUIDFn != nil {
		return m.findByUUIDFn(id, p...)
	}
	return nil, nil
}
func (m *mockTenantRepo) CreateOrUpdate(e *model.Tenant) (*model.Tenant, error) {
	if m.createOrUpdateFn != nil {
		return m.createOrUpdateFn(e)
	}
	return e, nil
}
func (m *mockTenantRepo) FindByName(name string) (*model.Tenant, error) {
	if m.findByNameFn != nil {
		return m.findByNameFn(name)
	}
	return nil, nil
}
func (m *mockTenantRepo) FindByIdentifier(id string) (*model.Tenant, error) {
	if m.findByIdentifierFn != nil {
		return m.findByIdentifierFn(id)
	}
	return nil, nil
}
func (m *mockTenantRepo) FindDefault() (*model.Tenant, error) {
	if m.findDefaultFn != nil {
		return m.findDefaultFn()
	}
	return nil, nil
}
func (m *mockTenantRepo) FindPaginated(f repository.TenantRepositoryGetFilter) (*repository.PaginationResult[model.Tenant], error) {
	if m.findPaginatedFn != nil {
		return m.findPaginatedFn(f)
	}
	return &repository.PaginationResult[model.Tenant]{}, nil
}
func (m *mockTenantRepo) SetStatusByUUID(id uuid.UUID, s string) error {
	if m.setStatusByUUIDFn != nil {
		return m.setStatusByUUIDFn(id, s)
	}
	return nil
}
func (m *mockTenantRepo) SetDefaultStatusByUUID(id uuid.UUID, v bool) error {
	if m.setDefaultStatusByUUIDFn != nil {
		return m.setDefaultStatusByUUIDFn(id, v)
	}
	return nil
}
func (m *mockTenantRepo) DeleteByUUID(id any) error {
	if m.deleteByUUIDFn != nil {
		return m.deleteByUUIDFn(id)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Mock: EmailTemplateRepository (no WithTx)
// ---------------------------------------------------------------------------

type mockEmailTemplateRepo struct {
	createFn                func(e *model.EmailTemplate) (*model.EmailTemplate, error)
	findByUUIDAndTenantIDFn func(uuid.UUID, int64, ...string) (*model.EmailTemplate, error)
	findPaginatedFn         func(repository.EmailTemplateRepositoryGetFilter) (*repository.PaginationResult[model.EmailTemplate], error)
	updateByUUIDFn          func(any, any) (*model.EmailTemplate, error)
	deleteByUUIDFn          func(any) error
	findByNameFn            func(string) (*model.EmailTemplate, error)
}

func (m *mockEmailTemplateRepo) CreateOrUpdate(e *model.EmailTemplate) (*model.EmailTemplate, error) {
	return nil, nil
}
func (m *mockEmailTemplateRepo) FindAll(p ...string) ([]model.EmailTemplate, error) { return nil, nil }
func (m *mockEmailTemplateRepo) FindByUUID(id any, p ...string) (*model.EmailTemplate, error) {
	return nil, nil
}
func (m *mockEmailTemplateRepo) FindByUUIDs(ids []string, p ...string) ([]model.EmailTemplate, error) {
	return nil, nil
}
func (m *mockEmailTemplateRepo) FindByID(id any, p ...string) (*model.EmailTemplate, error) {
	return nil, nil
}
func (m *mockEmailTemplateRepo) UpdateByID(id, data any) (*model.EmailTemplate, error) {
	return nil, nil
}
func (m *mockEmailTemplateRepo) DeleteByID(id any) error { return nil }
func (m *mockEmailTemplateRepo) Paginate(_ map[string]any, _, _ int, _ ...string) (*repository.PaginationResult[model.EmailTemplate], error) {
	return nil, nil
}
func (m *mockEmailTemplateRepo) FindByName(n string) (*model.EmailTemplate, error) {
	if m.findByNameFn != nil {
		return m.findByNameFn(n)
	}
	return nil, nil
}

func (m *mockEmailTemplateRepo) Create(e *model.EmailTemplate) (*model.EmailTemplate, error) {
	if m.createFn != nil {
		return m.createFn(e)
	}
	return e, nil
}
func (m *mockEmailTemplateRepo) FindByUUIDAndTenantID(id uuid.UUID, tID int64, p ...string) (*model.EmailTemplate, error) {
	if m.findByUUIDAndTenantIDFn != nil {
		return m.findByUUIDAndTenantIDFn(id, tID, p...)
	}
	return nil, nil
}
func (m *mockEmailTemplateRepo) FindPaginated(f repository.EmailTemplateRepositoryGetFilter) (*repository.PaginationResult[model.EmailTemplate], error) {
	if m.findPaginatedFn != nil {
		return m.findPaginatedFn(f)
	}
	return &repository.PaginationResult[model.EmailTemplate]{}, nil
}
func (m *mockEmailTemplateRepo) UpdateByUUID(id, data any) (*model.EmailTemplate, error) {
	if m.updateByUUIDFn != nil {
		return m.updateByUUIDFn(id, data)
	}
	return nil, nil
}
func (m *mockEmailTemplateRepo) DeleteByUUID(id any) error {
	if m.deleteByUUIDFn != nil {
		return m.deleteByUUIDFn(id)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Mock: PermissionRepository
// ---------------------------------------------------------------------------

type mockPermissionRepo struct {
	findByUUIDFn              func(any, ...string) (*model.Permission, error)
	findByUUIDsFn             func([]string, ...string) ([]model.Permission, error)
	findByUUIDAndTenantIDFn   func(uuid.UUID, int64) (*model.Permission, error)
	findByNameFn              func(string, int64) (*model.Permission, error)
	findPaginatedFn           func(repository.PermissionRepositoryGetFilter) (*repository.PaginationResult[model.Permission], error)
	createOrUpdateFn          func(*model.Permission) (*model.Permission, error)
	deleteByUUIDAndTenantIDFn func(uuid.UUID, int64) error
}

func (m *mockPermissionRepo) WithTx(_ *gorm.DB) repository.PermissionRepository     { return m }
func (m *mockPermissionRepo) Create(_ *model.Permission) (*model.Permission, error) { return nil, nil }
func (m *mockPermissionRepo) FindAll(_ ...string) ([]model.Permission, error)       { return nil, nil }
func (m *mockPermissionRepo) FindByUUID(id any, p ...string) (*model.Permission, error) {
	if m.findByUUIDFn != nil {
		return m.findByUUIDFn(id, p...)
	}
	return nil, nil
}
func (m *mockPermissionRepo) FindByUUIDs(ids []string, p ...string) ([]model.Permission, error) {
	if m.findByUUIDsFn != nil {
		return m.findByUUIDsFn(ids, p...)
	}
	return nil, nil
}
func (m *mockPermissionRepo) FindByID(_ any, _ ...string) (*model.Permission, error) { return nil, nil }
func (m *mockPermissionRepo) UpdateByUUID(_, _ any) (*model.Permission, error)       { return nil, nil }
func (m *mockPermissionRepo) UpdateByID(_, _ any) (*model.Permission, error)         { return nil, nil }
func (m *mockPermissionRepo) DeleteByUUID(_ any) error                               { return nil }
func (m *mockPermissionRepo) DeleteByID(_ any) error                                 { return nil }
func (m *mockPermissionRepo) Paginate(_ map[string]any, _, _ int, _ ...string) (*repository.PaginationResult[model.Permission], error) {
	return nil, nil
}

func (m *mockPermissionRepo) CreateOrUpdate(e *model.Permission) (*model.Permission, error) {
	if m.createOrUpdateFn != nil {
		return m.createOrUpdateFn(e)
	}
	return e, nil
}
func (m *mockPermissionRepo) FindByUUIDAndTenantID(id uuid.UUID, tID int64) (*model.Permission, error) {
	if m.findByUUIDAndTenantIDFn != nil {
		return m.findByUUIDAndTenantIDFn(id, tID)
	}
	return nil, nil
}
func (m *mockPermissionRepo) FindByName(name string, tID int64) (*model.Permission, error) {
	if m.findByNameFn != nil {
		return m.findByNameFn(name, tID)
	}
	return nil, nil
}
func (m *mockPermissionRepo) FindPaginated(f repository.PermissionRepositoryGetFilter) (*repository.PaginationResult[model.Permission], error) {
	if m.findPaginatedFn != nil {
		return m.findPaginatedFn(f)
	}
	return &repository.PaginationResult[model.Permission]{}, nil
}
func (m *mockPermissionRepo) DeleteByUUIDAndTenantID(id uuid.UUID, tID int64) error {
	if m.deleteByUUIDAndTenantIDFn != nil {
		return m.deleteByUUIDAndTenantIDFn(id, tID)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Mock: APIRepository
// ---------------------------------------------------------------------------

type mockAPIRepo struct {
	findByUUIDFn              func(id any, preloads ...string) (*model.API, error)
	findByUUIDAndTenantIDFn   func(uuid.UUID, int64) (*model.API, error)
	findPaginatedFn           func(repository.APIRepositoryGetFilter) (*repository.PaginationResult[model.API], error)
	findByNameFn              func(name string, tenantID int64) (*model.API, error)
	createOrUpdateFn          func(*model.API) (*model.API, error)
	deleteByUUIDAndTenantIDFn func(uuid.UUID, int64) error
	countByServiceIDFn        func(int64, int64) (int64, error)
}

func (m *mockAPIRepo) WithTx(_ *gorm.DB) repository.APIRepository { return m }
func (m *mockAPIRepo) Create(_ *model.API) (*model.API, error)    { return nil, nil }
func (m *mockAPIRepo) CreateOrUpdate(e *model.API) (*model.API, error) {
	if m.createOrUpdateFn != nil {
		return m.createOrUpdateFn(e)
	}
	return e, nil
}
func (m *mockAPIRepo) FindAll(_ ...string) ([]model.API, error)                 { return nil, nil }
func (m *mockAPIRepo) FindByUUIDs(_ []string, _ ...string) ([]model.API, error) { return nil, nil }
func (m *mockAPIRepo) FindByID(_ any, _ ...string) (*model.API, error)          { return nil, nil }
func (m *mockAPIRepo) UpdateByUUID(_, _ any) (*model.API, error)                { return nil, nil }
func (m *mockAPIRepo) UpdateByID(_, _ any) (*model.API, error)                  { return nil, nil }
func (m *mockAPIRepo) DeleteByUUID(_ any) error                                 { return nil }
func (m *mockAPIRepo) DeleteByID(_ any) error                                   { return nil }
func (m *mockAPIRepo) Paginate(_ map[string]any, _, _ int, _ ...string) (*repository.PaginationResult[model.API], error) {
	return nil, nil
}
func (m *mockAPIRepo) FindByName(name string, tID int64) (*model.API, error) {
	if m.findByNameFn != nil {
		return m.findByNameFn(name, tID)
	}
	return nil, nil
}
func (m *mockAPIRepo) FindByIdentifier(_ string, _ int64) (*model.API, error) { return nil, nil }
func (m *mockAPIRepo) SetStatusByUUID(_ uuid.UUID, _ int64, _ string) error   { return nil }
func (m *mockAPIRepo) CountByServiceID(sID int64, tID int64) (int64, error) {
	if m.countByServiceIDFn != nil {
		return m.countByServiceIDFn(sID, tID)
	}
	return 0, nil
}
func (m *mockAPIRepo) DeleteByUUIDAndTenantID(id uuid.UUID, tID int64) error {
	if m.deleteByUUIDAndTenantIDFn != nil {
		return m.deleteByUUIDAndTenantIDFn(id, tID)
	}
	return nil
}

func (m *mockAPIRepo) FindByUUID(id any, p ...string) (*model.API, error) {
	if m.findByUUIDFn != nil {
		return m.findByUUIDFn(id, p...)
	}
	return nil, nil
}
func (m *mockAPIRepo) FindByUUIDAndTenantID(id uuid.UUID, tID int64) (*model.API, error) {
	if m.findByUUIDAndTenantIDFn != nil {
		return m.findByUUIDAndTenantIDFn(id, tID)
	}
	return nil, nil
}
func (m *mockAPIRepo) FindPaginated(f repository.APIRepositoryGetFilter) (*repository.PaginationResult[model.API], error) {
	if m.findPaginatedFn != nil {
		return m.findPaginatedFn(f)
	}
	return &repository.PaginationResult[model.API]{}, nil
}

// ---------------------------------------------------------------------------
// Mock: RoleRepository
// ---------------------------------------------------------------------------

type mockRoleRepo struct {
	findByUUIDFn                 func(id any, preloads ...string) (*model.Role, error)
	findByNameAndTenantIDFn      func(string, int64) (*model.Role, error)
	findPaginatedFn              func(repository.RoleRepositoryGetFilter) (*repository.PaginationResult[model.Role], error)
	getPermsByRoleUUIDFn         func(repository.RoleRepositoryGetPermissionsFilter) (*repository.PaginationResult[model.Permission], error)
	createOrUpdateFn             func(*model.Role) (*model.Role, error)
	deleteByUUIDFn               func(any) error
	findByUUIDsFn                func([]string, ...string) ([]model.Role, error)
	findRegisteredRoleForSetupFn func(int64) (*model.Role, error)
	findSuperAdminRoleForSetupFn func(int64) (*model.Role, error)
}

func (m *mockRoleRepo) WithTx(_ *gorm.DB) repository.RoleRepository { return m }
func (m *mockRoleRepo) Create(_ *model.Role) (*model.Role, error)   { return nil, nil }
func (m *mockRoleRepo) FindAll(_ ...string) ([]model.Role, error)   { return nil, nil }
func (m *mockRoleRepo) FindByUUIDs(ids []string, p ...string) ([]model.Role, error) {
	if m.findByUUIDsFn != nil {
		return m.findByUUIDsFn(ids, p...)
	}
	return nil, nil
}
func (m *mockRoleRepo) FindByID(_ any, _ ...string) (*model.Role, error) { return nil, nil }
func (m *mockRoleRepo) UpdateByUUID(_, _ any) (*model.Role, error)       { return nil, nil }
func (m *mockRoleRepo) UpdateByID(_, _ any) (*model.Role, error)         { return nil, nil }
func (m *mockRoleRepo) DeleteByID(_ any) error                           { return nil }
func (m *mockRoleRepo) Paginate(_ map[string]any, _, _ int, _ ...string) (*repository.PaginationResult[model.Role], error) {
	return nil, nil
}
func (m *mockRoleRepo) FindAllByTenantID(_ int64) ([]model.Role, error)  { return nil, nil }
func (m *mockRoleRepo) SetStatusByUUID(_ uuid.UUID, _ string) error      { return nil }
func (m *mockRoleRepo) SetDefaultStatusByUUID(_ uuid.UUID, _ bool) error { return nil }
func (m *mockRoleRepo) SetSystemStatusByUUID(_ uuid.UUID, _ bool) error  { return nil }
func (m *mockRoleRepo) FindRegisteredRoleForSetup(tID int64) (*model.Role, error) {
	if m.findRegisteredRoleForSetupFn != nil {
		return m.findRegisteredRoleForSetupFn(tID)
	}
	return nil, nil
}
func (m *mockRoleRepo) FindSuperAdminRoleForSetup(tID int64) (*model.Role, error) {
	if m.findSuperAdminRoleForSetupFn != nil {
		return m.findSuperAdminRoleForSetupFn(tID)
	}
	return nil, nil
}

func (m *mockRoleRepo) FindByUUID(id any, p ...string) (*model.Role, error) {
	if m.findByUUIDFn != nil {
		return m.findByUUIDFn(id, p...)
	}
	return nil, nil
}
func (m *mockRoleRepo) CreateOrUpdate(e *model.Role) (*model.Role, error) {
	if m.createOrUpdateFn != nil {
		return m.createOrUpdateFn(e)
	}
	return e, nil
}
func (m *mockRoleRepo) FindByNameAndTenantID(name string, tID int64) (*model.Role, error) {
	if m.findByNameAndTenantIDFn != nil {
		return m.findByNameAndTenantIDFn(name, tID)
	}
	return nil, nil
}
func (m *mockRoleRepo) FindPaginated(f repository.RoleRepositoryGetFilter) (*repository.PaginationResult[model.Role], error) {
	if m.findPaginatedFn != nil {
		return m.findPaginatedFn(f)
	}
	return &repository.PaginationResult[model.Role]{}, nil
}
func (m *mockRoleRepo) GetPermissionsByRoleUUID(f repository.RoleRepositoryGetPermissionsFilter) (*repository.PaginationResult[model.Permission], error) {
	if m.getPermsByRoleUUIDFn != nil {
		return m.getPermsByRoleUUIDFn(f)
	}
	return &repository.PaginationResult[model.Permission]{}, nil
}
func (m *mockRoleRepo) DeleteByUUID(id any) error {
	if m.deleteByUUIDFn != nil {
		return m.deleteByUUIDFn(id)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Mock: RolePermissionRepository
// ---------------------------------------------------------------------------

type mockRolePermissionRepo struct {
	createFn                    func(e *model.RolePermission) (*model.RolePermission, error)
	findByRoleAndPermissionFn   func(roleID, permissionID int64) (*model.RolePermission, error)
	removeByRoleAndPermissionFn func(roleID, permissionID int64) error
}

func (m *mockRolePermissionRepo) WithTx(_ *gorm.DB) repository.RolePermissionRepository { return m }
func (m *mockRolePermissionRepo) CreateOrUpdate(_ *model.RolePermission) (*model.RolePermission, error) {
	return nil, nil
}
func (m *mockRolePermissionRepo) FindAll(_ ...string) ([]model.RolePermission, error) {
	return nil, nil
}
func (m *mockRolePermissionRepo) FindByUUID(_ any, _ ...string) (*model.RolePermission, error) {
	return nil, nil
}
func (m *mockRolePermissionRepo) FindByUUIDs(_ []string, _ ...string) ([]model.RolePermission, error) {
	return nil, nil
}
func (m *mockRolePermissionRepo) FindByID(_ any, _ ...string) (*model.RolePermission, error) {
	return nil, nil
}
func (m *mockRolePermissionRepo) UpdateByUUID(_, _ any) (*model.RolePermission, error) {
	return nil, nil
}
func (m *mockRolePermissionRepo) UpdateByID(_, _ any) (*model.RolePermission, error) { return nil, nil }
func (m *mockRolePermissionRepo) DeleteByUUID(_ any) error                           { return nil }
func (m *mockRolePermissionRepo) DeleteByID(_ any) error                             { return nil }
func (m *mockRolePermissionRepo) Paginate(_ map[string]any, _, _ int, _ ...string) (*repository.PaginationResult[model.RolePermission], error) {
	return nil, nil
}
func (m *mockRolePermissionRepo) Assign(_ *model.RolePermission) (*model.RolePermission, error) {
	return nil, nil
}
func (m *mockRolePermissionRepo) FindAllByRoleID(_ int64) ([]model.RolePermission, error) {
	return nil, nil
}
func (m *mockRolePermissionRepo) FindAllByPermissionID(_ int64) ([]model.RolePermission, error) {
	return nil, nil
}
func (m *mockRolePermissionRepo) SetDefaultStatusByUUID(_ uuid.UUID, _ bool) error { return nil }

func (m *mockRolePermissionRepo) Create(e *model.RolePermission) (*model.RolePermission, error) {
	if m.createFn != nil {
		return m.createFn(e)
	}
	return e, nil
}
func (m *mockRolePermissionRepo) FindByRoleAndPermission(roleID, permID int64) (*model.RolePermission, error) {
	if m.findByRoleAndPermissionFn != nil {
		return m.findByRoleAndPermissionFn(roleID, permID)
	}
	return nil, nil
}
func (m *mockRolePermissionRepo) RemoveByRoleAndPermission(roleID, permID int64) error {
	if m.removeByRoleAndPermissionFn != nil {
		return m.removeByRoleAndPermissionFn(roleID, permID)
	}
	return nil
}

// mockClientRepo is defined in login_service_test.go and shared across all service tests.

// ---------------------------------------------------------------------------
// Mock: ServiceRepository
// ---------------------------------------------------------------------------

type mockServiceRepo struct {
	findByUUIDFn               func(id any, preloads ...string) (*model.Service, error)
	findByNameFn               func(name string) (*model.Service, error)
	findByNameAndTenantIDFn    func(name string, tenantID int64) (*model.Service, error)
	createOrUpdateFn           func(*model.Service) (*model.Service, error)
	findServicesByPolicyUUIDFn func(uuid.UUID, repository.ServiceRepositoryGetFilter) (*repository.PaginationResult[model.Service], error)
	countPoliciesByServiceIDFn func(int64) (int64, error)
	findPaginatedFn            func(repository.ServiceRepositoryGetFilter) (*repository.PaginationResult[model.Service], error)
	deleteByUUIDFn             func(any) error
}

func (m *mockServiceRepo) WithTx(_ *gorm.DB) repository.ServiceRepository  { return m }
func (m *mockServiceRepo) Create(_ *model.Service) (*model.Service, error) { return nil, nil }
func (m *mockServiceRepo) CreateOrUpdate(e *model.Service) (*model.Service, error) {
	if m.createOrUpdateFn != nil {
		return m.createOrUpdateFn(e)
	}
	return e, nil
}
func (m *mockServiceRepo) FindAll(_ ...string) ([]model.Service, error) { return nil, nil }
func (m *mockServiceRepo) FindByUUIDs(_ []string, _ ...string) ([]model.Service, error) {
	return nil, nil
}
func (m *mockServiceRepo) FindByID(_ any, _ ...string) (*model.Service, error) { return nil, nil }
func (m *mockServiceRepo) UpdateByUUID(_, _ any) (*model.Service, error)       { return nil, nil }
func (m *mockServiceRepo) UpdateByID(_, _ any) (*model.Service, error)         { return nil, nil }
func (m *mockServiceRepo) DeleteByUUID(id any) error {
	if m.deleteByUUIDFn != nil {
		return m.deleteByUUIDFn(id)
	}
	return nil
}
func (m *mockServiceRepo) DeleteByID(_ any) error { return nil }
func (m *mockServiceRepo) Paginate(_ map[string]any, _, _ int, _ ...string) (*repository.PaginationResult[model.Service], error) {
	return nil, nil
}
func (m *mockServiceRepo) FindByNameAndTenantID(name string, tenantID int64) (*model.Service, error) {
	if m.findByNameAndTenantIDFn != nil {
		return m.findByNameAndTenantIDFn(name, tenantID)
	}
	return nil, nil
}
func (m *mockServiceRepo) FindByTenantID(_ int64) ([]model.Service, error) { return nil, nil }
func (m *mockServiceRepo) FindPaginated(f repository.ServiceRepositoryGetFilter) (*repository.PaginationResult[model.Service], error) {
	if m.findPaginatedFn != nil {
		return m.findPaginatedFn(f)
	}
	return &repository.PaginationResult[model.Service]{}, nil
}
func (m *mockServiceRepo) FindServicesByPolicyUUID(id uuid.UUID, f repository.ServiceRepositoryGetFilter) (*repository.PaginationResult[model.Service], error) {
	if m.findServicesByPolicyUUIDFn != nil {
		return m.findServicesByPolicyUUIDFn(id, f)
	}
	return &repository.PaginationResult[model.Service]{}, nil
}
func (m *mockServiceRepo) SetStatusByUUID(_ uuid.UUID, _ string) error { return nil }
func (m *mockServiceRepo) CountPoliciesByServiceID(sID int64) (int64, error) {
	if m.countPoliciesByServiceIDFn != nil {
		return m.countPoliciesByServiceIDFn(sID)
	}
	return 0, nil
}

func (m *mockServiceRepo) FindByUUID(id any, p ...string) (*model.Service, error) {
	if m.findByUUIDFn != nil {
		return m.findByUUIDFn(id, p...)
	}
	return nil, nil
}
func (m *mockServiceRepo) FindByName(name string) (*model.Service, error) {
	if m.findByNameFn != nil {
		return m.findByNameFn(name)
	}
	return nil, nil
}

// ---------------------------------------------------------------------------
// Mock: TenantServiceRepository
// ---------------------------------------------------------------------------

type mockTenantServiceRepo struct {
	findByTenantAndServiceFn func(tenantID int64, serviceID int64) (*model.TenantService, error)
	createOrUpdateFn         func(*model.TenantService) (*model.TenantService, error)
}

func (m *mockTenantServiceRepo) WithTx(_ *gorm.DB) repository.TenantServiceRepository { return m }
func (m *mockTenantServiceRepo) Create(_ *model.TenantService) (*model.TenantService, error) {
	return nil, nil
}
func (m *mockTenantServiceRepo) CreateOrUpdate(e *model.TenantService) (*model.TenantService, error) {
	if m.createOrUpdateFn != nil {
		return m.createOrUpdateFn(e)
	}
	return e, nil
}
func (m *mockTenantServiceRepo) FindAll(_ ...string) ([]model.TenantService, error) { return nil, nil }
func (m *mockTenantServiceRepo) FindByUUID(_ any, _ ...string) (*model.TenantService, error) {
	return nil, nil
}
func (m *mockTenantServiceRepo) FindByUUIDs(_ []string, _ ...string) ([]model.TenantService, error) {
	return nil, nil
}
func (m *mockTenantServiceRepo) FindByID(_ any, _ ...string) (*model.TenantService, error) {
	return nil, nil
}
func (m *mockTenantServiceRepo) UpdateByUUID(_, _ any) (*model.TenantService, error) { return nil, nil }
func (m *mockTenantServiceRepo) UpdateByID(_, _ any) (*model.TenantService, error)   { return nil, nil }
func (m *mockTenantServiceRepo) DeleteByUUID(_ any) error                            { return nil }
func (m *mockTenantServiceRepo) DeleteByID(_ any) error                              { return nil }
func (m *mockTenantServiceRepo) Paginate(_ map[string]any, _, _ int, _ ...string) (*repository.PaginationResult[model.TenantService], error) {
	return nil, nil
}
func (m *mockTenantServiceRepo) FindPaginated(_ repository.TenantServiceRepositoryGetFilter) (*repository.PaginationResult[model.TenantService], error) {
	return &repository.PaginationResult[model.TenantService]{}, nil
}
func (m *mockTenantServiceRepo) DeleteByTenantAndService(_ int64, _ int64) error { return nil }

func (m *mockTenantServiceRepo) FindByTenantAndService(tenantID int64, serviceID int64) (*model.TenantService, error) {
	if m.findByTenantAndServiceFn != nil {
		return m.findByTenantAndServiceFn(tenantID, serviceID)
	}
	return &model.TenantService{}, nil
}

// ---------------------------------------------------------------------------
// Mock: LoginTemplateRepository (no WithTx)
// ---------------------------------------------------------------------------

type mockLoginTemplateRepo struct {
	createFn                func(e *model.LoginTemplate) (*model.LoginTemplate, error)
	findByUUIDAndTenantIDFn func(uuid.UUID, int64, ...string) (*model.LoginTemplate, error)
	findPaginatedFn         func(repository.LoginTemplateRepositoryGetFilter) (*repository.PaginationResult[model.LoginTemplate], error)
	updateByUUIDFn          func(any, any) (*model.LoginTemplate, error)
	deleteByUUIDFn          func(any) error
}

func (m *mockLoginTemplateRepo) CreateOrUpdate(e *model.LoginTemplate) (*model.LoginTemplate, error) {
	return nil, nil
}
func (m *mockLoginTemplateRepo) FindAll(p ...string) ([]model.LoginTemplate, error) { return nil, nil }
func (m *mockLoginTemplateRepo) FindByUUID(id any, p ...string) (*model.LoginTemplate, error) {
	return nil, nil
}
func (m *mockLoginTemplateRepo) FindByUUIDs(ids []string, p ...string) ([]model.LoginTemplate, error) {
	return nil, nil
}
func (m *mockLoginTemplateRepo) FindByID(id any, p ...string) (*model.LoginTemplate, error) {
	return nil, nil
}
func (m *mockLoginTemplateRepo) UpdateByID(id, data any) (*model.LoginTemplate, error) {
	return nil, nil
}
func (m *mockLoginTemplateRepo) DeleteByID(id any) error { return nil }
func (m *mockLoginTemplateRepo) Paginate(_ map[string]any, _, _ int, _ ...string) (*repository.PaginationResult[model.LoginTemplate], error) {
	return nil, nil
}
func (m *mockLoginTemplateRepo) FindByName(_ string) (*model.LoginTemplate, error) { return nil, nil }

func (m *mockLoginTemplateRepo) Create(e *model.LoginTemplate) (*model.LoginTemplate, error) {
	if m.createFn != nil {
		return m.createFn(e)
	}
	return e, nil
}
func (m *mockLoginTemplateRepo) FindByUUIDAndTenantID(id uuid.UUID, tID int64, p ...string) (*model.LoginTemplate, error) {
	if m.findByUUIDAndTenantIDFn != nil {
		return m.findByUUIDAndTenantIDFn(id, tID, p...)
	}
	return nil, nil
}
func (m *mockLoginTemplateRepo) FindPaginated(f repository.LoginTemplateRepositoryGetFilter) (*repository.PaginationResult[model.LoginTemplate], error) {
	if m.findPaginatedFn != nil {
		return m.findPaginatedFn(f)
	}
	return &repository.PaginationResult[model.LoginTemplate]{}, nil
}
func (m *mockLoginTemplateRepo) UpdateByUUID(id, data any) (*model.LoginTemplate, error) {
	if m.updateByUUIDFn != nil {
		return m.updateByUUIDFn(id, data)
	}
	return nil, nil
}
func (m *mockLoginTemplateRepo) DeleteByUUID(id any) error {
	if m.deleteByUUIDFn != nil {
		return m.deleteByUUIDFn(id)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Mock: SmsTemplateRepository (no WithTx)
// ---------------------------------------------------------------------------

type mockSmsTemplateRepo struct {
	createFn                func(e *model.SmsTemplate) (*model.SmsTemplate, error)
	findByUUIDAndTenantIDFn func(string, int64) (*model.SmsTemplate, error)
	findPaginatedFn         func(repository.SmsTemplateRepositoryGetFilter) (*repository.PaginationResult[model.SmsTemplate], error)
	updateByUUIDFn          func(any, any) (*model.SmsTemplate, error)
	deleteByUUIDFn          func(any) error
}

func (m *mockSmsTemplateRepo) CreateOrUpdate(e *model.SmsTemplate) (*model.SmsTemplate, error) {
	return nil, nil
}
func (m *mockSmsTemplateRepo) FindAll(p ...string) ([]model.SmsTemplate, error) { return nil, nil }
func (m *mockSmsTemplateRepo) FindByUUID(id any, p ...string) (*model.SmsTemplate, error) {
	return nil, nil
}
func (m *mockSmsTemplateRepo) FindByUUIDs(ids []string, p ...string) ([]model.SmsTemplate, error) {
	return nil, nil
}
func (m *mockSmsTemplateRepo) FindByID(id any, p ...string) (*model.SmsTemplate, error) {
	return nil, nil
}
func (m *mockSmsTemplateRepo) UpdateByID(id, data any) (*model.SmsTemplate, error) { return nil, nil }
func (m *mockSmsTemplateRepo) DeleteByID(id any) error                             { return nil }
func (m *mockSmsTemplateRepo) Paginate(_ map[string]any, _, _ int, _ ...string) (*repository.PaginationResult[model.SmsTemplate], error) {
	return nil, nil
}
func (m *mockSmsTemplateRepo) FindByName(_ string) (*model.SmsTemplate, error) { return nil, nil }

func (m *mockSmsTemplateRepo) Create(e *model.SmsTemplate) (*model.SmsTemplate, error) {
	if m.createFn != nil {
		return m.createFn(e)
	}
	return e, nil
}
func (m *mockSmsTemplateRepo) FindByUUIDAndTenantID(id string, tID int64) (*model.SmsTemplate, error) {
	if m.findByUUIDAndTenantIDFn != nil {
		return m.findByUUIDAndTenantIDFn(id, tID)
	}
	return nil, nil
}
func (m *mockSmsTemplateRepo) FindPaginated(f repository.SmsTemplateRepositoryGetFilter) (*repository.PaginationResult[model.SmsTemplate], error) {
	if m.findPaginatedFn != nil {
		return m.findPaginatedFn(f)
	}
	return &repository.PaginationResult[model.SmsTemplate]{}, nil
}
func (m *mockSmsTemplateRepo) UpdateByUUID(id, data any) (*model.SmsTemplate, error) {
	if m.updateByUUIDFn != nil {
		return m.updateByUUIDFn(id, data)
	}
	return nil, nil
}
func (m *mockSmsTemplateRepo) DeleteByUUID(id any) error {
	if m.deleteByUUIDFn != nil {
		return m.deleteByUUIDFn(id)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Mock: IpRestrictionRuleRepository
// ---------------------------------------------------------------------------

type mockIpRestrictionRuleRepo struct {
	findByUUIDFn    func(id any, preloads ...string) (*model.IpRestrictionRule, error)
	findPaginatedFn func(repository.IpRestrictionRuleRepositoryGetFilter) (*repository.PaginationResult[model.IpRestrictionRule], error)
	createFn        func(e *model.IpRestrictionRule) (*model.IpRestrictionRule, error)
	updateByUUIDFn  func(any, any) (*model.IpRestrictionRule, error)
	deleteByUUIDFn  func(any) error
}

func (m *mockIpRestrictionRuleRepo) WithTx(_ *gorm.DB) repository.IpRestrictionRuleRepository {
	return m
}
func (m *mockIpRestrictionRuleRepo) CreateOrUpdate(e *model.IpRestrictionRule) (*model.IpRestrictionRule, error) {
	return nil, nil
}
func (m *mockIpRestrictionRuleRepo) FindAll(_ ...string) ([]model.IpRestrictionRule, error) {
	return nil, nil
}
func (m *mockIpRestrictionRuleRepo) FindByUUIDs(_ []string, _ ...string) ([]model.IpRestrictionRule, error) {
	return nil, nil
}
func (m *mockIpRestrictionRuleRepo) FindByID(_ any, _ ...string) (*model.IpRestrictionRule, error) {
	return nil, nil
}
func (m *mockIpRestrictionRuleRepo) UpdateByID(_, _ any) (*model.IpRestrictionRule, error) {
	return nil, nil
}
func (m *mockIpRestrictionRuleRepo) DeleteByID(_ any) error { return nil }
func (m *mockIpRestrictionRuleRepo) Paginate(_ map[string]any, _, _ int, _ ...string) (*repository.PaginationResult[model.IpRestrictionRule], error) {
	return nil, nil
}
func (m *mockIpRestrictionRuleRepo) FindByTenantID(_ int64) ([]model.IpRestrictionRule, error) {
	return nil, nil
}
func (m *mockIpRestrictionRuleRepo) FindByTenantIDAndStatus(_ int64, _ string) ([]model.IpRestrictionRule, error) {
	return nil, nil
}
func (m *mockIpRestrictionRuleRepo) FindByTenantIDAndType(_ int64, _ string) ([]model.IpRestrictionRule, error) {
	return nil, nil
}

func (m *mockIpRestrictionRuleRepo) FindByUUID(id any, p ...string) (*model.IpRestrictionRule, error) {
	if m.findByUUIDFn != nil {
		return m.findByUUIDFn(id, p...)
	}
	return nil, nil
}
func (m *mockIpRestrictionRuleRepo) FindPaginated(f repository.IpRestrictionRuleRepositoryGetFilter) (*repository.PaginationResult[model.IpRestrictionRule], error) {
	if m.findPaginatedFn != nil {
		return m.findPaginatedFn(f)
	}
	return &repository.PaginationResult[model.IpRestrictionRule]{}, nil
}
func (m *mockIpRestrictionRuleRepo) Create(e *model.IpRestrictionRule) (*model.IpRestrictionRule, error) {
	if m.createFn != nil {
		return m.createFn(e)
	}
	return e, nil
}
func (m *mockIpRestrictionRuleRepo) UpdateByUUID(id, data any) (*model.IpRestrictionRule, error) {
	if m.updateByUUIDFn != nil {
		return m.updateByUUIDFn(id, data)
	}
	return nil, nil
}
func (m *mockIpRestrictionRuleRepo) DeleteByUUID(id any) error {
	if m.deleteByUUIDFn != nil {
		return m.deleteByUUIDFn(id)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Mock: SecuritySettingRepository
// ---------------------------------------------------------------------------

type mockSecuritySettingRepo struct {
	findByTenantIDFn   func(tenantID int64) (*model.SecuritySetting, error)
	updateByUUIDFn     func(any, any) (*model.SecuritySetting, error)
	createFn           func(*model.SecuritySetting) (*model.SecuritySetting, error)
	createOrUpdateFn   func(*model.SecuritySetting) (*model.SecuritySetting, error)
	findByUUIDFn       func(any, ...string) (*model.SecuritySetting, error)
	incrementVersionFn func(int64) error
}

func (m *mockSecuritySettingRepo) WithTx(_ *gorm.DB) repository.SecuritySettingRepository { return m }
func (m *mockSecuritySettingRepo) Create(e *model.SecuritySetting) (*model.SecuritySetting, error) {
	if m.createFn != nil {
		return m.createFn(e)
	}
	return e, nil
}
func (m *mockSecuritySettingRepo) CreateOrUpdate(e *model.SecuritySetting) (*model.SecuritySetting, error) {
	if m.createOrUpdateFn != nil {
		return m.createOrUpdateFn(e)
	}
	return e, nil
}
func (m *mockSecuritySettingRepo) FindAll(_ ...string) ([]model.SecuritySetting, error) {
	return nil, nil
}
func (m *mockSecuritySettingRepo) FindByUUID(id any, p ...string) (*model.SecuritySetting, error) {
	if m.findByUUIDFn != nil {
		return m.findByUUIDFn(id, p...)
	}
	return nil, nil
}
func (m *mockSecuritySettingRepo) FindByUUIDs(_ []string, _ ...string) ([]model.SecuritySetting, error) {
	return nil, nil
}
func (m *mockSecuritySettingRepo) FindByID(_ any, _ ...string) (*model.SecuritySetting, error) {
	return nil, nil
}
func (m *mockSecuritySettingRepo) UpdateByID(_, _ any) (*model.SecuritySetting, error) {
	return nil, nil
}
func (m *mockSecuritySettingRepo) DeleteByUUID(_ any) error { return nil }
func (m *mockSecuritySettingRepo) DeleteByID(_ any) error   { return nil }
func (m *mockSecuritySettingRepo) Paginate(_ map[string]any, _, _ int, _ ...string) (*repository.PaginationResult[model.SecuritySetting], error) {
	return nil, nil
}
func (m *mockSecuritySettingRepo) FindPaginated(_ repository.SecuritySettingRepositoryGetFilter) (*repository.PaginationResult[model.SecuritySetting], error) {
	return nil, nil
}
func (m *mockSecuritySettingRepo) IncrementVersion(id int64) error {
	if m.incrementVersionFn != nil {
		return m.incrementVersionFn(id)
	}
	return nil
}

func (m *mockSecuritySettingRepo) FindByTenantID(tID int64) (*model.SecuritySetting, error) {
	if m.findByTenantIDFn != nil {
		return m.findByTenantIDFn(tID)
	}
	return nil, nil
}
func (m *mockSecuritySettingRepo) UpdateByUUID(id, data any) (*model.SecuritySetting, error) {
	if m.updateByUUIDFn != nil {
		return m.updateByUUIDFn(id, data)
	}
	return nil, nil
}

// ---------------------------------------------------------------------------
// Mock: SecuritySettingsAuditRepository
// ---------------------------------------------------------------------------

type mockSecuritySettingsAuditRepo struct {
	createFn func(e *model.SecuritySettingsAudit) (*model.SecuritySettingsAudit, error)
}

func (m *mockSecuritySettingsAuditRepo) WithTx(_ *gorm.DB) repository.SecuritySettingsAuditRepository {
	return m
}
func (m *mockSecuritySettingsAuditRepo) CreateOrUpdate(e *model.SecuritySettingsAudit) (*model.SecuritySettingsAudit, error) {
	return nil, nil
}
func (m *mockSecuritySettingsAuditRepo) FindAll(_ ...string) ([]model.SecuritySettingsAudit, error) {
	return nil, nil
}
func (m *mockSecuritySettingsAuditRepo) FindByUUID(_ any, _ ...string) (*model.SecuritySettingsAudit, error) {
	return nil, nil
}
func (m *mockSecuritySettingsAuditRepo) FindByUUIDs(_ []string, _ ...string) ([]model.SecuritySettingsAudit, error) {
	return nil, nil
}
func (m *mockSecuritySettingsAuditRepo) FindByID(_ any, _ ...string) (*model.SecuritySettingsAudit, error) {
	return nil, nil
}
func (m *mockSecuritySettingsAuditRepo) UpdateByUUID(_, _ any) (*model.SecuritySettingsAudit, error) {
	return nil, nil
}
func (m *mockSecuritySettingsAuditRepo) UpdateByID(_, _ any) (*model.SecuritySettingsAudit, error) {
	return nil, nil
}
func (m *mockSecuritySettingsAuditRepo) DeleteByUUID(_ any) error { return nil }
func (m *mockSecuritySettingsAuditRepo) DeleteByID(_ any) error   { return nil }
func (m *mockSecuritySettingsAuditRepo) Paginate(_ map[string]any, _, _ int, _ ...string) (*repository.PaginationResult[model.SecuritySettingsAudit], error) {
	return nil, nil
}
func (m *mockSecuritySettingsAuditRepo) FindBySecuritySettingID(_ int64) ([]model.SecuritySettingsAudit, error) {
	return nil, nil
}
func (m *mockSecuritySettingsAuditRepo) FindByTenantID(_ int64) ([]model.SecuritySettingsAudit, error) {
	return nil, nil
}
func (m *mockSecuritySettingsAuditRepo) FindPaginated(_ repository.SecuritySettingsAuditRepositoryGetFilter) (*repository.PaginationResult[model.SecuritySettingsAudit], error) {
	return nil, nil
}
func (m *mockSecuritySettingsAuditRepo) Create(e *model.SecuritySettingsAudit) (*model.SecuritySettingsAudit, error) {
	if m.createFn != nil {
		return m.createFn(e)
	}
	return e, nil
}

// ---------------------------------------------------------------------------
// Mock: ProfileRepository
// ---------------------------------------------------------------------------

type mockProfileRepo struct {
	findByUUIDFn          func(id any, preloads ...string) (*model.Profile, error)
	findByUserIDFn        func(userID int64) (*model.Profile, error)
	findDefaultByUserIDFn func(userID int64) (*model.Profile, error)
	findAllByUserIDFn     func(repository.ProfileRepositoryGetFilter) (*repository.PaginationResult[model.Profile], error)
	createFn              func(*model.Profile) (*model.Profile, error)
	createOrUpdateFn      func(*model.Profile) (*model.Profile, error)
	updateByUserIDFn      func(int64, *model.Profile) error
	unsetDefaultFn        func(userID int64) error
	deleteByUUIDFn        func(any) error
	deleteByUserIDFn      func(userID int64) error
}

func (m *mockProfileRepo) WithTx(_ *gorm.DB) repository.ProfileRepository { return m }
func (m *mockProfileRepo) Create(e *model.Profile) (*model.Profile, error) {
	if m.createFn != nil {
		return m.createFn(e)
	}
	return e, nil
}
func (m *mockProfileRepo) FindAll(_ ...string) ([]model.Profile, error) { return nil, nil }
func (m *mockProfileRepo) FindByUUIDs(_ []string, _ ...string) ([]model.Profile, error) {
	return nil, nil
}
func (m *mockProfileRepo) FindByID(_ any, _ ...string) (*model.Profile, error) { return nil, nil }
func (m *mockProfileRepo) UpdateByUUID(_, _ any) (*model.Profile, error)       { return nil, nil }
func (m *mockProfileRepo) UpdateByID(_, _ any) (*model.Profile, error)         { return nil, nil }
func (m *mockProfileRepo) DeleteByID(_ any) error                              { return nil }
func (m *mockProfileRepo) Paginate(_ map[string]any, _, _ int, _ ...string) (*repository.PaginationResult[model.Profile], error) {
	return nil, nil
}
func (m *mockProfileRepo) UpdateByUserID(uID int64, p *model.Profile) error {
	if m.updateByUserIDFn != nil {
		return m.updateByUserIDFn(uID, p)
	}
	return nil
}

func (m *mockProfileRepo) FindByUUID(id any, p ...string) (*model.Profile, error) {
	if m.findByUUIDFn != nil {
		return m.findByUUIDFn(id, p...)
	}
	return nil, nil
}
func (m *mockProfileRepo) FindByUserID(uID int64) (*model.Profile, error) {
	if m.findByUserIDFn != nil {
		return m.findByUserIDFn(uID)
	}
	return nil, nil
}
func (m *mockProfileRepo) FindDefaultByUserID(uID int64) (*model.Profile, error) {
	if m.findDefaultByUserIDFn != nil {
		return m.findDefaultByUserIDFn(uID)
	}
	return nil, nil
}
func (m *mockProfileRepo) FindAllByUserID(f repository.ProfileRepositoryGetFilter) (*repository.PaginationResult[model.Profile], error) {
	if m.findAllByUserIDFn != nil {
		return m.findAllByUserIDFn(f)
	}
	return &repository.PaginationResult[model.Profile]{}, nil
}
func (m *mockProfileRepo) CreateOrUpdate(e *model.Profile) (*model.Profile, error) {
	if m.createOrUpdateFn != nil {
		return m.createOrUpdateFn(e)
	}
	return e, nil
}
func (m *mockProfileRepo) UnsetDefaultProfiles(uID int64) error {
	if m.unsetDefaultFn != nil {
		return m.unsetDefaultFn(uID)
	}
	return nil
}
func (m *mockProfileRepo) DeleteByUUID(id any) error {
	if m.deleteByUUIDFn != nil {
		return m.deleteByUUIDFn(id)
	}
	return nil
}
func (m *mockProfileRepo) DeleteByUserID(uID int64) error {
	if m.deleteByUserIDFn != nil {
		return m.deleteByUserIDFn(uID)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Mock: UserSettingRepository
// ---------------------------------------------------------------------------

type mockUserSettingRepo struct {
	findByUUIDFn     func(id any, preloads ...string) (*model.UserSetting, error)
	findByUserIDFn   func(userID int64) (*model.UserSetting, error)
	createFn         func(*model.UserSetting) (*model.UserSetting, error)
	updateByUserIDFn func(int64, *model.UserSetting) error
	deleteByUUIDFn   func(any) error
}

func (m *mockUserSettingRepo) WithTx(_ *gorm.DB) repository.UserSettingRepository { return m }
func (m *mockUserSettingRepo) CreateOrUpdate(e *model.UserSetting) (*model.UserSetting, error) {
	return nil, nil
}
func (m *mockUserSettingRepo) FindAll(_ ...string) ([]model.UserSetting, error) { return nil, nil }
func (m *mockUserSettingRepo) FindByUUIDs(_ []string, _ ...string) ([]model.UserSetting, error) {
	return nil, nil
}
func (m *mockUserSettingRepo) FindByID(_ any, _ ...string) (*model.UserSetting, error) {
	return nil, nil
}
func (m *mockUserSettingRepo) UpdateByUUID(_, _ any) (*model.UserSetting, error) { return nil, nil }
func (m *mockUserSettingRepo) UpdateByID(_, _ any) (*model.UserSetting, error)   { return nil, nil }
func (m *mockUserSettingRepo) DeleteByID(_ any) error                            { return nil }
func (m *mockUserSettingRepo) Paginate(_ map[string]any, _, _ int, _ ...string) (*repository.PaginationResult[model.UserSetting], error) {
	return nil, nil
}
func (m *mockUserSettingRepo) UpdateByUserID(uID int64, e *model.UserSetting) error {
	if m.updateByUserIDFn != nil {
		return m.updateByUserIDFn(uID, e)
	}
	return nil
}
func (m *mockUserSettingRepo) DeleteByUserID(_ int64) error { return nil }

func (m *mockUserSettingRepo) FindByUUID(id any, p ...string) (*model.UserSetting, error) {
	if m.findByUUIDFn != nil {
		return m.findByUUIDFn(id, p...)
	}
	return nil, nil
}
func (m *mockUserSettingRepo) FindByUserID(uID int64) (*model.UserSetting, error) {
	if m.findByUserIDFn != nil {
		return m.findByUserIDFn(uID)
	}
	return nil, nil
}
func (m *mockUserSettingRepo) Create(e *model.UserSetting) (*model.UserSetting, error) {
	if m.createFn != nil {
		return m.createFn(e)
	}
	return e, nil
}
func (m *mockUserSettingRepo) DeleteByUUID(id any) error {
	if m.deleteByUUIDFn != nil {
		return m.deleteByUUIDFn(id)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Mock: TenantMemberRepository
// ---------------------------------------------------------------------------

type mockTenantMemberRepo struct {
	findByTenantMemberUUIDFn func(uuid.UUID) (*model.TenantMember, error)
	findByTenantAndUserFn    func(tenantID int64, userID int64) (*model.TenantMember, error)
	findAllByTenantFn        func(tenantID int64) ([]model.TenantMember, error)
	findAllByUserFn          func(userID int64) ([]model.TenantMember, error)
	createFn                 func(*model.TenantMember) (*model.TenantMember, error)
	createOrUpdateFn         func(*model.TenantMember) (*model.TenantMember, error)
	deleteByUUIDFn           func(any) error
}

func (m *mockTenantMemberRepo) WithTx(_ *gorm.DB) repository.TenantMemberRepository { return m }
func (m *mockTenantMemberRepo) CreateOrUpdate(e *model.TenantMember) (*model.TenantMember, error) {
	if m.createOrUpdateFn != nil {
		return m.createOrUpdateFn(e)
	}
	return e, nil
}
func (m *mockTenantMemberRepo) FindAll(_ ...string) ([]model.TenantMember, error) { return nil, nil }
func (m *mockTenantMemberRepo) FindByUUID(_ any, _ ...string) (*model.TenantMember, error) {
	return nil, nil
}
func (m *mockTenantMemberRepo) FindByUUIDs(_ []string, _ ...string) ([]model.TenantMember, error) {
	return nil, nil
}
func (m *mockTenantMemberRepo) FindByID(_ any, _ ...string) (*model.TenantMember, error) {
	return nil, nil
}
func (m *mockTenantMemberRepo) UpdateByUUID(_, _ any) (*model.TenantMember, error) { return nil, nil }
func (m *mockTenantMemberRepo) UpdateByID(_, _ any) (*model.TenantMember, error)   { return nil, nil }
func (m *mockTenantMemberRepo) DeleteByID(_ any) error                             { return nil }
func (m *mockTenantMemberRepo) Paginate(_ map[string]any, _, _ int, _ ...string) (*repository.PaginationResult[model.TenantMember], error) {
	return nil, nil
}
func (m *mockTenantMemberRepo) FindAllByUser(uID int64) ([]model.TenantMember, error) {
	if m.findAllByUserFn != nil {
		return m.findAllByUserFn(uID)
	}
	return nil, nil
}

func (m *mockTenantMemberRepo) FindByTenantMemberUUID(id uuid.UUID) (*model.TenantMember, error) {
	if m.findByTenantMemberUUIDFn != nil {
		return m.findByTenantMemberUUIDFn(id)
	}
	return nil, nil
}
func (m *mockTenantMemberRepo) FindByTenantAndUser(tID, uID int64) (*model.TenantMember, error) {
	if m.findByTenantAndUserFn != nil {
		return m.findByTenantAndUserFn(tID, uID)
	}
	return nil, nil
}
func (m *mockTenantMemberRepo) FindAllByTenant(tID int64) ([]model.TenantMember, error) {
	if m.findAllByTenantFn != nil {
		return m.findAllByTenantFn(tID)
	}
	return nil, nil
}
func (m *mockTenantMemberRepo) Create(e *model.TenantMember) (*model.TenantMember, error) {
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

// ---------------------------------------------------------------------------
// Mock: ServicePolicyRepository
// ---------------------------------------------------------------------------

type mockServicePolicyRepo struct {
	findByServiceAndPolicyFn func(serviceID int64, policyID int64) (*model.ServicePolicy, error)
	createFn                 func(*model.ServicePolicy) (*model.ServicePolicy, error)
	deleteByServiceAndPolicy func(serviceID int64, policyID int64) error
}

func (m *mockServicePolicyRepo) WithTx(_ *gorm.DB) repository.ServicePolicyRepository { return m }
func (m *mockServicePolicyRepo) CreateOrUpdate(e *model.ServicePolicy) (*model.ServicePolicy, error) {
	return nil, nil
}
func (m *mockServicePolicyRepo) FindAll(_ ...string) ([]model.ServicePolicy, error) { return nil, nil }
func (m *mockServicePolicyRepo) FindByUUID(_ any, _ ...string) (*model.ServicePolicy, error) {
	return nil, nil
}
func (m *mockServicePolicyRepo) FindByUUIDs(_ []string, _ ...string) ([]model.ServicePolicy, error) {
	return nil, nil
}
func (m *mockServicePolicyRepo) FindByID(_ any, _ ...string) (*model.ServicePolicy, error) {
	return nil, nil
}
func (m *mockServicePolicyRepo) UpdateByUUID(_, _ any) (*model.ServicePolicy, error) {
	return nil, nil
}
func (m *mockServicePolicyRepo) UpdateByID(_, _ any) (*model.ServicePolicy, error) { return nil, nil }
func (m *mockServicePolicyRepo) DeleteByUUID(_ any) error                          { return nil }
func (m *mockServicePolicyRepo) DeleteByID(_ any) error                            { return nil }
func (m *mockServicePolicyRepo) Paginate(_ map[string]any, _, _ int, _ ...string) (*repository.PaginationResult[model.ServicePolicy], error) {
	return nil, nil
}
func (m *mockServicePolicyRepo) FindPaginated(_ repository.ServicePolicyRepositoryGetFilter) (*repository.PaginationResult[model.ServicePolicy], error) {
	return nil, nil
}
func (m *mockServicePolicyRepo) FindPoliciesByServiceID(_ int64) ([]model.Policy, error) {
	return nil, nil
}
func (m *mockServicePolicyRepo) FindServicesByPolicyID(_ int64) ([]model.Service, error) {
	return nil, nil
}

func (m *mockServicePolicyRepo) FindByServiceAndPolicy(sID, pID int64) (*model.ServicePolicy, error) {
	if m.findByServiceAndPolicyFn != nil {
		return m.findByServiceAndPolicyFn(sID, pID)
	}
	return nil, nil
}
func (m *mockServicePolicyRepo) Create(e *model.ServicePolicy) (*model.ServicePolicy, error) {
	if m.createFn != nil {
		return m.createFn(e)
	}
	return e, nil
}
func (m *mockServicePolicyRepo) DeleteByServiceAndPolicy(sID, pID int64) error {
	if m.deleteByServiceAndPolicy != nil {
		return m.deleteByServiceAndPolicy(sID, pID)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Mock: PolicyRepository
// ---------------------------------------------------------------------------

type mockPolicyRepo struct {
	findByUUIDAndTenantIDFn func(uuid.UUID, int64) (*model.Policy, error)
	findByNameFn            func(string, int64) (*model.Policy, error)
	findByNameAndVersionFn  func(string, string, int64) (*model.Policy, error)
	findPaginatedFn         func(repository.PolicyRepositoryGetFilter) (*repository.PaginationResult[model.Policy], error)
	createFn                func(*model.Policy) (*model.Policy, error)
	updateByUUIDFn          func(any, any) (*model.Policy, error)
	setStatusByUUIDFn       func(uuid.UUID, int64, string) error
	deleteByUUIDAndTenantFn func(uuid.UUID, int64) error
}

func (m *mockPolicyRepo) WithTx(_ *gorm.DB) repository.PolicyRepository         { return m }
func (m *mockPolicyRepo) CreateOrUpdate(e *model.Policy) (*model.Policy, error) { return nil, nil }
func (m *mockPolicyRepo) FindAll(_ ...string) ([]model.Policy, error)           { return nil, nil }
func (m *mockPolicyRepo) FindByUUID(_ any, _ ...string) (*model.Policy, error)  { return nil, nil }
func (m *mockPolicyRepo) FindByUUIDs(_ []string, _ ...string) ([]model.Policy, error) {
	return nil, nil
}
func (m *mockPolicyRepo) FindByID(_ any, _ ...string) (*model.Policy, error) { return nil, nil }
func (m *mockPolicyRepo) UpdateByUUID(key, val any) (*model.Policy, error) {
	if m.updateByUUIDFn != nil {
		return m.updateByUUIDFn(key, val)
	}
	if p, ok := val.(*model.Policy); ok {
		return p, nil
	}
	return nil, nil
}
func (m *mockPolicyRepo) UpdateByID(_, _ any) (*model.Policy, error)         { return nil, nil }
func (m *mockPolicyRepo) DeleteByUUID(_ any) error                           { return nil }
func (m *mockPolicyRepo) DeleteByID(_ any) error                             { return nil }
func (m *mockPolicyRepo) FindSystemPolicies(_ int64) ([]model.Policy, error) { return nil, nil }
func (m *mockPolicyRepo) SetStatusByUUID(id uuid.UUID, tID int64, status string) error {
	if m.setStatusByUUIDFn != nil {
		return m.setStatusByUUIDFn(id, tID, status)
	}
	return nil
}
func (m *mockPolicyRepo) SetSystemStatusByUUID(_ uuid.UUID, _ int64, _ bool) error { return nil }
func (m *mockPolicyRepo) FindByNameAndVersion(name, version string, tID int64) (*model.Policy, error) {
	if m.findByNameAndVersionFn != nil {
		return m.findByNameAndVersionFn(name, version, tID)
	}
	return nil, nil
}
func (m *mockPolicyRepo) Paginate(_ map[string]any, _, _ int, _ ...string) (*repository.PaginationResult[model.Policy], error) {
	return nil, nil
}

func (m *mockPolicyRepo) FindByUUIDAndTenantID(id uuid.UUID, tID int64) (*model.Policy, error) {
	if m.findByUUIDAndTenantIDFn != nil {
		return m.findByUUIDAndTenantIDFn(id, tID)
	}
	return nil, nil
}
func (m *mockPolicyRepo) FindByName(name string, tID int64) (*model.Policy, error) {
	if m.findByNameFn != nil {
		return m.findByNameFn(name, tID)
	}
	return nil, nil
}
func (m *mockPolicyRepo) FindPaginated(f repository.PolicyRepositoryGetFilter) (*repository.PaginationResult[model.Policy], error) {
	if m.findPaginatedFn != nil {
		return m.findPaginatedFn(f)
	}
	return &repository.PaginationResult[model.Policy]{}, nil
}
func (m *mockPolicyRepo) Create(e *model.Policy) (*model.Policy, error) {
	if m.createFn != nil {
		return m.createFn(e)
	}
	return e, nil
}
func (m *mockPolicyRepo) DeleteByUUIDAndTenantID(id uuid.UUID, tID int64) error {
	if m.deleteByUUIDAndTenantFn != nil {
		return m.deleteByUUIDAndTenantFn(id, tID)
	}
	return nil
}

// Note: mockUserRepo, mockUserIdentityRepo, and mockIdentityProviderRepo are declared in login_service_test.go

// ---------------------------------------------------------------------------
// Mock: ClientURIRepository
// ---------------------------------------------------------------------------

type mockClientURIRepo struct {
	findByUUIDAndTenantIDFn   func(string, int64) (*model.ClientURI, error)
	findByURIAndTypeFn        func(string, string, int64, int64) (*model.ClientURI, error)
	findByClientIDAndTypeFn   func(int64, string, int64) ([]model.ClientURI, error)
	deleteByUUIDAndTenantIDFn func(string, int64) error
	createOrUpdateFn          func(*model.ClientURI) (*model.ClientURI, error)
}

func (m *mockClientURIRepo) WithTx(_ *gorm.DB) repository.ClientURIRepository    { return m }
func (m *mockClientURIRepo) Create(e *model.ClientURI) (*model.ClientURI, error) { return e, nil }
func (m *mockClientURIRepo) CreateOrUpdate(e *model.ClientURI) (*model.ClientURI, error) {
	if m.createOrUpdateFn != nil {
		return m.createOrUpdateFn(e)
	}
	return e, nil
}
func (m *mockClientURIRepo) FindAll(_ ...string) ([]model.ClientURI, error)          { return nil, nil }
func (m *mockClientURIRepo) FindByUUID(_ any, _ ...string) (*model.ClientURI, error) { return nil, nil }
func (m *mockClientURIRepo) FindByUUIDs(_ []string, _ ...string) ([]model.ClientURI, error) {
	return nil, nil
}
func (m *mockClientURIRepo) FindByID(_ any, _ ...string) (*model.ClientURI, error) { return nil, nil }
func (m *mockClientURIRepo) UpdateByUUID(_, _ any) (*model.ClientURI, error)       { return nil, nil }
func (m *mockClientURIRepo) UpdateByID(_, _ any) (*model.ClientURI, error)         { return nil, nil }
func (m *mockClientURIRepo) DeleteByUUID(_ any) error                              { return nil }
func (m *mockClientURIRepo) DeleteByID(_ any) error                                { return nil }
func (m *mockClientURIRepo) Paginate(_ map[string]any, _, _ int, _ ...string) (*repository.PaginationResult[model.ClientURI], error) {
	return nil, nil
}
func (m *mockClientURIRepo) FindByUUIDAndTenantID(id string, tID int64) (*model.ClientURI, error) {
	if m.findByUUIDAndTenantIDFn != nil {
		return m.findByUUIDAndTenantIDFn(id, tID)
	}
	return nil, nil
}
func (m *mockClientURIRepo) FindByURIAndType(uri, uriType string, clientID, tID int64) (*model.ClientURI, error) {
	if m.findByURIAndTypeFn != nil {
		return m.findByURIAndTypeFn(uri, uriType, clientID, tID)
	}
	return nil, nil
}
func (m *mockClientURIRepo) FindByClientIDAndType(clientID int64, uriType string, tID int64) ([]model.ClientURI, error) {
	if m.findByClientIDAndTypeFn != nil {
		return m.findByClientIDAndTypeFn(clientID, uriType, tID)
	}
	return nil, nil
}
func (m *mockClientURIRepo) DeleteByUUIDAndTenantID(id string, tID int64) error {
	if m.deleteByUUIDAndTenantIDFn != nil {
		return m.deleteByUUIDAndTenantIDFn(id, tID)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Mock: ClientPermissionRepository
// ---------------------------------------------------------------------------

type mockClientPermissionRepo struct {
	findByClientApiAndPermissionFn   func(int64, int64) (*model.ClientPermission, error)
	removeByClientApiAndPermissionFn func(int64, int64) error
	findByClientApiIDFn              func(int64) ([]model.ClientPermission, error)
	createFn                         func(*model.ClientPermission) (*model.ClientPermission, error)
}

func (m *mockClientPermissionRepo) WithTx(_ *gorm.DB) repository.ClientPermissionRepository {
	return m
}
func (m *mockClientPermissionRepo) Create(e *model.ClientPermission) (*model.ClientPermission, error) {
	if m.createFn != nil {
		return m.createFn(e)
	}
	return e, nil
}
func (m *mockClientPermissionRepo) CreateOrUpdate(e *model.ClientPermission) (*model.ClientPermission, error) {
	return e, nil
}
func (m *mockClientPermissionRepo) FindAll(_ ...string) ([]model.ClientPermission, error) {
	return nil, nil
}
func (m *mockClientPermissionRepo) FindByUUID(_ any, _ ...string) (*model.ClientPermission, error) {
	return nil, nil
}
func (m *mockClientPermissionRepo) FindByUUIDs(_ []string, _ ...string) ([]model.ClientPermission, error) {
	return nil, nil
}
func (m *mockClientPermissionRepo) FindByID(_ any, _ ...string) (*model.ClientPermission, error) {
	return nil, nil
}
func (m *mockClientPermissionRepo) UpdateByUUID(_, _ any) (*model.ClientPermission, error) {
	return nil, nil
}
func (m *mockClientPermissionRepo) UpdateByID(_, _ any) (*model.ClientPermission, error) {
	return nil, nil
}
func (m *mockClientPermissionRepo) DeleteByUUID(_ any) error { return nil }
func (m *mockClientPermissionRepo) DeleteByID(_ any) error   { return nil }
func (m *mockClientPermissionRepo) Paginate(_ map[string]any, _, _ int, _ ...string) (*repository.PaginationResult[model.ClientPermission], error) {
	return nil, nil
}
func (m *mockClientPermissionRepo) FindByClientApiAndPermission(caID, pID int64) (*model.ClientPermission, error) {
	if m.findByClientApiAndPermissionFn != nil {
		return m.findByClientApiAndPermissionFn(caID, pID)
	}
	return nil, nil
}
func (m *mockClientPermissionRepo) RemoveByClientApiAndPermission(caID, pID int64) error {
	if m.removeByClientApiAndPermissionFn != nil {
		return m.removeByClientApiAndPermissionFn(caID, pID)
	}
	return nil
}
func (m *mockClientPermissionRepo) FindByClientApiID(caID int64) ([]model.ClientPermission, error) {
	if m.findByClientApiIDFn != nil {
		return m.findByClientApiIDFn(caID)
	}
	return nil, nil
}

// ---------------------------------------------------------------------------
// Mock: ClientApiRepository
// ---------------------------------------------------------------------------

type mockClientApiRepo struct {
	findByClientAndApiFn           func(int64, int64) (*model.ClientApi, error)
	findByClientUUIDFn             func(uuid.UUID) ([]model.ClientApi, error)
	findByClientUUIDAndApiUUIDFn   func(uuid.UUID, uuid.UUID) (*model.ClientApi, error)
	removeByClientAndApiFn         func(int64, int64) error
	removeByClientUUIDAndApiUUIDFn func(uuid.UUID, uuid.UUID) error
	createFn                       func(*model.ClientApi) (*model.ClientApi, error)
}

func (m *mockClientApiRepo) WithTx(_ *gorm.DB) repository.ClientApiRepository { return m }
func (m *mockClientApiRepo) Create(e *model.ClientApi) (*model.ClientApi, error) {
	if m.createFn != nil {
		return m.createFn(e)
	}
	return e, nil
}
func (m *mockClientApiRepo) CreateOrUpdate(e *model.ClientApi) (*model.ClientApi, error) {
	return e, nil
}
func (m *mockClientApiRepo) FindAll(_ ...string) ([]model.ClientApi, error)          { return nil, nil }
func (m *mockClientApiRepo) FindByUUID(_ any, _ ...string) (*model.ClientApi, error) { return nil, nil }
func (m *mockClientApiRepo) FindByUUIDs(_ []string, _ ...string) ([]model.ClientApi, error) {
	return nil, nil
}
func (m *mockClientApiRepo) FindByID(_ any, _ ...string) (*model.ClientApi, error) { return nil, nil }
func (m *mockClientApiRepo) UpdateByUUID(_, _ any) (*model.ClientApi, error)       { return nil, nil }
func (m *mockClientApiRepo) UpdateByID(_, _ any) (*model.ClientApi, error)         { return nil, nil }
func (m *mockClientApiRepo) DeleteByUUID(_ any) error                              { return nil }
func (m *mockClientApiRepo) DeleteByID(_ any) error                                { return nil }
func (m *mockClientApiRepo) Paginate(_ map[string]any, _, _ int, _ ...string) (*repository.PaginationResult[model.ClientApi], error) {
	return nil, nil
}
func (m *mockClientApiRepo) FindByClientAndApi(cID, aID int64) (*model.ClientApi, error) {
	if m.findByClientAndApiFn != nil {
		return m.findByClientAndApiFn(cID, aID)
	}
	return nil, nil
}
func (m *mockClientApiRepo) FindByClientUUID(cUUID uuid.UUID) ([]model.ClientApi, error) {
	if m.findByClientUUIDFn != nil {
		return m.findByClientUUIDFn(cUUID)
	}
	return nil, nil
}
func (m *mockClientApiRepo) FindByClientUUIDAndApiUUID(cUUID, aUUID uuid.UUID) (*model.ClientApi, error) {
	if m.findByClientUUIDAndApiUUIDFn != nil {
		return m.findByClientUUIDAndApiUUIDFn(cUUID, aUUID)
	}
	return nil, nil
}
func (m *mockClientApiRepo) RemoveByClientAndApi(cID, aID int64) error {
	if m.removeByClientAndApiFn != nil {
		return m.removeByClientAndApiFn(cID, aID)
	}
	return nil
}
func (m *mockClientApiRepo) RemoveByClientUUIDAndApiUUID(cUUID, aUUID uuid.UUID) error {
	if m.removeByClientUUIDAndApiUUIDFn != nil {
		return m.removeByClientUUIDAndApiUUIDFn(cUUID, aUUID)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Mock: APIKeyRepository
// ---------------------------------------------------------------------------

type mockAPIKeyRepo struct {
	findByUUIDFn              func(any, ...string) (*model.APIKey, error)
	findByUUIDAndTenantIDFn   func(string, int64) (*model.APIKey, error)
	findByKeyHashFn           func(string) (*model.APIKey, error)
	findByKeyPrefixFn         func(string) (*model.APIKey, error)
	deleteByUUIDFn            func(any) error
	deleteByUUIDAndTenantIDFn func(string, int64) error
	findPaginatedFn           func(repository.APIKeyRepositoryGetFilter) (*repository.PaginationResult[model.APIKey], error)
	createFn                  func(*model.APIKey) (*model.APIKey, error)
	updateByUUIDFn            func(any, any) (*model.APIKey, error)
}

func (m *mockAPIKeyRepo) WithTx(_ *gorm.DB) repository.APIKeyRepository { return m }
func (m *mockAPIKeyRepo) Create(e *model.APIKey) (*model.APIKey, error) {
	if m.createFn != nil {
		return m.createFn(e)
	}
	return e, nil
}
func (m *mockAPIKeyRepo) CreateOrUpdate(e *model.APIKey) (*model.APIKey, error) { return e, nil }
func (m *mockAPIKeyRepo) FindAll(_ ...string) ([]model.APIKey, error)           { return nil, nil }
func (m *mockAPIKeyRepo) FindByUUID(id any, p ...string) (*model.APIKey, error) {
	if m.findByUUIDFn != nil {
		return m.findByUUIDFn(id, p...)
	}
	return nil, nil
}
func (m *mockAPIKeyRepo) FindByUUIDs(_ []string, _ ...string) ([]model.APIKey, error) {
	return nil, nil
}
func (m *mockAPIKeyRepo) FindByID(_ any, _ ...string) (*model.APIKey, error) { return nil, nil }
func (m *mockAPIKeyRepo) UpdateByUUID(id, data any) (*model.APIKey, error) {
	if m.updateByUUIDFn != nil {
		return m.updateByUUIDFn(id, data)
	}
	return nil, nil
}
func (m *mockAPIKeyRepo) UpdateByID(_, _ any) (*model.APIKey, error) { return nil, nil }
func (m *mockAPIKeyRepo) DeleteByUUID(id any) error {
	if m.deleteByUUIDFn != nil {
		return m.deleteByUUIDFn(id)
	}
	return nil
}
func (m *mockAPIKeyRepo) DeleteByID(_ any) error { return nil }
func (m *mockAPIKeyRepo) Paginate(_ map[string]any, _, _ int, _ ...string) (*repository.PaginationResult[model.APIKey], error) {
	return nil, nil
}
func (m *mockAPIKeyRepo) FindByUUIDAndTenantID(id string, tID int64) (*model.APIKey, error) {
	if m.findByUUIDAndTenantIDFn != nil {
		return m.findByUUIDAndTenantIDFn(id, tID)
	}
	return nil, nil
}
func (m *mockAPIKeyRepo) FindByKeyHash(h string) (*model.APIKey, error) {
	if m.findByKeyHashFn != nil {
		return m.findByKeyHashFn(h)
	}
	return nil, nil
}
func (m *mockAPIKeyRepo) FindByKeyPrefix(p string) (*model.APIKey, error) {
	if m.findByKeyPrefixFn != nil {
		return m.findByKeyPrefixFn(p)
	}
	return nil, nil
}
func (m *mockAPIKeyRepo) DeleteByUUIDAndTenantID(id string, tID int64) error {
	if m.deleteByUUIDAndTenantIDFn != nil {
		return m.deleteByUUIDAndTenantIDFn(id, tID)
	}
	return nil
}
func (m *mockAPIKeyRepo) FindPaginated(f repository.APIKeyRepositoryGetFilter) (*repository.PaginationResult[model.APIKey], error) {
	if m.findPaginatedFn != nil {
		return m.findPaginatedFn(f)
	}
	return &repository.PaginationResult[model.APIKey]{}, nil
}

// ---------------------------------------------------------------------------
// Mock: APIKeyApiRepository
// ---------------------------------------------------------------------------

type mockAPIKeyApiRepo struct {
	findByAPIKeyAndApiFn           func(int64, int64) (*model.APIKeyApi, error)
	findByAPIKeyUUIDFn             func(uuid.UUID) ([]model.APIKeyApi, error)
	findByAPIKeyUUIDPaginatedFn    func(uuid.UUID, int, int, string, string) (*repository.PaginationResult[model.APIKeyApi], error)
	findByAPIKeyUUIDAndApiUUIDFn   func(uuid.UUID, uuid.UUID) (*model.APIKeyApi, error)
	removeByAPIKeyAndApiFn         func(int64, int64) error
	removeByAPIKeyUUIDAndApiUUIDFn func(uuid.UUID, uuid.UUID) error
	createFn                       func(*model.APIKeyApi) (*model.APIKeyApi, error)
}

func (m *mockAPIKeyApiRepo) WithTx(_ *gorm.DB) repository.APIKeyApiRepository { return m }
func (m *mockAPIKeyApiRepo) Create(e *model.APIKeyApi) (*model.APIKeyApi, error) {
	if m.createFn != nil {
		return m.createFn(e)
	}
	return e, nil
}
func (m *mockAPIKeyApiRepo) CreateOrUpdate(e *model.APIKeyApi) (*model.APIKeyApi, error) {
	return e, nil
}
func (m *mockAPIKeyApiRepo) FindAll(_ ...string) ([]model.APIKeyApi, error)          { return nil, nil }
func (m *mockAPIKeyApiRepo) FindByUUID(_ any, _ ...string) (*model.APIKeyApi, error) { return nil, nil }
func (m *mockAPIKeyApiRepo) FindByUUIDs(_ []string, _ ...string) ([]model.APIKeyApi, error) {
	return nil, nil
}
func (m *mockAPIKeyApiRepo) FindByID(_ any, _ ...string) (*model.APIKeyApi, error) { return nil, nil }
func (m *mockAPIKeyApiRepo) UpdateByUUID(_, _ any) (*model.APIKeyApi, error)       { return nil, nil }
func (m *mockAPIKeyApiRepo) UpdateByID(_, _ any) (*model.APIKeyApi, error)         { return nil, nil }
func (m *mockAPIKeyApiRepo) DeleteByUUID(_ any) error                              { return nil }
func (m *mockAPIKeyApiRepo) DeleteByID(_ any) error                                { return nil }
func (m *mockAPIKeyApiRepo) Paginate(_ map[string]any, _, _ int, _ ...string) (*repository.PaginationResult[model.APIKeyApi], error) {
	return nil, nil
}
func (m *mockAPIKeyApiRepo) FindByAPIKeyAndApi(akID, aID int64) (*model.APIKeyApi, error) {
	if m.findByAPIKeyAndApiFn != nil {
		return m.findByAPIKeyAndApiFn(akID, aID)
	}
	return nil, nil
}
func (m *mockAPIKeyApiRepo) FindByAPIKeyUUID(akUUID uuid.UUID) ([]model.APIKeyApi, error) {
	if m.findByAPIKeyUUIDFn != nil {
		return m.findByAPIKeyUUIDFn(akUUID)
	}
	return nil, nil
}
func (m *mockAPIKeyApiRepo) FindByAPIKeyUUIDPaginated(akUUID uuid.UUID, page, limit int, sortBy, sortOrder string) (*repository.PaginationResult[model.APIKeyApi], error) {
	if m.findByAPIKeyUUIDPaginatedFn != nil {
		return m.findByAPIKeyUUIDPaginatedFn(akUUID, page, limit, sortBy, sortOrder)
	}
	return &repository.PaginationResult[model.APIKeyApi]{}, nil
}
func (m *mockAPIKeyApiRepo) FindByAPIKeyUUIDAndApiUUID(akUUID, aUUID uuid.UUID) (*model.APIKeyApi, error) {
	if m.findByAPIKeyUUIDAndApiUUIDFn != nil {
		return m.findByAPIKeyUUIDAndApiUUIDFn(akUUID, aUUID)
	}
	return nil, nil
}
func (m *mockAPIKeyApiRepo) RemoveByAPIKeyAndApi(akID, aID int64) error {
	if m.removeByAPIKeyAndApiFn != nil {
		return m.removeByAPIKeyAndApiFn(akID, aID)
	}
	return nil
}
func (m *mockAPIKeyApiRepo) RemoveByAPIKeyUUIDAndApiUUID(akUUID, aUUID uuid.UUID) error {
	if m.removeByAPIKeyUUIDAndApiUUIDFn != nil {
		return m.removeByAPIKeyUUIDAndApiUUIDFn(akUUID, aUUID)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Mock: APIKeyPermissionRepository
// ---------------------------------------------------------------------------

type mockAPIKeyPermissionRepo struct {
	createFn                         func(*model.APIKeyPermission) (*model.APIKeyPermission, error)
	findByAPIKeyApiAndPermissionFn   func(int64, int64) (*model.APIKeyPermission, error)
	removeByAPIKeyApiAndPermissionFn func(int64, int64) error
	findByAPIKeyApiIDFn              func(int64) ([]model.APIKeyPermission, error)
}

func (m *mockAPIKeyPermissionRepo) WithTx(_ *gorm.DB) repository.APIKeyPermissionRepository {
	return m
}
func (m *mockAPIKeyPermissionRepo) Create(e *model.APIKeyPermission) (*model.APIKeyPermission, error) {
	if m.createFn != nil {
		return m.createFn(e)
	}
	return e, nil
}
func (m *mockAPIKeyPermissionRepo) CreateOrUpdate(e *model.APIKeyPermission) (*model.APIKeyPermission, error) {
	return e, nil
}
func (m *mockAPIKeyPermissionRepo) FindAll(_ ...string) ([]model.APIKeyPermission, error) {
	return nil, nil
}
func (m *mockAPIKeyPermissionRepo) FindByUUID(_ any, _ ...string) (*model.APIKeyPermission, error) {
	return nil, nil
}
func (m *mockAPIKeyPermissionRepo) FindByUUIDs(_ []string, _ ...string) ([]model.APIKeyPermission, error) {
	return nil, nil
}
func (m *mockAPIKeyPermissionRepo) FindByID(_ any, _ ...string) (*model.APIKeyPermission, error) {
	return nil, nil
}
func (m *mockAPIKeyPermissionRepo) UpdateByUUID(_, _ any) (*model.APIKeyPermission, error) {
	return nil, nil
}
func (m *mockAPIKeyPermissionRepo) UpdateByID(_, _ any) (*model.APIKeyPermission, error) {
	return nil, nil
}
func (m *mockAPIKeyPermissionRepo) DeleteByUUID(_ any) error { return nil }
func (m *mockAPIKeyPermissionRepo) DeleteByID(_ any) error   { return nil }
func (m *mockAPIKeyPermissionRepo) Paginate(_ map[string]any, _, _ int, _ ...string) (*repository.PaginationResult[model.APIKeyPermission], error) {
	return nil, nil
}
func (m *mockAPIKeyPermissionRepo) FindByAPIKeyApiAndPermission(akaID, pID int64) (*model.APIKeyPermission, error) {
	if m.findByAPIKeyApiAndPermissionFn != nil {
		return m.findByAPIKeyApiAndPermissionFn(akaID, pID)
	}
	return nil, nil
}
func (m *mockAPIKeyPermissionRepo) RemoveByAPIKeyApiAndPermission(akaID, pID int64) error {
	if m.removeByAPIKeyApiAndPermissionFn != nil {
		return m.removeByAPIKeyApiAndPermissionFn(akaID, pID)
	}
	return nil
}
func (m *mockAPIKeyPermissionRepo) FindByAPIKeyApiID(akaID int64) ([]model.APIKeyPermission, error) {
	if m.findByAPIKeyApiIDFn != nil {
		return m.findByAPIKeyApiIDFn(akaID)
	}
	return nil, nil
}

// ---------------------------------------------------------------------------
// Mock: InviteRepository
// ---------------------------------------------------------------------------

type mockInviteRepo struct {
	findByUUIDAndTenantIDFn func(uuid.UUID, int64, ...string) (*model.Invite, error)
	findByTokenFn           func(string) (*model.Invite, error)
	markAsUsedFn            func(uuid.UUID) error
	revokeByUUIDFn          func(uuid.UUID) error
	createFn                func(*model.Invite) (*model.Invite, error)
}

func (m *mockInviteRepo) WithTx(_ *gorm.DB) repository.InviteRepository { return m }
func (m *mockInviteRepo) Create(e *model.Invite) (*model.Invite, error) {
	if m.createFn != nil {
		return m.createFn(e)
	}
	return e, nil
}
func (m *mockInviteRepo) CreateOrUpdate(e *model.Invite) (*model.Invite, error) { return e, nil }
func (m *mockInviteRepo) FindAll(_ ...string) ([]model.Invite, error)           { return nil, nil }
func (m *mockInviteRepo) FindByUUID(_ any, _ ...string) (*model.Invite, error)  { return nil, nil }
func (m *mockInviteRepo) FindByUUIDs(_ []string, _ ...string) ([]model.Invite, error) {
	return nil, nil
}
func (m *mockInviteRepo) FindByID(_ any, _ ...string) (*model.Invite, error) { return nil, nil }
func (m *mockInviteRepo) UpdateByUUID(_, _ any) (*model.Invite, error)       { return nil, nil }
func (m *mockInviteRepo) UpdateByID(_, _ any) (*model.Invite, error)         { return nil, nil }
func (m *mockInviteRepo) DeleteByUUID(_ any) error                           { return nil }
func (m *mockInviteRepo) DeleteByID(_ any) error                             { return nil }
func (m *mockInviteRepo) Paginate(_ map[string]any, _, _ int, _ ...string) (*repository.PaginationResult[model.Invite], error) {
	return nil, nil
}
func (m *mockInviteRepo) FindAllByClientID(_ int64) ([]model.Invite, error) { return nil, nil }
func (m *mockInviteRepo) FindAllByTenantID(_ int64) ([]model.Invite, error) { return nil, nil }
func (m *mockInviteRepo) FindByUUIDAndTenantID(id uuid.UUID, tID int64, p ...string) (*model.Invite, error) {
	if m.findByUUIDAndTenantIDFn != nil {
		return m.findByUUIDAndTenantIDFn(id, tID, p...)
	}
	return nil, nil
}
func (m *mockInviteRepo) FindByToken(token string) (*model.Invite, error) {
	if m.findByTokenFn != nil {
		return m.findByTokenFn(token)
	}
	return nil, nil
}
func (m *mockInviteRepo) MarkAsUsed(id uuid.UUID) error {
	if m.markAsUsedFn != nil {
		return m.markAsUsedFn(id)
	}
	return nil
}
func (m *mockInviteRepo) RevokeByUUID(id uuid.UUID) error {
	if m.revokeByUUIDFn != nil {
		return m.revokeByUUIDFn(id)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Mock: SignupFlowRepository
// ---------------------------------------------------------------------------

type mockSignupFlowRepo struct {
	findByUUIDAndTenantIDFn       func(uuid.UUID, int64, ...string) (*model.SignupFlow, error)
	findByIdentifierAndClientIDFn func(string, int64) (*model.SignupFlow, error)
	findByNameFn                  func(string) (*model.SignupFlow, error)
	findPaginatedFn               func(repository.SignupFlowRepositoryGetFilter) (*repository.PaginationResult[model.SignupFlow], error)
	createFn                      func(*model.SignupFlow) (*model.SignupFlow, error)
	createOrUpdateFn              func(*model.SignupFlow) (*model.SignupFlow, error)
	deleteByUUIDFn                func(any) error
}

func (m *mockSignupFlowRepo) WithTx(_ *gorm.DB) repository.SignupFlowRepository { return m }
func (m *mockSignupFlowRepo) Create(e *model.SignupFlow) (*model.SignupFlow, error) {
	if m.createFn != nil {
		return m.createFn(e)
	}
	return e, nil
}
func (m *mockSignupFlowRepo) CreateOrUpdate(e *model.SignupFlow) (*model.SignupFlow, error) {
	if m.createOrUpdateFn != nil {
		return m.createOrUpdateFn(e)
	}
	return e, nil
}
func (m *mockSignupFlowRepo) FindAll(_ ...string) ([]model.SignupFlow, error) { return nil, nil }
func (m *mockSignupFlowRepo) FindByUUID(_ any, _ ...string) (*model.SignupFlow, error) {
	return nil, nil
}
func (m *mockSignupFlowRepo) FindByUUIDs(_ []string, _ ...string) ([]model.SignupFlow, error) {
	return nil, nil
}
func (m *mockSignupFlowRepo) FindByID(_ any, _ ...string) (*model.SignupFlow, error) { return nil, nil }
func (m *mockSignupFlowRepo) UpdateByUUID(_, _ any) (*model.SignupFlow, error)       { return nil, nil }
func (m *mockSignupFlowRepo) UpdateByID(_, _ any) (*model.SignupFlow, error)         { return nil, nil }
func (m *mockSignupFlowRepo) DeleteByUUID(id any) error {
	if m.deleteByUUIDFn != nil {
		return m.deleteByUUIDFn(id)
	}
	return nil
}
func (m *mockSignupFlowRepo) DeleteByID(_ any) error { return nil }
func (m *mockSignupFlowRepo) Paginate(_ map[string]any, _, _ int, _ ...string) (*repository.PaginationResult[model.SignupFlow], error) {
	return nil, nil
}
func (m *mockSignupFlowRepo) FindPaginated(f repository.SignupFlowRepositoryGetFilter) (*repository.PaginationResult[model.SignupFlow], error) {
	if m.findPaginatedFn != nil {
		return m.findPaginatedFn(f)
	}
	return &repository.PaginationResult[model.SignupFlow]{}, nil
}
func (m *mockSignupFlowRepo) FindByUUIDAndTenantID(id uuid.UUID, tID int64, p ...string) (*model.SignupFlow, error) {
	if m.findByUUIDAndTenantIDFn != nil {
		return m.findByUUIDAndTenantIDFn(id, tID, p...)
	}
	return nil, nil
}
func (m *mockSignupFlowRepo) FindByIdentifierAndClientID(ident string, cID int64) (*model.SignupFlow, error) {
	if m.findByIdentifierAndClientIDFn != nil {
		return m.findByIdentifierAndClientIDFn(ident, cID)
	}
	return nil, nil
}
func (m *mockSignupFlowRepo) FindByName(name string) (*model.SignupFlow, error) {
	if m.findByNameFn != nil {
		return m.findByNameFn(name)
	}
	return nil, nil
}

// ---------------------------------------------------------------------------
// Mock: SignupFlowRoleRepository
// ---------------------------------------------------------------------------

type mockSignupFlowRoleRepo struct {
	findBySignupFlowIDFn            func(int64) ([]model.SignupFlowRole, error)
	deleteBySignupFlowIDAndRoleIDFn func(int64, int64) error
	findBySignupFlowIDAndRoleIDFn   func(int64, int64) (*model.SignupFlowRole, error)
	findBySignupFlowIDPaginatedFn   func(int64, int, int) ([]model.SignupFlowRole, int64, error)
	createFn                        func(*model.SignupFlowRole) (*model.SignupFlowRole, error)
}

func (m *mockSignupFlowRoleRepo) WithTx(_ *gorm.DB) repository.SignupFlowRoleRepository { return m }
func (m *mockSignupFlowRoleRepo) Create(e *model.SignupFlowRole) (*model.SignupFlowRole, error) {
	if m.createFn != nil {
		return m.createFn(e)
	}
	return e, nil
}
func (m *mockSignupFlowRoleRepo) CreateOrUpdate(e *model.SignupFlowRole) (*model.SignupFlowRole, error) {
	return e, nil
}
func (m *mockSignupFlowRoleRepo) FindAll(_ ...string) ([]model.SignupFlowRole, error) {
	return nil, nil
}
func (m *mockSignupFlowRoleRepo) FindByUUID(_ any, _ ...string) (*model.SignupFlowRole, error) {
	return nil, nil
}
func (m *mockSignupFlowRoleRepo) FindByUUIDs(_ []string, _ ...string) ([]model.SignupFlowRole, error) {
	return nil, nil
}
func (m *mockSignupFlowRoleRepo) FindByID(_ any, _ ...string) (*model.SignupFlowRole, error) {
	return nil, nil
}
func (m *mockSignupFlowRoleRepo) UpdateByUUID(_, _ any) (*model.SignupFlowRole, error) {
	return nil, nil
}
func (m *mockSignupFlowRoleRepo) UpdateByID(_, _ any) (*model.SignupFlowRole, error) { return nil, nil }
func (m *mockSignupFlowRoleRepo) DeleteByUUID(_ any) error                           { return nil }
func (m *mockSignupFlowRoleRepo) DeleteByID(_ any) error                             { return nil }
func (m *mockSignupFlowRoleRepo) Paginate(_ map[string]any, _, _ int, _ ...string) (*repository.PaginationResult[model.SignupFlowRole], error) {
	return nil, nil
}
func (m *mockSignupFlowRoleRepo) FindBySignupFlowID(sfID int64) ([]model.SignupFlowRole, error) {
	if m.findBySignupFlowIDFn != nil {
		return m.findBySignupFlowIDFn(sfID)
	}
	return nil, nil
}
func (m *mockSignupFlowRoleRepo) FindBySignupFlowIDPaginated(sfID int64, page, limit int) ([]model.SignupFlowRole, int64, error) {
	if m.findBySignupFlowIDPaginatedFn != nil {
		return m.findBySignupFlowIDPaginatedFn(sfID, page, limit)
	}
	return nil, 0, nil
}
func (m *mockSignupFlowRoleRepo) DeleteBySignupFlowIDAndRoleID(sfID, rID int64) error {
	if m.deleteBySignupFlowIDAndRoleIDFn != nil {
		return m.deleteBySignupFlowIDAndRoleIDFn(sfID, rID)
	}
	return nil
}
func (m *mockSignupFlowRoleRepo) FindBySignupFlowIDAndRoleID(sfID, rID int64) (*model.SignupFlowRole, error) {
	if m.findBySignupFlowIDAndRoleIDFn != nil {
		return m.findBySignupFlowIDAndRoleIDFn(sfID, rID)
	}
	return nil, nil
}

// ---------------------------------------------------------------------------
// Mock: TenantUserRepository
// ---------------------------------------------------------------------------

type mockTenantUserRepo struct {
	findByTenantAndUserFn func(int64, int64) (*model.TenantUser, error)
	createFn              func(*model.TenantUser) (*model.TenantUser, error)
}

func (m *mockTenantUserRepo) WithTx(_ *gorm.DB) repository.TenantUserRepository { return m }
func (m *mockTenantUserRepo) Create(e *model.TenantUser) (*model.TenantUser, error) {
	if m.createFn != nil {
		return m.createFn(e)
	}
	return e, nil
}
func (m *mockTenantUserRepo) CreateOrUpdate(e *model.TenantUser) (*model.TenantUser, error) {
	return e, nil
}
func (m *mockTenantUserRepo) FindAll(_ ...string) ([]model.TenantUser, error) { return nil, nil }
func (m *mockTenantUserRepo) FindByUUID(_ any, _ ...string) (*model.TenantUser, error) {
	return nil, nil
}
func (m *mockTenantUserRepo) FindByUUIDs(_ []string, _ ...string) ([]model.TenantUser, error) {
	return nil, nil
}
func (m *mockTenantUserRepo) FindByID(_ any, _ ...string) (*model.TenantUser, error) { return nil, nil }
func (m *mockTenantUserRepo) UpdateByUUID(_, _ any) (*model.TenantUser, error)       { return nil, nil }
func (m *mockTenantUserRepo) UpdateByID(_, _ any) (*model.TenantUser, error)         { return nil, nil }
func (m *mockTenantUserRepo) DeleteByUUID(_ any) error                               { return nil }
func (m *mockTenantUserRepo) DeleteByID(_ any) error                                 { return nil }
func (m *mockTenantUserRepo) Paginate(_ map[string]any, _, _ int, _ ...string) (*repository.PaginationResult[model.TenantUser], error) {
	return nil, nil
}
func (m *mockTenantUserRepo) FindByTenantUserUUID(_ uuid.UUID) (*model.TenantUser, error) {
	return nil, nil
}
func (m *mockTenantUserRepo) FindByTenantAndUser(tID, uID int64) (*model.TenantUser, error) {
	if m.findByTenantAndUserFn != nil {
		return m.findByTenantAndUserFn(tID, uID)
	}
	return nil, nil
}
func (m *mockTenantUserRepo) FindAllByTenant(_ int64) ([]model.TenantUser, error) { return nil, nil }
func (m *mockTenantUserRepo) FindAllByUser(_ int64) ([]model.TenantUser, error)   { return nil, nil }

// ---------------------------------------------------------------------------
// Mock: UserRoleRepository
// ---------------------------------------------------------------------------

type mockUserRoleRepo struct {
	findByUserIDFn            func(int64) ([]model.UserRole, error)
	findByUserIDAndRoleIDFn   func(int64, int64) (*model.UserRole, error)
	deleteByUserIDAndRoleIDFn func(int64, int64) error
	createFn                  func(*model.UserRole) (*model.UserRole, error)
}

func (m *mockUserRoleRepo) WithTx(_ *gorm.DB) repository.UserRoleRepository { return m }
func (m *mockUserRoleRepo) Create(e *model.UserRole) (*model.UserRole, error) {
	if m.createFn != nil {
		return m.createFn(e)
	}
	return e, nil
}
func (m *mockUserRoleRepo) CreateOrUpdate(e *model.UserRole) (*model.UserRole, error) { return e, nil }
func (m *mockUserRoleRepo) FindAll(_ ...string) ([]model.UserRole, error)             { return nil, nil }
func (m *mockUserRoleRepo) FindByUUID(_ any, _ ...string) (*model.UserRole, error)    { return nil, nil }
func (m *mockUserRoleRepo) FindByUUIDs(_ []string, _ ...string) ([]model.UserRole, error) {
	return nil, nil
}
func (m *mockUserRoleRepo) FindByID(_ any, _ ...string) (*model.UserRole, error) { return nil, nil }
func (m *mockUserRoleRepo) UpdateByUUID(_, _ any) (*model.UserRole, error)       { return nil, nil }
func (m *mockUserRoleRepo) UpdateByID(_, _ any) (*model.UserRole, error)         { return nil, nil }
func (m *mockUserRoleRepo) DeleteByUUID(_ any) error                             { return nil }
func (m *mockUserRoleRepo) DeleteByID(_ any) error                               { return nil }
func (m *mockUserRoleRepo) Paginate(_ map[string]any, _, _ int, _ ...string) (*repository.PaginationResult[model.UserRole], error) {
	return nil, nil
}
func (m *mockUserRoleRepo) FindByUserID(uID int64) ([]model.UserRole, error) {
	if m.findByUserIDFn != nil {
		return m.findByUserIDFn(uID)
	}
	return nil, nil
}
func (m *mockUserRoleRepo) FindByUserIDAndRoleID(uID, rID int64) (*model.UserRole, error) {
	if m.findByUserIDAndRoleIDFn != nil {
		return m.findByUserIDAndRoleIDFn(uID, rID)
	}
	return nil, nil
}
func (m *mockUserRoleRepo) FindDefaultRolesByUserID(_ int64) ([]model.UserRole, error) {
	return nil, nil
}
func (m *mockUserRoleRepo) DeleteByUserID(_ int64) error { return nil }
func (m *mockUserRoleRepo) DeleteByUserIDAndRoleID(uID, rID int64) error {
	if m.deleteByUserIDAndRoleIDFn != nil {
		return m.deleteByUserIDAndRoleIDFn(uID, rID)
	}
	return nil
}
