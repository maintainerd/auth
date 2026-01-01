package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/repository"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type APIKeyServiceDataResult struct {
	APIKeyUUID  uuid.UUID
	Name        string
	Description string
	KeyPrefix   string
	Config      datatypes.JSON
	ExpiresAt   *time.Time

	RateLimit *int
	Status    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type APIKeyApiServiceDataResult struct {
	APIKeyApiUUID uuid.UUID
	Api           APIServiceDataResult
	Permissions   []PermissionServiceDataResult
	CreatedAt     time.Time
}

// Paginated result for API Key APIs
type APIKeyApiServicePaginatedResult struct {
	Data       []APIKeyApiServiceDataResult
	Total      int64
	Page       int
	Limit      int
	TotalPages int
}

type APIKeyServiceGetFilter struct {
	TenantID    int64
	Name        *string
	Description *string
	Status      *string
	Page        int
	Limit       int
	SortBy      string
	SortOrder   string
}

type APIKeyServiceGetResult struct {
	Data       []APIKeyServiceDataResult
	Total      int64
	Page       int
	Limit      int
	TotalPages int
}

type APIKeyService interface {
	Get(filter APIKeyServiceGetFilter, requestingUserUUID uuid.UUID) (*APIKeyServiceGetResult, error)
	GetByUUID(apiKeyUUID uuid.UUID, tenantID int64, requestingUserUUID uuid.UUID) (*APIKeyServiceDataResult, error)

	GetConfigByUUID(apiKeyUUID uuid.UUID, tenantID int64) (datatypes.JSON, error)
	Create(tenantID int64, name, description string, config datatypes.JSON, expiresAt *time.Time, rateLimit *int, status string) (*APIKeyServiceDataResult, string, error)
	Update(apiKeyUUID uuid.UUID, tenantID int64, name, description *string, config datatypes.JSON, expiresAt *time.Time, rateLimit *int, status *string, updaterUserUUID uuid.UUID) (*APIKeyServiceDataResult, error)
	SetStatusByUUID(apiKeyUUID uuid.UUID, tenantID int64, status string) (*APIKeyServiceDataResult, error)
	Delete(apiKeyUUID uuid.UUID, tenantID int64, deleterUserUUID uuid.UUID) (*APIKeyServiceDataResult, error)
	ValidateAPIKey(keyHash string) (*APIKeyServiceDataResult, error)

	// API Key API methods
	GetAPIKeyApis(apiKeyUUID uuid.UUID, page, limit int, sortBy, sortOrder string) (*APIKeyApiServicePaginatedResult, error)
	AddAPIKeyApis(apiKeyUUID uuid.UUID, apiUUIDs []uuid.UUID) error
	RemoveAPIKeyApi(apiKeyUUID uuid.UUID, apiUUID uuid.UUID) error

	// API Key API Permission methods
	GetAPIKeyApiPermissions(apiKeyUUID uuid.UUID, apiUUID uuid.UUID) ([]PermissionServiceDataResult, error)
	AddAPIKeyApiPermissions(apiKeyUUID uuid.UUID, apiUUID uuid.UUID, permissionUUIDs []uuid.UUID) error
	RemoveAPIKeyApiPermission(apiKeyUUID uuid.UUID, apiUUID uuid.UUID, permissionUUID uuid.UUID) error
}

type apiKeyService struct {
	db                   *gorm.DB
	apiKeyRepo           repository.APIKeyRepository
	apiKeyApiRepo        repository.APIKeyApiRepository
	apiKeyPermissionRepo repository.APIKeyPermissionRepository
	apiRepo              repository.APIRepository
	userRepo             repository.UserRepository
	permissionRepo       repository.PermissionRepository
}

func NewAPIKeyService(
	db *gorm.DB,
	apiKeyRepo repository.APIKeyRepository,
	apiKeyApiRepo repository.APIKeyApiRepository,
	apiKeyPermissionRepo repository.APIKeyPermissionRepository,
	apiRepo repository.APIRepository,
	userRepo repository.UserRepository,
	permissionRepo repository.PermissionRepository,
) APIKeyService {
	return &apiKeyService{
		db:                   db,
		apiKeyRepo:           apiKeyRepo,
		apiKeyApiRepo:        apiKeyApiRepo,
		apiKeyPermissionRepo: apiKeyPermissionRepo,
		apiRepo:              apiRepo,
		userRepo:             userRepo,
		permissionRepo:       permissionRepo,
	}
}

// generateAPIKey generates a secure API key and returns the key and its hash
func (s *apiKeyService) generateAPIKey() (string, string, string, error) {
	// Generate 32 random bytes
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", "", "", err
	}

	// Create the API key with a prefix
	apiKey := "ak_" + hex.EncodeToString(bytes)

	// Create hash for storage
	hash := sha256.Sum256([]byte(apiKey))
	keyHash := hex.EncodeToString(hash[:])

	// Get prefix for identification (first 12 characters)
	keyPrefix := apiKey[:12]

	return apiKey, keyHash, keyPrefix, nil
}

func (s *apiKeyService) Get(filter APIKeyServiceGetFilter, requestingUserUUID uuid.UUID) (*APIKeyServiceGetResult, error) {
	// Get requesting user
	requestingUser, err := s.userRepo.FindByUUID(requestingUserUUID)
	if err != nil {
		return nil, err
	}
	if requestingUser == nil {
		return nil, errors.New("requesting user not found")
	}

	// Build repository filter
	repoFilter := repository.APIKeyRepositoryGetFilter{
		TenantID:    filter.TenantID,
		Name:        filter.Name,
		Description: filter.Description,
		Status:      filter.Status,
		Page:        filter.Page,
		Limit:       filter.Limit,
		SortBy:      filter.SortBy,
		SortOrder:   filter.SortOrder,
	}

	result, err := s.apiKeyRepo.FindPaginated(repoFilter)
	if err != nil {
		return nil, err
	}

	// Convert to service result
	var serviceResults []APIKeyServiceDataResult
	for _, apiKey := range result.Data {
		serviceResult := s.toServiceDataResult(apiKey)
		serviceResults = append(serviceResults, serviceResult)
	}

	return &APIKeyServiceGetResult{
		Data:       serviceResults,
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}, nil
}

// toServiceDataResult converts model to service data result
func (s *apiKeyService) toServiceDataResult(apiKey model.APIKey) APIKeyServiceDataResult {
	result := APIKeyServiceDataResult{
		APIKeyUUID:  apiKey.APIKeyUUID,
		Name:        apiKey.Name,
		Description: apiKey.Description,
		KeyPrefix:   apiKey.KeyPrefix,
		Config:      apiKey.Config,
		ExpiresAt:   apiKey.ExpiresAt,

		RateLimit: apiKey.RateLimit,
		Status:    apiKey.Status,
		CreatedAt: apiKey.CreatedAt,
		UpdatedAt: apiKey.UpdatedAt,
	}

	return result
}

func (s *apiKeyService) GetByUUID(apiKeyUUID uuid.UUID, tenantID int64, requestingUserUUID uuid.UUID) (*APIKeyServiceDataResult, error) {
	var result *APIKeyServiceDataResult

	err := s.db.Transaction(func(tx *gorm.DB) error {
		// Create repository with transaction
		apiKeyRepo := repository.NewAPIKeyRepository(tx)

		// Find API key by UUID and tenant
		apiKey, err := apiKeyRepo.FindByUUIDAndTenantID(apiKeyUUID.String(), tenantID)
		if err != nil {
			return err
		}
		if apiKey == nil {
			return errors.New("API key not found or access denied")
		}

		// Convert to service result
		result = &APIKeyServiceDataResult{
			APIKeyUUID:  apiKey.APIKeyUUID,
			Name:        apiKey.Name,
			Description: apiKey.Description,
			KeyPrefix:   apiKey.KeyPrefix,
			Config:      apiKey.Config,
			ExpiresAt:   apiKey.ExpiresAt,
			RateLimit:   apiKey.RateLimit,
			Status:      apiKey.Status,
			CreatedAt:   apiKey.CreatedAt,
			UpdatedAt:   apiKey.UpdatedAt,
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}

func (s *apiKeyService) GetConfigByUUID(apiKeyUUID uuid.UUID, tenantID int64) (datatypes.JSON, error) {
	apiKey, err := s.apiKeyRepo.FindByUUIDAndTenantID(apiKeyUUID.String(), tenantID)
	if err != nil || apiKey == nil {
		return nil, errors.New("API key not found or access denied")
	}

	return apiKey.Config, nil
}

func (s *apiKeyService) Create(tenantID int64, name, description string, config datatypes.JSON, expiresAt *time.Time, rateLimit *int, status string) (*APIKeyServiceDataResult, string, error) {
	var createdAPIKey *model.APIKey
	var plainKey string

	err := s.db.Transaction(func(tx *gorm.DB) error {
		apiKeyRepo := s.apiKeyRepo.WithTx(tx)

		// Generate API key
		var keyHash, keyPrefix string
		var err error
		plainKey, keyHash, keyPrefix, err = s.generateAPIKey()
		if err != nil {
			return err
		}

		// Create API key model
		apiKey := &model.APIKey{
			TenantID:    tenantID,
			Name:        name,
			Description: description,
			KeyHash:     keyHash,
			KeyPrefix:   keyPrefix,
			Config:      config,
			ExpiresAt:   expiresAt,
			RateLimit:   rateLimit,
			Status:      status,
		}

		// Save to database
		createdAPIKey, err = apiKeyRepo.Create(apiKey)
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, "", err
	}

	// Convert to service result
	result := &APIKeyServiceDataResult{
		APIKeyUUID:  createdAPIKey.APIKeyUUID,
		Name:        createdAPIKey.Name,
		Description: createdAPIKey.Description,
		KeyPrefix:   createdAPIKey.KeyPrefix,
		Config:      createdAPIKey.Config,
		ExpiresAt:   createdAPIKey.ExpiresAt,
		RateLimit:   createdAPIKey.RateLimit,
		Status:      createdAPIKey.Status,
		CreatedAt:   createdAPIKey.CreatedAt,
		UpdatedAt:   createdAPIKey.UpdatedAt,
	}

	return result, plainKey, nil
}

func (s *apiKeyService) Update(apiKeyUUID uuid.UUID, tenantID int64, name, description *string, config datatypes.JSON, expiresAt *time.Time, rateLimit *int, status *string, updaterUserUUID uuid.UUID) (*APIKeyServiceDataResult, error) {
	var result *APIKeyServiceDataResult

	err := s.db.Transaction(func(tx *gorm.DB) error {
		// Get the API key repository
		apiKeyRepo := s.apiKeyRepo

		// Check if API key exists and belongs to tenant
		_, err := apiKeyRepo.FindByUUIDAndTenantID(apiKeyUUID.String(), tenantID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("API key not found or access denied")
			}
			return err
		}

		// Prepare update data
		updateData := make(map[string]interface{})
		if name != nil {
			updateData["name"] = *name
		}
		if description != nil {
			updateData["description"] = *description
		}
		if config != nil {
			updateData["config"] = config
		}
		if expiresAt != nil {
			updateData["expires_at"] = expiresAt
		}
		if rateLimit != nil {
			updateData["rate_limit"] = rateLimit
		}
		if status != nil {
			updateData["status"] = *status
		}

		// Save the updated API key
		updatedAPIKey, err := apiKeyRepo.UpdateByUUID(apiKeyUUID, updateData)
		if err != nil {
			return err
		}

		// Convert to service result
		result = &APIKeyServiceDataResult{
			APIKeyUUID:  updatedAPIKey.APIKeyUUID,
			Name:        updatedAPIKey.Name,
			Description: updatedAPIKey.Description,
			KeyPrefix:   updatedAPIKey.KeyPrefix,
			Config:      updatedAPIKey.Config,
			ExpiresAt:   updatedAPIKey.ExpiresAt,
			RateLimit:   updatedAPIKey.RateLimit,
			Status:      updatedAPIKey.Status,
			CreatedAt:   updatedAPIKey.CreatedAt,
			UpdatedAt:   updatedAPIKey.UpdatedAt,
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}

func (s *apiKeyService) Delete(apiKeyUUID uuid.UUID, tenantID int64, deleterUserUUID uuid.UUID) (*APIKeyServiceDataResult, error) {
	var result *APIKeyServiceDataResult
	err := s.db.Transaction(func(tx *gorm.DB) error {
		apiKeyRepo := s.apiKeyRepo.WithTx(tx)

		// Get API key before deletion to return its data
		apiKey, err := apiKeyRepo.FindByUUIDAndTenantID(apiKeyUUID.String(), tenantID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("API key not found or access denied")
			}
			return err
		}

		// Map to result before deletion
		result = &APIKeyServiceDataResult{
			APIKeyUUID:  apiKey.APIKeyUUID,
			Name:        apiKey.Name,
			Description: apiKey.Description,
			KeyPrefix:   apiKey.KeyPrefix,
			Config:      apiKey.Config,
			ExpiresAt:   apiKey.ExpiresAt,
			RateLimit:   apiKey.RateLimit,
			Status:      apiKey.Status,
			CreatedAt:   apiKey.CreatedAt,
			UpdatedAt:   apiKey.UpdatedAt,
		}

		// Delete the API key (CASCADE will handle related records)
		err = apiKeyRepo.DeleteByUUID(apiKeyUUID)
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}

func (s *apiKeyService) SetStatusByUUID(apiKeyUUID uuid.UUID, tenantID int64, status string) (*APIKeyServiceDataResult, error) {
	var result *APIKeyServiceDataResult
	err := s.db.Transaction(func(tx *gorm.DB) error {
		apiKeyRepo := s.apiKeyRepo.WithTx(tx)

		// Check if API key exists and belongs to tenant
		_, err := apiKeyRepo.FindByUUIDAndTenantID(apiKeyUUID.String(), tenantID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("API key not found or access denied")
			}
			return err
		}

		// Update status using base repository method
		updateData := map[string]interface{}{
			"status": status,
		}
		updatedAPIKey, err := apiKeyRepo.UpdateByUUID(apiKeyUUID, updateData)
		if err != nil {
			return err
		}

		// Map to result
		result = &APIKeyServiceDataResult{
			APIKeyUUID:  updatedAPIKey.APIKeyUUID,
			Name:        updatedAPIKey.Name,
			Description: updatedAPIKey.Description,
			KeyPrefix:   updatedAPIKey.KeyPrefix,
			Config:      updatedAPIKey.Config,
			ExpiresAt:   updatedAPIKey.ExpiresAt,
			RateLimit:   updatedAPIKey.RateLimit,
			Status:      updatedAPIKey.Status,
			CreatedAt:   updatedAPIKey.CreatedAt,
			UpdatedAt:   updatedAPIKey.UpdatedAt,
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}

// Get APIs assigned to API key with pagination
func (s *apiKeyService) GetAPIKeyApis(apiKeyUUID uuid.UUID, page, limit int, sortBy, sortOrder string) (*APIKeyApiServicePaginatedResult, error) {
	// Set defaults for pagination
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 10
	}

	// Get API key APIs with pagination
	result, err := s.apiKeyApiRepo.FindByAPIKeyUUIDPaginated(apiKeyUUID, page, limit, sortBy, sortOrder)
	if err != nil {
		return nil, err
	}

	// Convert to service data result
	results := make([]APIKeyApiServiceDataResult, len(result.Data))
	for i, apiKeyApi := range result.Data {
		// Convert API
		apiDto := APIServiceDataResult{
			APIUUID:     apiKeyApi.API.APIUUID,
			Name:        apiKeyApi.API.Name,
			DisplayName: apiKeyApi.API.DisplayName,
			Description: apiKeyApi.API.Description,
			APIType:     apiKeyApi.API.APIType,
			Identifier:  apiKeyApi.API.Identifier,
			Status:      apiKeyApi.API.Status,
			IsDefault:   apiKeyApi.API.IsDefault,
			IsSystem:    apiKeyApi.API.IsSystem,
			CreatedAt:   apiKeyApi.API.CreatedAt,
			UpdatedAt:   apiKeyApi.API.UpdatedAt,
		}

		// Convert permissions
		permissions := make([]PermissionServiceDataResult, len(apiKeyApi.Permissions))
		for j, perm := range apiKeyApi.Permissions {
			permissions[j] = PermissionServiceDataResult{
				PermissionUUID: perm.Permission.PermissionUUID,
				Name:           perm.Permission.Name,
				Description:    perm.Permission.Description,
				Status:         perm.Permission.Status,
				IsDefault:      perm.Permission.IsDefault,
				IsSystem:       perm.Permission.IsSystem,
				CreatedAt:      perm.Permission.CreatedAt,
				UpdatedAt:      perm.Permission.UpdatedAt,
			}
		}

		results[i] = APIKeyApiServiceDataResult{
			APIKeyApiUUID: apiKeyApi.APIKeyApiUUID,
			Api:           apiDto,
			Permissions:   permissions,
			CreatedAt:     apiKeyApi.CreatedAt,
		}
	}

	return &APIKeyApiServicePaginatedResult{
		Data:       results,
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}, nil
}

// Add APIs to API key
func (s *apiKeyService) AddAPIKeyApis(apiKeyUUID uuid.UUID, apiUUIDs []uuid.UUID) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		apiKeyRepo := s.apiKeyRepo.WithTx(tx)
		apiKeyApiRepo := s.apiKeyApiRepo.WithTx(tx)
		apiRepo := s.apiRepo.WithTx(tx)

		// Get API key
		apiKey, err := apiKeyRepo.FindByUUID(apiKeyUUID)
		if err != nil {
			return err
		}
		if apiKey == nil {
			return errors.New("API key not found")
		}

		// Process each API UUID
		for _, apiUUID := range apiUUIDs {
			// Get API
			api, err := apiRepo.FindByUUID(apiUUID)
			if err != nil {
				return err
			}
			if api == nil {
				return errors.New("API not found: " + apiUUID.String())
			}

			// Check if relationship already exists
			existing, err := apiKeyApiRepo.FindByAPIKeyAndApi(apiKey.APIKeyID, api.APIID)
			if err != nil {
				return err
			}
			if existing != nil {
				continue // Skip if already exists
			}

			// Create API key API relationship
			apiKeyApi := &model.APIKeyApi{
				APIKeyID: apiKey.APIKeyID,
				APIID:    api.APIID,
			}

			_, err = apiKeyApiRepo.Create(apiKeyApi)
			if err != nil {
				return err
			}
		}

		return nil
	})
}

// Remove API from API key
func (s *apiKeyService) RemoveAPIKeyApi(apiKeyUUID uuid.UUID, apiUUID uuid.UUID) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		apiKeyApiRepo := s.apiKeyApiRepo.WithTx(tx)

		// First check if the relationship exists
		apiKeyApi, err := apiKeyApiRepo.FindByAPIKeyUUIDAndApiUUID(apiKeyUUID, apiUUID)
		if err != nil {
			return err
		}
		if apiKeyApi == nil {
			return errors.New("API key API relationship not found")
		}

		// Remove the API key API relationship (CASCADE will handle permissions)
		return apiKeyApiRepo.RemoveByAPIKeyUUIDAndApiUUID(apiKeyUUID, apiUUID)
	})
}

// Get permissions for a specific API assigned to API key
func (s *apiKeyService) GetAPIKeyApiPermissions(apiKeyUUID uuid.UUID, apiUUID uuid.UUID) ([]PermissionServiceDataResult, error) {
	// Get API key API relationship
	apiKeyApi, err := s.apiKeyApiRepo.FindByAPIKeyUUIDAndApiUUID(apiKeyUUID, apiUUID)
	if err != nil {
		return nil, err
	}
	if apiKeyApi == nil {
		return nil, errors.New("API key API relationship not found")
	}

	// Get permissions for this API key API
	permissions, err := s.apiKeyPermissionRepo.FindByAPIKeyApiID(apiKeyApi.APIKeyApiID)
	if err != nil {
		return nil, err
	}

	// Convert to service data result
	results := make([]PermissionServiceDataResult, len(permissions))
	for i, perm := range permissions {
		results[i] = PermissionServiceDataResult{
			PermissionUUID: perm.Permission.PermissionUUID,
			Name:           perm.Permission.Name,
			Description:    perm.Permission.Description,
			Status:         perm.Permission.Status,
			IsDefault:      perm.Permission.IsDefault,
			IsSystem:       perm.Permission.IsSystem,
			CreatedAt:      perm.Permission.CreatedAt,
			UpdatedAt:      perm.Permission.UpdatedAt,
		}
	}

	return results, nil
}

// Add permissions to a specific API for API key
func (s *apiKeyService) AddAPIKeyApiPermissions(apiKeyUUID uuid.UUID, apiUUID uuid.UUID, permissionUUIDs []uuid.UUID) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		apiKeyApiRepo := s.apiKeyApiRepo.WithTx(tx)
		apiKeyPermissionRepo := s.apiKeyPermissionRepo.WithTx(tx)
		permissionRepo := s.permissionRepo.WithTx(tx)
		apiRepo := s.apiRepo.WithTx(tx)

		// Get API key API relationship
		apiKeyApi, err := apiKeyApiRepo.FindByAPIKeyUUIDAndApiUUID(apiKeyUUID, apiUUID)
		if err != nil {
			return err
		}
		if apiKeyApi == nil {
			return errors.New("API key API relationship not found")
		}

		// Get the API to validate permissions belong to it
		api, err := apiRepo.FindByUUID(apiUUID)
		if err != nil {
			return err
		}
		if api == nil {
			return errors.New("API not found")
		}

		// Process each permission UUID
		for _, permissionUUID := range permissionUUIDs {
			// Get permission
			permission, err := permissionRepo.FindByUUID(permissionUUID)
			if err != nil {
				return err
			}
			if permission == nil {
				return errors.New("permission not found: " + permissionUUID.String())
			}

			// Validate that the permission belongs to the specified API
			if permission.APIID != api.APIID {
				return errors.New("permission does not belong to the specified API: " + permissionUUID.String())
			}

			// Check if relationship already exists
			existing, err := apiKeyPermissionRepo.FindByAPIKeyApiAndPermission(apiKeyApi.APIKeyApiID, permission.PermissionID)
			if err != nil {
				return err
			}
			if existing != nil {
				continue // Skip if already exists
			}

			// Create API key permission relationship
			apiKeyPermission := &model.APIKeyPermission{
				APIKeyApiID:  apiKeyApi.APIKeyApiID,
				PermissionID: permission.PermissionID,
			}

			_, err = apiKeyPermissionRepo.Create(apiKeyPermission)
			if err != nil {
				return err
			}
		}

		return nil
	})
}

// Remove permission from a specific API for API key
func (s *apiKeyService) RemoveAPIKeyApiPermission(apiKeyUUID uuid.UUID, apiUUID uuid.UUID, permissionUUID uuid.UUID) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		apiKeyApiRepo := s.apiKeyApiRepo.WithTx(tx)
		apiKeyPermissionRepo := s.apiKeyPermissionRepo.WithTx(tx)
		permissionRepo := s.permissionRepo.WithTx(tx)
		apiRepo := s.apiRepo.WithTx(tx)

		// Get API key API relationship
		apiKeyApi, err := apiKeyApiRepo.FindByAPIKeyUUIDAndApiUUID(apiKeyUUID, apiUUID)
		if err != nil {
			return err
		}
		if apiKeyApi == nil {
			return errors.New("API key API relationship not found")
		}

		// Get the API to validate permission belongs to it
		api, err := apiRepo.FindByUUID(apiUUID)
		if err != nil {
			return err
		}
		if api == nil {
			return errors.New("API not found")
		}

		// Get permission
		permission, err := permissionRepo.FindByUUID(permissionUUID)
		if err != nil {
			return err
		}
		if permission == nil {
			return errors.New("permission not found")
		}

		// Validate that the permission belongs to the specified API
		if permission.APIID != api.APIID {
			return errors.New("permission does not belong to the specified API")
		}

		// Remove the permission
		return apiKeyPermissionRepo.RemoveByAPIKeyApiAndPermission(apiKeyApi.APIKeyApiID, permission.PermissionID)
	})
}

func (s *apiKeyService) ValidateAPIKey(keyHash string) (*APIKeyServiceDataResult, error) {
	return nil, errors.New("not implemented")
}
