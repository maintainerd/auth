package repository

import (
	"errors"

	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

type ClientPermissionRepository interface {
	BaseRepositoryMethods[model.ClientPermission]
	WithTx(tx *gorm.DB) ClientPermissionRepository
	FindByClientApiAndPermission(clientApiID int64, permissionID int64) (*model.ClientPermission, error)
	RemoveByClientApiAndPermission(clientApiID int64, permissionID int64) error
	FindByClientApiID(clientApiID int64) ([]model.ClientPermission, error)
}

type clientPermissionRepository struct {
	*BaseRepository[model.ClientPermission]
}

func NewClientPermissionRepository(db *gorm.DB) ClientPermissionRepository {
	return &clientPermissionRepository{
		BaseRepository: NewBaseRepository[model.ClientPermission](db, "client_permission_uuid", "client_permission_id"),
	}
}

func (r *clientPermissionRepository) WithTx(tx *gorm.DB) ClientPermissionRepository {
	return &clientPermissionRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

func (r *clientPermissionRepository) FindByClientApiAndPermission(clientApiID int64, permissionID int64) (*model.ClientPermission, error) {
	var clientPermission model.ClientPermission
	err := r.DB().Where("client_api_id = ? AND permission_id = ?", clientApiID, permissionID).First(&clientPermission).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &clientPermission, err
}

func (r *clientPermissionRepository) RemoveByClientApiAndPermission(clientApiID int64, permissionID int64) error {
	return r.DB().
		Where("client_api_id = ? AND permission_id = ?", clientApiID, permissionID).
		Unscoped().Delete(&model.ClientPermission{}).Error
}

func (r *clientPermissionRepository) FindByClientApiID(clientApiID int64) ([]model.ClientPermission, error) {
	var permissions []model.ClientPermission
	err := r.DB().Where("client_api_id = ?", clientApiID).
		Preload("Permission").
		Find(&permissions).Error

	if err != nil {
		return nil, err
	}

	return permissions, nil
}
