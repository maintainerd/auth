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
	LastUsedAt  *time.Time
	UsageCount  int
	RateLimit   *int
	Status      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
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
	GetByUUID(apiKeyUUID uuid.UUID, requestingUserUUID uuid.UUID) (*APIKeyServiceDataResult, error)

	GetConfigByUUID(apiKeyUUID uuid.UUID) (datatypes.JSON, error)
	Create(name, description string, config datatypes.JSON, expiresAt *time.Time, rateLimit *int, status string) (*APIKeyServiceDataResult, string, error)
	Update(apiKeyUUID uuid.UUID, name, description *string, config datatypes.JSON, expiresAt *time.Time, rateLimit *int, status *string, updaterUserUUID uuid.UUID) (*APIKeyServiceDataResult, error)
	Delete(apiKeyUUID uuid.UUID, deleterUserUUID uuid.UUID) (*APIKeyServiceDataResult, error)
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
		LastUsedAt:  apiKey.LastUsedAt,
		UsageCount:  apiKey.UsageCount,
		RateLimit:   apiKey.RateLimit,
		Status:      apiKey.Status,
		CreatedAt:   apiKey.CreatedAt,
		UpdatedAt:   apiKey.UpdatedAt,
	}

	return result
}

func (s *apiKeyService) GetByUUID(apiKeyUUID uuid.UUID, requestingUserUUID uuid.UUID) (*APIKeyServiceDataResult, error) {
	var result *APIKeyServiceDataResult

	err := s.db.Transaction(func(tx *gorm.DB) error {
		// Create repository with transaction
		apiKeyRepo := repository.NewAPIKeyRepository(tx)

		// Find API key by UUID
		apiKey, err := apiKeyRepo.FindByUUID(apiKeyUUID)
		if err != nil {
			return err
		}
		if apiKey == nil {
			return errors.New("API key not found")
		}

		// Convert to service result
		result = &APIKeyServiceDataResult{
			APIKeyUUID:  apiKey.APIKeyUUID,
			Name:        apiKey.Name,
			Description: apiKey.Description,
			KeyPrefix:   apiKey.KeyPrefix,
			Config:      apiKey.Config,
			ExpiresAt:   apiKey.ExpiresAt,
			LastUsedAt:  apiKey.LastUsedAt,
			UsageCount:  apiKey.UsageCount,
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

func (s *apiKeyService) GetConfigByUUID(apiKeyUUID uuid.UUID) (datatypes.JSON, error) {
	apiKey, err := s.apiKeyRepo.FindByUUID(apiKeyUUID)
	if err != nil || apiKey == nil {
		return nil, errors.New("API key not found")
	}

	return apiKey.Config, nil
}

func (s *apiKeyService) Create(name, description string, config datatypes.JSON, expiresAt *time.Time, rateLimit *int, status string) (*APIKeyServiceDataResult, string, error) {
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
		LastUsedAt:  createdAPIKey.LastUsedAt,
		UsageCount:  createdAPIKey.UsageCount,
		RateLimit:   createdAPIKey.RateLimit,
		Status:      createdAPIKey.Status,
		CreatedAt:   createdAPIKey.CreatedAt,
		UpdatedAt:   createdAPIKey.UpdatedAt,
	}

	return result, plainKey, nil
}

func (s *apiKeyService) Update(apiKeyUUID uuid.UUID, name, description *string, config datatypes.JSON, expiresAt *time.Time, rateLimit *int, status *string, updaterUserUUID uuid.UUID) (*APIKeyServiceDataResult, error) {
	return nil, errors.New("not implemented")
}

func (s *apiKeyService) Delete(apiKeyUUID uuid.UUID, deleterUserUUID uuid.UUID) (*APIKeyServiceDataResult, error) {
	return nil, errors.New("not implemented")
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

		// Remove the API key API relationship
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

		// Get API key API relationship
		apiKeyApi, err := apiKeyApiRepo.FindByAPIKeyUUIDAndApiUUID(apiKeyUUID, apiUUID)
		if err != nil {
			return err
		}
		if apiKeyApi == nil {
			return errors.New("API key API relationship not found")
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

		// Get API key API relationship
		apiKeyApi, err := apiKeyApiRepo.FindByAPIKeyUUIDAndApiUUID(apiKeyUUID, apiUUID)
		if err != nil {
			return err
		}
		if apiKeyApi == nil {
			return errors.New("API key API relationship not found")
		}

		// Get permission
		permission, err := permissionRepo.FindByUUID(permissionUUID)
		if err != nil {
			return err
		}
		if permission == nil {
			return errors.New("permission not found")
		}

		// Remove the permission
		return apiKeyPermissionRepo.RemoveByAPIKeyApiAndPermission(apiKeyApi.APIKeyApiID, permission.PermissionID)
	})
}

func (s *apiKeyService) ValidateAPIKey(keyHash string) (*APIKeyServiceDataResult, error) {
	return nil, errors.New("not implemented")
}
