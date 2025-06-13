package repository

import (
	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

type APIRepository interface {
	BaseRepositoryMethods[model.API]
	FindByName(apiName string, authContainerID int64) (*model.API, error)
	FindByIdentifier(identifier string, authContainerID int64) (*model.API, error)
	FindAllByServiceID(serviceID int64) ([]model.API, error)
	FindAllByAuthContainerID(authContainerID int64) ([]model.API, error)
	FindDefaultByServiceID(serviceID int64) (*model.API, error)
	SetActiveStatusByUUID(apiUUID uuid.UUID, isActive bool) error
	SetDefaultStatusByUUID(apiUUID uuid.UUID, isDefault bool) error
}

type apiRepository struct {
	*BaseRepository[model.API]
	db *gorm.DB
}

func NewAPIRepository(db *gorm.DB) APIRepository {
	return &apiRepository{
		BaseRepository: NewBaseRepository[model.API](db, "api_uuid", "api_id"),
		db:             db,
	}
}

func (r *apiRepository) FindByName(apiName string, authContainerID int64) (*model.API, error) {
	var api model.API
	err := r.db.
		Where("api_name = ? AND auth_container_id = ?", apiName, authContainerID).
		First(&api).Error
	return &api, err
}

func (r *apiRepository) FindByIdentifier(identifier string, authContainerID int64) (*model.API, error) {
	var api model.API
	err := r.db.
		Where("identifier = ? AND auth_container_id = ?", identifier, authContainerID).
		First(&api).Error
	return &api, err
}

func (r *apiRepository) FindAllByServiceID(serviceID int64) ([]model.API, error) {
	var apis []model.API
	err := r.db.
		Where("service_id = ?", serviceID).
		Find(&apis).Error
	return apis, err
}

func (r *apiRepository) FindAllByAuthContainerID(authContainerID int64) ([]model.API, error) {
	var apis []model.API
	err := r.db.
		Where("auth_container_id = ?", authContainerID).
		Find(&apis).Error
	return apis, err
}

func (r *apiRepository) FindDefaultByServiceID(serviceID int64) (*model.API, error) {
	var api model.API
	err := r.db.
		Where("service_id = ? AND is_default = true", serviceID).
		First(&api).Error
	return &api, err
}

func (r *apiRepository) SetActiveStatusByUUID(apiUUID uuid.UUID, isActive bool) error {
	return r.db.Model(&model.API{}).
		Where("api_uuid = ?", apiUUID).
		Update("is_active", isActive).Error
}

func (r *apiRepository) SetDefaultStatusByUUID(apiUUID uuid.UUID, isDefault bool) error {
	return r.db.Model(&model.API{}).
		Where("api_uuid = ?", apiUUID).
		Update("is_default", isDefault).Error
}
