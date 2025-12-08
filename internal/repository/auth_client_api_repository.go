package repository

import (
	"errors"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

type AuthClientApiRepository interface {
	BaseRepositoryMethods[model.AuthClientApi]
	WithTx(tx *gorm.DB) AuthClientApiRepository
	FindByAuthClientAndApi(authClientID int64, apiID int64) (*model.AuthClientApi, error)
	FindByAuthClientUUID(authClientUUID uuid.UUID) ([]model.AuthClientApi, error)
	FindByAuthClientUUIDAndApiUUID(authClientUUID uuid.UUID, apiUUID uuid.UUID) (*model.AuthClientApi, error)
	RemoveByAuthClientAndApi(authClientID int64, apiID int64) error
	RemoveByAuthClientUUIDAndApiUUID(authClientUUID uuid.UUID, apiUUID uuid.UUID) error
}

type authClientApiRepository struct {
	*BaseRepository[model.AuthClientApi]
	db *gorm.DB
}

func NewAuthClientApiRepository(db *gorm.DB) AuthClientApiRepository {
	return &authClientApiRepository{
		BaseRepository: NewBaseRepository[model.AuthClientApi](db, "auth_client_api_uuid", "auth_client_api_id"),
		db:             db,
	}
}

func (r *authClientApiRepository) WithTx(tx *gorm.DB) AuthClientApiRepository {
	return &authClientApiRepository{
		BaseRepository: NewBaseRepository[model.AuthClientApi](tx, "auth_client_api_uuid", "auth_client_api_id"),
		db:             tx,
	}
}

func (r *authClientApiRepository) FindByAuthClientAndApi(authClientID int64, apiID int64) (*model.AuthClientApi, error) {
	var authClientApi model.AuthClientApi
	err := r.db.Where("auth_client_id = ? AND api_id = ?", authClientID, apiID).First(&authClientApi).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &authClientApi, err
}

func (r *authClientApiRepository) FindByAuthClientUUID(authClientUUID uuid.UUID) ([]model.AuthClientApi, error) {
	var authClientApis []model.AuthClientApi
	err := r.db.Joins("JOIN auth_clients ON auth_clients.auth_client_id = auth_client_apis.auth_client_id").
		Where("auth_clients.auth_client_uuid = ?", authClientUUID).
		Preload("API").
		Preload("Permissions.Permission").
		Find(&authClientApis).Error

	if err != nil {
		return nil, err
	}

	return authClientApis, nil
}

func (r *authClientApiRepository) FindByAuthClientUUIDAndApiUUID(authClientUUID uuid.UUID, apiUUID uuid.UUID) (*model.AuthClientApi, error) {
	var authClientApi model.AuthClientApi
	err := r.db.Joins("JOIN auth_clients ON auth_clients.auth_client_id = auth_client_apis.auth_client_id").
		Joins("JOIN apis ON apis.api_id = auth_client_apis.api_id").
		Where("auth_clients.auth_client_uuid = ? AND apis.api_uuid = ?", authClientUUID, apiUUID).
		Preload("API").
		Preload("Permissions.Permission").
		First(&authClientApi).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &authClientApi, nil
}

func (r *authClientApiRepository) RemoveByAuthClientAndApi(authClientID int64, apiID int64) error {
	return r.db.
		Where("auth_client_id = ? AND api_id = ?", authClientID, apiID).
		Unscoped().Delete(&model.AuthClientApi{}).Error
}

func (r *authClientApiRepository) RemoveByAuthClientUUIDAndApiUUID(authClientUUID uuid.UUID, apiUUID uuid.UUID) error {
	// First, find the auth_client_api record to get the IDs
	var authClientApi model.AuthClientApi
	err := r.db.Joins("JOIN auth_clients ON auth_clients.auth_client_id = auth_client_apis.auth_client_id").
		Joins("JOIN apis ON apis.api_id = auth_client_apis.api_id").
		Where("auth_clients.auth_client_uuid = ? AND apis.api_uuid = ?", authClientUUID, apiUUID).
		First(&authClientApi).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil // Already deleted or doesn't exist
		}
		return err
	}

	// Now delete using the primary key
	return r.db.Unscoped().Delete(&authClientApi).Error
}
