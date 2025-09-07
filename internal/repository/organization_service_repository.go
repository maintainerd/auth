package repository

import (
	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

type OrganizationServiceRepository interface {
	BaseRepositoryMethods[model.OrganizationService]
	WithTx(tx *gorm.DB) OrganizationServiceRepository
	FindByOrganizationID(orgID int64) ([]model.OrganizationService, error)
	FindByServiceID(serviceID int64) ([]model.OrganizationService, error)
	FindByOrganizationUUID(orgUUID uuid.UUID) ([]model.OrganizationService, error)
	FindByServiceUUID(serviceUUID uuid.UUID) ([]model.OrganizationService, error)
	DeleteByOrganizationAndServiceID(orgID, serviceID int64) error
}

type organizationServiceRepository struct {
	*BaseRepository[model.OrganizationService]
	db *gorm.DB
}

func NewOrganizationServiceRepository(db *gorm.DB) OrganizationServiceRepository {
	return &organizationServiceRepository{
		BaseRepository: NewBaseRepository[model.OrganizationService](db, "organization_service_uuid", "organization_service_id"),
		db:             db,
	}
}

func (r *organizationServiceRepository) WithTx(tx *gorm.DB) OrganizationServiceRepository {
	return &organizationServiceRepository{
		BaseRepository: NewBaseRepository[model.OrganizationService](tx, "organization_service_uuid", "organization_service_id"),
		db:             tx,
	}
}

// Find all services for a given organization ID
func (r *organizationServiceRepository) FindByOrganizationID(orgID int64) ([]model.OrganizationService, error) {
	var orgServices []model.OrganizationService
	err := r.db.Where("organization_id = ?", orgID).Find(&orgServices).Error
	return orgServices, err
}

// Find all organizations for a given service ID
func (r *organizationServiceRepository) FindByServiceID(serviceID int64) ([]model.OrganizationService, error) {
	var orgServices []model.OrganizationService
	err := r.db.Where("service_id = ?", serviceID).Find(&orgServices).Error
	return orgServices, err
}

// Find by organization UUID
func (r *organizationServiceRepository) FindByOrganizationUUID(orgUUID uuid.UUID) ([]model.OrganizationService, error) {
	var org model.Organization
	if err := r.db.Select("organization_id").Where("organization_uuid = ?", orgUUID).First(&org).Error; err != nil {
		return nil, err
	}
	return r.FindByOrganizationID(org.OrganizationID)
}

// Find by service UUID
func (r *organizationServiceRepository) FindByServiceUUID(serviceUUID uuid.UUID) ([]model.OrganizationService, error) {
	var svc model.Service
	if err := r.db.Select("service_id").Where("service_uuid = ?", serviceUUID).First(&svc).Error; err != nil {
		return nil, err
	}
	return r.FindByServiceID(svc.ServiceID)
}

// Delete a specific organization-service relation by orgID and serviceID
func (r *organizationServiceRepository) DeleteByOrganizationAndServiceID(orgID, serviceID int64) error {
	return r.db.Where("organization_id = ? AND service_id = ?", orgID, serviceID).
		Delete(&model.OrganizationService{}).Error
}
