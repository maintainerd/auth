package repository

import (
	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

type RoleRepository interface {
	BaseRepositoryMethods[model.Role]
	FindByName(name string, authContainerID int64) (*model.Role, error)
	FindAllByAuthContainerID(authContainerID int64) ([]model.Role, error)
	FindDefaultRolesByAuthContainerID(authContainerID int64) ([]model.Role, error)
	SetActiveStatusByUUID(roleUUID uuid.UUID, isActive bool) error
	SetDefaultStatusByUUID(roleUUID uuid.UUID, isDefault bool) error
	FindAllWithPermissionsByAuthContainerID(authContainerID int64) ([]model.Role, error)
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

func (r *roleRepository) FindByName(name string, authContainerID int64) (*model.Role, error) {
	var role model.Role
	err := r.db.
		Where("name = ? AND auth_container_id = ?", name, authContainerID).
		First(&role).Error
	return &role, err
}

func (r *roleRepository) FindAllByAuthContainerID(authContainerID int64) ([]model.Role, error) {
	var roles []model.Role
	err := r.db.
		Where("auth_container_id = ?", authContainerID).
		Find(&roles).Error
	return roles, err
}

func (r *roleRepository) FindDefaultRolesByAuthContainerID(authContainerID int64) ([]model.Role, error) {
	var roles []model.Role
	err := r.db.
		Where("auth_container_id = ? AND is_default = true", authContainerID).
		Find(&roles).Error
	return roles, err
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

func (r *roleRepository) FindAllWithPermissionsByAuthContainerID(authContainerID int64) ([]model.Role, error) {
	var roles []model.Role
	err := r.db.
		Preload("Permissions").
		Where("auth_container_id = ?", authContainerID).
		Find(&roles).Error
	return roles, err
}
