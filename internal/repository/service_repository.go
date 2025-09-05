package repository

import (
	"errors"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

type ServiceRepositoryGetFilter struct {
	Name        *string
	DisplayName *string
	Description *string
	Version     *string
	IsDefault   *bool
	IsActive    *bool
	IsPublic    *bool
	Page        int
	Limit       int
	SortBy      string
	SortOrder   string
}

type ServiceRepository interface {
	BaseRepositoryMethods[model.Service]
	WithTx(tx *gorm.DB) ServiceRepository
	FindByName(serviceName string) (*model.Service, error)
	FindDefaultServices() ([]model.Service, error)
	FindPaginated(filter ServiceRepositoryGetFilter) (*PaginationResult[model.Service], error)
	SetActiveStatusByUUID(serviceUUID uuid.UUID, isActive bool) error
	SetDefaultStatusByUUID(serviceUUID uuid.UUID, isDefault bool) error
}

type serviceRepository struct {
	*BaseRepository[model.Service]
	db *gorm.DB
}

func NewServiceRepository(db *gorm.DB) ServiceRepository {
	return &serviceRepository{
		BaseRepository: NewBaseRepository[model.Service](db, "service_uuid", "service_id"),
		db:             db,
	}
}

func (r *serviceRepository) WithTx(tx *gorm.DB) ServiceRepository {
	return &serviceRepository{
		BaseRepository: NewBaseRepository[model.Service](tx, "service_uuid", "service_id"),
		db:             tx,
	}
}

func (r *serviceRepository) FindByName(serviceName string) (*model.Service, error) {
	var service model.Service
	err := r.db.Where("name = ?", serviceName).First(&service).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &service, nil
}

func (r *serviceRepository) FindDefaultServices() ([]model.Service, error) {
	var services []model.Service
	err := r.db.Where("is_default = true").Find(&services).Error
	return services, err
}

func (r *serviceRepository) FindPaginated(filter ServiceRepositoryGetFilter) (*PaginationResult[model.Service], error) {
	conditions := map[string]any{}
	if filter.Name != nil {
		conditions["name"] = *filter.Name
	}
	if filter.DisplayName != nil {
		conditions["display_name"] = *filter.DisplayName
	}
	if filter.Description != nil {
		conditions["description"] = *filter.Description
	}
	if filter.Version != nil {
		conditions["version"] = *filter.Version
	}
	if filter.IsDefault != nil {
		conditions["is_default"] = *filter.IsDefault
	}
	if filter.IsActive != nil {
		conditions["is_active"] = *filter.IsActive
	}
	if filter.IsPublic != nil {
		conditions["is_public"] = *filter.IsPublic
	}

	// Sorting
	orderBy := filter.SortBy + " " + filter.SortOrder

	query := r.db.Model(&model.Service{}).Where(conditions).Order(orderBy)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	offset := (filter.Page - 1) * filter.Limit
	var services []model.Service
	if err := query.Limit(filter.Limit).Offset(offset).Find(&services).Error; err != nil {
		return nil, err
	}

	totalPages := int((total + int64(filter.Limit) - 1) / int64(filter.Limit))

	return &PaginationResult[model.Service]{
		Data:       services,
		Total:      total,
		Page:       filter.Page,
		Limit:      filter.Limit,
		TotalPages: totalPages,
	}, nil
}

func (r *serviceRepository) SetActiveStatusByUUID(serviceUUID uuid.UUID, isActive bool) error {
	return r.db.Model(&model.Service{}).
		Where("service_uuid = ?", serviceUUID).
		Update("is_active", isActive).Error
}

func (r *serviceRepository) SetDefaultStatusByUUID(serviceUUID uuid.UUID, isDefault bool) error {
	return r.db.Model(&model.Service{}).
		Where("service_uuid = ?", serviceUUID).
		Update("is_default", isDefault).Error
}
