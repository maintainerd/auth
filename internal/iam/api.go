package iam

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/platform/apperror"
	"github.com/maintainerd/auth/internal/platform/crypto"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"gorm.io/gorm"
)

type APIServiceDataResult struct {
	APIUUID     uuid.UUID
	Name        string
	DisplayName string
	Description string
	APIType     string
	Identifier  string
	Service     *ServiceServiceDataResult
	Status      string
	IsSystem    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type APIServiceGetFilter struct {
	TenantID    int64
	Name        *string
	DisplayName *string
	APIType     *string
	Identifier  *string
	ServiceID   *int64
	Status      []string
	IsSystem    *bool
	Page        int
	Limit       int
	SortBy      string
	SortOrder   string
}

type APIServiceGetResult struct {
	Data       []APIServiceDataResult
	Total      int64
	Page       int
	Limit      int
	TotalPages int
}

type APIService interface {
	Get(ctx context.Context, filter APIServiceGetFilter) (*APIServiceGetResult, error)
	GetByUUID(ctx context.Context, apiUUID uuid.UUID, tenantID int64) (*APIServiceDataResult, error)
	GetServiceIDByUUID(ctx context.Context, serviceUUID uuid.UUID) (int64, error)
	Create(ctx context.Context, tenantID int64, name string, displayName string, description string, apiType string, status string, isSystem bool, serviceUUID string) (*APIServiceDataResult, error)
	Update(ctx context.Context, apiUUID uuid.UUID, tenantID int64, name string, displayName string, description string, apiType string, status string, serviceUUID string) (*APIServiceDataResult, error)
	SetStatusByUUID(ctx context.Context, apiUUID uuid.UUID, tenantID int64, status string) (*APIServiceDataResult, error)
	DeleteByUUID(ctx context.Context, apiUUID uuid.UUID, tenantID int64) (*APIServiceDataResult, error)
}

type apiService struct {
	db                *gorm.DB
	apiRepo           APIRepository
	serviceRepo       ServiceRepository
	tenantServiceRepo TenantServiceRepository
}

func NewAPIService(
	db *gorm.DB,
	apiRepo APIRepository,
	serviceRepo ServiceRepository,
	tenantServiceRepo TenantServiceRepository,
) APIService {
	return &apiService{
		db:                db,
		apiRepo:           apiRepo,
		serviceRepo:       serviceRepo,
		tenantServiceRepo: tenantServiceRepo,
	}
}

func (s *apiService) Get(ctx context.Context, filter APIServiceGetFilter) (*APIServiceGetResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "api.list")
	defer span.End()
	span.SetAttributes(attribute.Int64("tenant.id", filter.TenantID))

	apiFilter := APIRepositoryGetFilter{
		TenantID:    filter.TenantID,
		Name:        filter.Name,
		DisplayName: filter.DisplayName,
		APIType:     filter.APIType,
		Identifier:  filter.Identifier,
		ServiceID:   filter.ServiceID,
		Status:      filter.Status,
		IsSystem:    filter.IsSystem,
		Page:        filter.Page,
		Limit:       filter.Limit,
		SortBy:      filter.SortBy,
		SortOrder:   filter.SortOrder,
	}

	result, err := s.apiRepo.FindPaginated(apiFilter)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to fetch apis")
		return nil, err
	}

	apis := make([]APIServiceDataResult, len(result.Data))
	for i, api := range result.Data {
		apis[i] = *toAPIServiceDataResult(&api)
	}

	span.SetStatus(codes.Ok, "")
	return &APIServiceGetResult{
		Data:       apis,
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}, nil
}

func (s *apiService) GetByUUID(ctx context.Context, apiUUID uuid.UUID, tenantID int64) (*APIServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "api.get")
	defer span.End()
	span.SetAttributes(
		attribute.String("api.uuid", apiUUID.String()),
		attribute.Int64("tenant.id", tenantID),
	)

	api, err := s.apiRepo.FindByUUIDAndTenantID(apiUUID, tenantID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to fetch api")
		return nil, err
	}
	if api == nil {
		span.SetStatus(codes.Error, "api not found")
		return nil, apperror.NewNotFound("api not found")
	}

	span.SetStatus(codes.Ok, "")
	return toAPIServiceDataResult(api), nil
}

func (s *apiService) GetServiceIDByUUID(ctx context.Context, serviceUUID uuid.UUID) (int64, error) {
	_, span := otel.Tracer("service").Start(ctx, "api.getServiceID")
	defer span.End()
	span.SetAttributes(attribute.String("uuid", serviceUUID.String()))

	service, err := s.serviceRepo.FindByUUID(serviceUUID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to fetch service")
		return 0, apperror.NewNotFound("service not found")
	}
	if service == nil {
		span.SetStatus(codes.Error, "service not found")
		return 0, apperror.NewNotFound("service not found")
	}

	span.SetStatus(codes.Ok, "")
	return service.ServiceID, nil
}

func (s *apiService) Create(ctx context.Context, tenantID int64, name string, displayName string, description string, apiType string, status string, isSystem bool, serviceUUID string) (*APIServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "api.create")
	defer span.End()
	span.SetAttributes(
		attribute.Int64("tenant.id", tenantID),
		attribute.String("api.name", name),
	)

	var createdAPI *API

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txAPIRepo := s.apiRepo.WithTx(tx)
		txServiceRepo := s.serviceRepo.WithTx(tx)

		// Check if api already exists
		existingAPI, err := txAPIRepo.FindByName(name, tenantID)
		if err != nil {
			return err
		}
		if existingAPI != nil {
			return apperror.NewConflict(name + " api already exists")
		}

		// Check if service exist
		service, err := txServiceRepo.FindByUUID(serviceUUID)
		if err != nil || service == nil {
			return apperror.NewNotFound("service not found")
		}

		// Generate identifier
		idSuffix, err := crypto.GenerateIdentifier(12)
		if err != nil {
			return err
		}
		identifier := fmt.Sprintf("api-%s", idSuffix)

		// Create api
		newAPI := &API{
			Name:        name,
			DisplayName: displayName,
			Description: description,
			APIType:     apiType,
			Identifier:  identifier,
			ServiceID:   service.ServiceID,
			TenantID:    tenantID,
			Status:      status,
			IsSystem:    isSystem,
		}

		_, err = txAPIRepo.CreateOrUpdate(newAPI)
		if err != nil {
			return err
		}

		// Fetch API with Service preloaded
		createdAPI, err = txAPIRepo.FindByUUID(newAPI.APIUUID, "Service")
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to create api")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return toAPIServiceDataResult(createdAPI), nil
}

func (s *apiService) Update(ctx context.Context, apiUUID uuid.UUID, tenantID int64, name string, displayName string, description string, apiType string, status string, serviceUUID string) (*APIServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "api.update")
	defer span.End()
	span.SetAttributes(
		attribute.String("api.uuid", apiUUID.String()),
		attribute.Int64("tenant.id", tenantID),
	)

	var updatedAPI *API

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txAPIRepo := s.apiRepo.WithTx(tx)
		txServiceRepo := s.serviceRepo.WithTx(tx)
		txTenantServiceRepo := s.tenantServiceRepo.WithTx(tx)

		// Get api and validate tenant ownership
		api, err := txAPIRepo.FindByUUIDAndTenantID(apiUUID, tenantID)
		if err != nil {
			return err
		}
		if api == nil {
			return apperror.NewNotFoundWithReason("api not found or access denied")
		}

		// Verify API's service belongs to tenant
		if api.ServiceID > 0 {
			tenantService, err := txTenantServiceRepo.FindByTenantAndService(tenantID, api.ServiceID)
			if err != nil || tenantService == nil {
				return apperror.NewNotFoundWithReason("api not found or access denied")
			}
		}

		// Check if API is a system record (critical for app functionality)
		if api.IsSystem {
			return apperror.NewValidation("system API cannot be updated")
		}

		// Validate service exists
		serviceUUIDParsed, err := uuid.Parse(serviceUUID)
		if err != nil {
			return apperror.NewValidation("invalid service UUID format")
		}

		service, err := txServiceRepo.FindByUUID(serviceUUIDParsed)
		if err != nil || service == nil {
			return apperror.NewNotFound("service not found")
		}

		// Verify new service belongs to tenant
		tenantService, err := txTenantServiceRepo.FindByTenantAndService(tenantID, service.ServiceID)
		if err != nil || tenantService == nil {
			return apperror.NewNotFoundWithReason("service not found or access denied")
		}

		// Check if api already exist
		if api.Name != name {
			existingAPI, err := txAPIRepo.FindByName(name, tenantID)
			if err != nil {
				return err
			}
			if existingAPI != nil && existingAPI.APIUUID != apiUUID {
				return apperror.NewConflict(name + " api already exists")
			}
		}

		// Set values
		api.Name = name
		api.DisplayName = displayName
		api.Description = description
		api.APIType = apiType
		api.Status = status
		api.ServiceID = service.ServiceID // Update the service assignment

		// Update
		_, err = txAPIRepo.CreateOrUpdate(api)
		if err != nil {
			return err
		}

		updatedAPI = api

		return nil
	})

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to update api")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return toAPIServiceDataResult(updatedAPI), nil
}

func (s *apiService) SetStatusByUUID(ctx context.Context, apiUUID uuid.UUID, tenantID int64, status string) (*APIServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "api.setStatus")
	defer span.End()
	span.SetAttributes(
		attribute.String("api.uuid", apiUUID.String()),
		attribute.Int64("tenant.id", tenantID),
		attribute.String("api.status", status),
	)

	var updatedAPI *API

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txAPIRepo := s.apiRepo.WithTx(tx)
		txTenantServiceRepo := s.tenantServiceRepo.WithTx(tx)

		// Get api and validate tenant ownership
		api, err := txAPIRepo.FindByUUIDAndTenantID(apiUUID, tenantID)
		if err != nil {
			return err
		}
		if api == nil {
			return apperror.NewNotFoundWithReason("api not found or access denied")
		}

		// Verify API's service belongs to tenant
		if api.ServiceID > 0 {
			tenantService, err := txTenantServiceRepo.FindByTenantAndService(tenantID, api.ServiceID)
			if err != nil || tenantService == nil {
				return apperror.NewNotFoundWithReason("api not found or access denied")
			}
		}

		// Check if API is a system record (critical for app functionality)
		if api.IsSystem {
			return apperror.NewValidation("system API status cannot be updated")
		}

		api.Status = status

		_, err = txAPIRepo.CreateOrUpdate(api)
		if err != nil {
			return err
		}

		updatedAPI = api

		return nil
	})

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to update api status")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return toAPIServiceDataResult(updatedAPI), nil
}

func (s *apiService) DeleteByUUID(ctx context.Context, apiUUID uuid.UUID, tenantID int64) (*APIServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "api.delete")
	defer span.End()
	span.SetAttributes(
		attribute.String("api.uuid", apiUUID.String()),
		attribute.Int64("tenant.id", tenantID),
	)

	// Get api and validate tenant ownership
	api, err := s.apiRepo.FindByUUIDAndTenantID(apiUUID, tenantID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to fetch api")
		return nil, err
	}
	if api == nil {
		span.SetStatus(codes.Error, "api not found or access denied")
		return nil, apperror.NewNotFoundWithReason("api not found or access denied")
	}

	// Verify API's service belongs to tenant
	if api.ServiceID > 0 {
		tenantService, err := s.tenantServiceRepo.FindByTenantAndService(tenantID, api.ServiceID)
		if err != nil || tenantService == nil {
			span.SetStatus(codes.Error, "api not found or access denied")
			return nil, apperror.NewNotFoundWithReason("api not found or access denied")
		}
	}

	// Check if API is a system record (critical for app functionality)
	if api.IsSystem {
		span.SetStatus(codes.Error, "system API cannot be deleted")
		return nil, apperror.NewValidation("system API cannot be deleted")
	}

	err = s.apiRepo.DeleteByUUIDAndTenantID(apiUUID, tenantID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to delete api")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return toAPIServiceDataResult(api), nil
}

// Reponse builder
func toAPIServiceDataResult(api *API) *APIServiceDataResult {
	if api == nil {
		return nil
	}

	result := &APIServiceDataResult{
		APIUUID:     api.APIUUID,
		Name:        api.Name,
		DisplayName: api.DisplayName,
		Description: api.Description,
		APIType:     api.APIType,
		Identifier:  api.Identifier,
		Status:      api.Status,
		IsSystem:    api.IsSystem,
		CreatedAt:   api.CreatedAt,
		UpdatedAt:   api.UpdatedAt,
	}

	if api.Service != nil {
		result.Service = &ServiceServiceDataResult{
			ServiceUUID: api.Service.ServiceUUID,
			Name:        api.Service.Name,
			DisplayName: api.Service.DisplayName,
			Description: api.Service.Description,
			Version:     api.Service.Version,
			IsSystem:    api.Service.IsSystem,
			Status:      api.Service.Status,
			APICount:    0, // Not needed in API context
			PolicyCount: 0, // Not needed in API context
			CreatedAt:   api.Service.CreatedAt,
			UpdatedAt:   api.Service.UpdatedAt,
		}
	}

	return result
}

type API struct {
	APIID       int64          `gorm:"column:api_id;primaryKey"`
	APIUUID     uuid.UUID      `gorm:"column:api_uuid"`
	TenantID    int64          `gorm:"column:tenant_id;not null"`
	ServiceID   int64          `gorm:"column:service_id"`
	Name        string         `gorm:"column:name"`
	DisplayName string         `gorm:"column:display_name"`
	Description string         `gorm:"column:description"`
	APIType     string         `gorm:"column:api_type"`
	Identifier  string         `gorm:"column:identifier"`
	Status      string         `gorm:"column:status;default:'inactive'"`
	IsSystem    bool           `gorm:"column:is_system;default:false"`
	CreatedBy   *int64         `gorm:"column:created_by"`
	UpdatedBy   *int64         `gorm:"column:updated_by"`
	CreatedAt   time.Time      `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt   time.Time      `gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt   gorm.DeletedAt `gorm:"column:deleted_at;index"`

	// Relationships
	Service *Service `gorm:"foreignKey:ServiceID;references:ServiceID"`
}

func (API) TableName() string {
	return "apis"
}

func (a *API) BeforeCreate(tx *gorm.DB) (err error) {
	if a.APIUUID == uuid.Nil {
		a.APIUUID = uuid.New()
	}
	return
}
