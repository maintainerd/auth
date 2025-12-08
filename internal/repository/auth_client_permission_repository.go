package repository

import (
	"errors"

	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

type AuthClientPermissionRepository interface {
	BaseRepositoryMethods[model.AuthClientPermission]
	WithTx(tx *gorm.DB) AuthClientPermissionRepository
	FindByAuthClientApiAndPermission(authClientApiID int64, permissionID int64) (*model.AuthClientPermission, error)
	RemoveByAuthClientApiAndPermission(authClientApiID int64, permissionID int64) error
	FindByAuthClientApiID(authClientApiID int64) ([]model.AuthClientPermission, error)
}

type authClientPermissionRepository struct {
	*BaseRepository[model.AuthClientPermission]
	db *gorm.DB
}

func NewAuthClientPermissionRepository(db *gorm.DB) AuthClientPermissionRepository {
	return &authClientPermissionRepository{
		BaseRepository: NewBaseRepository[model.AuthClientPermission](db, "auth_client_permission_uuid", "auth_client_permission_id"),
		db:             db,
	}
}

func (r *authClientPermissionRepository) WithTx(tx *gorm.DB) AuthClientPermissionRepository {
	return &authClientPermissionRepository{
		BaseRepository: NewBaseRepository[model.AuthClientPermission](tx, "auth_client_permission_uuid", "auth_client_permission_id"),
		db:             tx,
	}
}

func (r *authClientPermissionRepository) FindByAuthClientApiAndPermission(authClientApiID int64, permissionID int64) (*model.AuthClientPermission, error) {
	var authClientPermission model.AuthClientPermission
	err := r.db.Where("auth_client_api_id = ? AND permission_id = ?", authClientApiID, permissionID).First(&authClientPermission).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &authClientPermission, err
}

func (r *authClientPermissionRepository) RemoveByAuthClientApiAndPermission(authClientApiID int64, permissionID int64) error {
	return r.db.
		Where("auth_client_api_id = ? AND permission_id = ?", authClientApiID, permissionID).
		Unscoped().Delete(&model.AuthClientPermission{}).Error
}

func (r *authClientPermissionRepository) FindByAuthClientApiID(authClientApiID int64) ([]model.AuthClientPermission, error) {
	var permissions []model.AuthClientPermission
	err := r.db.Where("auth_client_api_id = ?", authClientApiID).
		Preload("Permission").
		Find(&permissions).Error

	if err != nil {
		return nil, err
	}

	return permissions, nil
}
