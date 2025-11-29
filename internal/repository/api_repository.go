package repository

import (
	"errors"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

type APIRepositoryGetFilter struct {
	Name        *string
	DisplayName *string
	APIType     *string
	Identifier  *string
	ServiceID   *int64
	IsActive    *bool
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
	FindByName(apiName string) (*model.API, error)
	FindByIdentifier(identifier string) (*model.API, error)
	FindDefaultByServiceID(serviceID int64) (*model.API, error)
	FindPaginated(filter APIRepositoryGetFilter) (*PaginationResult[model.API], error)
	SetActiveStatusByUUID(apiUUID uuid.UUID, isActive bool) error
	SetDefaultStatusByUUID(apiUUID uuid.UUID, isDefault bool) error
	CountByServiceID(serviceID int64) (int64, error)
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

func (r *apiRepository) FindByName(apiName string) (*model.API, error) {
	var api model.API
	err := r.db.
		Preload("Service").
		Where("name = ?", apiName).
		First(&api).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &api, err
}

func (r *apiRepository) FindByIdentifier(identifier string) (*model.API, error) {
	var api model.API
	err := r.db.
		Where("identifier = ?", identifier).
		First(&api).Error
	return &api, err
}

func (r *apiRepository) FindDefaultByServiceID(serviceID int64) (*model.API, error) {
	var api model.API
	err := r.db.
		Where("service_id = ? AND is_default = true", serviceID).
		First(&api).Error
	return &api, err
}

func (r *apiRepository) FindPaginated(filter APIRepositoryGetFilter) (*PaginationResult[model.API], error) {
	query := r.db.Model(&model.API{})

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
	if filter.IsActive != nil {
		query = query.Where("is_active = ?", *filter.IsActive)
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

func (r *apiRepository) CountByServiceID(serviceID int64) (int64, error) {
	var count int64
	err := r.db.Model(&model.API{}).
		Where("service_id = ?", serviceID).
		Count(&count).Error
	return count, err
}
