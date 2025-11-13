package repository

import (
	"errors"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

type RoleRepositoryGetFilter struct {
	Name        *string
	Description *string
	IsDefault   *bool
	IsSystem    *bool
	IsActive    *bool
	TenantID    int64
	Page        int
	Limit       int
	SortBy      string
	SortOrder   string
}

type RoleRepository interface {
	BaseRepositoryMethods[model.Role]
	WithTx(tx *gorm.DB) RoleRepository
	FindByNameAndTenantID(name string, tenantID int64) (*model.Role, error)
	FindAllByTenantID(tenantID int64) ([]model.Role, error)
	FindPaginated(filter RoleRepositoryGetFilter) (*PaginationResult[model.Role], error)
	SetActiveStatusByUUID(roleUUID uuid.UUID, isActive bool) error
	SetDefaultStatusByUUID(roleUUID uuid.UUID, isDefault bool) error
	SetSystemStatusByUUID(roleUUID uuid.UUID, isSystem bool) error
	FindRegisteredRoleForSetup(tenantID int64) (*model.Role, error)
	FindSuperAdminRoleForSetup(tenantID int64) (*model.Role, error)
}

type roleRepository struct {
	*BaseRepository[model.Role]
	db *gorm.DB
}

func NewRoleRepository(db *gorm.DB) RoleRepository {
	return &roleRepository{
		BaseRepository: NewBaseRepository[model.Role](db, "role_uuid", "role_id"),
		db:             db,
	}
}

func (r *roleRepository) WithTx(tx *gorm.DB) RoleRepository {
	return &roleRepository{
		BaseRepository: NewBaseRepository[model.Role](tx, "role_uuid", "role_id"),
		db:             tx,
	}
}

func (r *roleRepository) FindByNameAndTenantID(name string, tenantID int64) (*model.Role, error) {
	var role model.Role
	err := r.db.
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

func (r *roleRepository) FindAllByTenantID(tenantID int64) ([]model.Role, error) {
	var roles []model.Role
	err := r.db.
		Where("tenant_id = ?", tenantID).
		Find(&roles).Error
	return roles, err
}

func (r *roleRepository) FindPaginated(filter RoleRepositoryGetFilter) (*PaginationResult[model.Role], error) {
	query := r.db.Model(&model.Role{})

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
	if filter.IsActive != nil {
		query = query.Where("is_active = ?", *filter.IsActive)
	}

	// Sorting
	if filter.SortBy != "" {
		order := "ASC"
		if filter.SortOrder == "desc" {
			order = "DESC"
		}
		query = query.Order(filter.SortBy + " " + order)
	} else {
		query = query.Order("created_at DESC")
	}

	// Count
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// Pagination
	offset := (filter.Page - 1) * filter.Limit
	var roles []model.Role
	if err := query.Limit(filter.Limit).Offset(offset).Find(&roles).Error; err != nil {
		return nil, err
	}

	totalPages := int((total + int64(filter.Limit) - 1) / int64(filter.Limit))

	return &PaginationResult[model.Role]{
		Data:       roles,
		Total:      total,
		Page:       filter.Page,
		Limit:      filter.Limit,
		TotalPages: totalPages,
	}, nil
}

func (r *roleRepository) SetActiveStatusByUUID(roleUUID uuid.UUID, isActive bool) error {
	return r.db.Model(&model.Role{}).
		Where("role_uuid = ?", roleUUID).
		Update("is_active", isActive).Error
}

func (r *roleRepository) SetDefaultStatusByUUID(roleUUID uuid.UUID, isDefault bool) error {
	return r.db.Model(&model.Role{}).
		Where("role_uuid = ?", roleUUID).
		Update("is_default", isDefault).Error
}

func (r *roleRepository) SetSystemStatusByUUID(roleUUID uuid.UUID, isSystem bool) error {
	return r.db.Model(&model.Role{}).
		Where("role_uuid = ?", roleUUID).
		Update("is_system", isSystem).Error
}

func (r *roleRepository) FindRegisteredRoleForSetup(tenantID int64) (*model.Role, error) {
	var role model.Role
	err := r.db.Where("tenant_id = ? AND name = ? AND is_default = ? AND is_system = ?",
		tenantID, "registered", true, true).First(&role).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &role, nil
}

func (r *roleRepository) FindSuperAdminRoleForSetup(tenantID int64) (*model.Role, error) {
	var role model.Role
	err := r.db.Where("tenant_id = ? AND name = ? AND is_system = ?",
		tenantID, "super-admin", true).First(&role).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &role, nil
}
