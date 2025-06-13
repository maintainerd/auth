package repository

import (
	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

type UserPermissionRepository interface {
	BaseRepositoryMethods[model.UserPermission]
	FindByUserID(userID int64) ([]model.UserPermission, error)
	FindByUserIDAndPermissionID(userID int64, permissionID int64) (*model.UserPermission, error)
	FindDefaultPermissionsByUserID(userID int64) ([]model.UserPermission, error)
	DeleteByUserID(userID int64) error
	DeleteByUserIDAndPermissionID(userID int64, permissionID int64) error
}

type userPermissionRepository struct {
	*BaseRepository[model.UserPermission]
	db *gorm.DB
}

func NewUserPermissionRepository(db *gorm.DB) UserPermissionRepository {
	return &userPermissionRepository{
		BaseRepository: NewBaseRepository[model.UserPermission](db, "user_permission_uuid", "user_permission_id"),
		db:             db,
	}
}

func (r *userPermissionRepository) FindByUserID(userID int64) ([]model.UserPermission, error) {
	var userPermissions []model.UserPermission
	err := r.db.Where("user_id = ?", userID).Find(&userPermissions).Error
	return userPermissions, err
}

func (r *userPermissionRepository) FindByUserIDAndPermissionID(userID int64, permissionID int64) (*model.UserPermission, error) {
	var up model.UserPermission
	err := r.db.
		Where("user_id = ? AND permission_id = ?", userID, permissionID).
		First(&up).Error
	return &up, err
}

func (r *userPermissionRepository) FindDefaultPermissionsByUserID(userID int64) ([]model.UserPermission, error) {
	var userPermissions []model.UserPermission
	err := r.db.
		Where("user_id = ? AND is_default = true", userID).
		Find(&userPermissions).Error
	return userPermissions, err
}

func (r *userPermissionRepository) DeleteByUserID(userID int64) error {
	return r.db.Where("user_id = ?", userID).Delete(&model.UserPermission{}).Error
}

func (r *userPermissionRepository) DeleteByUserIDAndPermissionID(userID int64, permissionID int64) error {
	return r.db.
		Where("user_id = ? AND permission_id = ?", userID, permissionID).
		Delete(&model.UserPermission{}).Error
}
