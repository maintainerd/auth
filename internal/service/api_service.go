package service

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/repository"
	"github.com/maintainerd/auth/internal/util"
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
	IsDefault   bool
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
	IsDefault   *bool
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
	Get(filter APIServiceGetFilter) (*APIServiceGetResult, error)
	GetByUUID(apiUUID uuid.UUID, tenantID int64) (*APIServiceDataResult, error)
	GetServiceIDByUUID(serviceUUID uuid.UUID) (int64, error)
	Create(tenantID int64, name string, displayName string, description string, apiType string, status string, isSystem bool, serviceUUID string) (*APIServiceDataResult, error)
	Update(apiUUID uuid.UUID, tenantID int64, name string, displayName string, description string, apiType string, status string, serviceUUID string) (*APIServiceDataResult, error)
	SetStatusByUUID(apiUUID uuid.UUID, tenantID int64, status string) (*APIServiceDataResult, error)
	DeleteByUUID(apiUUID uuid.UUID, tenantID int64) (*APIServiceDataResult, error)
}

type apiService struct {
	db          *gorm.DB
	apiRepo     repository.APIRepository
	serviceRepo repository.ServiceRepository
}

func NewAPIService(
	db *gorm.DB,
	apiRepo repository.APIRepository,
	serviceRepo repository.ServiceRepository,
) APIService {
	return &apiService{
		db:          db,
		apiRepo:     apiRepo,
		serviceRepo: serviceRepo,
	}
}

func (s *apiService) Get(filter APIServiceGetFilter) (*APIServiceGetResult, error) {
	apiFilter := repository.APIRepositoryGetFilter{
		TenantID:    filter.TenantID,
		Name:        filter.Name,
		DisplayName: filter.DisplayName,
		APIType:     filter.APIType,
		Identifier:  filter.Identifier,
		ServiceID:   filter.ServiceID,
		Status:      filter.Status,
		IsDefault:   filter.IsDefault,
		IsSystem:    filter.IsSystem,
		Page:        filter.Page,
		Limit:       filter.Limit,
		SortBy:      filter.SortBy,
		SortOrder:   filter.SortOrder,
	}

	result, err := s.apiRepo.FindPaginated(apiFilter)
	if err != nil {
		return nil, err
	}

	apis := make([]APIServiceDataResult, len(result.Data))
	for i, api := range result.Data {
		apis[i] = *toAPIServiceDataResult(&api)
	}

	return &APIServiceGetResult{
		Data:       apis,
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}, nil
}

func (s *apiService) GetByUUID(apiUUID uuid.UUID, tenantID int64) (*APIServiceDataResult, error) {
	api, err := s.apiRepo.FindByUUIDAndTenantID(apiUUID, tenantID)
	if err != nil {
		return nil, err
	}
	if api == nil {
		return nil, errors.New("api not found")
	}

	return toAPIServiceDataResult(api), nil
}

func (s *apiService) GetServiceIDByUUID(serviceUUID uuid.UUID) (int64, error) {
	service, err := s.serviceRepo.FindByUUID(serviceUUID)
	if err != nil || service == nil {
		return 0, errors.New("service not found")
	}

	return service.ServiceID, nil
}

func (s *apiService) Create(tenantID int64, name string, displayName string, description string, apiType string, status string, isSystem bool, serviceUUID string) (*APIServiceDataResult, error) {
	var createdAPI *model.API

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txAPIRepo := s.apiRepo.WithTx(tx)
		txServiceRepo := s.serviceRepo.WithTx(tx)

		// Check if api already exists
		existingAPI, err := txAPIRepo.FindByName(name, tenantID)
		if err != nil {
			return err
		}
		if existingAPI != nil {
			return errors.New(name + " api already exists")
		}

		// Check if service exist
		service, err := txServiceRepo.FindByUUID(serviceUUID)
		if err != nil || service == nil {
			return errors.New("service not found")
		}

		// Generate identifier
		identifier := fmt.Sprintf("api-%s", util.GenerateIdentifier(12))

		// Create api
		newAPI := &model.API{
			Name:        name,
			DisplayName: displayName,
			Description: description,
			APIType:     apiType,
			Identifier:  identifier,
			ServiceID:   service.ServiceID,
			TenantID:    tenantID,
			Status:      status,
			IsDefault:   false, // System-managed field, always default to false for user-created APIs
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
		return nil, err
	}

	return toAPIServiceDataResult(createdAPI), nil
}

func (s *apiService) Update(apiUUID uuid.UUID, tenantID int64, name string, displayName string, description string, apiType string, status string, serviceUUID string) (*APIServiceDataResult, error) {
	var updatedAPI *model.API

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txAPIRepo := s.apiRepo.WithTx(tx)

		// Get api and validate tenant ownership
		api, err := txAPIRepo.FindByUUIDAndTenantID(apiUUID, tenantID)
		if err != nil {
			return err
		}
		if api == nil {
			return errors.New("api not found or access denied")
		}

		// Check if default
		if api.IsDefault {
			return errors.New("default api cannot be updated")
		}

		// Validate service exists
		txServiceRepo := s.serviceRepo.WithTx(tx)
		serviceUUIDParsed, err := uuid.Parse(serviceUUID)
		if err != nil {
			return errors.New("invalid service UUID format")
		}

		service, err := txServiceRepo.FindByUUID(serviceUUIDParsed)
		if err != nil || service == nil {
			return errors.New("service not found")
		}

		// Check if api already exist
		if api.Name != name {
			existingAPI, err := txAPIRepo.FindByName(name, tenantID)
			if err != nil {
				return err
			}
			if existingAPI != nil && existingAPI.APIUUID != apiUUID {
				return errors.New(name + " api already exists")
			}
		}

		// Set values
		api.Name = name
		api.DisplayName = displayName
		api.Description = description
		api.APIType = apiType
		api.Status = status
		api.ServiceID = service.ServiceID // Update the service assignment
		// IsDefault is system-managed, don't update it in user requests

		// Update
		_, err = txAPIRepo.CreateOrUpdate(api)
		if err != nil {
			return err
		}

		updatedAPI = api

		return nil
	})

	if err != nil {
		return nil, err
	}

	return toAPIServiceDataResult(updatedAPI), nil
}

func (s *apiService) SetStatusByUUID(apiUUID uuid.UUID, tenantID int64, status string) (*APIServiceDataResult, error) {
	var updatedAPI *model.API

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txAPIRepo := s.apiRepo.WithTx(tx)

		// Get api and validate tenant ownership
		api, err := txAPIRepo.FindByUUIDAndTenantID(apiUUID, tenantID)
		if err != nil {
			return err
		}
		if api == nil {
			return errors.New("api not found or access denied")
		}

		// Check if default
		if api.IsDefault {
			return errors.New("default api cannot be updated")
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
		return nil, err
	}

	return toAPIServiceDataResult(updatedAPI), nil
}

func (s *apiService) DeleteByUUID(apiUUID uuid.UUID, tenantID int64) (*APIServiceDataResult, error) {
	// Get api and validate tenant ownership
	api, err := s.apiRepo.FindByUUIDAndTenantID(apiUUID, tenantID)
	if err != nil {
		return nil, err
	}
	if api == nil {
		return nil, errors.New("api not found or access denied")
	}

	// Check if default
	if api.IsDefault {
		return nil, errors.New("default api cannot be deleted")
	}

	err = s.apiRepo.DeleteByUUIDAndTenantID(apiUUID, tenantID)
	if err != nil {
		return nil, err
	}

	return toAPIServiceDataResult(api), nil
}

// Reponse builder
func toAPIServiceDataResult(api *model.API) *APIServiceDataResult {
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
		IsDefault:   api.IsDefault,
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
