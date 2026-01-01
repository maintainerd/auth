package repository

import (
	"errors"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

type APIRepositoryGetFilter struct {
	TenantID    int64
	Name        *string
	DisplayName *string
	APIType     *string
	Identifier  *string
	ServiceID   *int64
	Status      []string
	IsDefault   *bool
	IsSystem    *bool
	Page        int
	Limit       int
	SortBy      string
	SortOrder   string
}

type APIRepository interface {
	BaseRepositoryMethods[model.API]
	WithTx(tx *gorm.DB) APIRepository
	FindByUUIDAndTenantID(apiUUID uuid.UUID, tenantID int64) (*model.API, error)
	FindByName(apiName string, tenantID int64) (*model.API, error)
	FindByIdentifier(identifier string, tenantID int64) (*model.API, error)
	FindDefaultByServiceID(serviceID int64, tenantID int64) (*model.API, error)
	FindPaginated(filter APIRepositoryGetFilter) (*PaginationResult[model.API], error)
	SetStatusByUUID(apiUUID uuid.UUID, tenantID int64, status string) error
	SetDefaultStatusByUUID(apiUUID uuid.UUID, tenantID int64, isDefault bool) error
	CountByServiceID(serviceID int64, tenantID int64) (int64, error)
	DeleteByUUIDAndTenantID(apiUUID uuid.UUID, tenantID int64) error
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

func (r *apiRepository) WithTx(tx *gorm.DB) APIRepository {
	return &apiRepository{
		BaseRepository: NewBaseRepository[model.API](tx, "api_uuid", "api_id"),
		db:             tx,
	}
}

func (r *apiRepository) FindByUUIDAndTenantID(apiUUID uuid.UUID, tenantID int64) (*model.API, error) {
	var api model.API
	err := r.db.
		Preload("Service").
		Where("api_uuid = ? AND tenant_id = ?", apiUUID, tenantID).
		First(&api).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &api, nil
}

func (r *apiRepository) FindByName(apiName string, tenantID int64) (*model.API, error) {
	var api model.API
	err := r.db.
		Preload("Service").
		Where("name = ? AND tenant_id = ?", apiName, tenantID).
		First(&api).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &api, err
}

func (r *apiRepository) FindByIdentifier(identifier string, tenantID int64) (*model.API, error) {
	var api model.API
	err := r.db.
		Where("identifier = ? AND tenant_id = ?", identifier, tenantID).
		First(&api).Error
	return &api, err
}

func (r *apiRepository) FindDefaultByServiceID(serviceID int64, tenantID int64) (*model.API, error) {
	var api model.API
	err := r.db.
		Where("service_id = ? AND tenant_id = ? AND is_default = true", serviceID, tenantID).
		First(&api).Error
	return &api, err
}

func (r *apiRepository) FindPaginated(filter APIRepositoryGetFilter) (*PaginationResult[model.API], error) {
	query := r.db.Model(&model.API{})

	// Filter by tenant_id
	query = query.Where("tenant_id = ?", filter.TenantID)

	// Filters with LIKE
	if filter.Name != nil {
		query = query.Where("name ILIKE ?", "%"+*filter.Name+"%")
	}
	if filter.DisplayName != nil {
		query = query.Where("display_name ILIKE ?", "%"+*filter.DisplayName+"%")
	}

	// Filters with exact match
	if filter.APIType != nil {
		query = query.Where("api_type = ?", *filter.APIType)
	}
	if filter.Identifier != nil {
		query = query.Where("identifier = ?", *filter.Identifier)
	}
	if filter.ServiceID != nil {
		query = query.Where("service_id = ?", *filter.ServiceID)
	}
	if len(filter.Status) > 0 {
		query = query.Where("status IN ?", filter.Status)
	}
	if filter.IsDefault != nil {
		query = query.Where("is_default = ?", *filter.IsDefault)
	}
	if filter.IsSystem != nil {
		query = query.Where("is_system = ?", *filter.IsSystem)
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
	var apis []model.API
	if err := query.Preload("Service").Limit(filter.Limit).Offset(offset).Find(&apis).Error; err != nil {
		return nil, err
	}

	totalPages := int((total + int64(filter.Limit) - 1) / int64(filter.Limit))

	return &PaginationResult[model.API]{
		Data:       apis,
		Total:      total,
		Page:       filter.Page,
		Limit:      filter.Limit,
		TotalPages: totalPages,
	}, nil
}

func (r *apiRepository) SetStatusByUUID(apiUUID uuid.UUID, tenantID int64, status string) error {
	return r.db.Model(&model.API{}).
		Where("api_uuid = ? AND tenant_id = ?", apiUUID, tenantID).
		Update("status", status).Error
}

func (r *apiRepository) SetDefaultStatusByUUID(apiUUID uuid.UUID, tenantID int64, isDefault bool) error {
	return r.db.Model(&model.API{}).
		Where("api_uuid = ? AND tenant_id = ?", apiUUID, tenantID).
		Update("is_default", isDefault).Error
}

func (r *apiRepository) CountByServiceID(serviceID int64, tenantID int64) (int64, error) {
	var count int64
	err := r.db.Model(&model.API{}).
		Where("service_id = ? AND tenant_id = ?", serviceID, tenantID).
		Count(&count).Error
	return count, err
}

func (r *apiRepository) DeleteByUUIDAndTenantID(apiUUID uuid.UUID, tenantID int64) error {
	return r.db.Where("api_uuid = ? AND tenant_id = ?", apiUUID, tenantID).Delete(&model.API{}).Error
}
