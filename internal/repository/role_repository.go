package repository

import (
	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

type RoleRepository interface {
	Create(role *model.Role) error
	FindAll() ([]model.Role, error)
	FindByUUID(roleUUID uuid.UUID) (*model.Role, error)
	UpdateByUUID(roleUUID uuid.UUID, updatedRole *model.Role) error
	DeleteByUUID(roleUUID uuid.UUID) error
	FindByName(name string, authContainerID int64) (*model.Role, error)
}

type roleRepository struct {
	db *gorm.DB
}

func NewRoleRepository(db *gorm.DB) RoleRepository {
	return &roleRepository{db}
}

func (r *roleRepository) Create(role *model.Role) error {
	return r.db.Create(role).Error
}

func (r *roleRepository) FindAll() ([]model.Role, error) {
	var roles []model.Role
	err := r.db.Find(&roles).Error
	return roles, err
}

func (r *roleRepository) FindByUUID(roleUUID uuid.UUID) (*model.Role, error) {
	var role model.Role
	err := r.db.Where("role_uuid = ?", roleUUID).First(&role).Error
	return &role, err
}

func (r *roleRepository) UpdateByUUID(roleUUID uuid.UUID, updatedRole *model.Role) error {
	// Ensure we target the correct row by UUID
	return r.db.Model(&model.Role{}).
		Where("role_uuid = ?", roleUUID).
		Updates(updatedRole).Error
}

func (r *roleRepository) DeleteByUUID(roleUUID uuid.UUID) error {
	return r.db.Where("role_uuid = ?", roleUUID).Delete(&model.Role{}).Error
}

func (r *roleRepository) FindByName(name string, authContainerID int64) (*model.Role, error) {
	var role model.Role
	err := r.db.
		Where("name = ? AND auth_container_id = ?", name, authContainerID).
		First(&role).Error
	return &role, err
}
