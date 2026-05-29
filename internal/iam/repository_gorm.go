package iam

import (
	"errors"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/shared"
	"gorm.io/gorm"
)

type APIRepositoryGetFilter struct {
	TenantID    int64
	Name        *string
	DisplayName *string
	APIType     *string
	Identifier  *string
	ServiceID   *int64
	Status      []string
	IsSystem    *bool
	Page        int
	Limit       int
	SortBy      string
	SortOrder   string
}

type APIRepository interface {
	BaseRepositoryMethods[API]
	WithTx(tx *gorm.DB) APIRepository
	FindByUUIDAndTenantID(apiUUID uuid.UUID, tenantID int64) (*API, error)
	FindByName(apiName string, tenantID int64) (*API, error)
	FindByIdentifier(identifier string, tenantID int64) (*API, error)
	FindPaginated(filter APIRepositoryGetFilter) (*PaginationResult[API], error)
	SetStatusByUUID(apiUUID uuid.UUID, tenantID int64, status string) error
	CountByServiceID(serviceID int64, tenantID int64) (int64, error)
	DeleteByUUIDAndTenantID(apiUUID uuid.UUID, tenantID int64) error
}

type apiRepository struct {
	*BaseRepository[API]
}

func NewAPIRepository(db *gorm.DB) APIRepository {
	return &apiRepository{
		BaseRepository: NewBaseRepository[API](db, "api_uuid", "api_id"),
	}
}

func (r *apiRepository) WithTx(tx *gorm.DB) APIRepository {
	return &apiRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

func (r *apiRepository) FindByUUIDAndTenantID(apiUUID uuid.UUID, tenantID int64) (*API, error) {
	var api API
	err := r.DB().
		Preload("Service").
		Where("api_uuid = ? AND tenant_id = ?", apiUUID, tenantID).
		First(&api).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &api, nil
}

func (r *apiRepository) FindByName(apiName string, tenantID int64) (*API, error) {
	var api API
	err := r.DB().
		Preload("Service").
		Where("name = ? AND tenant_id = ?", apiName, tenantID).
		First(&api).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &api, err
}

func (r *apiRepository) FindByIdentifier(identifier string, tenantID int64) (*API, error) {
	var api API
	err := r.DB().
		Where("identifier = ? AND tenant_id = ?", identifier, tenantID).
		First(&api).Error
	return &api, err
}

func (r *apiRepository) FindPaginated(filter APIRepositoryGetFilter) (*PaginationResult[API], error) {
	query := r.DB().Model(&API{})

	// Filter by tenant_id
	query = query.Where("tenant_id = ?", filter.TenantID)

	// Filters with LIKE
	if filter.Name != nil {
		query = query.Where("name ILIKE ?", "%"+*filter.Name+"%")
	}
	if filter.DisplayName != nil {
		query = query.Where("display_name ILIKE ?", "%"+*filter.DisplayName+"%")
	}

	// Filters with exact match
	if filter.APIType != nil {
		query = query.Where("api_type = ?", *filter.APIType)
	}
	if filter.Identifier != nil {
		query = query.Where("identifier = ?", *filter.Identifier)
	}
	if filter.ServiceID != nil {
		query = query.Where("service_id = ?", *filter.ServiceID)
	}
	if len(filter.Status) > 0 {
		query = query.Where("status IN ?", filter.Status)
	}
	if filter.IsSystem != nil {
		query = query.Where("is_system = ?", *filter.IsSystem)
	}

	// Sorting — protected against SQL injection via allowlist
	query = query.Order(sanitizeOrder(filter.SortBy, filter.SortOrder, "created_at DESC"))

	// Count
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// Pagination guards prevent division-by-zero and negative offsets
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit < 1 {
		filter.Limit = 10
	}
	offset := (filter.Page - 1) * filter.Limit
	var apis []API
	if err := query.Preload("Service").Limit(filter.Limit).Offset(offset).Find(&apis).Error; err != nil {
		return nil, err
	}

	totalPages := int((total + int64(filter.Limit) - 1) / int64(filter.Limit))

	return &PaginationResult[API]{
		Data:       apis,
		Total:      total,
		Page:       filter.Page,
		Limit:      filter.Limit,
		TotalPages: totalPages,
	}, nil
}

func (r *apiRepository) SetStatusByUUID(apiUUID uuid.UUID, tenantID int64, status string) error {
	return r.DB().Model(&API{}).
		Where("api_uuid = ? AND tenant_id = ?", apiUUID, tenantID).
		Update("status", status).Error
}

func (r *apiRepository) CountByServiceID(serviceID int64, tenantID int64) (int64, error) {
	var count int64
	err := r.DB().Model(&API{}).
		Where("service_id = ? AND tenant_id = ?", serviceID, tenantID).
		Count(&count).Error
	return count, err
}

func (r *apiRepository) DeleteByUUIDAndTenantID(apiUUID uuid.UUID, tenantID int64) error {
	return r.DB().Where("api_uuid = ? AND tenant_id = ?", apiUUID, tenantID).Delete(&API{}).Error
}

type PermissionRepositoryGetFilter struct {
	TenantID    int64
	Name        *string
	Description *string
	APIID       *int64
	RoleID      *int64
	Status      *string
	IsDefault   *bool
	IsSystem    *bool
	Page        int
	Limit       int
	SortBy      string
	SortOrder   string
}

type PermissionRepository interface {
	BaseRepositoryMethods[Permission]
	WithTx(tx *gorm.DB) PermissionRepository
	FindByUUIDAndTenantID(permissionUUID uuid.UUID, tenantID int64) (*Permission, error)
	FindByName(name string, tenantID int64) (*Permission, error)
	FindPaginated(filter PermissionRepositoryGetFilter) (*PaginationResult[Permission], error)
	DeleteByUUIDAndTenantID(permissionUUID uuid.UUID, tenantID int64) error
}

type permissionRepository struct {
	*BaseRepository[Permission]
}

func NewPermissionRepository(db *gorm.DB) PermissionRepository {
	return &permissionRepository{
		BaseRepository: NewBaseRepository[Permission](db, "permission_uuid", "permission_id"),
	}
}

func (r *permissionRepository) WithTx(tx *gorm.DB) PermissionRepository {
	return &permissionRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

func (r *permissionRepository) FindByUUIDAndTenantID(permissionUUID uuid.UUID, tenantID int64) (*Permission, error) {
	var permission Permission
	err := r.DB().
		Preload("API").
		Where("permission_uuid = ? AND tenant_id = ?", permissionUUID, tenantID).
		First(&permission).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &permission, nil
}

func (r *permissionRepository) FindByName(name string, tenantID int64) (*Permission, error) {
	var permission Permission
	err := r.DB().Where("name = ? AND tenant_id = ?", name, tenantID).First(&permission).Error

	// If no record is found, return nil record and nil error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &permission, err
}

func (r *permissionRepository) FindPaginated(filter PermissionRepositoryGetFilter) (*PaginationResult[Permission], error) {
	query := r.DB().Model(&Permission{}).Where("tenant_id = ?", filter.TenantID)

	// Filters with LIKE
	if filter.Name != nil {
		query = query.Where("name ILIKE ?", "%"+*filter.Name+"%")
	}
	if filter.Description != nil {
		query = query.Where("description ILIKE ?", "%"+*filter.Description+"%")
	}

	// Filters with exact match
	if filter.APIID != nil {
		query = query.Where("api_id = ?", *filter.APIID)
	}
	if filter.Status != nil {
		query = query.Where("status = ?", *filter.Status)
	}
	if filter.IsDefault != nil {
		query = query.Where("is_default = ?", *filter.IsDefault)
	}
	if filter.IsSystem != nil {
		query = query.Where("is_system = ?", *filter.IsSystem)
	}

	// Joined table filter
	if filter.RoleID != nil {
		query = query.Joins(
			"JOIN role_permissions rp ON rp.permission_id = permissions.permission_id",
		).Where("rp.role_id = ?", *filter.RoleID)
	}

	// Sorting — protected against SQL injection via allowlist
	query = query.Order(sanitizeOrder(filter.SortBy, filter.SortOrder, "created_at DESC"))

	// Count
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// Pagination guards prevent division-by-zero and negative offsets
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit < 1 {
		filter.Limit = 10
	}
	offset := (filter.Page - 1) * filter.Limit
	var permissions []Permission
	if err := query.Preload("API").Limit(filter.Limit).Offset(offset).Find(&permissions).Error; err != nil {
		return nil, err
	}

	totalPages := int((total + int64(filter.Limit) - 1) / int64(filter.Limit))

	return &PaginationResult[Permission]{
		Data:       permissions,
		Total:      total,
		Page:       filter.Page,
		Limit:      filter.Limit,
		TotalPages: totalPages,
	}, nil
}

func (r *permissionRepository) DeleteByUUIDAndTenantID(permissionUUID uuid.UUID, tenantID int64) error {
	result := r.DB().Where("permission_uuid = ? AND tenant_id = ?", permissionUUID, tenantID).Delete(&Permission{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

type PolicyRepositoryGetFilter struct {
	TenantID    int64
	Name        *string
	Description *string
	Version     *string
	Status      []string
	IsSystem    *bool
	ServiceID   *uuid.UUID
	Page        int
	Limit       int
	SortBy      string
	SortOrder   string
}

type PolicyRepository interface {
	BaseRepositoryMethods[Policy]
	WithTx(tx *gorm.DB) PolicyRepository
	FindByUUIDAndTenantID(policyUUID uuid.UUID, tenantID int64) (*Policy, error)
	FindByName(policyName string, tenantID int64) (*Policy, error)
	FindByNameAndVersion(policyName string, version string, tenantID int64) (*Policy, error)
	FindSystemPolicies(tenantID int64) ([]Policy, error)
	FindPaginated(filter PolicyRepositoryGetFilter) (*PaginationResult[Policy], error)
	SetStatusByUUID(policyUUID uuid.UUID, tenantID int64, status string) error
	SetSystemStatusByUUID(policyUUID uuid.UUID, tenantID int64, isSystem bool) error
	DeleteByUUIDAndTenantID(policyUUID uuid.UUID, tenantID int64) error
}

type policyRepository struct {
	*BaseRepository[Policy]
}

func NewPolicyRepository(db *gorm.DB) PolicyRepository {
	return &policyRepository{
		BaseRepository: NewBaseRepository[Policy](db, "policy_uuid", "policy_id"),
	}
}

func (r *policyRepository) WithTx(tx *gorm.DB) PolicyRepository {
	return &policyRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

func (r *policyRepository) FindByUUIDAndTenantID(policyUUID uuid.UUID, tenantID int64) (*Policy, error) {
	var policy Policy
	err := r.DB().Where("policy_uuid = ? AND tenant_id = ?", policyUUID, tenantID).First(&policy).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &policy, nil
}

func (r *policyRepository) FindByName(policyName string, tenantID int64) (*Policy, error) {
	var policy Policy
	err := r.DB().Where("name = ? AND tenant_id = ?", policyName, tenantID).First(&policy).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &policy, nil
}

func (r *policyRepository) FindByNameAndVersion(policyName string, version string, tenantID int64) (*Policy, error) {
	var policy Policy
	err := r.DB().Where("name = ? AND version = ? AND tenant_id = ?", policyName, version, tenantID).First(&policy).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &policy, nil
}

func (r *policyRepository) FindSystemPolicies(tenantID int64) ([]Policy, error) {
	var policies []Policy
	err := r.DB().Where("is_system = ? AND tenant_id = ?", true, tenantID).Find(&policies).Error
	return policies, err
}

func (r *policyRepository) SetStatusByUUID(policyUUID uuid.UUID, tenantID int64, status string) error {
	return r.DB().Model(&Policy{}).
		Where("policy_uuid = ? AND tenant_id = ?", policyUUID, tenantID).
		Update("status", status).Error
}

func (r *policyRepository) SetSystemStatusByUUID(policyUUID uuid.UUID, tenantID int64, isSystem bool) error {
	return r.DB().Model(&Policy{}).
		Where("policy_uuid = ? AND tenant_id = ?", policyUUID, tenantID).
		Update("is_system", isSystem).Error
}

func (r *policyRepository) FindPaginated(filter PolicyRepositoryGetFilter) (*PaginationResult[Policy], error) {
	query := r.DB().Model(&Policy{})

	// Filter by tenant_id
	query = query.Where("tenant_id = ?", filter.TenantID)

	// Apply filters
	if filter.Name != nil {
		query = query.Where("name ILIKE ?", "%"+*filter.Name+"%")
	}
	if filter.Description != nil {
		query = query.Where("description ILIKE ?", "%"+*filter.Description+"%")
	}
	if filter.Version != nil {
		query = query.Where("version ILIKE ?", "%"+*filter.Version+"%")
	}
	if len(filter.Status) > 0 {
		query = query.Where("status IN ?", filter.Status)
	}
	if filter.IsSystem != nil {
		query = query.Where("is_system = ?", *filter.IsSystem)
	}
	if filter.ServiceID != nil {
		// Join with service_policies and services tables to filter by service UUID
		query = query.Joins("INNER JOIN service_policies ON policies.policy_id = service_policies.policy_id").
			Joins("INNER JOIN services ON service_policies.service_id = services.service_id").
			Where("services.service_uuid = ?", *filter.ServiceID)
	}

	// Count total records
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// Apply sorting — protected against SQL injection via allowlist
	query = query.Order(sanitizeOrder(filter.SortBy, filter.SortOrder, "created_at DESC"))

	// Pagination
	filter.Page, filter.Limit = normalizePagination(filter.Page, filter.Limit)
	offset := (filter.Page - 1) * filter.Limit
	var policies []Policy
	if err := query.Limit(filter.Limit).Offset(offset).Find(&policies).Error; err != nil {
		return nil, err
	}

	totalPages := int((total + int64(filter.Limit) - 1) / int64(filter.Limit))

	return &PaginationResult[Policy]{
		Data:       policies,
		Total:      total,
		Page:       filter.Page,
		Limit:      filter.Limit,
		TotalPages: totalPages,
	}, nil
}

func (r *policyRepository) DeleteByUUIDAndTenantID(policyUUID uuid.UUID, tenantID int64) error {
	return r.DB().Where("policy_uuid = ? AND tenant_id = ?", policyUUID, tenantID).Delete(&Policy{}).Error
}

type RoleRepositoryGetFilter struct {
	Name        *string
	Description *string
	IsDefault   *bool
	IsSystem    *bool
	Status      *string
	TenantID    int64
	Page        int
	Limit       int
	SortBy      string
	SortOrder   string
}

type RoleRepositoryGetPermissionsFilter struct {
	RoleUUID  uuid.UUID
	Status    *string
	Page      int
	Limit     int
	SortBy    string
	SortOrder string
}

type RoleRepository interface {
	BaseRepositoryMethods[Role]
	WithTx(tx *gorm.DB) RoleRepository
	FindByNameAndTenantID(name string, tenantID int64) (*Role, error)
	FindAllByTenantID(tenantID int64) ([]Role, error)
	FindPaginated(filter RoleRepositoryGetFilter) (*PaginationResult[Role], error)
	GetPermissionsByRoleUUID(filter RoleRepositoryGetPermissionsFilter) (*PaginationResult[Permission], error)
	SetStatusByUUID(roleUUID uuid.UUID, status string) error
	SetDefaultStatusByUUID(roleUUID uuid.UUID, isDefault bool) error
	SetSystemStatusByUUID(roleUUID uuid.UUID, isSystem bool) error
	FindRegisteredRoleForSetup(tenantID int64) (*Role, error)
	FindSuperAdminRoleForSetup(tenantID int64) (*Role, error)
}

type roleRepository struct {
	*BaseRepository[Role]
}

func NewRoleRepository(db *gorm.DB) RoleRepository {
	return &roleRepository{
		BaseRepository: NewBaseRepository[Role](db, "role_uuid", "role_id"),
	}
}

func (r *roleRepository) WithTx(tx *gorm.DB) RoleRepository {
	return &roleRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

func (r *roleRepository) FindByNameAndTenantID(name string, tenantID int64) (*Role, error) {
	var role Role
	err := r.DB().
		Where("name = ? AND tenant_id = ?", name, tenantID).
		First(&role).Error

	// If no record is found, return nil record and nil error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		// For all other errors, return nil record and the actual error
		return nil, err
	}

	return &role, nil
}

func (r *roleRepository) FindAllByTenantID(tenantID int64) ([]Role, error) {
	var roles []Role
	err := r.DB().
		Where("tenant_id = ?", tenantID).
		Find(&roles).Error
	return roles, err
}

func (r *roleRepository) FindPaginated(filter RoleRepositoryGetFilter) (*PaginationResult[Role], error) {
	query := r.DB().Model(&Role{})

	// Always filter
	query = query.Where("tenant_id = ?", filter.TenantID)

	// Filters with LIKE
	if filter.Name != nil {
		query = query.Where("name ILIKE ?", "%"+*filter.Name+"%")
	}
	if filter.Description != nil {
		query = query.Where("description ILIKE ?", "%"+*filter.Description+"%")
	}

	// Filters with exact match
	if filter.IsDefault != nil {
		query = query.Where("is_default = ?", *filter.IsDefault)
	}
	if filter.IsSystem != nil {
		query = query.Where("is_system = ?", *filter.IsSystem)
	}
	if filter.Status != nil {
		query = query.Where("status = ?", *filter.Status)
	}

	// Sorting — protected against SQL injection via allowlist
	query = query.Order(sanitizeOrder(filter.SortBy, filter.SortOrder, "created_at DESC"))

	// Count
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// Pagination
	filter.Page, filter.Limit = normalizePagination(filter.Page, filter.Limit)
	offset := (filter.Page - 1) * filter.Limit
	var roles []Role
	if err := query.Limit(filter.Limit).Offset(offset).Find(&roles).Error; err != nil {
		return nil, err
	}

	totalPages := int((total + int64(filter.Limit) - 1) / int64(filter.Limit))

	return &PaginationResult[Role]{
		Data:       roles,
		Total:      total,
		Page:       filter.Page,
		Limit:      filter.Limit,
		TotalPages: totalPages,
	}, nil
}

func (r *roleRepository) SetStatusByUUID(roleUUID uuid.UUID, status string) error {
	return r.DB().Model(&Role{}).
		Where("role_uuid = ?", roleUUID).
		Update("status", status).Error
}

func (r *roleRepository) SetDefaultStatusByUUID(roleUUID uuid.UUID, isDefault bool) error {
	return r.DB().Model(&Role{}).
		Where("role_uuid = ?", roleUUID).
		Update("is_default", isDefault).Error
}

func (r *roleRepository) SetSystemStatusByUUID(roleUUID uuid.UUID, isSystem bool) error {
	return r.DB().Model(&Role{}).
		Where("role_uuid = ?", roleUUID).
		Update("is_system", isSystem).Error
}

func (r *roleRepository) FindRegisteredRoleForSetup(tenantID int64) (*Role, error) {
	var role Role
	err := r.DB().Where("tenant_id = ? AND name = ? AND is_default = ? AND is_system = ?",
		tenantID, shared.RoleRegistered, true, true).First(&role).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &role, nil
}

func (r *roleRepository) FindSuperAdminRoleForSetup(tenantID int64) (*Role, error) {
	var role Role
	err := r.DB().Where("tenant_id = ? AND name = ? AND is_system = ?",
		tenantID, shared.RoleSuperAdmin, true).First(&role).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &role, nil
}

func (r *roleRepository) GetPermissionsByRoleUUID(filter RoleRepositoryGetPermissionsFilter) (*PaginationResult[Permission], error) {
	// Single-query JOIN: no round trip to fetch role.RoleID first.
	query := r.DB().Model(&Permission{}).
		Joins("JOIN role_permissions ON role_permissions.permission_id = permissions.permission_id").
		Joins("JOIN roles ON roles.role_id = role_permissions.role_id").
		Where("roles.role_uuid = ?", filter.RoleUUID)

	// Apply filters
	if filter.Status != nil {
		query = query.Where("permissions.status = ?", *filter.Status)
	}

	// Sorting — protected against SQL injection via allowlist
	query = query.Order(sanitizeOrderPrefixed("permissions.", filter.SortBy, filter.SortOrder, "permissions.created_at DESC"))

	// Count
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// Pagination
	filter.Page, filter.Limit = normalizePagination(filter.Page, filter.Limit)
	offset := (filter.Page - 1) * filter.Limit
	var permissions []Permission
	if err := query.Preload("API").Limit(filter.Limit).Offset(offset).Find(&permissions).Error; err != nil {
		return nil, err
	}

	totalPages := int((total + int64(filter.Limit) - 1) / int64(filter.Limit))

	return &PaginationResult[Permission]{
		Data:       permissions,
		Total:      total,
		Page:       filter.Page,
		Limit:      filter.Limit,
		TotalPages: totalPages,
	}, nil
}

type RolePermissionRepository interface {
	BaseRepositoryMethods[RolePermission]
	WithTx(tx *gorm.DB) RolePermissionRepository
	Assign(rolePermission *RolePermission) (*RolePermission, error)
	FindByRoleAndPermission(roleID int64, permissionID int64) (*RolePermission, error)
	FindAllByRoleID(roleID int64) ([]RolePermission, error)
	FindAllByPermissionID(permissionID int64) ([]RolePermission, error)
	RemoveByRoleAndPermission(roleID int64, permissionID int64) error
	SetDefaultStatusByUUID(rolePermissionUUID uuid.UUID, isDefault bool) error
}

type rolePermissionRepository struct {
	*BaseRepository[RolePermission]
}

func NewRolePermissionRepository(db *gorm.DB) RolePermissionRepository {
	return &rolePermissionRepository{
		BaseRepository: NewBaseRepository[RolePermission](db, "role_permission_uuid", "role_permission_id"),
	}
}

func (r *rolePermissionRepository) WithTx(tx *gorm.DB) RolePermissionRepository {
	return &rolePermissionRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

// Assign a role-permission pair and return the created record
func (r *rolePermissionRepository) Assign(rolePermission *RolePermission) (*RolePermission, error) {
	return r.Create(rolePermission)
}

func (r *rolePermissionRepository) FindByRoleAndPermission(roleID int64, permissionID int64) (*RolePermission, error) {
	var rp RolePermission
	err := r.DB().
		Where("role_id = ? AND permission_id = ?", roleID, permissionID).
		First(&rp).Error

	// If no record is found, return nil record and nil error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		// For all other errors, return nil record and the actual error
		return nil, err
	}

	return &rp, nil
}

func (r *rolePermissionRepository) FindAllByRoleID(roleID int64) ([]RolePermission, error) {
	var rps []RolePermission
	err := r.DB().Where("role_id = ?", roleID).Find(&rps).Error
	return rps, err
}

func (r *rolePermissionRepository) FindAllByPermissionID(permissionID int64) ([]RolePermission, error) {
	var rps []RolePermission
	err := r.DB().Where("permission_id = ?", permissionID).Find(&rps).Error
	return rps, err
}

func (r *rolePermissionRepository) RemoveByRoleAndPermission(roleID int64, permissionID int64) error {
	return r.DB().
		Where("role_id = ? AND permission_id = ?", roleID, permissionID).
		Unscoped().Delete(&RolePermission{}).Error
}

func (r *rolePermissionRepository) SetDefaultStatusByUUID(rolePermissionUUID uuid.UUID, isDefault bool) error {
	return r.DB().Model(&RolePermission{}).
		Where("role_permission_uuid = ?", rolePermissionUUID).
		Update("is_default", isDefault).Error
}

type ServiceRepositoryGetFilter struct {
	Name        *string
	DisplayName *string
	Description *string
	Version     *string
	TenantID    *int64
	IsSystem    *bool
	Status      []string
	Page        int
	Limit       int
	SortBy      string
	SortOrder   string
}

type ServiceRepository interface {
	BaseRepositoryMethods[Service]
	WithTx(tx *gorm.DB) ServiceRepository
	FindByName(serviceName string) (*Service, error)
	FindByNameAndTenantID(serviceName string, tenantID int64) (*Service, error)
	FindByTenantID(tenantID int64) ([]Service, error)
	FindPaginated(filter ServiceRepositoryGetFilter) (*PaginationResult[Service], error)
	FindServicesByPolicyUUID(policyUUID uuid.UUID, filter ServiceRepositoryGetFilter) (*PaginationResult[Service], error)
	SetStatusByUUID(serviceUUID uuid.UUID, status string) error
	CountPoliciesByServiceID(serviceID int64) (int64, error)
}

type serviceRepository struct {
	*BaseRepository[Service]
}

func NewServiceRepository(db *gorm.DB) ServiceRepository {
	return &serviceRepository{
		BaseRepository: NewBaseRepository[Service](db, "service_uuid", "service_id"),
	}
}

func (r *serviceRepository) WithTx(tx *gorm.DB) ServiceRepository {
	return &serviceRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

func (r *serviceRepository) FindByName(serviceName string) (*Service, error) {
	var service Service
	err := r.DB().Where("name = ?", serviceName).First(&service).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &service, nil
}

func (r *serviceRepository) FindByNameAndTenantID(serviceName string, tenantID int64) (*Service, error) {
	var service Service
	err := r.DB().
		Joins("JOIN tenant_services ON services.service_id = tenant_services.service_id").
		Where("services.name = ? AND tenant_services.tenant_id = ?", serviceName, tenantID).
		First(&service).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &service, nil
}

func (r *serviceRepository) FindByTenantID(tenantID int64) ([]Service, error) {
	var services []Service
	err := r.DB().
		Joins("JOIN tenant_services ON services.service_id = tenant_services.service_id").
		Where("tenant_services.tenant_id = ?", tenantID).
		Find(&services).Error
	return services, err
}

func (r *serviceRepository) FindPaginated(filter ServiceRepositoryGetFilter) (*PaginationResult[Service], error) {
	query := r.DB().Model(&Service{})

	// Filters with LIKE
	if filter.Name != nil {
		query = query.Where("name ILIKE ?", "%"+*filter.Name+"%")
	}
	if filter.DisplayName != nil {
		query = query.Where("display_name ILIKE ?", "%"+*filter.DisplayName+"%")
	}
	if filter.Description != nil {
		query = query.Where("description ILIKE ?", "%"+*filter.Description+"%")
	}
	if filter.Version != nil {
		query = query.Where("version ILIKE ?", "%"+*filter.Version+"%")
	}

	// Filters with exact match
	if filter.TenantID != nil {
		query = query.Joins("JOIN tenant_services ON services.service_id = tenant_services.service_id").
			Where("tenant_services.tenant_id = ?", *filter.TenantID)
	}
	if len(filter.Status) > 0 {
		query = query.Where("status IN ?", filter.Status)
	}
	if filter.IsSystem != nil {
		query = query.Where("is_system = ?", *filter.IsSystem)
	}

	// Sorting — protected against SQL injection via allowlist
	query = query.Order(sanitizeOrder(filter.SortBy, filter.SortOrder, "created_at DESC"))

	// Count
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// Pagination guards prevent division-by-zero and negative offsets
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit < 1 {
		filter.Limit = 10
	}
	offset := (filter.Page - 1) * filter.Limit
	var services []Service
	if err := query.Limit(filter.Limit).Offset(offset).Find(&services).Error; err != nil {
		return nil, err
	}

	totalPages := int((total + int64(filter.Limit) - 1) / int64(filter.Limit))

	return &PaginationResult[Service]{
		Data:       services,
		Total:      total,
		Page:       filter.Page,
		Limit:      filter.Limit,
		TotalPages: totalPages,
	}, nil
}

func (r *serviceRepository) FindServicesByPolicyUUID(policyUUID uuid.UUID, filter ServiceRepositoryGetFilter) (*PaginationResult[Service], error) {
	query := r.DB().Model(&Service{}).
		Joins("INNER JOIN service_policies ON services.service_id = service_policies.service_id").
		Joins("INNER JOIN policies ON service_policies.policy_id = policies.policy_id").
		Where("policies.policy_uuid = ?", policyUUID)

	// Apply filters with LIKE
	if filter.Name != nil {
		query = query.Where("services.name ILIKE ?", "%"+*filter.Name+"%")
	}
	if filter.DisplayName != nil {
		query = query.Where("services.display_name ILIKE ?", "%"+*filter.DisplayName+"%")
	}
	if filter.Description != nil {
		query = query.Where("services.description ILIKE ?", "%"+*filter.Description+"%")
	}
	if filter.Version != nil {
		query = query.Where("services.version ILIKE ?", "%"+*filter.Version+"%")
	}

	// Status filter (multiple values)
	if len(filter.Status) > 0 {
		query = query.Where("services.status IN ?", filter.Status)
	}

	// Boolean filters
	if filter.IsSystem != nil {
		query = query.Where("services.is_system = ?", *filter.IsSystem)
	}

	// Count total records
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// Apply sorting — protected against SQL injection via allowlist
	query = query.Order(sanitizeOrderPrefixed("services.", filter.SortBy, filter.SortOrder, "services.created_at DESC"))

	// Pagination guards prevent division-by-zero and negative offsets
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit < 1 {
		filter.Limit = 10
	}
	offset := (filter.Page - 1) * filter.Limit
	var services []Service
	if err := query.Limit(filter.Limit).Offset(offset).Find(&services).Error; err != nil {
		return nil, err
	}

	totalPages := int((total + int64(filter.Limit) - 1) / int64(filter.Limit))

	return &PaginationResult[Service]{
		Data:       services,
		Total:      total,
		Page:       filter.Page,
		Limit:      filter.Limit,
		TotalPages: totalPages,
	}, nil
}

func (r *serviceRepository) SetStatusByUUID(serviceUUID uuid.UUID, status string) error {
	return r.DB().Model(&Service{}).
		Where("service_uuid = ?", serviceUUID).
		Update("status", status).Error
}

func (r *serviceRepository) CountPoliciesByServiceID(serviceID int64) (int64, error) {
	var count int64
	err := r.DB().Model(&ServicePolicy{}).
		Where("service_id = ?", serviceID).
		Count(&count).Error
	return count, err
}

type ServicePolicyRepositoryGetFilter struct {
	ServiceID *int64
	PolicyID  *int64
	Page      int
	Limit     int
	SortBy    string
	SortOrder string
}

type ServicePolicyRepository interface {
	BaseRepositoryMethods[ServicePolicy]
	WithTx(tx *gorm.DB) ServicePolicyRepository
	FindPaginated(filter ServicePolicyRepositoryGetFilter) (*PaginationResult[ServicePolicy], error)
	FindByServiceAndPolicy(serviceID int64, policyID int64) (*ServicePolicy, error)
	DeleteByServiceAndPolicy(serviceID int64, policyID int64) error
	FindPoliciesByServiceID(serviceID int64) ([]Policy, error)
	FindServicesByPolicyID(policyID int64) ([]Service, error)
}

type servicePolicyRepository struct {
	*BaseRepository[ServicePolicy]
}

func NewServicePolicyRepository(db *gorm.DB) ServicePolicyRepository {
	return &servicePolicyRepository{
		BaseRepository: NewBaseRepository[ServicePolicy](db, "service_policy_uuid", "service_policy_id"),
	}
}

func (r *servicePolicyRepository) WithTx(tx *gorm.DB) ServicePolicyRepository {
	return &servicePolicyRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

func (r *servicePolicyRepository) FindByServiceAndPolicy(serviceID int64, policyID int64) (*ServicePolicy, error) {
	var servicePolicy ServicePolicy
	err := r.DB().Where("service_id = ? AND policy_id = ?", serviceID, policyID).First(&servicePolicy).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &servicePolicy, nil
}

func (r *servicePolicyRepository) DeleteByServiceAndPolicy(serviceID int64, policyID int64) error {
	return r.DB().Where("service_id = ? AND policy_id = ?", serviceID, policyID).Delete(&ServicePolicy{}).Error
}

func (r *servicePolicyRepository) FindPoliciesByServiceID(serviceID int64) ([]Policy, error) {
	var policies []Policy
	err := r.DB().Table("policies").
		Joins("INNER JOIN service_policies ON policies.policy_id = service_policies.policy_id").
		Where("service_policies.service_id = ?", serviceID).
		Find(&policies).Error
	return policies, err
}

func (r *servicePolicyRepository) FindServicesByPolicyID(policyID int64) ([]Service, error) {
	var services []Service
	err := r.DB().Table("services").
		Joins("INNER JOIN service_policies ON services.service_id = service_policies.service_id").
		Where("service_policies.policy_id = ?", policyID).
		Find(&services).Error
	return services, err
}

func (r *servicePolicyRepository) FindPaginated(filter ServicePolicyRepositoryGetFilter) (*PaginationResult[ServicePolicy], error) {
	query := r.DB().Model(&ServicePolicy{})

	// Apply filters
	if filter.ServiceID != nil {
		query = query.Where("service_id = ?", *filter.ServiceID)
	}
	if filter.PolicyID != nil {
		query = query.Where("policy_id = ?", *filter.PolicyID)
	}

	// Count total records
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// Apply sorting — protected against SQL injection via allowlist
	query = query.Order(sanitizeOrder(filter.SortBy, filter.SortOrder, "created_at DESC"))

	// Pagination guards prevent division-by-zero and negative offsets
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit < 1 {
		filter.Limit = 10
	}
	offset := (filter.Page - 1) * filter.Limit
	var servicePolicies []ServicePolicy
	if err := query.Limit(filter.Limit).Offset(offset).Find(&servicePolicies).Error; err != nil {
		return nil, err
	}

	totalPages := int((total + int64(filter.Limit) - 1) / int64(filter.Limit))

	return &PaginationResult[ServicePolicy]{
		Data:       servicePolicies,
		Total:      total,
		Page:       filter.Page,
		Limit:      filter.Limit,
		TotalPages: totalPages,
	}, nil
}
