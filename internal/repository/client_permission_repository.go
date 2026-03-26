package repository

import (
	"errors"

	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

type ClientPermissionRepository interface {
	BaseRepositoryMethods[model.ClientPermission]
	WithTx(tx *gorm.DB) ClientPermissionRepository
	FindByClientApiAndPermission(ClientApiID int64, permissionID int64) (*model.ClientPermission, error)
	RemoveByClientApiAndPermission(ClientApiID int64, permissionID int64) error
	FindByClientApiID(ClientApiID int64) ([]model.ClientPermission, error)
}

type clientPermissionRepository struct {
	*BaseRepository[model.ClientPermission]
	db *gorm.DB
}

func NewClientPermissionRepository(db *gorm.DB) ClientPermissionRepository {
	return &clientPermissionRepository{
		BaseRepository: NewBaseRepository[model.ClientPermission](db, "client_permission_uuid", "client_permission_id"),
		db:             db,
	}
}

func (r *clientPermissionRepository) WithTx(tx *gorm.DB) ClientPermissionRepository {
	return &clientPermissionRepository{
		BaseRepository: NewBaseRepository[model.ClientPermission](tx, "client_permission_uuid", "client_permission_id"),
		db:             tx,
	}
}

func (r *clientPermissionRepository) FindByClientApiAndPermission(ClientApiID int64, permissionID int64) (*model.ClientPermission, error) {
	var ClientPermission model.ClientPermission
	err := r.db.Where("client_api_id = ? AND permission_id = ?", ClientApiID, permissionID).First(&ClientPermission).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &ClientPermission, err
}

func (r *clientPermissionRepository) RemoveByClientApiAndPermission(ClientApiID int64, permissionID int64) error {
	return r.db.
		Where("client_api_id = ? AND permission_id = ?", ClientApiID, permissionID).
		Unscoped().Delete(&model.ClientPermission{}).Error
}

func (r *clientPermissionRepository) FindByClientApiID(ClientApiID int64) ([]model.ClientPermission, error) {
	var permissions []model.ClientPermission
	err := r.db.Where("client_api_id = ?", ClientApiID).
		Preload("Permission").
		Find(&permissions).Error

	if err != nil {
		return nil, err
	}

	return permissions, nil
}
