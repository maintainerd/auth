package repository

import (
	"errors"

	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

type APIKeyRepositoryGetFilter struct {
	TenantID    int64
	Name        *string
	Description *string
	Status      *string
	Page        int
	Limit       int
	SortBy      string
	SortOrder   string
}

type APIKeyRepository interface {
	BaseRepositoryMethods[model.APIKey]
	WithTx(tx *gorm.DB) APIKeyRepository
	FindByUUIDAndTenantID(uuid string, tenantID int64) (*model.APIKey, error)
	FindByKeyHash(keyHash string) (*model.APIKey, error)
	FindByKeyPrefix(keyPrefix string) (*model.APIKey, error)
	DeleteByUUIDAndTenantID(uuid string, tenantID int64) error
	FindPaginated(filter APIKeyRepositoryGetFilter) (*PaginationResult[model.APIKey], error)
}

type apiKeyRepository struct {
	*BaseRepository[model.APIKey]
	db *gorm.DB
}

func NewAPIKeyRepository(db *gorm.DB) APIKeyRepository {
	return &apiKeyRepository{
		BaseRepository: NewBaseRepository[model.APIKey](db, "api_key_uuid", "api_key_id"),
		db:             db,
	}
}

func (r *apiKeyRepository) WithTx(tx *gorm.DB) APIKeyRepository {
	return &apiKeyRepository{
		BaseRepository: NewBaseRepository[model.APIKey](tx, "api_key_uuid", "api_key_id"),
		db:             tx,
	}
}

func (r *apiKeyRepository) FindByUUIDAndTenantID(uuid string, tenantID int64) (*model.APIKey, error) {
	var apiKey model.APIKey
	err := r.db.Where("api_key_uuid = ? AND tenant_id = ?", uuid, tenantID).First(&apiKey).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &apiKey, nil
}

func (r *apiKeyRepository) FindByKeyHash(keyHash string) (*model.APIKey, error) {
	var apiKey model.APIKey
	if err := r.db.Where("key_hash = ?", keyHash).First(&apiKey).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &apiKey, nil
}

func (r *apiKeyRepository) DeleteByUUIDAndTenantID(uuid string, tenantID int64) error {
	result := r.db.Where("api_key_uuid = ? AND tenant_id = ?", uuid, tenantID).Delete(&model.APIKey{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *apiKeyRepository) FindByKeyPrefix(keyPrefix string) (*model.APIKey, error) {
	var apiKey model.APIKey
	if err := r.db.Where("key_prefix = ?", keyPrefix).First(&apiKey).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &apiKey, nil
}

func (r *apiKeyRepository) FindPaginated(filter APIKeyRepositoryGetFilter) (*PaginationResult[model.APIKey], error) {
	query := r.db.Model(&model.APIKey{})

	// Always filter by tenant
	query = query.Where("tenant_id = ?", filter.TenantID)

	// Apply filters
	if filter.Name != nil {
		query = query.Where("name ILIKE ?", "%"+*filter.Name+"%")
	}
	if filter.Description != nil {
		query = query.Where("description ILIKE ?", "%"+*filter.Description+"%")
	}
	if filter.Status != nil {
		query = query.Where("status = ?", *filter.Status)
	}

	// Sorting
	orderBy := filter.SortBy + " " + filter.SortOrder
	query = query.Order(orderBy)

	// Count
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// Pagination
	offset := (filter.Page - 1) * filter.Limit
	var apiKeys []model.APIKey
	if err := query.Limit(filter.Limit).Offset(offset).Find(&apiKeys).Error; err != nil {
		return nil, err
	}

	totalPages := int((total + int64(filter.Limit) - 1) / int64(filter.Limit))

	return &PaginationResult[model.APIKey]{
		Data:       apiKeys,
		Total:      total,
		Page:       filter.Page,
		Limit:      filter.Limit,
		TotalPages: totalPages,
	}, nil
}
