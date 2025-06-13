package repository

import (
	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

type ServiceRepository interface {
	BaseRepositoryMethods[model.Service]
	FindByName(serviceName string) (*model.Service, error)
	FindByType(serviceType string) ([]model.Service, error)
	FindDefaultServices() ([]model.Service, error)
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

func (r *serviceRepository) FindByName(serviceName string) (*model.Service, error) {
	var service model.Service
	err := r.db.Where("service_name = ?", serviceName).First(&service).Error
	return &service, err
}

func (r *serviceRepository) FindByType(serviceType string) ([]model.Service, error) {
	var services []model.Service
	err := r.db.Where("service_type = ?", serviceType).Find(&services).Error
	return services, err
}

func (r *serviceRepository) FindDefaultServices() ([]model.Service, error) {
	var services []model.Service
	err := r.db.Where("is_default = true").Find(&services).Error
	return services, err
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
