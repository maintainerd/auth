package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/apperror"
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

type APIKeyAPIServiceDataResult struct {
	APIKeyAPIUUID uuid.UUID
	Api           APIServiceDataResult
	Permissions   []PermissionServiceDataResult
	CreatedAt     time.Time
}

// Paginated result for API Key APIs
type APIKeyAPIServicePaginatedResult struct {
	Data       []APIKeyAPIServiceDataResult
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
	GetAPIKeyAPIs(apiKeyUUID uuid.UUID, page, limit int, sortBy, sortOrder string) (*APIKeyAPIServicePaginatedResult, error)
	AddAPIKeyAPIs(apiKeyUUID uuid.UUID, apiUUIDs []uuid.UUID) error
	RemoveAPIKeyAPI(apiKeyUUID uuid.UUID, apiUUID uuid.UUID) error

	// API Key API Permission methods
	GetAPIKeyAPIPermissions(apiKeyUUID uuid.UUID, apiUUID uuid.UUID) ([]PermissionServiceDataResult, error)
	AddAPIKeyAPIPermissions(apiKeyUUID uuid.UUID, apiUUID uuid.UUID, permissionUUIDs []uuid.UUID) error
	RemoveAPIKeyAPIPermission(apiKeyUUID uuid.UUID, apiUUID uuid.UUID, permissionUUID uuid.UUID) error
}

type apiKeyService struct {
	db                   *gorm.DB
	apiKeyRepo           repository.APIKeyRepository
	apiKeyAPIRepo        repository.APIKeyAPIRepository
	apiKeyPermissionRepo repository.APIKeyPermissionRepository
	apiRepo              repository.APIRepository
	userRepo             repository.UserRepository
	permissionRepo       repository.PermissionRepository
}

func NewAPIKeyService(
	db *gorm.DB,
	apiKeyRepo repository.APIKeyRepository,
	apiKeyAPIRepo repository.APIKeyAPIRepository,
	apiKeyPermissionRepo repository.APIKeyPermissionRepository,
	apiRepo repository.APIRepository,
	userRepo repository.UserRepository,
	permissionRepo repository.PermissionRepository,
) APIKeyService {
	return &apiKeyService{
		db:                   db,
		apiKeyRepo:           apiKeyRepo,
		apiKeyAPIRepo:        apiKeyAPIRepo,
		apiKeyPermissionRepo: apiKeyPermissionRepo,
		apiRepo:              apiRepo,
		userRepo:             userRepo,
		permissionRepo:       permissionRepo,
	}
}

// generateAPIKey generates a secure API key and returns the key and its hash
func (s *apiKeyService) generateAPIKey() (string, string, string) {
	// Generate 32 random bytes
	bytes := make([]byte, 32)
	_, _ = rand.Read(bytes)

	// Create the API key with a prefix
	apiKey := "ak_" + hex.EncodeToString(bytes)

	// Create hash for storage
	hash := sha256.Sum256([]byte(apiKey))
	keyHash := hex.EncodeToString(hash[:])

	// Get prefix for identification (first 12 characters)
	keyPrefix := apiKey[:12]

	return apiKey, keyHash, keyPrefix
}

func (s *apiKeyService) Get(filter APIKeyServiceGetFilter, requestingUserUUID uuid.UUID) (*APIKeyServiceGetResult, error) {
	// Get requesting user
	requestingUser, err := s.userRepo.FindByUUID(requestingUserUUID)
	if err != nil {
		return nil, err
	}
	if requestingUser == nil {
		return nil, apperror.NewNotFoundWithReason("requesting user not found")
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
		apiKeyRepo := s.apiKeyRepo.WithTx(tx)

		// Find API key by UUID and tenant
		apiKey, err := apiKeyRepo.FindByUUIDAndTenantID(apiKeyUUID.String(), tenantID)
		if err != nil {
			return err
		}
		if apiKey == nil {
			return apperror.NewNotFoundWithReason("API key not found or access denied")
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
		return nil, apperror.NewNotFoundWithReason("API key not found or access denied")
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
		plainKey, keyHash, keyPrefix = s.generateAPIKey()

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
		var err error
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
		// Get the API key repository with transaction
		apiKeyRepo := s.apiKeyRepo.WithTx(tx)

		// Check if API key exists and belongs to tenant
		existing, err := apiKeyRepo.FindByUUIDAndTenantID(apiKeyUUID.String(), tenantID)
		if err != nil {
			return err
		}
		if existing == nil {
			return apperror.NewNotFoundWithReason("API key not found or access denied")
		}

		// Prepare update data
		updateData := make(map[string]any)
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
			return err
		}
		if apiKey == nil {
			return apperror.NewNotFoundWithReason("API key not found or access denied")
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
		existing, err := apiKeyRepo.FindByUUIDAndTenantID(apiKeyUUID.String(), tenantID)
		if err != nil {
			return err
		}
		if existing == nil {
			return apperror.NewNotFoundWithReason("API key not found or access denied")
		}

		// Update status using base repository method
		updateData := map[string]any{
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
func (s *apiKeyService) GetAPIKeyAPIs(apiKeyUUID uuid.UUID, page, limit int, sortBy, sortOrder string) (*APIKeyAPIServicePaginatedResult, error) {
	// Set defaults for pagination
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 10
	}

	// Get API key APIs with pagination
	result, err := s.apiKeyAPIRepo.FindByAPIKeyUUIDPaginated(apiKeyUUID, page, limit, sortBy, sortOrder)
	if err != nil {
		return nil, err
	}

	// Convert to service data result
	results := make([]APIKeyAPIServiceDataResult, len(result.Data))
	for i, apiKeyAPI := range result.Data {
		// Convert API
		apiDTO := APIServiceDataResult{
			APIUUID:     apiKeyAPI.API.APIUUID,
			Name:        apiKeyAPI.API.Name,
			DisplayName: apiKeyAPI.API.DisplayName,
			Description: apiKeyAPI.API.Description,
			APIType:     apiKeyAPI.API.APIType,
			Identifier:  apiKeyAPI.API.Identifier,
			Status:      apiKeyAPI.API.Status,
			IsSystem:    apiKeyAPI.API.IsSystem,
			CreatedAt:   apiKeyAPI.API.CreatedAt,
			UpdatedAt:   apiKeyAPI.API.UpdatedAt,
		}

		// Convert permissions
		permissions := make([]PermissionServiceDataResult, len(apiKeyAPI.Permissions))
		for j, perm := range apiKeyAPI.Permissions {
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

		results[i] = APIKeyAPIServiceDataResult{
			APIKeyAPIUUID: apiKeyAPI.APIKeyAPIUUID,
			Api:           apiDTO,
			Permissions:   permissions,
			CreatedAt:     apiKeyAPI.CreatedAt,
		}
	}

	return &APIKeyAPIServicePaginatedResult{
		Data:       results,
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}, nil
}

// Add APIs to API key
func (s *apiKeyService) AddAPIKeyAPIs(apiKeyUUID uuid.UUID, apiUUIDs []uuid.UUID) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		apiKeyRepo := s.apiKeyRepo.WithTx(tx)
		apiKeyAPIRepo := s.apiKeyAPIRepo.WithTx(tx)
		apiRepo := s.apiRepo.WithTx(tx)

		// Get API key
		apiKey, err := apiKeyRepo.FindByUUID(apiKeyUUID)
		if err != nil {
			return err
		}
		if apiKey == nil {
			return apperror.NewNotFoundWithReason("API key not found")
		}

		// Process each API UUID
		for _, apiUUID := range apiUUIDs {
			// Get API
			api, err := apiRepo.FindByUUID(apiUUID)
			if err != nil {
				return err
			}
			if api == nil {
				return apperror.NewNotFoundWithReason("API not found: " + apiUUID.String())
			}

			// Check if relationship already exists
			existing, err := apiKeyAPIRepo.FindByAPIKeyAndAPI(apiKey.APIKeyID, api.APIID)
			if err != nil {
				return err
			}
			if existing != nil {
				continue // Skip if already exists
			}

			// Create API key API relationship
			apiKeyAPI := &model.APIKeyAPI{
				APIKeyID: apiKey.APIKeyID,
				APIID:    api.APIID,
			}

			_, err = apiKeyAPIRepo.Create(apiKeyAPI)
			if err != nil {
				return err
			}
		}

		return nil
	})
}

// Remove API from API key
func (s *apiKeyService) RemoveAPIKeyAPI(apiKeyUUID uuid.UUID, apiUUID uuid.UUID) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		apiKeyAPIRepo := s.apiKeyAPIRepo.WithTx(tx)

		// First check if the relationship exists
		apiKeyAPI, err := apiKeyAPIRepo.FindByAPIKeyUUIDAndAPIUUID(apiKeyUUID, apiUUID)
		if err != nil {
			return err
		}
		if apiKeyAPI == nil {
			return apperror.NewNotFoundWithReason("API key API relationship not found")
		}

		// Remove the API key API relationship (CASCADE will handle permissions)
		return apiKeyAPIRepo.RemoveByAPIKeyUUIDAndAPIUUID(apiKeyUUID, apiUUID)
	})
}

// Get permissions for a specific API assigned to API key
func (s *apiKeyService) GetAPIKeyAPIPermissions(apiKeyUUID uuid.UUID, apiUUID uuid.UUID) ([]PermissionServiceDataResult, error) {
	// Get API key API relationship
	apiKeyAPI, err := s.apiKeyAPIRepo.FindByAPIKeyUUIDAndAPIUUID(apiKeyUUID, apiUUID)
	if err != nil {
		return nil, err
	}
	if apiKeyAPI == nil {
		return nil, apperror.NewNotFoundWithReason("API key API relationship not found")
	}

	// Get permissions for this API key API
	permissions, err := s.apiKeyPermissionRepo.FindByAPIKeyAPIID(apiKeyAPI.APIKeyAPIID)
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
func (s *apiKeyService) AddAPIKeyAPIPermissions(apiKeyUUID uuid.UUID, apiUUID uuid.UUID, permissionUUIDs []uuid.UUID) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		apiKeyAPIRepo := s.apiKeyAPIRepo.WithTx(tx)
		apiKeyPermissionRepo := s.apiKeyPermissionRepo.WithTx(tx)
		permissionRepo := s.permissionRepo.WithTx(tx)
		apiRepo := s.apiRepo.WithTx(tx)

		// Get API key API relationship
		apiKeyAPI, err := apiKeyAPIRepo.FindByAPIKeyUUIDAndAPIUUID(apiKeyUUID, apiUUID)
		if err != nil {
			return err
		}
		if apiKeyAPI == nil {
			return apperror.NewNotFoundWithReason("API key API relationship not found")
		}

		// Get the API to validate permissions belong to it
		api, err := apiRepo.FindByUUID(apiUUID)
		if err != nil {
			return err
		}
		if api == nil {
			return apperror.NewNotFoundWithReason("API not found")
		}

		// Process each permission UUID
		for _, permissionUUID := range permissionUUIDs {
			// Get permission
			permission, err := permissionRepo.FindByUUID(permissionUUID)
			if err != nil {
				return err
			}
			if permission == nil {
				return apperror.NewNotFoundWithReason("permission not found: " + permissionUUID.String())
			}

			// Validate that the permission belongs to the specified API
			if permission.APIID != api.APIID {
				return apperror.NewValidation("permission does not belong to the specified API: " + permissionUUID.String())
			}

			// Check if relationship already exists
			existing, err := apiKeyPermissionRepo.FindByAPIKeyAPIAndPermission(apiKeyAPI.APIKeyAPIID, permission.PermissionID)
			if err != nil {
				return err
			}
			if existing != nil {
				continue // Skip if already exists
			}

			// Create API key permission relationship
			apiKeyPermission := &model.APIKeyPermission{
				APIKeyAPIID:  apiKeyAPI.APIKeyAPIID,
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
func (s *apiKeyService) RemoveAPIKeyAPIPermission(apiKeyUUID uuid.UUID, apiUUID uuid.UUID, permissionUUID uuid.UUID) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		apiKeyAPIRepo := s.apiKeyAPIRepo.WithTx(tx)
		apiKeyPermissionRepo := s.apiKeyPermissionRepo.WithTx(tx)
		permissionRepo := s.permissionRepo.WithTx(tx)
		apiRepo := s.apiRepo.WithTx(tx)

		// Get API key API relationship
		apiKeyAPI, err := apiKeyAPIRepo.FindByAPIKeyUUIDAndAPIUUID(apiKeyUUID, apiUUID)
		if err != nil {
			return err
		}
		if apiKeyAPI == nil {
			return apperror.NewNotFoundWithReason("API key API relationship not found")
		}

		// Get the API to validate permission belongs to it
		api, err := apiRepo.FindByUUID(apiUUID)
		if err != nil {
			return err
		}
		if api == nil {
			return apperror.NewNotFoundWithReason("API not found")
		}

		// Get permission
		permission, err := permissionRepo.FindByUUID(permissionUUID)
		if err != nil {
			return err
		}
		if permission == nil {
			return apperror.NewNotFound("permission not found")
		}

		// Validate that the permission belongs to the specified API
		if permission.APIID != api.APIID {
			return apperror.NewValidation("permission does not belong to the specified API")
		}

		// Remove the permission
		return apiKeyPermissionRepo.RemoveByAPIKeyAPIAndPermission(apiKeyAPI.APIKeyAPIID, permission.PermissionID)
	})
}

func (s *apiKeyService) ValidateAPIKey(keyHash string) (*APIKeyServiceDataResult, error) {
	return nil, apperror.NewValidation("not implemented")
}
