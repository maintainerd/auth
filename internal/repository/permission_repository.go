package repository

import (
	"errors"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

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
	BaseRepositoryMethods[model.Permission]
	WithTx(tx *gorm.DB) PermissionRepository
	FindByUUIDAndTenantID(permissionUUID uuid.UUID, tenantID int64) (*model.Permission, error)
	FindByName(name string, tenantID int64) (*model.Permission, error)
	FindPaginated(filter PermissionRepositoryGetFilter) (*PaginationResult[model.Permission], error)
	DeleteByUUIDAndTenantID(permissionUUID uuid.UUID, tenantID int64) error
}

type permissionRepository struct {
	*BaseRepository[model.Permission]
	db *gorm.DB
}

func NewPermissionRepository(db *gorm.DB) PermissionRepository {
	return &permissionRepository{
		BaseRepository: NewBaseRepository[model.Permission](db, "permission_uuid", "permission_id"),
		db:             db,
	}
}

func (r *permissionRepository) WithTx(tx *gorm.DB) PermissionRepository {
	return &permissionRepository{
		BaseRepository: NewBaseRepository[model.Permission](tx, "permission_uuid", "permission_id"),
		db:             tx,
	}
}

func (r *permissionRepository) FindByUUIDAndTenantID(permissionUUID uuid.UUID, tenantID int64) (*model.Permission, error) {
	var permission model.Permission
	err := r.db.
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

func (r *permissionRepository) FindByName(name string, tenantID int64) (*model.Permission, error) {
	var permission model.Permission
	err := r.db.Where("name = ? AND tenant_id = ?", name, tenantID).First(&permission).Error

	// If no record is found, return nil record and nil error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &permission, err
}

func (r *permissionRepository) FindPaginated(filter PermissionRepositoryGetFilter) (*PaginationResult[model.Permission], error) {
	query := r.db.Model(&model.Permission{}).Where("tenant_id = ?", filter.TenantID)

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

	// Sorting
	orderBy := filter.SortBy + " " + filter.SortOrder
	query = query.Order(orderBy)

	// Count
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// Pagination
	offset := (filter.Page - 1) * filter.Limit
	var permissions []model.Permission
	if err := query.Preload("API").Limit(filter.Limit).Offset(offset).Find(&permissions).Error; err != nil {
		return nil, err
	}

	totalPages := int((total + int64(filter.Limit) - 1) / int64(filter.Limit))

	return &PaginationResult[model.Permission]{
		Data:       permissions,
		Total:      total,
		Page:       filter.Page,
		Limit:      filter.Limit,
		TotalPages: totalPages,
	}, nil
}

func (r *permissionRepository) DeleteByUUIDAndTenantID(permissionUUID uuid.UUID, tenantID int64) error {
	result := r.db.Where("permission_uuid = ? AND tenant_id = ?", permissionUUID, tenantID).Delete(&model.Permission{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
