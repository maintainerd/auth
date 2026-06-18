package client

import (
	"errors"

	"github.com/maintainerd/auth/internal/platform/database"
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
	BaseRepositoryMethods[APIKey]
	UpdateByUUID(uuid any, updatedData any) (*APIKey, error)
	DeleteByUUID(uuid any) error
	WithTx(tx *gorm.DB) APIKeyRepository
	FindByUUIDAndTenantID(uuid string, tenantID int64) (*APIKey, error)
	FindByKeyHash(keyHash string) (*APIKey, error)
	FindByKeyPrefix(keyPrefix string) (*APIKey, error)
	DeleteByUUIDAndTenantID(uuid string, tenantID int64) error
	FindPaginated(filter APIKeyRepositoryGetFilter) (*PaginationResult[APIKey], error)
}

type apiKeyRepository struct {
	*BaseRepository[APIKey]
}

func NewAPIKeyRepository(db *gorm.DB) APIKeyRepository {
	return &apiKeyRepository{
		BaseRepository: database.NewBaseRepository[APIKey](db, "api_key_uuid", "api_key_id"),
	}
}

func (r *apiKeyRepository) WithTx(tx *gorm.DB) APIKeyRepository {
	return &apiKeyRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

func (r *apiKeyRepository) FindByUUIDAndTenantID(uuid string, tenantID int64) (*APIKey, error) {
	var apiKey APIKey
	err := r.DB().Where("api_key_uuid = ? AND tenant_id = ?", uuid, tenantID).First(&apiKey).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &apiKey, nil
}

func (r *apiKeyRepository) FindByKeyHash(keyHash string) (*APIKey, error) {
	var apiKey APIKey
	if err := r.DB().Where("key_hash = ?", keyHash).First(&apiKey).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &apiKey, nil
}

func (r *apiKeyRepository) DeleteByUUIDAndTenantID(uuid string, tenantID int64) error {
	result := r.DB().Where("api_key_uuid = ? AND tenant_id = ?", uuid, tenantID).Delete(&APIKey{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *apiKeyRepository) FindByKeyPrefix(keyPrefix string) (*APIKey, error) {
	var apiKey APIKey
	if err := r.DB().Where("key_prefix = ?", keyPrefix).First(&apiKey).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &apiKey, nil
}

func (r *apiKeyRepository) FindPaginated(filter APIKeyRepositoryGetFilter) (*PaginationResult[APIKey], error) {
	query := r.DB().Model(&APIKey{})

	// Always filter by tenant
	query = query.Where("tenant_id = ?", filter.TenantID)

	// Apply filters
	query = database.ApplyILike(query, "name", filter.Name)
	query = database.ApplyILike(query, "description", filter.Description)
	if filter.Status != nil {
		query = query.Where("status = ?", *filter.Status)
	}

	// Sorting — protected against SQL injection via allowlist
	query = query.Order(database.SanitizeOrder(filter.SortBy, filter.SortOrder, "created_at DESC"))

	return database.PaginateQuery[APIKey](query, filter.Page, filter.Limit)
}
