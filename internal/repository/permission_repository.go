package repository

import (
	"errors"

	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

type PermissionRepositoryGetFilter struct {
	Name        *string
	Description *string
	APIID       *int64
	IsActive    *bool
	IsDefault   *bool
	Page        int
	Limit       int
	SortBy      string
	SortOrder   string
}

type PermissionRepository interface {
	BaseRepositoryMethods[model.Permission]
	WithTx(tx *gorm.DB) PermissionRepository
	FindByName(name string) (*model.Permission, error)
	FindPaginated(filter PermissionRepositoryGetFilter) (*PaginationResult[model.Permission], error)
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

func (r *permissionRepository) FindByName(name string) (*model.Permission, error) {
	var permission model.Permission
	err := r.db.Where("name = ?", name).First(&permission).Error

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
	query := r.db.Model(&model.Permission{})

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
	if filter.IsActive != nil {
		query = query.Where("is_active = ?", *filter.IsActive)
	}
	if filter.IsDefault != nil {
		query = query.Where("is_default = ?", *filter.IsDefault)
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
