package repository

import (
	"errors"

	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

type APIKeyPermissionRepository interface {
	BaseRepositoryMethods[model.APIKeyPermission]
	WithTx(tx *gorm.DB) APIKeyPermissionRepository
	FindByAPIKeyAPIAndPermission(apiKeyAPIID int64, permissionID int64) (*model.APIKeyPermission, error)
	RemoveByAPIKeyAPIAndPermission(apiKeyAPIID int64, permissionID int64) error
	FindByAPIKeyAPIID(apiKeyAPIID int64) ([]model.APIKeyPermission, error)
}

type apiKeyPermissionRepository struct {
	*BaseRepository[model.APIKeyPermission]
}

func NewAPIKeyPermissionRepository(db *gorm.DB) APIKeyPermissionRepository {
	return &apiKeyPermissionRepository{
		BaseRepository: NewBaseRepository[model.APIKeyPermission](db, "api_key_permission_uuid", "api_key_permission_id"),
	}
}

func (r *apiKeyPermissionRepository) WithTx(tx *gorm.DB) APIKeyPermissionRepository {
	return &apiKeyPermissionRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

func (r *apiKeyPermissionRepository) FindByAPIKeyAPIAndPermission(apiKeyAPIID int64, permissionID int64) (*model.APIKeyPermission, error) {
	var apiKeyPermission model.APIKeyPermission
	if err := r.DB().Where("api_key_api_id = ? AND permission_id = ?", apiKeyAPIID, permissionID).First(&apiKeyPermission).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &apiKeyPermission, nil
}

func (r *apiKeyPermissionRepository) RemoveByAPIKeyAPIAndPermission(apiKeyAPIID int64, permissionID int64) error {
	return r.DB().Where("api_key_api_id = ? AND permission_id = ?", apiKeyAPIID, permissionID).Delete(&model.APIKeyPermission{}).Error
}

func (r *apiKeyPermissionRepository) FindByAPIKeyAPIID(apiKeyAPIID int64) ([]model.APIKeyPermission, error) {
	var apiKeyPermissions []model.APIKeyPermission
	if err := r.DB().Where("api_key_api_id = ?", apiKeyAPIID).Preload("Permission").Find(&apiKeyPermissions).Error; err != nil {
		return nil, err
	}
	return apiKeyPermissions, nil
}
