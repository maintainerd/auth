package iam

import (
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RolePermissionRepository interface {
	BaseRepositoryMethods[RolePermission]
	WithTx(tx *gorm.DB) RolePermissionRepository
	Assign(rolePermission *RolePermission) (*RolePermission, error)
	FindByRoleAndPermission(roleID int64, permissionID int64) (*RolePermission, error)
	FindAllByRoleID(roleID int64) ([]RolePermission, error)
	FindAllByPermissionID(permissionID int64) ([]RolePermission, error)
	RemoveByRoleAndPermission(roleID int64, permissionID int64) error
	SetDefaultStatusByUUID(rolePermissionUUID uuid.UUID, isDefault bool) error
}

type rolePermissionRepository struct {
	*BaseRepository[RolePermission]
}

func NewRolePermissionRepository(db *gorm.DB) RolePermissionRepository {
	return &rolePermissionRepository{
		BaseRepository: NewBaseRepository[RolePermission](db, "role_permission_uuid", "role_permission_id"),
	}
}

func (r *rolePermissionRepository) WithTx(tx *gorm.DB) RolePermissionRepository {
	return &rolePermissionRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

// Assign a role-permission pair and return the created record
func (r *rolePermissionRepository) Assign(rolePermission *RolePermission) (*RolePermission, error) {
	return r.Create(rolePermission)
}

func (r *rolePermissionRepository) FindByRoleAndPermission(roleID int64, permissionID int64) (*RolePermission, error) {
	var rp RolePermission
	err := r.DB().
		Where("role_id = ? AND permission_id = ?", roleID, permissionID).
		First(&rp).Error

	// If no record is found, return nil record and nil error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		// For all other errors, return nil record and the actual error
		return nil, err
	}

	return &rp, nil
}

func (r *rolePermissionRepository) FindAllByRoleID(roleID int64) ([]RolePermission, error) {
	var rps []RolePermission
	err := r.DB().Where("role_id = ?", roleID).Find(&rps).Error
	return rps, err
}

func (r *rolePermissionRepository) FindAllByPermissionID(permissionID int64) ([]RolePermission, error) {
	var rps []RolePermission
	err := r.DB().Where("permission_id = ?", permissionID).Find(&rps).Error
	return rps, err
}

func (r *rolePermissionRepository) RemoveByRoleAndPermission(roleID int64, permissionID int64) error {
	return r.DB().
		Where("role_id = ? AND permission_id = ?", roleID, permissionID).
		Unscoped().Delete(&RolePermission{}).Error
}

func (r *rolePermissionRepository) SetDefaultStatusByUUID(rolePermissionUUID uuid.UUID, isDefault bool) error {
	return r.DB().Model(&RolePermission{}).
		Where("role_permission_uuid = ?", rolePermissionUUID).
		Update("is_default", isDefault).Error
}
