package repository

import (
	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

type ServiceRepository interface {
	Create(service *model.Service) error
	FindAll() ([]model.Service, error)
	FindByUUID(serviceUUID uuid.UUID) (*model.Service, error)
	FindByName(serviceName string) (*model.Service, error)
	UpdateByUUID(serviceUUID uuid.UUID, updatedService *model.Service) error
	DeleteByUUID(serviceUUID uuid.UUID) error
}

type serviceRepository struct {
	db *gorm.DB
}

func NewServiceRepository(db *gorm.DB) ServiceRepository {
	return &serviceRepository{db}
}

func (r *serviceRepository) Create(service *model.Service) error {
	return r.db.Create(service).Error
}

func (r *serviceRepository) FindAll() ([]model.Service, error) {
	var services []model.Service
	err := r.db.Find(&services).Error
	return services, err
}

func (r *serviceRepository) FindByUUID(serviceUUID uuid.UUID) (*model.Service, error) {
	var service model.Service
	err := r.db.Where("service_uuid = ?", serviceUUID).First(&service).Error
	return &service, err
}

func (r *serviceRepository) FindByName(serviceName string) (*model.Service, error) {
	var service model.Service
	err := r.db.Where("service_name = ?", serviceName).First(&service).Error
	return &service, err
}

func (r *serviceRepository) UpdateByUUID(serviceUUID uuid.UUID, updatedService *model.Service) error {
	return r.db.Model(&model.Service{}).
		Where("service_uuid = ?", serviceUUID).
		Updates(updatedService).Error
}

func (r *serviceRepository) DeleteByUUID(serviceUUID uuid.UUID) error {
	return r.db.Where("service_uuid = ?", serviceUUID).
		Delete(&model.Service{}).Error
}
