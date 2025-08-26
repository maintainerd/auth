package repository

import (
	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

type OrganizationRepository interface {
	BaseRepositoryMethods[model.Organization]
	FindByName(name string) (*model.Organization, error)
	FindByEmail(email string) (*model.Organization, error)
	FindByExternalReferenceID(refID string) (*model.Organization, error)
	FindAllActive() ([]model.Organization, error)
	FindDefaultOrganization() (*model.Organization, error)
	FindOrganizationWithServices(organizationUUID uuid.UUID) (*model.Organization, error)
	FindDefaultOrganizationWithServices() (*model.Organization, error)
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

func (r *organizationRepository) FindByName(name string) (*model.Organization, error) {
	var org model.Organization
	err := r.db.
		Where("name = ?", name).
		First(&org).Error
	return &org, err
}

func (r *organizationRepository) FindByEmail(email string) (*model.Organization, error) {
	var org model.Organization
	err := r.db.
		Where("email = ?", email).
		First(&org).Error
	return &org, err
}

func (r *organizationRepository) FindByExternalReferenceID(refID string) (*model.Organization, error) {
	var org model.Organization
	err := r.db.
		Where("external_reference_id = ?", refID).
		First(&org).Error
	return &org, err
}

func (r *organizationRepository) FindAllActive() ([]model.Organization, error) {
	var orgs []model.Organization
	err := r.db.
		Where("is_active = true").
		Find(&orgs).Error
	return orgs, err
}

func (r *organizationRepository) FindDefaultOrganization() (*model.Organization, error) {
	var org model.Organization
	err := r.db.
		Where("is_default = true").
		First(&org).Error
	return &org, err
}

func (r *organizationRepository) FindOrganizationWithServices(organizationUUID uuid.UUID) (*model.Organization, error) {
	var org model.Organization
	err := r.db.
		Preload("Services").
		Where("organization_uuid = ?", organizationUUID).
		First(&org).Error
	return &org, err
}

func (r *organizationRepository) FindDefaultOrganizationWithServices() (*model.Organization, error) {
	var org model.Organization
	err := r.db.
		Preload("Services").
		Where("is_default = true").
		First(&org).Error
	return &org, err
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
