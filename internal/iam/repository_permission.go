package iam

import (
	"errors"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/platform/database"
	"gorm.io/gorm"
)

type PermissionRepositoryGetFilter struct {
	TenantID    int64
	Name        *string
	Description *string
	APIID       *int64
	RoleID      *int64
	Status      *string
	IsSystem    *bool
	Page        int
	Limit       int
	SortBy      string
	SortOrder   string
}

type PermissionRepository interface {
	BaseRepositoryMethods[Permission]
	FindByUUID(uuid any, preloads ...string) (*Permission, error)
	FindByUUIDs(uuids []string, preloads ...string) ([]Permission, error)
	WithTx(tx *gorm.DB) PermissionRepository
	FindByUUIDAndTenantID(permissionUUID uuid.UUID, tenantID int64) (*Permission, error)
	FindByUUIDsAndTenantID(uuids []string, tenantID int64) ([]Permission, error)
	FindByName(name string, tenantID int64) (*Permission, error)
	FindPaginated(filter PermissionRepositoryGetFilter) (*PaginationResult[Permission], error)
	DeleteByUUIDAndTenantID(permissionUUID uuid.UUID, tenantID int64) error
}

type permissionRepository struct {
	*BaseRepository[Permission]
}

func NewPermissionRepository(db *gorm.DB) PermissionRepository {
	return &permissionRepository{
		BaseRepository: database.NewBaseRepository[Permission](db, "permission_uuid", "permission_id"),
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

func (r *permissionRepository) FindByUUIDsAndTenantID(uuids []string, tenantID int64) ([]Permission, error) {
	var permissions []Permission
	err := r.DB().
		Where("permission_uuid IN ? AND tenant_id = ?", uuids, tenantID).
		Find(&permissions).Error
	return permissions, err
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
	query = database.ApplyILike(query, "name", filter.Name)
	query = database.ApplyILike(query, "description", filter.Description)

	// Filters with exact match
	if filter.APIID != nil {
		query = query.Where("api_id = ?", *filter.APIID)
	}
	if filter.Status != nil {
		query = query.Where("status = ?", *filter.Status)
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
	query = query.Order(database.SanitizeOrder(filter.SortBy, filter.SortOrder, "created_at DESC")).Preload("API")

	return database.PaginateQuery[Permission](query, filter.Page, filter.Limit)
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
