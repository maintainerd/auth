package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/apperror"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/repository"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
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
	Get(ctx context.Context, filter APIKeyServiceGetFilter, requestingUserUUID uuid.UUID) (*APIKeyServiceGetResult, error)
	GetByUUID(ctx context.Context, apiKeyUUID uuid.UUID, tenantID int64, requestingUserUUID uuid.UUID) (*APIKeyServiceDataResult, error)

	GetConfigByUUID(ctx context.Context, apiKeyUUID uuid.UUID, tenantID int64) (datatypes.JSON, error)
	Create(ctx context.Context, tenantID int64, name, description string, config datatypes.JSON, expiresAt *time.Time, rateLimit *int, status string) (*APIKeyServiceDataResult, string, error)
	Update(ctx context.Context, apiKeyUUID uuid.UUID, tenantID int64, name, description *string, config datatypes.JSON, expiresAt *time.Time, rateLimit *int, status *string, updaterUserUUID uuid.UUID) (*APIKeyServiceDataResult, error)
	SetStatusByUUID(ctx context.Context, apiKeyUUID uuid.UUID, tenantID int64, status string) (*APIKeyServiceDataResult, error)
	Delete(ctx context.Context, apiKeyUUID uuid.UUID, tenantID int64, deleterUserUUID uuid.UUID) (*APIKeyServiceDataResult, error)
	ValidateAPIKey(ctx context.Context, keyHash string) (*APIKeyServiceDataResult, error)

	// API Key API methods
	GetAPIKeyAPIs(ctx context.Context, apiKeyUUID uuid.UUID, page, limit int, sortBy, sortOrder string) (*APIKeyAPIServicePaginatedResult, error)
	AddAPIKeyAPIs(ctx context.Context, apiKeyUUID uuid.UUID, apiUUIDs []uuid.UUID) error
	RemoveAPIKeyAPI(ctx context.Context, apiKeyUUID uuid.UUID, apiUUID uuid.UUID) error

	// API Key API Permission methods
	GetAPIKeyAPIPermissions(ctx context.Context, apiKeyUUID uuid.UUID, apiUUID uuid.UUID) ([]PermissionServiceDataResult, error)
	AddAPIKeyAPIPermissions(ctx context.Context, apiKeyUUID uuid.UUID, apiUUID uuid.UUID, permissionUUIDs []uuid.UUID) error
	RemoveAPIKeyAPIPermission(ctx context.Context, apiKeyUUID uuid.UUID, apiUUID uuid.UUID, permissionUUID uuid.UUID) error
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

func (s *apiKeyService) Get(ctx context.Context, filter APIKeyServiceGetFilter, requestingUserUUID uuid.UUID) (*APIKeyServiceGetResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "api_key.list")
	defer span.End()
	span.SetAttributes(attribute.Int64("tenant.id", filter.TenantID))

	// Get requesting user
	requestingUser, err := s.userRepo.FindByUUID(requestingUserUUID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to fetch requesting user")
		return nil, err
	}
	if requestingUser == nil {
		span.SetStatus(codes.Error, "requesting user not found")
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
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to fetch api keys")
		return nil, err
	}

	// Convert to service result
	var serviceResults []APIKeyServiceDataResult
	for _, apiKey := range result.Data {
		serviceResult := s.toServiceDataResult(apiKey)
		serviceResults = append(serviceResults, serviceResult)
	}

	span.SetStatus(codes.Ok, "")
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

func (s *apiKeyService) GetByUUID(ctx context.Context, apiKeyUUID uuid.UUID, tenantID int64, requestingUserUUID uuid.UUID) (*APIKeyServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "api_key.get")
	defer span.End()
	span.SetAttributes(
		attribute.String("api_key.uuid", apiKeyUUID.String()),
		attribute.Int64("tenant.id", tenantID),
	)

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
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to fetch api key")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return result, nil
}

func (s *apiKeyService) GetConfigByUUID(ctx context.Context, apiKeyUUID uuid.UUID, tenantID int64) (datatypes.JSON, error) {
	_, span := otel.Tracer("service").Start(ctx, "api_key.getConfig")
	defer span.End()
	span.SetAttributes(
		attribute.String("api_key.uuid", apiKeyUUID.String()),
		attribute.Int64("tenant.id", tenantID),
	)

	apiKey, err := s.apiKeyRepo.FindByUUIDAndTenantID(apiKeyUUID.String(), tenantID)
	if err != nil || apiKey == nil {
		span.SetStatus(codes.Error, "api key not found or access denied")
		return nil, apperror.NewNotFoundWithReason("API key not found or access denied")
	}

	span.SetStatus(codes.Ok, "")
	return apiKey.Config, nil
}

func (s *apiKeyService) Create(ctx context.Context, tenantID int64, name, description string, config datatypes.JSON, expiresAt *time.Time, rateLimit *int, status string) (*APIKeyServiceDataResult, string, error) {
	_, span := otel.Tracer("service").Start(ctx, "api_key.create")
	defer span.End()
	span.SetAttributes(
		attribute.Int64("tenant.id", tenantID),
		attribute.String("api_key.name", name),
	)

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
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to create api key")
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

	span.SetStatus(codes.Ok, "")
	return result, plainKey, nil
}

func (s *apiKeyService) Update(ctx context.Context, apiKeyUUID uuid.UUID, tenantID int64, name, description *string, config datatypes.JSON, expiresAt *time.Time, rateLimit *int, status *string, updaterUserUUID uuid.UUID) (*APIKeyServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "api_key.update")
	defer span.End()
	span.SetAttributes(
		attribute.String("api_key.uuid", apiKeyUUID.String()),
		attribute.Int64("tenant.id", tenantID),
	)

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
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to update api key")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return result, nil
}

func (s *apiKeyService) Delete(ctx context.Context, apiKeyUUID uuid.UUID, tenantID int64, deleterUserUUID uuid.UUID) (*APIKeyServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "api_key.delete")
	defer span.End()
	span.SetAttributes(
		attribute.String("api_key.uuid", apiKeyUUID.String()),
		attribute.Int64("tenant.id", tenantID),
	)

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
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to delete api key")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return result, nil
}

func (s *apiKeyService) SetStatusByUUID(ctx context.Context, apiKeyUUID uuid.UUID, tenantID int64, status string) (*APIKeyServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "api_key.setStatus")
	defer span.End()
	span.SetAttributes(
		attribute.String("api_key.uuid", apiKeyUUID.String()),
		attribute.Int64("tenant.id", tenantID),
		attribute.String("api_key.status", status),
	)

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
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to update api key status")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return result, nil
}

// Get APIs assigned to API key with pagination
func (s *apiKeyService) GetAPIKeyAPIs(ctx context.Context, apiKeyUUID uuid.UUID, page, limit int, sortBy, sortOrder string) (*APIKeyAPIServicePaginatedResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "api_key.getAPIs")
	defer span.End()
	span.SetAttributes(attribute.String("api_key.uuid", apiKeyUUID.String()))
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
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to fetch api key apis")
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

	span.SetStatus(codes.Ok, "")
	return &APIKeyAPIServicePaginatedResult{
		Data:       results,
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}, nil
}

// Add APIs to API key
func (s *apiKeyService) AddAPIKeyAPIs(ctx context.Context, apiKeyUUID uuid.UUID, apiUUIDs []uuid.UUID) error {
	_, span := otel.Tracer("service").Start(ctx, "api_key.addAPIs")
	defer span.End()
	span.SetAttributes(attribute.String("api_key.uuid", apiKeyUUID.String()))

	err := s.db.Transaction(func(tx *gorm.DB) error {
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
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to add apis to api key")
		return err
	}
	span.SetStatus(codes.Ok, "")
	return nil
}

// Remove API from API key
func (s *apiKeyService) RemoveAPIKeyAPI(ctx context.Context, apiKeyUUID uuid.UUID, apiUUID uuid.UUID) error {
	_, span := otel.Tracer("service").Start(ctx, "api_key.removeAPI")
	defer span.End()
	span.SetAttributes(
		attribute.String("api_key.uuid", apiKeyUUID.String()),
		attribute.String("api.uuid", apiUUID.String()),
	)

	err := s.db.Transaction(func(tx *gorm.DB) error {
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
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to remove api from api key")
		return err
	}
	span.SetStatus(codes.Ok, "")
	return nil
}

// Get permissions for a specific API assigned to API key
func (s *apiKeyService) GetAPIKeyAPIPermissions(ctx context.Context, apiKeyUUID uuid.UUID, apiUUID uuid.UUID) ([]PermissionServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "api_key.getAPIPermissions")
	defer span.End()
	span.SetAttributes(
		attribute.String("api_key.uuid", apiKeyUUID.String()),
		attribute.String("api.uuid", apiUUID.String()),
	)

	// Get API key API relationship
	apiKeyAPI, err := s.apiKeyAPIRepo.FindByAPIKeyUUIDAndAPIUUID(apiKeyUUID, apiUUID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to fetch api key api relationship")
		return nil, err
	}
	if apiKeyAPI == nil {
		span.SetStatus(codes.Error, "api key api relationship not found")
		return nil, apperror.NewNotFoundWithReason("API key API relationship not found")
	}

	// Get permissions for this API key API
	permissions, err := s.apiKeyPermissionRepo.FindByAPIKeyAPIID(apiKeyAPI.APIKeyAPIID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to fetch permissions")
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

	span.SetStatus(codes.Ok, "")
	return results, nil
}

// Add permissions to a specific API for API key
func (s *apiKeyService) AddAPIKeyAPIPermissions(ctx context.Context, apiKeyUUID uuid.UUID, apiUUID uuid.UUID, permissionUUIDs []uuid.UUID) error {
	_, span := otel.Tracer("service").Start(ctx, "api_key.addAPIPermissions")
	defer span.End()
	span.SetAttributes(
		attribute.String("api_key.uuid", apiKeyUUID.String()),
		attribute.String("api.uuid", apiUUID.String()),
	)

	err := s.db.Transaction(func(tx *gorm.DB) error {
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
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to add permissions to api key api")
		return err
	}
	span.SetStatus(codes.Ok, "")
	return nil
}

// Remove permission from a specific API for API key
func (s *apiKeyService) RemoveAPIKeyAPIPermission(ctx context.Context, apiKeyUUID uuid.UUID, apiUUID uuid.UUID, permissionUUID uuid.UUID) error {
	_, span := otel.Tracer("service").Start(ctx, "api_key.removeAPIPermission")
	defer span.End()
	span.SetAttributes(
		attribute.String("api_key.uuid", apiKeyUUID.String()),
		attribute.String("api.uuid", apiUUID.String()),
		attribute.String("permission.uuid", permissionUUID.String()),
	)

	err := s.db.Transaction(func(tx *gorm.DB) error {
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
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to remove permission from api key api")
		return err
	}
	span.SetStatus(codes.Ok, "")
	return nil
}

func (s *apiKeyService) ValidateAPIKey(ctx context.Context, keyHash string) (*APIKeyServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "api_key.validate")
	defer span.End()
	span.SetStatus(codes.Error, "not implemented")
	return nil, apperror.NewValidation("not implemented")
}
