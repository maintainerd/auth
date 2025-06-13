package repository

import (
	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

type PermissionRepository interface {
	BaseRepositoryMethods[model.Permission]
	FindByName(name string) (*model.Permission, error)
	FindAllByAPIID(apiID int64) ([]model.Permission, error)
	FindAllByAuthContainerID(authContainerID int64) ([]model.Permission, error)
	FindDefaultPermissionsByAPIID(apiID int64) ([]model.Permission, error)
	SetActiveStatusByUUID(permissionUUID uuid.UUID, isActive bool) error
	SetDefaultStatusByUUID(permissionUUID uuid.UUID, isDefault bool) error
	FindAllByRoleID(roleID int64) ([]model.Permission, error)
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

func (r *permissionRepository) FindByName(name string) (*model.Permission, error) {
	var permission model.Permission
	err := r.db.Where("name = ?", name).First(&permission).Error
	return &permission, err
}

func (r *permissionRepository) FindAllByAPIID(apiID int64) ([]model.Permission, error) {
	var permissions []model.Permission
	err := r.db.Where("api_id = ?", apiID).Find(&permissions).Error
	return permissions, err
}

func (r *permissionRepository) FindAllByAuthContainerID(authContainerID int64) ([]model.Permission, error) {
	var permissions []model.Permission
	err := r.db.Where("auth_container_id = ?", authContainerID).Find(&permissions).Error
	return permissions, err
}

func (r *permissionRepository) FindDefaultPermissionsByAPIID(apiID int64) ([]model.Permission, error) {
	var permissions []model.Permission
	err := r.db.Where("api_id = ? AND is_default = true", apiID).Find(&permissions).Error
	return permissions, err
}

func (r *permissionRepository) SetActiveStatusByUUID(permissionUUID uuid.UUID, isActive bool) error {
	return r.db.Model(&model.Permission{}).
		Where("permission_uuid = ?", permissionUUID).
		Update("is_active", isActive).Error
}

func (r *permissionRepository) SetDefaultStatusByUUID(permissionUUID uuid.UUID, isDefault bool) error {
	return r.db.Model(&model.Permission{}).
		Where("permission_uuid = ?", permissionUUID).
		Update("is_default", isDefault).Error
}

func (r *permissionRepository) FindAllByRoleID(roleID int64) ([]model.Permission, error) {
	var permissions []model.Permission
	err := r.db.Joins("JOIN role_permissions ON role_permissions.permission_id = permissions.permission_id").
		Where("role_permissions.role_id = ?", roleID).
		Find(&permissions).Error
	return permissions, err
}
