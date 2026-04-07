package repository

import (
	"errors"

	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

type ClientPermissionRepository interface {
	BaseRepositoryMethods[model.ClientPermission]
	WithTx(tx *gorm.DB) ClientPermissionRepository
	FindByClientAPIAndPermission(clientAPIID int64, permissionID int64) (*model.ClientPermission, error)
	RemoveByClientAPIAndPermission(clientAPIID int64, permissionID int64) error
	FindByClientAPIID(clientAPIID int64) ([]model.ClientPermission, error)
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

func (r *clientPermissionRepository) FindByClientAPIAndPermission(clientAPIID int64, permissionID int64) (*model.ClientPermission, error) {
	var clientPermission model.ClientPermission
	err := r.DB().Where("client_api_id = ? AND permission_id = ?", clientAPIID, permissionID).First(&clientPermission).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &clientPermission, err
}

func (r *clientPermissionRepository) RemoveByClientAPIAndPermission(clientAPIID int64, permissionID int64) error {
	return r.DB().
		Where("client_api_id = ? AND permission_id = ?", clientAPIID, permissionID).
		Unscoped().Delete(&model.ClientPermission{}).Error
}

func (r *clientPermissionRepository) FindByClientAPIID(clientAPIID int64) ([]model.ClientPermission, error) {
	var permissions []model.ClientPermission
	err := r.DB().Where("client_api_id = ?", clientAPIID).
		Preload("Permission").
		Find(&permissions).Error

	if err != nil {
		return nil, err
	}

	return permissions, nil
}
