package repository

import (
	"errors"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

type ClientAPIRepository interface {
	BaseRepositoryMethods[model.ClientAPI]
	WithTx(tx *gorm.DB) ClientAPIRepository
	FindByClientAndAPI(clientID int64, apiID int64) (*model.ClientAPI, error)
	FindByClientUUID(clientUUID uuid.UUID) ([]model.ClientAPI, error)
	FindByClientUUIDAndAPIUUID(clientUUID uuid.UUID, apiUUID uuid.UUID) (*model.ClientAPI, error)
	RemoveByClientAndAPI(clientID int64, apiID int64) error
	RemoveByClientUUIDAndAPIUUID(clientUUID uuid.UUID, apiUUID uuid.UUID) error
}

type clientAPIRepository struct {
	*BaseRepository[model.ClientAPI]
}

func NewClientAPIRepository(db *gorm.DB) ClientAPIRepository {
	return &clientAPIRepository{
		BaseRepository: NewBaseRepository[model.ClientAPI](db, "client_api_uuid", "client_api_id"),
	}
}

func (r *clientAPIRepository) WithTx(tx *gorm.DB) ClientAPIRepository {
	return &clientAPIRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

func (r *clientAPIRepository) FindByClientAndAPI(clientID int64, apiID int64) (*model.ClientAPI, error) {
	var clientAPI model.ClientAPI
	err := r.DB().Where("client_id = ? AND api_id = ?", clientID, apiID).First(&clientAPI).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &clientAPI, err
}

func (r *clientAPIRepository) FindByClientUUID(clientUUID uuid.UUID) ([]model.ClientAPI, error) {
	var clientAPIs []model.ClientAPI
	err := r.DB().Joins("JOIN clients ON clients.client_id = client_apis.client_id").
		Where("clients.client_uuid = ?", clientUUID).
		Preload("API").
		Preload("Permissions.Permission").
		Find(&clientAPIs).Error

	if err != nil {
		return nil, err
	}

	return clientAPIs, nil
}

func (r *clientAPIRepository) FindByClientUUIDAndAPIUUID(clientUUID uuid.UUID, apiUUID uuid.UUID) (*model.ClientAPI, error) {
	var clientAPI model.ClientAPI
	err := r.DB().Joins("JOIN clients ON clients.client_id = client_apis.client_id").
		Joins("JOIN apis ON apis.api_id = client_apis.api_id").
		Where("clients.client_uuid = ? AND apis.api_uuid = ?", clientUUID, apiUUID).
		Preload("API").
		Preload("Permissions.Permission").
		First(&clientAPI).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &clientAPI, nil
}

func (r *clientAPIRepository) RemoveByClientAndAPI(clientID int64, apiID int64) error {
	return r.DB().
		Where("client_id = ? AND api_id = ?", clientID, apiID).
		Unscoped().Delete(&model.ClientAPI{}).Error
}

func (r *clientAPIRepository) RemoveByClientUUIDAndAPIUUID(clientUUID uuid.UUID, apiUUID uuid.UUID) error {
	// First, find the client_api record to get the IDs
	var clientAPI model.ClientAPI
	err := r.DB().Joins("JOIN clients ON clients.client_id = client_apis.client_id").
		Joins("JOIN apis ON apis.api_id = client_apis.api_id").
		Where("clients.client_uuid = ? AND apis.api_uuid = ?", clientUUID, apiUUID).
		First(&clientAPI).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil // Already deleted or doesn't exist
		}
		return err
	}

	// Now delete using the primary key
	return r.DB().Unscoped().Delete(&clientAPI).Error
}
