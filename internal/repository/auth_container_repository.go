package repository

import (
	"errors"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

type AuthContainerRepositoryGetFilter struct {
	Name           *string
	Description    *string
	APIType        *string
	Identifier     *string
	OrganizationID *int64
	IsActive       *bool
	IsPublic       *bool
	IsDefault      *bool
	Page           int
	Limit          int
	SortBy         string
	SortOrder      string
}

type AuthContainerRepository interface {
	BaseRepositoryMethods[model.AuthContainer]
	WithTx(tx *gorm.DB) AuthContainerRepository
	FindByName(name string) (*model.AuthContainer, error)
	FindByIdentifier(identifier string) (*model.AuthContainer, error)
	FindDefault() (*model.AuthContainer, error)
	FindDefaultByOrganizationID(organizationID int64) (*model.AuthContainer, error)
	FindPaginated(filter AuthContainerRepositoryGetFilter) (*PaginationResult[model.AuthContainer], error)
	SetActiveStatusByUUID(authContainerUUID uuid.UUID, isActive bool) error
	SetDefaultStatusByUUID(authContainerUUID uuid.UUID, isDefault bool) error
}

type authContainerRepository struct {
	*BaseRepository[model.AuthContainer]
	db *gorm.DB
}

func NewAuthContainerRepository(db *gorm.DB) AuthContainerRepository {
	return &authContainerRepository{
		BaseRepository: NewBaseRepository[model.AuthContainer](db, "auth_container_uuid", "auth_container_id"),
		db:             db,
	}
}

func (r *authContainerRepository) WithTx(tx *gorm.DB) AuthContainerRepository {
	return &authContainerRepository{
		BaseRepository: NewBaseRepository[model.AuthContainer](tx, "auth_container_uuid", "auth_container_id"),
		db:             tx,
	}
}

func (r *authContainerRepository) FindByName(name string) (*model.AuthContainer, error) {
	var authContainer model.AuthContainer
	err := r.db.
		Where("name = ?", name).
		First(&authContainer).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &authContainer, err
}

func (r *authContainerRepository) FindByIdentifier(identifier string) (*model.AuthContainer, error) {
	var authContainer model.AuthContainer
	err := r.db.
		Where("identifier = ?", identifier).
		First(&authContainer).Error
	return &authContainer, err
}

func (r *authContainerRepository) FindDefault() (*model.AuthContainer, error) {
	var authContainer model.AuthContainer
	err := r.db.
		Where("is_default = ? AND is_active = ?", true, true).
		First(&authContainer).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &authContainer, nil
}

func (r *authContainerRepository) FindDefaultByOrganizationID(organizationID int64) (*model.AuthContainer, error) {
	var authContainer model.AuthContainer
	err := r.db.
		Where("organization_id = ?", organizationID).
		First(&authContainer).Error
	return &authContainer, err
}

func (r *authContainerRepository) FindPaginated(filter AuthContainerRepositoryGetFilter) (*PaginationResult[model.AuthContainer], error) {
	query := r.db.Model(&model.AuthContainer{})

	// Filters with LIKE
	if filter.Name != nil {
		query = query.Where("name ILIKE ?", "%"+*filter.Name+"%")
	}
	if filter.Description != nil {
		query = query.Where("description ILIKE ?", "%"+*filter.Description+"%")
	}

	// Filters with exact match
	if filter.Identifier != nil {
		query = query.Where("identifier = ?", *filter.Identifier)
	}
	if filter.OrganizationID != nil {
		query = query.Where("organization_id = ?", *filter.OrganizationID)
	}
	if filter.IsActive != nil {
		query = query.Where("is_active = ?", *filter.IsActive)
	}
	if filter.IsPublic != nil {
		query = query.Where("is_public = ?", *filter.IsPublic)
	}
	if filter.IsDefault != nil {
		query = query.Where("is_default = ?", *filter.IsDefault)
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
	var authContainers []model.AuthContainer
	if err := query.Preload("Organization").Limit(filter.Limit).Offset(offset).Find(&authContainers).Error; err != nil {
		return nil, err
	}

	totalPages := int((total + int64(filter.Limit) - 1) / int64(filter.Limit))

	return &PaginationResult[model.AuthContainer]{
		Data:       authContainers,
		Total:      total,
		Page:       filter.Page,
		Limit:      filter.Limit,
		TotalPages: totalPages,
	}, nil
}

func (r *authContainerRepository) SetActiveStatusByUUID(authContainerUUID uuid.UUID, isActive bool) error {
	return r.db.Model(&model.AuthContainer{}).
		Where("auth_container_uuid = ?", authContainerUUID).
		Update("is_active", isActive).Error
}

func (r *authContainerRepository) SetDefaultStatusByUUID(authContainerUUID uuid.UUID, isDefault bool) error {
	return r.db.Model(&model.AuthContainer{}).
		Where("auth_container_uuid = ?", authContainerUUID).
		Update("is_default", isDefault).Error
}
