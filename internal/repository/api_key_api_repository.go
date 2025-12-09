package repository

import (
	"errors"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

type APIKeyApiRepository interface {
	BaseRepositoryMethods[model.APIKeyApi]
	WithTx(tx *gorm.DB) APIKeyApiRepository
	FindByAPIKeyAndApi(apiKeyID int64, apiID int64) (*model.APIKeyApi, error)
	FindByAPIKeyUUID(apiKeyUUID uuid.UUID) ([]model.APIKeyApi, error)
	FindByAPIKeyUUIDPaginated(apiKeyUUID uuid.UUID, page, limit int, sortBy, sortOrder string) (*PaginationResult[model.APIKeyApi], error)
	FindByAPIKeyUUIDAndApiUUID(apiKeyUUID uuid.UUID, apiUUID uuid.UUID) (*model.APIKeyApi, error)
	RemoveByAPIKeyAndApi(apiKeyID int64, apiID int64) error
	RemoveByAPIKeyUUIDAndApiUUID(apiKeyUUID uuid.UUID, apiUUID uuid.UUID) error
}

type apiKeyApiRepository struct {
	*BaseRepository[model.APIKeyApi]
	db *gorm.DB
}

func NewAPIKeyApiRepository(db *gorm.DB) APIKeyApiRepository {
	return &apiKeyApiRepository{
		BaseRepository: NewBaseRepository[model.APIKeyApi](db, "api_key_api_uuid", "api_key_api_id"),
		db:             db,
	}
}

func (r *apiKeyApiRepository) WithTx(tx *gorm.DB) APIKeyApiRepository {
	return &apiKeyApiRepository{
		BaseRepository: NewBaseRepository[model.APIKeyApi](tx, "api_key_api_uuid", "api_key_api_id"),
		db:             tx,
	}
}

func (r *apiKeyApiRepository) FindByAPIKeyAndApi(apiKeyID int64, apiID int64) (*model.APIKeyApi, error) {
	var apiKeyApi model.APIKeyApi
	if err := r.db.Where("api_key_id = ? AND api_id = ?", apiKeyID, apiID).First(&apiKeyApi).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &apiKeyApi, nil
}

func (r *apiKeyApiRepository) FindByAPIKeyUUID(apiKeyUUID uuid.UUID) ([]model.APIKeyApi, error) {
	var apiKeyApis []model.APIKeyApi
	err := r.db.Joins("JOIN api_keys ON api_keys.api_key_id = api_key_apis.api_key_id").
		Where("api_keys.api_key_uuid = ?", apiKeyUUID).
		Preload("API").
		Preload("Permissions.Permission").
		Find(&apiKeyApis).Error

	if err != nil {
		return nil, err
	}

	return apiKeyApis, nil
}

func (r *apiKeyApiRepository) FindByAPIKeyUUIDPaginated(apiKeyUUID uuid.UUID, page, limit int, sortBy, sortOrder string) (*PaginationResult[model.APIKeyApi], error) {
	var apiKeyApis []model.APIKeyApi
	var total int64

	// Base query
	query := r.db.Joins("JOIN api_keys ON api_keys.api_key_id = api_key_apis.api_key_id").
		Where("api_keys.api_key_uuid = ?", apiKeyUUID).
		Preload("API").
		Preload("Permissions.Permission")

	// Count total records
	if err := query.Model(&model.APIKeyApi{}).Count(&total).Error; err != nil {
		return nil, err
	}

	// Apply sorting
	if sortBy != "" {
		orderClause := sortBy
		if sortOrder == "desc" {
			orderClause += " DESC"
		} else {
			orderClause += " ASC"
		}
		query = query.Order(orderClause)
	} else {
		query = query.Order("api_key_apis.created_at DESC") // Default sorting
	}

	// Apply pagination
	offset := (page - 1) * limit
	if err := query.Limit(limit).Offset(offset).Find(&apiKeyApis).Error; err != nil {
		return nil, err
	}

	// Calculate total pages
	totalPages := int((total + int64(limit) - 1) / int64(limit))

	return &PaginationResult[model.APIKeyApi]{
		Data:       apiKeyApis,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	}, nil
}

func (r *apiKeyApiRepository) FindByAPIKeyUUIDAndApiUUID(apiKeyUUID uuid.UUID, apiUUID uuid.UUID) (*model.APIKeyApi, error) {
	var apiKeyApi model.APIKeyApi
	err := r.db.Joins("JOIN api_keys ON api_keys.api_key_id = api_key_apis.api_key_id").
		Joins("JOIN apis ON apis.api_id = api_key_apis.api_id").
		Where("api_keys.api_key_uuid = ? AND apis.api_uuid = ?", apiKeyUUID, apiUUID).
		Preload("API").
		Preload("Permissions.Permission").
		First(&apiKeyApi).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &apiKeyApi, nil
}

func (r *apiKeyApiRepository) RemoveByAPIKeyAndApi(apiKeyID int64, apiID int64) error {
	return r.db.Where("api_key_id = ? AND api_id = ?", apiKeyID, apiID).Delete(&model.APIKeyApi{}).Error
}

func (r *apiKeyApiRepository) RemoveByAPIKeyUUIDAndApiUUID(apiKeyUUID uuid.UUID, apiUUID uuid.UUID) error {
	return r.db.Joins("JOIN api_keys ON api_keys.api_key_id = api_key_apis.api_key_id").
		Joins("JOIN apis ON apis.api_id = api_key_apis.api_id").
		Where("api_keys.api_key_uuid = ? AND apis.api_uuid = ?", apiKeyUUID, apiUUID).
		Delete(&model.APIKeyApi{}).Error
}
