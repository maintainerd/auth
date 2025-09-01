package repository

import (
	"errors"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

type RoleRepositoryGetFilter struct {
	Name            *string
	Description     *string
	IsDefault       *bool
	IsActive        *bool
	AuthContainerID int64
	Page            int
	Limit           int
	SortBy          string
	SortOrder       string
}

type RoleRepository interface {
	BaseRepositoryMethods[model.Role]
	WithTx(tx *gorm.DB) RoleRepository
	FindByNameAndAuthContainerID(name string, authContainerID int64) (*model.Role, error)
	FindAllByAuthContainerID(authContainerID int64) ([]model.Role, error)
	FindPaginated(filter RoleRepositoryGetFilter) (*PaginationResult[model.Role], error)
	SetActiveStatusByUUID(roleUUID uuid.UUID, isActive bool) error
	SetDefaultStatusByUUID(roleUUID uuid.UUID, isDefault bool) error
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

func (r *roleRepository) FindByNameAndAuthContainerID(name string, authContainerID int64) (*model.Role, error) {
	var role model.Role
	err := r.db.
		Where("name = ? AND auth_container_id = ?", name, authContainerID).
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

func (r *roleRepository) FindAllByAuthContainerID(authContainerID int64) ([]model.Role, error) {
	var roles []model.Role
	err := r.db.
		Where("auth_container_id = ?", authContainerID).
		Find(&roles).Error
	return roles, err
}

func (r *roleRepository) FindPaginated(filter RoleRepositoryGetFilter) (*PaginationResult[model.Role], error) {
	conditions := map[string]any{
		"auth_container_id": filter.AuthContainerID,
	}
	if filter.Name != nil {
		conditions["name"] = *filter.Name
	}
	if filter.Description != nil {
		conditions["description"] = *filter.Description
	}
	if filter.IsDefault != nil {
		conditions["is_default"] = *filter.IsDefault
	}
	if filter.IsActive != nil {
		conditions["is_active"] = *filter.IsActive
	}

	// Sorting
	orderBy := filter.SortBy + " " + filter.SortOrder

	query := r.db.Model(&model.Role{}).Where(conditions).Order(orderBy)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

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
