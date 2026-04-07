package repository

import (
	"errors"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

type APIKeyAPIRepository interface {
	BaseRepositoryMethods[model.APIKeyAPI]
	WithTx(tx *gorm.DB) APIKeyAPIRepository
	FindByAPIKeyAndAPI(apiKeyID int64, apiID int64) (*model.APIKeyAPI, error)
	FindByAPIKeyUUID(apiKeyUUID uuid.UUID) ([]model.APIKeyAPI, error)
	FindByAPIKeyUUIDPaginated(apiKeyUUID uuid.UUID, page, limit int, sortBy, sortOrder string) (*PaginationResult[model.APIKeyAPI], error)
	FindByAPIKeyUUIDAndAPIUUID(apiKeyUUID uuid.UUID, apiUUID uuid.UUID) (*model.APIKeyAPI, error)
	RemoveByAPIKeyAndAPI(apiKeyID int64, apiID int64) error
	RemoveByAPIKeyUUIDAndAPIUUID(apiKeyUUID uuid.UUID, apiUUID uuid.UUID) error
}

type apiKeyAPIRepository struct {
	*BaseRepository[model.APIKeyAPI]
}

func NewAPIKeyAPIRepository(db *gorm.DB) APIKeyAPIRepository {
	return &apiKeyAPIRepository{
		BaseRepository: NewBaseRepository[model.APIKeyAPI](db, "api_key_api_uuid", "api_key_api_id"),
	}
}

func (r *apiKeyAPIRepository) WithTx(tx *gorm.DB) APIKeyAPIRepository {
	return &apiKeyAPIRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

func (r *apiKeyAPIRepository) FindByAPIKeyAndAPI(apiKeyID int64, apiID int64) (*model.APIKeyAPI, error) {
	var apiKeyAPI model.APIKeyAPI
	if err := r.DB().Where("api_key_id = ? AND api_id = ?", apiKeyID, apiID).First(&apiKeyAPI).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &apiKeyAPI, nil
}

func (r *apiKeyAPIRepository) FindByAPIKeyUUID(apiKeyUUID uuid.UUID) ([]model.APIKeyAPI, error) {
	var apiKeyAPIs []model.APIKeyAPI
	err := r.DB().Joins("JOIN api_keys ON api_keys.api_key_id = api_key_apis.api_key_id").
		Where("api_keys.api_key_uuid = ?", apiKeyUUID).
		Preload("API").
		Preload("Permissions.Permission").
		Find(&apiKeyAPIs).Error

	if err != nil {
		return nil, err
	}

	return apiKeyAPIs, nil
}

func (r *apiKeyAPIRepository) FindByAPIKeyUUIDPaginated(apiKeyUUID uuid.UUID, page, limit int, sortBy, sortOrder string) (*PaginationResult[model.APIKeyAPI], error) {
	var apiKeyAPIs []model.APIKeyAPI
	var total int64

	// Base query
	query := r.DB().Joins("JOIN api_keys ON api_keys.api_key_id = api_key_apis.api_key_id").
		Where("api_keys.api_key_uuid = ?", apiKeyUUID).
		Preload("API").
		Preload("Permissions.Permission")

	// Count total records
	if err := query.Model(&model.APIKeyAPI{}).Count(&total).Error; err != nil {
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

	// Pagination guards prevent division-by-zero and negative offsets
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}
	offset := (page - 1) * limit
	if err := query.Limit(limit).Offset(offset).Find(&apiKeyAPIs).Error; err != nil {
		return nil, err
	}

	// Calculate total pages
	totalPages := int((total + int64(limit) - 1) / int64(limit))

	return &PaginationResult[model.APIKeyAPI]{
		Data:       apiKeyAPIs,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	}, nil
}

func (r *apiKeyAPIRepository) FindByAPIKeyUUIDAndAPIUUID(apiKeyUUID uuid.UUID, apiUUID uuid.UUID) (*model.APIKeyAPI, error) {
	var apiKeyAPI model.APIKeyAPI
	err := r.DB().Joins("JOIN api_keys ON api_keys.api_key_id = api_key_apis.api_key_id").
		Joins("JOIN apis ON apis.api_id = api_key_apis.api_id").
		Where("api_keys.api_key_uuid = ? AND apis.api_uuid = ?", apiKeyUUID, apiUUID).
		Preload("API").
		Preload("Permissions.Permission").
		First(&apiKeyAPI).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &apiKeyAPI, nil
}

func (r *apiKeyAPIRepository) RemoveByAPIKeyAndAPI(apiKeyID int64, apiID int64) error {
	return r.DB().Where("api_key_id = ? AND api_id = ?", apiKeyID, apiID).Delete(&model.APIKeyAPI{}).Error
}

func (r *apiKeyAPIRepository) RemoveByAPIKeyUUIDAndAPIUUID(apiKeyUUID uuid.UUID, apiUUID uuid.UUID) error {
	// First find the record to get its ID
	apiKeyAPI, err := r.FindByAPIKeyUUIDAndAPIUUID(apiKeyUUID, apiUUID)
	if err != nil {
		return err
	}
	if apiKeyAPI == nil {
		return errors.New("API key API relationship not found")
	}

	// Delete by ID (more reliable than complex JOINs in DELETE)
	return r.DB().Delete(&model.APIKeyAPI{}, apiKeyAPI.APIKeyAPIID).Error
}
