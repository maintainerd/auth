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
	IsSystem    bool
	Status      string
	APICount    int64
	PolicyCount int64
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type ServiceServiceGetFilter struct {
	Name        *string
	DisplayName *string
	Description *string
	Version     *string
	IsSystem    *bool
	Status      []string
	TenantID    *int64
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
	GetByUUID(serviceUUID uuid.UUID, tenantID int64) (*ServiceServiceDataResult, error)
	Create(name string, displayName string, description string, version string, isSystem bool, status string, tenantID int64) (*ServiceServiceDataResult, error)
	Update(serviceUUID uuid.UUID, tenantID int64, name string, displayName string, description string, version string, isSystem bool, status string) (*ServiceServiceDataResult, error)
	SetStatusByUUID(serviceUUID uuid.UUID, tenantID int64, status string) (*ServiceServiceDataResult, error)
	DeleteByUUID(serviceUUID uuid.UUID, tenantID int64) (*ServiceServiceDataResult, error)
	AssignPolicy(serviceUUID uuid.UUID, policyUUID uuid.UUID, tenantID int64) error
	RemovePolicy(serviceUUID uuid.UUID, policyUUID uuid.UUID, tenantID int64) error
}

type serviceService struct {
	db                *gorm.DB
	serviceRepo       repository.ServiceRepository
	tenantServiceRepo repository.TenantServiceRepository
	apiRepo           repository.APIRepository
	servicePolicyRepo repository.ServicePolicyRepository
	policyRepo        repository.PolicyRepository
}

func NewServiceService(
	db *gorm.DB,
	serviceRepo repository.ServiceRepository,
	tenantServiceRepo repository.TenantServiceRepository,
	apiRepo repository.APIRepository,
	servicePolicyRepo repository.ServicePolicyRepository,
	policyRepo repository.PolicyRepository,
) ServiceService {
	return &serviceService{
		db:                db,
		serviceRepo:       serviceRepo,
		tenantServiceRepo: tenantServiceRepo,
		apiRepo:           apiRepo,
		servicePolicyRepo: servicePolicyRepo,
		policyRepo:        policyRepo,
	}
}

func (s *serviceService) Get(filter ServiceServiceGetFilter) (*ServiceServiceGetResult, error) {
	serviceFilter := repository.ServiceRepositoryGetFilter{
		Name:        filter.Name,
		DisplayName: filter.DisplayName,
		Description: filter.Description,
		Version:     filter.Version,
		IsSystem:    filter.IsSystem,
		Status:      filter.Status,
		TenantID:    filter.TenantID,
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
		services[i] = *s.toServiceServiceDataResult(&svc)
	}

	return &ServiceServiceGetResult{
		Data:       services,
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}, nil
}

func (s *serviceService) GetByUUID(serviceUUID uuid.UUID, tenantID int64) (*ServiceServiceDataResult, error) {
	service, err := s.serviceRepo.FindByUUID(serviceUUID)
	if err != nil || service == nil {
		return nil, errors.New("service not found")
	}

	// Verify service belongs to tenant by checking tenant_services relationship
	tenantService, err := s.tenantServiceRepo.FindByTenantAndService(tenantID, service.ServiceID)
	if err != nil || tenantService == nil {
		return nil, errors.New("service not found or access denied")
	}

	return s.toServiceServiceDataResult(service), nil
}

func (s *serviceService) Create(name string, displayName string, description string, version string, isSystem bool, status string, tenantID int64) (*ServiceServiceDataResult, error) {
	var createdService *model.Service

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txServiceRepo := s.serviceRepo.WithTx(tx)

		// Check if service already exists for this tenant
		existingService, err := txServiceRepo.FindByNameAndTenantID(name, tenantID)
		if err != nil {
			return err
		}
		if existingService != nil {
			return errors.New(name + " service already exists for this tenant")
		}

		// Create service
		newService := &model.Service{
			Name:        name,
			DisplayName: displayName,
			Description: description,
			Version:     version,
			IsSystem:    isSystem,
			Status:      status,
		}

		_, err = txServiceRepo.CreateOrUpdate(newService)
		if err != nil {
			return err
		}

		// Create tenant-service relationship
		txTenantServiceRepo := s.tenantServiceRepo.WithTx(tx)
		tenantService := &model.TenantService{
			TenantID:  tenantID,
			ServiceID: newService.ServiceID,
		}

		_, err = txTenantServiceRepo.CreateOrUpdate(tenantService)
		if err != nil {
			return err
		}

		createdService = newService

		return nil
	})

	if err != nil {
		return nil, err
	}

	return s.toServiceServiceDataResult(createdService), nil
}

func (s *serviceService) Update(serviceUUID uuid.UUID, tenantID int64, name string, displayName string, description string, version string, isSystem bool, status string) (*ServiceServiceDataResult, error) {
	var updatedService *model.Service

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txServiceRepo := s.serviceRepo.WithTx(tx)
		txTenantServiceRepo := s.tenantServiceRepo.WithTx(tx)

		service, err := txServiceRepo.FindByUUID(serviceUUID)
		if err != nil || service == nil {
			return errors.New("service not found")
		}

		// Verify service belongs to tenant
		tenantService, err := txTenantServiceRepo.FindByTenantAndService(tenantID, service.ServiceID)
		if err != nil || tenantService == nil {
			return errors.New("service not found or access denied")
		}

		// Check if service is a system record (critical for app functionality)
		if service.IsSystem {
			return errors.New("system service cannot be updated")
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
		service.IsSystem = isSystem
		service.Status = status

		_, err = txServiceRepo.CreateOrUpdate(service)
		if err != nil {
			return err
		}

		updatedService = service

		return nil
	})

	if err != nil {
		return nil, err
	}

	return s.toServiceServiceDataResult(updatedService), nil
}

func (s *serviceService) SetStatusByUUID(serviceUUID uuid.UUID, tenantID int64, status string) (*ServiceServiceDataResult, error) {
	var updatedService *model.Service

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txServiceRepo := s.serviceRepo.WithTx(tx)
		txTenantServiceRepo := s.tenantServiceRepo.WithTx(tx)

		service, err := txServiceRepo.FindByUUID(serviceUUID)
		if err != nil || service == nil {
			return errors.New("service not found")
		}

		// Verify service belongs to tenant
		tenantService, err := txTenantServiceRepo.FindByTenantAndService(tenantID, service.ServiceID)
		if err != nil || tenantService == nil {
			return errors.New("service not found or access denied")
		}

		// Check if service is a system record (critical for app functionality)
		if service.IsSystem {
			return errors.New("system service status cannot be updated")
		}

		service.Status = status

		_, err = txServiceRepo.CreateOrUpdate(service)
		if err != nil {
			return err
		}

		updatedService = service

		return nil
	})

	if err != nil {
		return nil, err
	}

	return s.toServiceServiceDataResult(updatedService), nil
}

func (s *serviceService) DeleteByUUID(serviceUUID uuid.UUID, tenantID int64) (*ServiceServiceDataResult, error) {
	service, err := s.serviceRepo.FindByUUID(serviceUUID)
	if err != nil || service == nil {
		return nil, errors.New("service not found")
	}

	// Verify service belongs to tenant
	tenantService, err := s.tenantServiceRepo.FindByTenantAndService(tenantID, service.ServiceID)
	if err != nil || tenantService == nil {
		return nil, errors.New("service not found or access denied")
	}

	// Check if service is a system record (critical for app functionality)
	if service.IsSystem {
		return nil, errors.New("system service cannot be deleted")
	}

	err = s.serviceRepo.DeleteByUUID(serviceUUID)
	if err != nil {
		return nil, err
	}

	return s.toServiceServiceDataResult(service), nil
}

// Helper function to convert model.Service to ServiceServiceDataResult with counts
func (s *serviceService) toServiceServiceDataResult(service *model.Service) *ServiceServiceDataResult {
	// Get API count for this service
	apiCount, _ := s.apiRepo.CountByServiceID(service.ServiceID, 0) // TODO: Fix to use actual tenant_id

	// Get policy count for this service
	policyCount, _ := s.serviceRepo.CountPoliciesByServiceID(service.ServiceID)

	return &ServiceServiceDataResult{
		ServiceUUID: service.ServiceUUID,
		Name:        service.Name,
		DisplayName: service.DisplayName,
		Description: service.Description,
		Version:     service.Version,
		IsSystem:    service.IsSystem,
		Status:      service.Status,
		APICount:    apiCount,
		PolicyCount: policyCount,
		CreatedAt:   service.CreatedAt,
		UpdatedAt:   service.UpdatedAt,
	}
}

func (s *serviceService) AssignPolicy(serviceUUID uuid.UUID, policyUUID uuid.UUID, tenantID int64) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		txServiceRepo := s.serviceRepo.WithTx(tx)
		txPolicyRepo := s.policyRepo.WithTx(tx)
		txServicePolicyRepo := s.servicePolicyRepo.WithTx(tx)

		// Check if service exists
		service, err := txServiceRepo.FindByUUID(serviceUUID)
		if err != nil {
			return err
		}
		if service == nil {
			return errors.New("service not found")
		}

		// Check if policy exists and belongs to the same tenant
		policy, err := txPolicyRepo.FindByUUIDAndTenantID(policyUUID, tenantID)
		if err != nil {
			return err
		}
		if policy == nil {
			return errors.New("policy not found or access denied")
		}

		// Check if assignment already exists
		existing, err := txServicePolicyRepo.FindByServiceAndPolicy(service.ServiceID, policy.PolicyID)
		if err != nil {
			return err
		}
		if existing != nil {
			// Assignment already exists, return success for idempotency
			return nil
		}

		// Create new service-policy assignment
		servicePolicy := &model.ServicePolicy{
			ServicePolicyUUID: uuid.New(),
			ServiceID:         service.ServiceID,
			PolicyID:          policy.PolicyID,
		}

		_, err = txServicePolicyRepo.Create(servicePolicy)
		return err
	})
}

func (s *serviceService) RemovePolicy(serviceUUID uuid.UUID, policyUUID uuid.UUID, tenantID int64) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		txServiceRepo := s.serviceRepo.WithTx(tx)
		txPolicyRepo := s.policyRepo.WithTx(tx)
		txServicePolicyRepo := s.servicePolicyRepo.WithTx(tx)

		// Check if service exists
		service, err := txServiceRepo.FindByUUID(serviceUUID)
		if err != nil {
			return err
		}
		if service == nil {
			return errors.New("service not found")
		}

		// Check if policy exists and belongs to the same tenant
		policy, err := txPolicyRepo.FindByUUIDAndTenantID(policyUUID, tenantID)
		if err != nil {
			return err
		}
		if policy == nil {
			return errors.New("policy not found or access denied")
		}

		// Check if assignment exists
		existing, err := txServicePolicyRepo.FindByServiceAndPolicy(service.ServiceID, policy.PolicyID)
		if err != nil {
			return err
		}
		if existing == nil {
			// Assignment doesn't exist, return success for idempotency
			return nil
		}

		// Remove the service-policy assignment
		return txServicePolicyRepo.DeleteByServiceAndPolicy(service.ServiceID, policy.PolicyID)
	})
}
