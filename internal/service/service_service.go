package service

import (
	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/repository"
)

type ServiceService interface {
	Create(service *model.Service) error
	GetAll() ([]model.Service, error)
	GetByUUID(serviceUUID uuid.UUID) (*model.Service, error)
	GetByName(serviceName string) (*model.Service, error)
	UpdateByUUID(serviceUUID uuid.UUID, updatedService *model.Service) error
	DeleteByUUID(serviceUUID uuid.UUID) error
}

type serviceService struct {
	repo repository.ServiceRepository
}

func NewServiceService(repo repository.ServiceRepository) ServiceService {
	return &serviceService{repo}
}

func (s *serviceService) Create(service *model.Service) error {
	service.ServiceUUID = uuid.New()
	return s.repo.Create(service)
}

func (s *serviceService) GetAll() ([]model.Service, error) {
	return s.repo.FindAll()
}

func (s *serviceService) GetByUUID(serviceUUID uuid.UUID) (*model.Service, error) {
	return s.repo.FindByUUID(serviceUUID)
}

func (s *serviceService) GetByName(serviceName string) (*model.Service, error) {
	return s.repo.FindByName(serviceName)
}

func (s *serviceService) UpdateByUUID(serviceUUID uuid.UUID, updatedService *model.Service) error {
	return s.repo.UpdateByUUID(serviceUUID, updatedService)
}

func (s *serviceService) DeleteByUUID(serviceUUID uuid.UUID) error {
	return s.repo.DeleteByUUID(serviceUUID)
}
