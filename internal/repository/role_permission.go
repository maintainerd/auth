package repository

import (
	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

type RolePermissionRepository interface {
	BaseRepositoryMethods[model.RolePermission]
	Assign(rolePermission *model.RolePermission) (*model.RolePermission, error)
	FindByRoleAndPermission(roleID int64, permissionID int64) (*model.RolePermission, error)
	FindAllByRoleID(roleID int64) ([]model.RolePermission, error)
	FindAllByPermissionID(permissionID int64) ([]model.RolePermission, error)
	RemoveByRoleAndPermission(roleID int64, permissionID int64) error
	SetDefaultStatusByUUID(rolePermissionUUID uuid.UUID, isDefault bool) error
}

type rolePermissionRepository struct {
	*BaseRepository[model.RolePermission]
	db *gorm.DB
}

func NewRolePermissionRepository(db *gorm.DB) RolePermissionRepository {
	return &rolePermissionRepository{
		BaseRepository: NewBaseRepository[model.RolePermission](db, "role_permission_uuid", "role_permission_id"),
		db:             db,
	}
}

// Assign a role-permission pair and return the created record
func (r *rolePermissionRepository) Assign(rolePermission *model.RolePermission) (*model.RolePermission, error) {
	return r.Create(rolePermission)
}

func (r *rolePermissionRepository) FindByRoleAndPermission(roleID int64, permissionID int64) (*model.RolePermission, error) {
	var rp model.RolePermission
	err := r.db.
		Where("role_id = ? AND permission_id = ?", roleID, permissionID).
		First(&rp).Error
	return &rp, err
}

func (r *rolePermissionRepository) FindAllByRoleID(roleID int64) ([]model.RolePermission, error) {
	var rps []model.RolePermission
	err := r.db.Where("role_id = ?", roleID).Find(&rps).Error
	return rps, err
}

func (r *rolePermissionRepository) FindAllByPermissionID(permissionID int64) ([]model.RolePermission, error) {
	var rps []model.RolePermission
	err := r.db.Where("permission_id = ?", permissionID).Find(&rps).Error
	return rps, err
}

func (r *rolePermissionRepository) RemoveByRoleAndPermission(roleID int64, permissionID int64) error {
	return r.db.
		Where("role_id = ? AND permission_id = ?", roleID, permissionID).
		Delete(&model.RolePermission{}).Error
}

func (r *rolePermissionRepository) SetDefaultStatusByUUID(rolePermissionUUID uuid.UUID, isDefault bool) error {
	return r.db.Model(&model.RolePermission{}).
		Where("role_permission_uuid = ?", rolePermissionUUID).
		Update("is_default", isDefault).Error
}
