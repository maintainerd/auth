package service

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/repository"
	"gorm.io/gorm"
)

type ServiceServiceDataResult struct {
	ServiceUUID uuid.UUID
	Name        string
	DisplayName string
	Description string
	Version     string
	IsDefault   bool
	IsActive    bool
	IsPublic    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type ServiceServiceGetFilter struct {
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

type ServiceServiceGetResult struct {
	Data       []ServiceServiceDataResult
	Total      int64
	Page       int
	Limit      int
	TotalPages int
}

type ServiceService interface {
	Get(filter ServiceServiceGetFilter) (*ServiceServiceGetResult, error)
	GetByUUID(serviceUUID uuid.UUID) (*ServiceServiceDataResult, error)
	Create(name string, displayName string, description string, version string, isDefault bool, isActive bool, isPublic bool) (*ServiceServiceDataResult, error)
	Update(serviceUUID uuid.UUID, name string, displayName string, description string, version string, isDefault bool, isActive bool, isPublic bool) (*ServiceServiceDataResult, error)
	SetActiveStatusByUUID(serviceUUID uuid.UUID) (*ServiceServiceDataResult, error)
	DeleteByUUID(serviceUUID uuid.UUID) (*ServiceServiceDataResult, error)
}

type serviceService struct {
	db          *gorm.DB
	serviceRepo repository.ServiceRepository
}

func NewServiceService(
	db *gorm.DB,
	serviceRepo repository.ServiceRepository,
) ServiceService {
	return &serviceService{
		db:          db,
		serviceRepo: serviceRepo,
	}
}

func (s *serviceService) Get(filter ServiceServiceGetFilter) (*ServiceServiceGetResult, error) {
	serviceFilter := repository.ServiceRepositoryGetFilter{
		Name:        filter.Name,
		DisplayName: filter.DisplayName,
		Description: filter.Description,
		Version:     filter.Version,
		IsDefault:   filter.IsDefault,
		IsActive:    filter.IsActive,
		IsPublic:    filter.IsPublic,
		Page:        filter.Page,
		Limit:       filter.Limit,
		SortBy:      filter.SortBy,
		SortOrder:   filter.SortOrder,
	}

	result, err := s.serviceRepo.FindPaginated(serviceFilter)
	if err != nil {
		return nil, err
	}

	services := make([]ServiceServiceDataResult, len(result.Data))
	for i, svc := range result.Data {
		services[i] = ServiceServiceDataResult{
			ServiceUUID: svc.ServiceUUID,
			Name:        svc.Name,
			DisplayName: svc.DisplayName,
			Description: svc.Description,
			Version:     svc.Version,
			IsDefault:   svc.IsDefault,
			IsActive:    svc.IsActive,
			IsPublic:    svc.IsPublic,
			CreatedAt:   svc.CreatedAt,
			UpdatedAt:   svc.UpdatedAt,
		}
	}

	return &ServiceServiceGetResult{
		Data:       services,
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}, nil
}

func (s *serviceService) GetByUUID(serviceUUID uuid.UUID) (*ServiceServiceDataResult, error) {
	service, err := s.serviceRepo.FindByUUID(serviceUUID)
	if err != nil {
		return nil, err
	}
	if service == nil {
		return nil, errors.New("service not found")
	}

	return &ServiceServiceDataResult{
		ServiceUUID: service.ServiceUUID,
		Name:        service.Name,
		DisplayName: service.DisplayName,
		Description: service.Description,
		Version:     service.Version,
		IsDefault:   service.IsDefault,
		IsActive:    service.IsActive,
		IsPublic:    service.IsPublic,
		CreatedAt:   service.CreatedAt,
		UpdatedAt:   service.UpdatedAt,
	}, nil
}

func (s *serviceService) Create(name string, displayName string, description string, version string, isDefault bool, isActive bool, isPublic bool) (*ServiceServiceDataResult, error) {
	var createdService *model.Service

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txServiceRepo := s.serviceRepo.WithTx(tx)

		// Check if service already exists
		existingService, err := txServiceRepo.FindByName(name)
		if err != nil {
			return err
		}
		if existingService != nil {
			return errors.New(name + " service already exists")
		}

		// Create service
		newService := &model.Service{
			Name:        name,
			DisplayName: displayName,
			Description: description,
			Version:     version,
			IsDefault:   isDefault,
			IsActive:    isActive,
			IsPublic:    isPublic,
		}

		if err := txServiceRepo.CreateOrUpdate(newService); err != nil {
			return err
		}

		createdService = newService
		return nil
	})

	if err != nil {
		return nil, err
	}

	return &ServiceServiceDataResult{
		ServiceUUID: createdService.ServiceUUID,
		Name:        createdService.Name,
		DisplayName: createdService.DisplayName,
		Description: createdService.Description,
		Version:     createdService.Version,
		IsDefault:   createdService.IsDefault,
		IsActive:    createdService.IsActive,
		IsPublic:    createdService.IsPublic,
		CreatedAt:   createdService.CreatedAt,
		UpdatedAt:   createdService.UpdatedAt,
	}, nil
}

func (s *serviceService) Update(serviceUUID uuid.UUID, name string, displayName string, description string, version string, isDefault bool, isActive bool, isPublic bool) (*ServiceServiceDataResult, error) {
	var updatedService *model.Service

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txServiceRepo := s.serviceRepo.WithTx(tx)

		service, err := txServiceRepo.FindByUUID(serviceUUID)
		if err != nil {
			return err
		}
		if service == nil {
			return errors.New("service not found")
		}

		if service.IsDefault {
			return errors.New("default service cannot be updated")
		}

		if service.Name != name {
			existingService, err := txServiceRepo.FindByName(name)
			if err != nil {
				return err
			}
			if existingService != nil && existingService.ServiceUUID != serviceUUID {
				return errors.New(name + " service already exists")
			}
		}

		service.Name = name
		service.DisplayName = displayName
		service.Description = description
		service.Version = version
		service.IsDefault = isDefault
		service.IsActive = isActive
		service.IsPublic = isPublic

		if err := txServiceRepo.CreateOrUpdate(service); err != nil {
			return err
		}

		updatedService = service
		return nil
	})

	if err != nil {
		return nil, err
	}

	return &ServiceServiceDataResult{
		ServiceUUID: updatedService.ServiceUUID,
		Name:        updatedService.Name,
		DisplayName: updatedService.DisplayName,
		Description: updatedService.Description,
		Version:     updatedService.Version,
		IsDefault:   updatedService.IsDefault,
		IsActive:    updatedService.IsActive,
		IsPublic:    updatedService.IsPublic,
		CreatedAt:   updatedService.CreatedAt,
		UpdatedAt:   updatedService.UpdatedAt,
	}, nil
}

func (s *serviceService) SetActiveStatusByUUID(serviceUUID uuid.UUID) (*ServiceServiceDataResult, error) {
	var updatedService *model.Service

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txServiceRepo := s.serviceRepo.WithTx(tx)

		service, err := txServiceRepo.FindByUUID(serviceUUID)
		if err != nil {
			return err
		}
		if service == nil {
			return errors.New("service not found")
		}

		if service.IsDefault {
			return errors.New("default service cannot be updated")
		}

		service.IsActive = !service.IsActive

		if err := txServiceRepo.CreateOrUpdate(service); err != nil {
			return err
		}

		updatedService = service
		return nil
	})

	if err != nil {
		return nil, err
	}

	return &ServiceServiceDataResult{
		ServiceUUID: updatedService.ServiceUUID,
		Name:        updatedService.Name,
		DisplayName: updatedService.DisplayName,
		Description: updatedService.Description,
		Version:     updatedService.Version,
		IsDefault:   updatedService.IsDefault,
		IsActive:    updatedService.IsActive,
		IsPublic:    updatedService.IsPublic,
		CreatedAt:   updatedService.CreatedAt,
		UpdatedAt:   updatedService.UpdatedAt,
	}, nil
}

func (s *serviceService) DeleteByUUID(serviceUUID uuid.UUID) (*ServiceServiceDataResult, error) {
	service, err := s.serviceRepo.FindByUUID(serviceUUID)
	if err != nil {
		return nil, err
	}
	if service == nil {
		return nil, errors.New("service not found")
	}

	if service.IsDefault {
		return nil, errors.New("default service cannot be deleted")
	}

	err = s.serviceRepo.DeleteByUUID(serviceUUID)
	if err != nil {
		return nil, err
	}

	return &ServiceServiceDataResult{
		ServiceUUID: service.ServiceUUID,
		Name:        service.Name,
		DisplayName: service.DisplayName,
		Description: service.Description,
		Version:     service.Version,
		IsDefault:   service.IsDefault,
		IsActive:    service.IsActive,
		IsPublic:    service.IsPublic,
		CreatedAt:   service.CreatedAt,
		UpdatedAt:   service.UpdatedAt,
	}, nil
}
