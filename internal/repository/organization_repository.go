package repository

import (
	"errors"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

type OrganizationRepositoryGetFilter struct {
	Name        *string
	Description *string
	Email       *string
	Phone       *string
	IsActive    *bool
	IsDefault   *bool
	IsRoot      *bool
	Page        int
	Limit       int
	SortBy      string
	SortOrder   string
}

type OrganizationRepository interface {
	BaseRepositoryMethods[model.Organization]
	WithTx(tx *gorm.DB) OrganizationRepository
	FindByName(name string) (*model.Organization, error)
	FindByEmail(email string) (*model.Organization, error)
	FindDefaultOrganization() (*model.Organization, error)
	FindPaginated(filter OrganizationRepositoryGetFilter) (*PaginationResult[model.Organization], error)
	CanUserUpdateOrganization(userUUID uuid.UUID, organizationUUID uuid.UUID) (bool, error)
	SetActiveStatusByUUID(organizationUUID uuid.UUID, isActive bool) error
	SetDefaultStatusByUUID(organizationUUID uuid.UUID, isDefault bool) error
}

type organizationRepository struct {
	*BaseRepository[model.Organization]
	db *gorm.DB
}

func NewOrganizationRepository(db *gorm.DB) OrganizationRepository {
	return &organizationRepository{
		BaseRepository: NewBaseRepository[model.Organization](db, "organization_uuid", "organization_id"),
		db:             db,
	}
}

func (r *organizationRepository) WithTx(tx *gorm.DB) OrganizationRepository {
	return &organizationRepository{
		BaseRepository: NewBaseRepository[model.Organization](tx, "organization_uuid", "organization_id"),
		db:             tx,
	}
}

func (r *organizationRepository) FindByName(name string) (*model.Organization, error) {
	var org model.Organization
	err := r.db.
		Where("name = ?", name).
		First(&org).Error

	// If no record is found, return nil record and nil error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &org, err
}

func (r *organizationRepository) FindByEmail(email string) (*model.Organization, error) {
	var org model.Organization
	err := r.db.
		Where("email = ?", email).
		First(&org).Error
	return &org, err
}

func (r *organizationRepository) FindDefaultOrganization() (*model.Organization, error) {
	var org model.Organization
	err := r.db.
		Where("is_default = true").
		First(&org).Error
	return &org, err
}

func (r *organizationRepository) FindPaginated(filter OrganizationRepositoryGetFilter) (*PaginationResult[model.Organization], error) {
	query := r.db.Model(&model.Organization{})

	// Filters with LIKE
	if filter.Name != nil {
		query = query.Where("name ILIKE ?", "%"+*filter.Name+"%")
	}
	if filter.Description != nil {
		query = query.Where("description ILIKE ?", "%"+*filter.Description+"%")
	}
	if filter.Email != nil {
		query = query.Where("email ILIKE ?", "%"+*filter.Email+"%")
	}
	if filter.Phone != nil {
		query = query.Where("phone ILIKE ?", "%"+*filter.Phone+"%")
	}

	// Filters with exact match
	if filter.IsActive != nil {
		query = query.Where("is_active = ?", *filter.IsActive)
	}
	if filter.IsDefault != nil {
		query = query.Where("is_default = ?", *filter.IsDefault)
	}
	if filter.IsRoot != nil {
		query = query.Where("is_root = ?", *filter.IsRoot)
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
	var orgs []model.Organization
	if err := query.Limit(filter.Limit).Offset(offset).Find(&orgs).Error; err != nil {
		return nil, err
	}

	totalPages := int((total + int64(filter.Limit) - 1) / int64(filter.Limit))

	return &PaginationResult[model.Organization]{
		Data:       orgs,
		Total:      total,
		Page:       filter.Page,
		Limit:      filter.Limit,
		TotalPages: totalPages,
	}, nil
}

func (r *organizationRepository) CanUserUpdateOrganization(userUUID uuid.UUID, organizationUUID uuid.UUID) (bool, error) {
	// Fetch user with their AuthContainer and its Organization
	var user model.User
	err := r.db.
		Preload("AuthContainer.Organization").
		Where("user_uuid = ?", userUUID).
		First(&user).Error
	if err != nil {
		return false, err
	}

	if user.AuthContainer == nil {
		return false, errors.New("user has no auth container")
	}

	// Fetch organization
	var targetOrg model.Organization
	if err := r.db.Where("organization_uuid = ?", organizationUUID).First(&targetOrg).Error; err != nil {
		return false, err
	}

	// users in the global default auth container can update any org (except default)
	if user.AuthContainer.IsDefault {
		return true, nil
	}

	// users in a non-default auth container can only update their own org
	if user.AuthContainer.OrganizationID == targetOrg.OrganizationID {
		return true, nil
	}

	return false, nil
}

func (r *organizationRepository) SetActiveStatusByUUID(organizationUUID uuid.UUID, isActive bool) error {
	return r.db.Model(&model.Organization{}).
		Where("organization_uuid = ?", organizationUUID).
		Update("is_active", isActive).Error
}

func (r *organizationRepository) SetDefaultStatusByUUID(organizationUUID uuid.UUID, isDefault bool) error {
	return r.db.Model(&model.Organization{}).
		Where("organization_uuid = ?", organizationUUID).
		Update("is_default", isDefault).Error
}
