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
	FindByClientAndApi(ClientID int64, apiID int64) (*model.ClientApi, error)
	FindByClientUUID(ClientUUID uuid.UUID) ([]model.ClientApi, error)
	FindByClientUUIDAndApiUUID(ClientUUID uuid.UUID, apiUUID uuid.UUID) (*model.ClientApi, error)
	RemoveByClientAndApi(ClientID int64, apiID int64) error
	RemoveByClientUUIDAndApiUUID(ClientUUID uuid.UUID, apiUUID uuid.UUID) error
}

type clientApiRepository struct {
	*BaseRepository[model.ClientApi]
	db *gorm.DB
}

func NewClientApiRepository(db *gorm.DB) ClientApiRepository {
	return &clientApiRepository{
		BaseRepository: NewBaseRepository[model.ClientApi](db, "client_api_uuid", "client_api_id"),
		db:             db,
	}
}

func (r *clientApiRepository) WithTx(tx *gorm.DB) ClientApiRepository {
	return &clientApiRepository{
		BaseRepository: NewBaseRepository[model.ClientApi](tx, "client_api_uuid", "client_api_id"),
		db:             tx,
	}
}

func (r *clientApiRepository) FindByClientAndApi(ClientID int64, apiID int64) (*model.ClientApi, error) {
	var ClientApi model.ClientApi
	err := r.db.Where("client_id = ? AND api_id = ?", ClientID, apiID).First(&ClientApi).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &ClientApi, err
}

func (r *clientApiRepository) FindByClientUUID(ClientUUID uuid.UUID) ([]model.ClientApi, error) {
	var ClientApis []model.ClientApi
	err := r.db.Joins("JOIN clients ON clients.client_id = client_apis.client_id").
		Where("clients.client_uuid = ?", ClientUUID).
		Preload("API").
		Preload("Permissions.Permission").
		Find(&ClientApis).Error

	if err != nil {
		return nil, err
	}

	return ClientApis, nil
}

func (r *clientApiRepository) FindByClientUUIDAndApiUUID(ClientUUID uuid.UUID, apiUUID uuid.UUID) (*model.ClientApi, error) {
	var ClientApi model.ClientApi
	err := r.db.Joins("JOIN clients ON clients.client_id = client_apis.client_id").
		Joins("JOIN apis ON apis.api_id = client_apis.api_id").
		Where("clients.client_uuid = ? AND apis.api_uuid = ?", ClientUUID, apiUUID).
		Preload("API").
		Preload("Permissions.Permission").
		First(&ClientApi).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &ClientApi, nil
}

func (r *clientApiRepository) RemoveByClientAndApi(ClientID int64, apiID int64) error {
	return r.db.
		Where("client_id = ? AND api_id = ?", ClientID, apiID).
		Unscoped().Delete(&model.ClientApi{}).Error
}

func (r *clientApiRepository) RemoveByClientUUIDAndApiUUID(ClientUUID uuid.UUID, apiUUID uuid.UUID) error {
	// First, find the client_api record to get the IDs
	var ClientApi model.ClientApi
	err := r.db.Joins("JOIN clients ON clients.client_id = client_apis.client_id").
		Joins("JOIN apis ON apis.api_id = client_apis.api_id").
		Where("clients.client_uuid = ? AND apis.api_uuid = ?", ClientUUID, apiUUID).
		First(&ClientApi).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil // Already deleted or doesn't exist
		}
		return err
	}

	// Now delete using the primary key
	return r.db.Unscoped().Delete(&ClientApi).Error
}
