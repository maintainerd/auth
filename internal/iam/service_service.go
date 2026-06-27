package iam

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/authevent"
	"github.com/maintainerd/auth/internal/event"
	"github.com/maintainerd/auth/internal/platform/apperror"
	"github.com/maintainerd/auth/internal/platform/ptr"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
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
	Get(ctx context.Context, filter ServiceServiceGetFilter) (*ServiceServiceGetResult, error)
	GetByUUID(ctx context.Context, serviceUUID uuid.UUID, tenantID int64) (*ServiceServiceDataResult, error)
	Create(ctx context.Context, name string, displayName string, description string, version string, isSystem bool, status string, tenantID int64) (*ServiceServiceDataResult, error)
	Update(ctx context.Context, serviceUUID uuid.UUID, tenantID int64, name string, displayName string, description string, version string, isSystem bool, status string) (*ServiceServiceDataResult, error)
	SetStatusByUUID(ctx context.Context, serviceUUID uuid.UUID, tenantID int64, status string) (*ServiceServiceDataResult, error)
	DeleteByUUID(ctx context.Context, serviceUUID uuid.UUID, tenantID int64) (*ServiceServiceDataResult, error)
	AssignPolicy(ctx context.Context, serviceUUID uuid.UUID, policyUUID uuid.UUID, tenantID int64) error
	RemovePolicy(ctx context.Context, serviceUUID uuid.UUID, policyUUID uuid.UUID, tenantID int64) error
}

type serviceService struct {
	db                *gorm.DB
	serviceRepo       ServiceRepository
	tenantServiceRepo TenantServiceRepository
	apiRepo           APIRepository
	servicePolicyRepo ServicePolicyRepository
	policyRepo        PolicyRepository
	authEventService  authevent.AuthEventService
	eventService      event.EventService
}

func NewServiceService(
	db *gorm.DB,
	serviceRepo ServiceRepository,
	tenantServiceRepo TenantServiceRepository,
	apiRepo APIRepository,
	servicePolicyRepo ServicePolicyRepository,
	policyRepo PolicyRepository,
	authEventService ...authevent.AuthEventService,
) ServiceService {
	eventSvc := authevent.NoopService()
	if len(authEventService) > 0 && authEventService[0] != nil {
		eventSvc = authEventService[0]
	}
	return &serviceService{
		db:                db,
		serviceRepo:       serviceRepo,
		tenantServiceRepo: tenantServiceRepo,
		apiRepo:           apiRepo,
		servicePolicyRepo: servicePolicyRepo,
		policyRepo:        policyRepo,
		authEventService:  eventSvc,
		eventService:      nil,
	}
}

// SetServiceEventService injects the event service after construction.
func SetServiceEventService(svc ServiceService, es event.EventService) {
	if impl, ok := svc.(*serviceService); ok {
		impl.eventService = es
	}
}

func (s *serviceService) Get(ctx context.Context, filter ServiceServiceGetFilter) (*ServiceServiceGetResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "list")
	defer span.End()

	serviceFilter := ServiceRepositoryGetFilter{
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
		span.RecordError(err)
		span.SetStatus(codes.Error, "list services failed")
		return nil, err
	}

	var filterTenantID int64
	if filter.TenantID != nil {
		filterTenantID = *filter.TenantID
	}

	services := make([]ServiceServiceDataResult, len(result.Data))
	for i, svc := range result.Data {
		services[i] = *s.toServiceServiceDataResult(&svc, filterTenantID)
	}

	span.SetStatus(codes.Ok, "")
	return &ServiceServiceGetResult{
		Data:       services,
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}, nil
}

func (s *serviceService) GetByUUID(ctx context.Context, serviceUUID uuid.UUID, tenantID int64) (*ServiceServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "getByUUID")
	defer span.End()
	span.SetAttributes(attribute.String("uuid", serviceUUID.String()), attribute.Int64("tenant.id", tenantID))

	service, err := s.serviceRepo.FindByUUID(serviceUUID)
	if err != nil || service == nil {
		span.SetStatus(codes.Error, "service not found")
		return nil, apperror.NewNotFound("service not found")
	}

	// Verify service belongs to tenant by checking tenant_services relationship
	tenantService, err := s.tenantServiceRepo.FindByTenantAndService(tenantID, service.ServiceID)
	if err != nil || tenantService == nil {
		span.SetStatus(codes.Error, "service not found or access denied")
		return nil, apperror.NewNotFoundWithReason("service not found or access denied")
	}

	span.SetStatus(codes.Ok, "")
	return s.toServiceServiceDataResult(service, tenantID), nil
}

func (s *serviceService) Create(ctx context.Context, name string, displayName string, description string, version string, isSystem bool, status string, tenantID int64) (*ServiceServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "create")
	defer span.End()
	span.SetAttributes(attribute.Int64("tenant.id", tenantID))

	var createdService *Service

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txServiceRepo := s.serviceRepo.WithTx(tx)

		// Check if service already exists for this tenant
		existingService, err := txServiceRepo.FindByNameAndTenantID(name, tenantID)
		if err != nil {
			return err
		}
		if existingService != nil {
			return apperror.NewConflict(name + " service already exists for this tenant")
		}

		// Create service
		newService := &Service{
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
		tenantService := &TenantService{
			TenantID:  tenantID,
			ServiceID: newService.ServiceID,
		}

		_, err = txTenantServiceRepo.CreateOrUpdate(tenantService)
		if err != nil {
			return err
		}

		createdService = newService

		// Emit service.created inside the transaction
		if s.eventService != nil {
			if _, emitErr := s.eventService.Emit(ctx, tx, event.NewIntegrationEvent(
				event.EventTypeServiceCreated, 1, tenantID,
			).SetSubject(&createdService.ServiceUUID, "service")); emitErr != nil {
				return emitErr
			}
		}

		return nil
	})

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "create service failed")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return s.toServiceServiceDataResult(createdService, tenantID), nil
}

func (s *serviceService) Update(ctx context.Context, serviceUUID uuid.UUID, tenantID int64, name string, displayName string, description string, version string, isSystem bool, status string) (*ServiceServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "update")
	defer span.End()
	span.SetAttributes(attribute.String("uuid", serviceUUID.String()), attribute.Int64("tenant.id", tenantID))

	var updatedService *Service

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txServiceRepo := s.serviceRepo.WithTx(tx)
		txTenantServiceRepo := s.tenantServiceRepo.WithTx(tx)

		service, err := txServiceRepo.FindByUUID(serviceUUID)
		if err != nil || service == nil {
			return apperror.NewNotFound("service not found")
		}

		// Verify service belongs to tenant
		tenantService, err := txTenantServiceRepo.FindByTenantAndService(tenantID, service.ServiceID)
		if err != nil || tenantService == nil {
			return apperror.NewNotFoundWithReason("service not found or access denied")
		}

		// Check if service is a system record (critical for app functionality)
		if service.IsSystem {
			return apperror.NewValidation("system service cannot be updated")
		}

		if service.Name != name {
			existingService, err := txServiceRepo.FindByName(name)
			if err != nil {
				return err
			}
			if existingService != nil && existingService.ServiceUUID != serviceUUID {
				return apperror.NewConflict(name + " service already exists")
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

		// Emit service.updated inside the mutation transaction so the data
		// change and the event commit (or roll back) together.
		if s.eventService != nil {
			if _, emitErr := s.eventService.Emit(ctx, tx, event.NewIntegrationEvent(
				event.EventTypeServiceUpdated, 1, tenantID,
			).SetSubject(&service.ServiceUUID, "service")); emitErr != nil {
				return emitErr
			}
		}
		return nil
	})

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "update service failed")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return s.toServiceServiceDataResult(updatedService, tenantID), nil
}

func (s *serviceService) SetStatusByUUID(ctx context.Context, serviceUUID uuid.UUID, tenantID int64, status string) (*ServiceServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "setStatus")
	defer span.End()
	span.SetAttributes(attribute.String("uuid", serviceUUID.String()), attribute.Int64("tenant.id", tenantID))

	var updatedService *Service

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txServiceRepo := s.serviceRepo.WithTx(tx)
		txTenantServiceRepo := s.tenantServiceRepo.WithTx(tx)

		service, err := txServiceRepo.FindByUUID(serviceUUID)
		if err != nil || service == nil {
			return apperror.NewNotFound("service not found")
		}

		// Verify service belongs to tenant
		tenantService, err := txTenantServiceRepo.FindByTenantAndService(tenantID, service.ServiceID)
		if err != nil || tenantService == nil {
			return apperror.NewNotFoundWithReason("service not found or access denied")
		}

		// Check if service is a system record (critical for app functionality)
		if service.IsSystem {
			return apperror.NewValidation("system service status cannot be updated")
		}

		service.Status = status

		_, err = txServiceRepo.CreateOrUpdate(service)
		if err != nil {
			return err
		}

		updatedService = service

		if s.eventService != nil {
			if _, emitErr := s.eventService.Emit(ctx, tx, event.NewIntegrationEvent(
				event.EventTypeServiceStatusChanged, 1, tenantID,
			).SetSubject(&service.ServiceUUID, "service").SetChangedFields("status")); emitErr != nil {
				return emitErr
			}
		}
		return nil
	})

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "set service status failed")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return s.toServiceServiceDataResult(updatedService, tenantID), nil
}

func (s *serviceService) DeleteByUUID(ctx context.Context, serviceUUID uuid.UUID, tenantID int64) (*ServiceServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "delete")
	defer span.End()
	span.SetAttributes(attribute.String("uuid", serviceUUID.String()), attribute.Int64("tenant.id", tenantID))

	service, err := s.serviceRepo.FindByUUID(serviceUUID)
	if err != nil || service == nil {
		span.SetStatus(codes.Error, "service not found")
		return nil, apperror.NewNotFound("service not found")
	}

	// Verify service belongs to tenant
	tenantService, err := s.tenantServiceRepo.FindByTenantAndService(tenantID, service.ServiceID)
	if err != nil || tenantService == nil {
		span.SetStatus(codes.Error, "service not found or access denied")
		return nil, apperror.NewNotFoundWithReason("service not found or access denied")
	}

	// Check if service is a system record (critical for app functionality)
	if service.IsSystem {
		span.SetStatus(codes.Error, "system service cannot be deleted")
		return nil, apperror.NewValidation("system service cannot be deleted")
	}

	// Wrap delete + event emission in one transaction so they commit together.
	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := s.serviceRepo.WithTx(tx).DeleteByUUID(serviceUUID); err != nil {
			return err
		}
		if s.eventService != nil {
			if _, emitErr := s.eventService.Emit(ctx, tx, event.NewIntegrationEvent(
				event.EventTypeServiceDeleted, 1, tenantID,
			).SetSubject(&service.ServiceUUID, "service")); emitErr != nil {
				return emitErr
			}
		}
		return nil
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "delete service failed")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return s.toServiceServiceDataResult(service, tenantID), nil
}

// Helper function to convert Service to ServiceServiceDataResult with counts
func (s *serviceService) toServiceServiceDataResult(service *Service, tenantID int64) *ServiceServiceDataResult {
	// Get API count for this service scoped to the caller's tenant
	apiCount, _ := s.apiRepo.CountByServiceID(service.ServiceID, tenantID)

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

func (s *serviceService) AssignPolicy(ctx context.Context, serviceUUID uuid.UUID, policyUUID uuid.UUID, tenantID int64) error {
	_, span := otel.Tracer("service").Start(ctx, "assignPolicy")
	defer span.End()
	span.SetAttributes(attribute.String("uuid", serviceUUID.String()), attribute.String("policy.uuid", policyUUID.String()), attribute.Int64("tenant.id", tenantID))

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txServiceRepo := s.serviceRepo.WithTx(tx)
		txPolicyRepo := s.policyRepo.WithTx(tx)
		txServicePolicyRepo := s.servicePolicyRepo.WithTx(tx)

		// Check if service exists
		service, err := txServiceRepo.FindByUUID(serviceUUID)
		if err != nil {
			return err
		}
		if service == nil {
			return apperror.NewNotFound("service not found")
		}
		// Tenant isolation: the service must belong to the caller's tenant,
		// otherwise a tenant could attach/detach its policy on another tenant's
		// (or a system) service.
		if service.TenantID != tenantID {
			return apperror.NewNotFoundWithReason("service not found or access denied")
		}

		// Check if policy exists and belongs to the same tenant
		policy, err := txPolicyRepo.FindByUUIDAndTenantID(policyUUID, tenantID)
		if err != nil {
			return err
		}
		if policy == nil {
			return apperror.NewNotFoundWithReason("policy not found or access denied")
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
		servicePolicy := &ServicePolicy{
			ServicePolicyUUID: uuid.New(),
			ServiceID:         service.ServiceID,
			PolicyID:          policy.PolicyID,
		}

		_, err = txServicePolicyRepo.Create(servicePolicy)
		return err
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "assign policy failed")
		return err
	}
	span.SetStatus(codes.Ok, "")
	s.authEventService.Log(ctx, authevent.AuthEventInput{
		TenantID:    tenantID,
		Category:    authevent.AuthEventCategoryAuthz,
		EventType:   authevent.AuthEventTypeIAMServicePolicyAssigned,
		Severity:    authevent.AuthEventSeverityInfo,
		Result:      authevent.AuthEventResultSuccess,
		Description: ptr.Ptr("Service policy assigned"),
	})
	return nil
}

func (s *serviceService) RemovePolicy(ctx context.Context, serviceUUID uuid.UUID, policyUUID uuid.UUID, tenantID int64) error {
	_, span := otel.Tracer("service").Start(ctx, "removePolicy")
	defer span.End()
	span.SetAttributes(attribute.String("uuid", serviceUUID.String()), attribute.String("policy.uuid", policyUUID.String()), attribute.Int64("tenant.id", tenantID))

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txServiceRepo := s.serviceRepo.WithTx(tx)
		txPolicyRepo := s.policyRepo.WithTx(tx)
		txServicePolicyRepo := s.servicePolicyRepo.WithTx(tx)

		// Check if service exists
		service, err := txServiceRepo.FindByUUID(serviceUUID)
		if err != nil {
			return err
		}
		if service == nil {
			return apperror.NewNotFound("service not found")
		}
		// Tenant isolation: the service must belong to the caller's tenant,
		// otherwise a tenant could attach/detach its policy on another tenant's
		// (or a system) service.
		if service.TenantID != tenantID {
			return apperror.NewNotFoundWithReason("service not found or access denied")
		}

		// Check if policy exists and belongs to the same tenant
		policy, err := txPolicyRepo.FindByUUIDAndTenantID(policyUUID, tenantID)
		if err != nil {
			return err
		}
		if policy == nil {
			return apperror.NewNotFoundWithReason("policy not found or access denied")
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
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "remove policy failed")
		return err
	}
	span.SetStatus(codes.Ok, "")
	s.authEventService.Log(ctx, authevent.AuthEventInput{
		TenantID:    tenantID,
		Category:    authevent.AuthEventCategoryAuthz,
		EventType:   authevent.AuthEventTypeIAMServicePolicyRemoved,
		Severity:    authevent.AuthEventSeverityInfo,
		Result:      authevent.AuthEventResultSuccess,
		Description: ptr.Ptr("Service policy removed"),
	})
	return nil
}
