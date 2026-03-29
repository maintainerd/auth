package repository

import (
	"errors"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

type ClientApiRepository interface {
	BaseRepositoryMethods[model.ClientApi]
	WithTx(tx *gorm.DB) ClientApiRepository
	FindByClientAndApi(clientID int64, apiID int64) (*model.ClientApi, error)
	FindByClientUUID(clientUUID uuid.UUID) ([]model.ClientApi, error)
	FindByClientUUIDAndApiUUID(clientUUID uuid.UUID, apiUUID uuid.UUID) (*model.ClientApi, error)
	RemoveByClientAndApi(clientID int64, apiID int64) error
	RemoveByClientUUIDAndApiUUID(clientUUID uuid.UUID, apiUUID uuid.UUID) error
}

type clientApiRepository struct {
	*BaseRepository[model.ClientApi]
}

func NewClientApiRepository(db *gorm.DB) ClientApiRepository {
	return &clientApiRepository{
		BaseRepository: NewBaseRepository[model.ClientApi](db, "client_api_uuid", "client_api_id"),
	}
}

func (r *clientApiRepository) WithTx(tx *gorm.DB) ClientApiRepository {
	return &clientApiRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

func (r *clientApiRepository) FindByClientAndApi(clientID int64, apiID int64) (*model.ClientApi, error) {
	var clientApi model.ClientApi
	err := r.DB().Where("client_id = ? AND api_id = ?", clientID, apiID).First(&clientApi).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &clientApi, err
}

func (r *clientApiRepository) FindByClientUUID(clientUUID uuid.UUID) ([]model.ClientApi, error) {
	var clientApis []model.ClientApi
	err := r.DB().Joins("JOIN clients ON clients.client_id = client_apis.client_id").
		Where("clients.client_uuid = ?", clientUUID).
		Preload("API").
		Preload("Permissions.Permission").
		Find(&clientApis).Error

	if err != nil {
		return nil, err
	}

	return clientApis, nil
}

func (r *clientApiRepository) FindByClientUUIDAndApiUUID(clientUUID uuid.UUID, apiUUID uuid.UUID) (*model.ClientApi, error) {
	var clientApi model.ClientApi
	err := r.DB().Joins("JOIN clients ON clients.client_id = client_apis.client_id").
		Joins("JOIN apis ON apis.api_id = client_apis.api_id").
		Where("clients.client_uuid = ? AND apis.api_uuid = ?", clientUUID, apiUUID).
		Preload("API").
		Preload("Permissions.Permission").
		First(&clientApi).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &clientApi, nil
}

func (r *clientApiRepository) RemoveByClientAndApi(clientID int64, apiID int64) error {
	return r.DB().
		Where("client_id = ? AND api_id = ?", clientID, apiID).
		Unscoped().Delete(&model.ClientApi{}).Error
}

func (r *clientApiRepository) RemoveByClientUUIDAndApiUUID(clientUUID uuid.UUID, apiUUID uuid.UUID) error {
	// First, find the client_api record to get the IDs
	var clientApi model.ClientApi
	err := r.DB().Joins("JOIN clients ON clients.client_id = client_apis.client_id").
		Joins("JOIN apis ON apis.api_id = client_apis.api_id").
		Where("clients.client_uuid = ? AND apis.api_uuid = ?", clientUUID, apiUUID).
		First(&clientApi).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil // Already deleted or doesn't exist
		}
		return err
	}

	// Now delete using the primary key
	return r.DB().Unscoped().Delete(&clientApi).Error
}
