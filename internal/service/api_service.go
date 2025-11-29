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
	IsActive    bool
	IsDefault   bool
	IsSystem    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type APIServiceGetFilter struct {
	Name        *string
	DisplayName *string
	APIType     *string
	Identifier  *string
	ServiceID   *int64
	IsActive    *bool
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
	GetByUUID(apiUUID uuid.UUID) (*APIServiceDataResult, error)
	Create(name string, displayName string, description string, apiType string, isActive bool, isDefault bool, isSystem bool, serviceUUID string) (*APIServiceDataResult, error)
	Update(apiUUID uuid.UUID, name string, displayName string, description string, apiType string, isActive bool, isDefault bool) (*APIServiceDataResult, error)
	SetActiveStatusByUUID(apiUUID uuid.UUID) (*APIServiceDataResult, error)
	DeleteByUUID(apiUUID uuid.UUID) (*APIServiceDataResult, error)
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
		Name:        filter.Name,
		DisplayName: filter.DisplayName,
		APIType:     filter.APIType,
		Identifier:  filter.Identifier,
		ServiceID:   filter.ServiceID,
		IsActive:    filter.IsActive,
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

func (s *apiService) GetByUUID(apiUUID uuid.UUID) (*APIServiceDataResult, error) {
	api, err := s.apiRepo.FindByUUID(apiUUID, "Service")
	if err != nil || api == nil {
		return nil, errors.New("api not found")
	}

	return toAPIServiceDataResult(api), nil
}

func (s *apiService) Create(name string, displayName string, description string, apiType string, isActive bool, isDefault bool, isSystem bool, serviceUUID string) (*APIServiceDataResult, error) {
	var createdAPI *model.API

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txAPIRepo := s.apiRepo.WithTx(tx)
		txServiceRepo := s.serviceRepo.WithTx(tx)

		// Check if api already exists
		existingAPI, err := txAPIRepo.FindByName(name)
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
			IsActive:    isActive,
			IsDefault:   isDefault,
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

func (s *apiService) Update(apiUUID uuid.UUID, name string, displayName string, description string, apiType string, isActive bool, isDefault bool) (*APIServiceDataResult, error) {
	var updatedAPI *model.API

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txAPIRepo := s.apiRepo.WithTx(tx)

		// Get api
		api, err := txAPIRepo.FindByUUID(apiUUID, "Service")
		if err != nil || api == nil {
			return errors.New("api not found")
		}

		// Check if default
		if api.IsDefault {
			return errors.New("default api cannot be updated")
		}

		// Check if api already exist
		if api.Name != name {
			existingAPI, err := txAPIRepo.FindByName(name)
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
		api.IsActive = isActive
		api.IsDefault = isDefault

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

func (s *apiService) SetActiveStatusByUUID(apiUUID uuid.UUID) (*APIServiceDataResult, error) {
	var updatedAPI *model.API

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txAPIRepo := s.apiRepo.WithTx(tx)

		// Get api
		api, err := txAPIRepo.FindByUUID(apiUUID, "Service")
		if err != nil || api == nil {
			return errors.New("api not found")
		}

		// Check if default
		if api.IsDefault {
			return errors.New("default api cannot be updated")
		}

		api.IsActive = !api.IsActive

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

func (s *apiService) DeleteByUUID(apiUUID uuid.UUID) (*APIServiceDataResult, error) {
	// Get api
	api, err := s.apiRepo.FindByUUID(apiUUID, "Service")
	if err != nil || api == nil {
		return nil, errors.New("api not found")
	}

	// Check if default
	if api.IsDefault {
		return nil, errors.New("default api cannot be deleted")
	}

	err = s.apiRepo.DeleteByUUID(apiUUID)
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
		IsActive:    api.IsActive,
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
			IsDefault:   api.Service.IsDefault,
			IsSystem:    api.Service.IsSystem,
			Status:      api.Service.Status,
			IsPublic:    api.Service.IsPublic,
			APICount:    0, // Not needed in API context
			PolicyCount: 0, // Not needed in API context
			CreatedAt:   api.Service.CreatedAt,
			UpdatedAt:   api.Service.UpdatedAt,
		}
	}

	return result
}
