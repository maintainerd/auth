package iam

import (
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/platform/database"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"gorm.io/gorm"
)

type RoleRepositoryGetFilter struct {
	Search      *string
	Name        *string
	Description *string
	IsDefault   *bool
	IsSystem    *bool
	Status      []string
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
	FindByUUID(uuid any, preloads ...string) (*Role, error)
	FindByUUIDs(uuids []string, preloads ...string) ([]Role, error)
	FindAll(preloads ...string) ([]Role, error)
	FindByID(id any, preloads ...string) (*Role, error)
	UpdateByUUID(uuid any, updatedData any) (*Role, error)
	UpdateByID(id any, updatedData any) (*Role, error)
	DeleteByUUID(uuid any) error
	DeleteByID(id any) error
	Paginate(conditions map[string]any, page int, limit int, preloads ...string) (*PaginationResult[Role], error)
	WithTx(tx *gorm.DB) RoleRepository
	FindByNameAndTenantID(name string, tenantID int64) (*Role, error)
	FindByUUIDAndTenantID(roleUUID uuid.UUID, tenantID int64) (*Role, error)
	FindAllByTenantID(tenantID int64) ([]Role, error)
	FindPaginated(filter RoleRepositoryGetFilter) (*PaginationResult[Role], error)
	GetPermissionsByRoleUUID(filter RoleRepositoryGetPermissionsFilter) (*PaginationResult[Permission], error)
	// SetStatusByUUID / SetDefaultStatusByUUID / SetSystemStatusByUUID are gone.
	// All three matched on role_uuid alone with no tenant predicate — cross-tenant
	// writers sitting in the interface waiting for a first caller — and none had
	// one. serviceRepository.SetStatusByUUID was scoped for the same reason; these
	// were removed instead because nothing goes through them (role status changes
	// run through roleService.SetStatusByUUID → CreateOrUpdate).
	FindRegisteredRoleForSetup(tenantID int64) (*Role, error)
	FindSuperAdminRoleForSetup(tenantID int64) (*Role, error)
}

type roleRepository struct {
	*BaseRepository[Role]
}

func NewRoleRepository(db *gorm.DB) RoleRepository {
	return &roleRepository{
		BaseRepository: database.NewBaseRepository[Role](db, "role_uuid", "role_id"),
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

func (r *roleRepository) FindByUUIDAndTenantID(roleUUID uuid.UUID, tenantID int64) (*Role, error) {
	var role Role
	err := r.DB().
		Where("role_uuid = ? AND tenant_id = ?", roleUUID, tenantID).
		First(&role).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
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

// roleSortColumns is this table's own sort allowlist. The global set in
// platform/database is a union across every table — it contains email, username
// and client_id, none of which exist on `roles` — so GET /roles?sort_by=email
// reached Postgres as an undefined column (42703) and surfaced as a 500 rather
// than a 400.
var roleSortColumns = map[string]struct{}{
	"created_at": {}, "updated_at": {}, "name": {}, "description": {},
	"status": {}, "is_default": {}, "is_system": {}, "tenant_id": {},
}

func (r *roleRepository) FindPaginated(filter RoleRepositoryGetFilter) (*PaginationResult[Role], error) {
	query := r.DB().Model(&Role{})

	// Always filter
	query = query.Where("tenant_id = ?", filter.TenantID)

	// Search across name and description with OR
	if filter.Search != nil && *filter.Search != "" {
		like := "%" + strings.ToLower(*filter.Search) + "%"
		query = query.Where("LOWER(name) LIKE ? OR LOWER(description) LIKE ?", like, like)
	} else {
		query = database.ApplyILike(query, "name", filter.Name)
		query = database.ApplyILike(query, "description", filter.Description)
	}

	// Filters with exact match
	if filter.IsDefault != nil {
		query = query.Where("is_default = ?", *filter.IsDefault)
	}
	if filter.IsSystem != nil {
		query = query.Where("is_system = ?", *filter.IsSystem)
	}
	if len(filter.Status) > 0 {
		query = query.Where("status IN ?", filter.Status)
	}

	// Sorting — protected against SQL injection via allowlist
	query = query.Order(database.SanitizeOrderIn(roleSortColumns, filter.SortBy, filter.SortOrder, "created_at DESC"))

	return database.PaginateQuery[Role](query, filter.Page, filter.Limit)
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

	// Sorting — protected against SQL injection via allowlist. The rows are
	// permissions, so the allowlist is the permissions one, not roleSortColumns.
	query = query.Order(database.SanitizeOrderInPrefixed(permissionSortColumns, "permissions.", filter.SortBy, filter.SortOrder, "permissions.created_at DESC")).Preload("API")

	return database.PaginateQuery[Permission](query, filter.Page, filter.Limit)
}
