package service

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/repository"
	"github.com/maintainerd/auth/internal/util"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type AuthClientSecretServiceDataResult struct {
	ClientID     string
	ClientSecret *string
}

type AuthClientURIServiceDataResult struct {
	AuthClientURIUUID uuid.UUID
	URI               string
	Type              string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type AuthClientServiceDataResult struct {
	AuthClientUUID   uuid.UUID
	Name             string
	DisplayName      string
	ClientType       string
	Domain           *string
	AuthClientURIs   *[]AuthClientURIServiceDataResult
	IdentityProvider *IdentityProviderServiceDataResult
	Permissions      *[]PermissionServiceDataResult
	Status           string
	IsDefault        bool
	IsSystem         bool
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type AuthClientApiServiceDataResult struct {
	AuthClientApiUUID uuid.UUID
	Api               APIServiceDataResult
	Permissions       []PermissionServiceDataResult
	CreatedAt         time.Time
}

type AuthClientServiceGetFilter struct {
	TenantID             int64
	Name                 *string
	DisplayName          *string
	ClientType           []string
	IdentityProviderUUID *string
	Status               []string
	IsDefault            *bool
	IsSystem             *bool
	Page                 int
	Limit                int
	SortBy               string
	SortOrder            string
}

type AuthClientServiceGetResult struct {
	Data       []AuthClientServiceDataResult
	Total      int64
	Page       int
	Limit      int
	TotalPages int
}

type AuthClientService interface {
	Get(filter AuthClientServiceGetFilter) (*AuthClientServiceGetResult, error)
	GetByUUID(authClientUUID uuid.UUID, tenantID int64) (*AuthClientServiceDataResult, error)
	GetSecretByUUID(authClientUUID uuid.UUID, tenantID int64) (*AuthClientSecretServiceDataResult, error)
	GetConfigByUUID(authClientUUID uuid.UUID, tenantID int64) (datatypes.JSON, error)
	Create(tenantID int64, name string, displayName string, clientType string, domain string, config datatypes.JSON, status string, isDefault bool, identityProviderUUID string, actorUserUUID uuid.UUID) (*AuthClientServiceDataResult, error)
	Update(authClientUUID uuid.UUID, tenantID int64, name string, displayName string, clientType string, domain string, config datatypes.JSON, status string, isDefault bool, actorUserUUID uuid.UUID) (*AuthClientServiceDataResult, error)
	SetStatusByUUID(authClientUUID uuid.UUID, tenantID int64, status string, actorUserUUID uuid.UUID) (*AuthClientServiceDataResult, error)
	DeleteByUUID(authClientUUID uuid.UUID, tenantID int64, actorUserUUID uuid.UUID) (*AuthClientServiceDataResult, error)
	CreateURI(authClientUUID uuid.UUID, tenantID int64, uri string, uriType string, actorUserUUID uuid.UUID) (*AuthClientServiceDataResult, error)
	UpdateURI(authClientUUID uuid.UUID, tenantID int64, authClientURIUUID uuid.UUID, uri string, uriType string, actorUserUUID uuid.UUID) (*AuthClientServiceDataResult, error)
	DeleteURI(authClientUUID uuid.UUID, tenantID int64, authClientURIUUID uuid.UUID, actorUserUUID uuid.UUID) (*AuthClientServiceDataResult, error)

	// Auth Client API methods
	GetAuthClientApis(tenantID int64, authClientUUID uuid.UUID) ([]AuthClientApiServiceDataResult, error)
	AddAuthClientApis(tenantID int64, authClientUUID uuid.UUID, apiUUIDs []uuid.UUID) error
	RemoveAuthClientApi(tenantID int64, authClientUUID uuid.UUID, apiUUID uuid.UUID) error

	// Auth Client API Permission methods
	GetAuthClientApiPermissions(tenantID int64, authClientUUID uuid.UUID, apiUUID uuid.UUID) ([]PermissionServiceDataResult, error)
	AddAuthClientApiPermissions(tenantID int64, authClientUUID uuid.UUID, apiUUID uuid.UUID, permissionUUIDs []uuid.UUID) error
	RemoveAuthClientApiPermission(tenantID int64, authClientUUID uuid.UUID, apiUUID uuid.UUID, permissionUUID uuid.UUID) error
}

type authClientService struct {
	db                       *gorm.DB
	authClientRepo           repository.AuthClientRepository
	authClientUriRepo        repository.AuthClientURIRepository
	idpRepo                  repository.IdentityProviderRepository
	permissionRepo           repository.PermissionRepository
	authClientPermissionRepo repository.AuthClientPermissionRepository
	authClientApiRepo        repository.AuthClientApiRepository
	apiRepo                  repository.APIRepository
	userRepo                 repository.UserRepository
	tenantRepo               repository.TenantRepository
}

func NewAuthClientService(
	db *gorm.DB,
	authClientRepo repository.AuthClientRepository,
	authClientUriRepo repository.AuthClientURIRepository,
	idpRepo repository.IdentityProviderRepository,
	permissionRepo repository.PermissionRepository,
	authClientPermissionRepo repository.AuthClientPermissionRepository,
	authClientApiRepo repository.AuthClientApiRepository,
	apiRepo repository.APIRepository,
	userRepo repository.UserRepository,
	tenantRepo repository.TenantRepository,
) AuthClientService {
	return &authClientService{
		db:                       db,
		authClientRepo:           authClientRepo,
		authClientUriRepo:        authClientUriRepo,
		idpRepo:                  idpRepo,
		permissionRepo:           permissionRepo,
		authClientPermissionRepo: authClientPermissionRepo,
		authClientApiRepo:        authClientApiRepo,
		apiRepo:                  apiRepo,
		userRepo:                 userRepo,
		tenantRepo:               tenantRepo,
	}
}

func (s *authClientService) Get(filter AuthClientServiceGetFilter) (*AuthClientServiceGetResult, error) {
	var idpID *int64

	// Get identity provider
	if filter.IdentityProviderUUID != nil {
		idp, err := s.idpRepo.FindByUUID(*filter.IdentityProviderUUID)
		if err != nil || idp == nil {
			// Return empty result instead of error when identity provider not found
			return &AuthClientServiceGetResult{
				Data:       []AuthClientServiceDataResult{},
				Total:      0,
				Page:       filter.Page,
				Limit:      filter.Limit,
				TotalPages: 0,
			}, nil
		}
		idpID = &idp.IdentityProviderID
	}

	// Build query filter
	queryFilter := repository.AuthClientRepositoryGetFilter{
		TenantID:           filter.TenantID,
		Name:               filter.Name,
		DisplayName:        filter.DisplayName,
		ClientType:         filter.ClientType,
		IdentityProviderID: idpID,
		Status:             filter.Status,
		IsDefault:          filter.IsDefault,
		IsSystem:           filter.IsSystem,
		Page:               filter.Page,
		Limit:              filter.Limit,
		SortBy:             filter.SortBy,
		SortOrder:          filter.SortOrder,
	}

	result, err := s.authClientRepo.FindPaginated(queryFilter)
	if err != nil {
		return nil, err
	}

	// Build response data
	resData := make([]AuthClientServiceDataResult, len(result.Data))
	for i, rdata := range result.Data {
		resData[i] = *ToAuthClientServiceDataResult(&rdata)
	}

	return &AuthClientServiceGetResult{
		Data:       resData,
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}, nil
}

func (s *authClientService) GetByUUID(authClientUUID uuid.UUID, tenantID int64) (*AuthClientServiceDataResult, error) {
	authClient, err := s.authClientRepo.FindByUUIDAndTenantID(authClientUUID, tenantID)
	if err != nil {
		return nil, err
	}
	if authClient == nil {
		return nil, errors.New("auth client not found or access denied")
	}

	return ToAuthClientServiceDataResult(authClient), nil
}

func (s *authClientService) GetSecretByUUID(authClientUUID uuid.UUID, tenantID int64) (*AuthClientSecretServiceDataResult, error) {
	authClient, err := s.authClientRepo.FindByUUIDAndTenantID(authClientUUID, tenantID)
	if err != nil {
		return nil, err
	}
	if authClient == nil {
		return nil, errors.New("auth client not found or access denied")
	}

	return &AuthClientSecretServiceDataResult{
		ClientID:     *authClient.ClientID,
		ClientSecret: authClient.ClientSecret,
	}, nil
}

func (s *authClientService) GetConfigByUUID(authClientUUID uuid.UUID, tenantID int64) (datatypes.JSON, error) {
	authClient, err := s.authClientRepo.FindByUUIDAndTenantID(authClientUUID, tenantID)
	if err != nil {
		return nil, err
	}
	if authClient == nil {
		return nil, errors.New("auth client not found or access denied")
	}

	return authClient.Config, nil
}

func (s *authClientService) Create(tenantID int64, name string, displayName string, clientType string, domain string, config datatypes.JSON, status string, isDefault bool, identityProviderUUID string, actorUserUUID uuid.UUID) (*AuthClientServiceDataResult, error) {
	var createdAuthClient *model.AuthClient

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txAuthClientRepo := s.authClientRepo.WithTx(tx)
		txIdpRepo := s.idpRepo.WithTx(tx)
		txUserRepo := s.userRepo.WithTx(tx)

		// Parse identity provider UUID
		idpUUIDParsed, err := uuid.Parse(identityProviderUUID)
		if err != nil {
			return errors.New("invalid identity provider UUID")
		}

		// Check if identity provider exists
		identityProvider, err := txIdpRepo.FindByUUID(idpUUIDParsed, "Tenant")
		if err != nil || identityProvider == nil {
			return errors.New("identity provider not found")
		}

		// Get actor user with tenant info
		actorUser, err := txUserRepo.FindByUUID(actorUserUUID, "Tenant")
		if err != nil || actorUser == nil {
			return errors.New("actor user not found")
		}

		// Validate tenant access permissions
		if err := ValidateTenantAccess(actorUser, identityProvider.Tenant); err != nil {
			return err
		}

		// Check if auth client already exists
		existingAuthClient, err := txAuthClientRepo.FindByNameAndIdentityProvider(name, identityProvider.IdentityProviderID, tenantID)
		if err != nil {
			return err
		}
		if existingAuthClient != nil {
			return errors.New(name + " auth client already exists")
		}

		// Generate identifier
		clientId := util.GenerateIdentifier(12)
		clientSecret := util.GenerateIdentifier(64)

		// Create auth client
		newAuthClient := &model.AuthClient{
			Name:               name,
			DisplayName:        displayName,
			ClientType:         clientType,
			Domain:             &domain,
			ClientID:           &clientId,
			ClientSecret:       &clientSecret,
			Config:             config,
			TenantID:           tenantID,
			IdentityProviderID: identityProvider.IdentityProviderID,
			Status:             status,
			IsDefault:          isDefault,
			IsSystem:           false,
		}

		_, err = txAuthClientRepo.CreateOrUpdate(newAuthClient)
		if err != nil {
			return err
		}

		// Fetch AuthClient with Service preloaded
		createdAuthClient, err = txAuthClientRepo.FindByUUID(newAuthClient.AuthClientUUID, "IdentityProvider", "AuthClientURIs")
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return ToAuthClientServiceDataResult(createdAuthClient), nil
}

func (s *authClientService) Update(authClientUUID uuid.UUID, tenantID int64, name string, displayName string, clientType string, domain string, config datatypes.JSON, status string, isDefault bool, actorUserUUID uuid.UUID) (*AuthClientServiceDataResult, error) {
	var updatedAuthClient *model.AuthClient

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txAuthClientRepo := s.authClientRepo.WithTx(tx)
		txUserRepo := s.userRepo.WithTx(tx)

		// Get auth client
		authClient, err := txAuthClientRepo.FindByUUID(authClientUUID, "IdentityProvider.Tenant", "AuthClientURIs")
		if err != nil || authClient == nil {
			return errors.New("auth client not found")
		}

		// Get actor user with tenant info
		actorUser, err := txUserRepo.FindByUUID(actorUserUUID, "Tenant")
		if err != nil || actorUser == nil {
			return errors.New("actor user not found")
		}

		// Validate tenant access permissions
		if err := ValidateTenantAccess(actorUser, authClient.IdentityProvider.Tenant); err != nil {
			return err
		}

		// Check if default
		if authClient.IsDefault {
			return errors.New("default auth client cannot cannot be updated")
		}

		// Check if auth client already exist
		if authClient.Name != name {
			existingAuthClient, err := txAuthClientRepo.FindByNameAndIdentityProvider(name, authClient.IdentityProviderID, tenantID)
			if err != nil {
				return err
			}
			if existingAuthClient != nil && existingAuthClient.AuthClientUUID != authClientUUID {
				return errors.New(name + " auth client already exists")
			}
		}

		// Set values
		authClient.Name = name
		authClient.DisplayName = displayName
		authClient.ClientType = clientType
		authClient.Domain = &domain
		authClient.Config = config
		authClient.Status = status
		authClient.IsDefault = isDefault

		// Update
		_, err = txAuthClientRepo.CreateOrUpdate(authClient)
		if err != nil {
			return err
		}

		updatedAuthClient = authClient

		return nil
	})

	if err != nil {
		return nil, err
	}

	return ToAuthClientServiceDataResult(updatedAuthClient), nil
}

func (s *authClientService) SetStatusByUUID(authClientUUID uuid.UUID, tenantID int64, status string, actorUserUUID uuid.UUID) (*AuthClientServiceDataResult, error) {
	var updatedAuthClient *model.AuthClient

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txAuthClientRepo := s.authClientRepo.WithTx(tx)
		txUserRepo := s.userRepo.WithTx(tx)

		// Get auth client
		authClient, err := txAuthClientRepo.FindByUUID(authClientUUID, "IdentityProvider.Tenant", "AuthClientURIs")
		if err != nil || authClient == nil {
			return errors.New("auth client not found")
		}

		// Get actor user with tenant info
		actorUser, err := txUserRepo.FindByUUID(actorUserUUID, "Tenant")
		if err != nil || actorUser == nil {
			return errors.New("actor user not found")
		}

		// Validate tenant access permissions
		if err := ValidateTenantAccess(actorUser, authClient.IdentityProvider.Tenant); err != nil {
			return err
		}

		// Check if default or system
		if authClient.IsDefault {
			return errors.New("default auth client cannot be updated")
		}
		if authClient.IsSystem {
			return errors.New("system auth client cannot be updated")
		}

		// Set values
		authClient.Status = status

		// Update
		_, err = txAuthClientRepo.CreateOrUpdate(authClient)
		if err != nil {
			return err
		}

		updatedAuthClient = authClient

		return nil
	})

	if err != nil {
		return nil, err
	}

	return ToAuthClientServiceDataResult(updatedAuthClient), nil
}

func (s *authClientService) DeleteByUUID(authClientUUID uuid.UUID, tenantID int64, actorUserUUID uuid.UUID) (*AuthClientServiceDataResult, error) {
	var deletedAuthClient *model.AuthClient

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txAuthClientRepo := s.authClientRepo.WithTx(tx)
		txUserRepo := s.userRepo.WithTx(tx)

		// Get auth client
		authClient, err := txAuthClientRepo.FindByUUID(authClientUUID, "IdentityProvider.Tenant", "AuthClientURIs")
		if err != nil || authClient == nil {
			return errors.New("auth client not found")
		}

		// Get actor user with tenant info
		actorUser, err := txUserRepo.FindByUUID(actorUserUUID, "Tenant")
		if err != nil || actorUser == nil {
			return errors.New("actor user not found")
		}

		// Validate tenant access permissions
		if err := ValidateTenantAccess(actorUser, authClient.IdentityProvider.Tenant); err != nil {
			return err
		}

		// Check if default
		if authClient.IsDefault {
			return errors.New("default auth client cannot be deleted")
		}

		// Delete
		if err := txAuthClientRepo.DeleteByUUID(authClientUUID); err != nil {
			return err
		}

		deletedAuthClient = authClient

		return nil
	})

	if err != nil {
		return nil, err
	}

	return ToAuthClientServiceDataResult(deletedAuthClient), nil
}

func (s *authClientService) CreateURI(authClientUUID uuid.UUID, tenantID int64, uri string, uriType string, actorUserUUID uuid.UUID) (*AuthClientServiceDataResult, error) {
	var createdAuthClient *model.AuthClient

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txAuthClientRepo := s.authClientRepo.WithTx(tx)
		txURIRepo := s.authClientUriRepo.WithTx(tx)
		txUserRepo := s.userRepo.WithTx(tx)

		// Find the auth client by UUID
		authClient, err := txAuthClientRepo.FindByUUID(authClientUUID, "IdentityProvider.Tenant")
		if err != nil || authClient == nil {
			return errors.New("auth client not found")
		}

		// Get actor user with tenant info
		actorUser, err := txUserRepo.FindByUUID(actorUserUUID, "Tenant")
		if err != nil || actorUser == nil {
			return errors.New("actor user not found")
		}

		// Validate tenant access permissions
		if err := ValidateTenantAccess(actorUser, authClient.IdentityProvider.Tenant); err != nil {
			return err
		}

		// Create the URI entry
		newURI := &model.AuthClientURI{
			AuthClientID: authClient.AuthClientID,
			URI:          uri,
			Type:         uriType,
		}

		_, err = txURIRepo.CreateOrUpdate(newURI)
		if err != nil {
			return err
		}

		// Find the auth client updated with the new URI
		authClientCreated, err := txAuthClientRepo.FindByUUID(authClientUUID, "IdentityProvider", "AuthClientURIs")
		if err != nil || authClientCreated == nil {
			return errors.New("auth client not found")
		}

		createdAuthClient = authClientCreated

		return nil
	})

	if err != nil {
		return nil, err
	}

	return ToAuthClientServiceDataResult(createdAuthClient), nil
}

func (s *authClientService) UpdateURI(authClientUUID uuid.UUID, tenantID int64, authClientURIUUID uuid.UUID, uri string, uriType string, actorUserUUID uuid.UUID) (*AuthClientServiceDataResult, error) {
	var updatedAuthClient *model.AuthClient

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txAuthClientRepo := s.authClientRepo.WithTx(tx)
		txURIRepo := s.authClientUriRepo.WithTx(tx)
		txUserRepo := s.userRepo.WithTx(tx)

		// Find the auth client by UUID
		authClient, err := txAuthClientRepo.FindByUUID(authClientUUID, "IdentityProvider.Tenant")
		if err != nil || authClient == nil {
			return errors.New("auth client not found")
		}

		// Get actor user with tenant info
		actorUser, err := txUserRepo.FindByUUID(actorUserUUID, "Tenant")
		if err != nil || actorUser == nil {
			return errors.New("actor user not found")
		}

		// Validate tenant access permissions
		if err := ValidateTenantAccess(actorUser, authClient.IdentityProvider.Tenant); err != nil {
			return err
		}

		// Find the URI entry by UUID
		existingURI, err := txURIRepo.FindByUUID(authClientURIUUID)
		if err != nil || existingURI == nil {
			return errors.New("URI not found")
		}

		// Check if the URI belongs to the auth client
		if existingURI.AuthClientID != authClient.AuthClientID {
			return errors.New("URI does not belong to the specified auth client")
		}

		// Set new values
		existingURI.URI = uri
		existingURI.Type = uriType

		// Update
		_, err = txURIRepo.CreateOrUpdate(existingURI)
		if err != nil {
			return err
		}

		// Find the auth client updated with the new URI
		authClientUpdated, err := txAuthClientRepo.FindByUUID(authClientUUID, "IdentityProvider", "AuthClientURIs")
		if err != nil || authClientUpdated == nil {
			return errors.New("auth client not found")
		}

		updatedAuthClient = authClientUpdated

		return nil
	})

	if err != nil {
		return nil, err
	}

	return ToAuthClientServiceDataResult(updatedAuthClient), nil
}

func (s *authClientService) DeleteURI(authClientUUID uuid.UUID, tenantID int64, authClientURIUUID uuid.UUID, actorUserUUID uuid.UUID) (*AuthClientServiceDataResult, error) {
	var deletedAuthClient *model.AuthClient

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txAuthClientRepo := s.authClientRepo.WithTx(tx)
		txURIRepo := s.authClientUriRepo.WithTx(tx)
		txUserRepo := s.userRepo.WithTx(tx)

		// Find the auth client by UUID
		authClient, err := txAuthClientRepo.FindByUUID(authClientUUID, "IdentityProvider.Tenant", "AuthClientURIs")
		if err != nil || authClient == nil {
			return errors.New("auth client not found")
		}

		// Get actor user with tenant info
		actorUser, err := txUserRepo.FindByUUID(actorUserUUID, "Tenant")
		if err != nil || actorUser == nil {
			return errors.New("actor user not found")
		}

		// Validate tenant access permissions
		if err := ValidateTenantAccess(actorUser, authClient.IdentityProvider.Tenant); err != nil {
			return err
		}

		// Find the URI entry by UUID
		existingURI, err := txURIRepo.FindByUUID(authClientURIUUID)
		if err != nil || existingURI == nil {
			return errors.New("URI not found")
		}

		// Check if the URI belongs to the auth client
		if existingURI.AuthClientID != authClient.AuthClientID {
			return errors.New("URI does not belong to the specified auth client")
		}

		// Delete the entry
		if err := txURIRepo.DeleteByUUID(authClientURIUUID); err != nil {
			return err
		}

		deletedAuthClient = authClient

		return nil
	})

	if err != nil {
		return nil, err
	}

	return ToAuthClientServiceDataResult(deletedAuthClient), nil
}

// Response builder - made public for use in other services
func ToAuthClientServiceDataResult(authClient *model.AuthClient) *AuthClientServiceDataResult {
	if authClient == nil {
		return nil
	}

	result := &AuthClientServiceDataResult{
		AuthClientUUID: authClient.AuthClientUUID,
		Name:           authClient.Name,
		DisplayName:    authClient.DisplayName,
		ClientType:     authClient.ClientType,
		Domain:         authClient.Domain,
		Status:         authClient.Status,
		IsDefault:      authClient.IsDefault,
		IsSystem:       authClient.IsSystem,
		CreatedAt:      authClient.CreatedAt,
		UpdatedAt:      authClient.UpdatedAt,
	}

	if authClient.IdentityProvider != nil {
		result.IdentityProvider = &IdentityProviderServiceDataResult{
			IdentityProviderUUID: authClient.IdentityProvider.IdentityProviderUUID,
			Name:                 authClient.IdentityProvider.Name,
			DisplayName:          authClient.IdentityProvider.DisplayName,
			Provider:             authClient.IdentityProvider.Provider,
			ProviderType:         authClient.IdentityProvider.ProviderType,
			Identifier:           authClient.IdentityProvider.Identifier,
			Status:               authClient.IdentityProvider.Status,
			IsDefault:            authClient.IdentityProvider.IsDefault,
			IsSystem:             authClient.IdentityProvider.IsSystem,
			CreatedAt:            authClient.IdentityProvider.CreatedAt,
			UpdatedAt:            authClient.IdentityProvider.UpdatedAt,
		}
	}

	// Map URIs
	if authClient.AuthClientURIs != nil && len(*authClient.AuthClientURIs) > 0 {
		uris := make([]AuthClientURIServiceDataResult, len(*authClient.AuthClientURIs))
		for i, u := range *authClient.AuthClientURIs {
			uris[i] = AuthClientURIServiceDataResult{
				AuthClientURIUUID: u.AuthClientURIUUID,
				URI:               u.URI,
				Type:              u.Type,
				CreatedAt:         u.CreatedAt,
				UpdatedAt:         u.UpdatedAt,
			}
		}
		result.AuthClientURIs = &uris
	}

	// TODO: Permissions are now accessed via AuthClientApis relationship
	// This will be implemented when we complete the new API structure

	return result
}

// Get APIs assigned to auth client
func (s *authClientService) GetAuthClientApis(tenantID int64, authClientUUID uuid.UUID) ([]AuthClientApiServiceDataResult, error) {
	// Get auth client APIs from repository
	authClientApis, err := s.authClientApiRepo.FindByAuthClientUUID(authClientUUID)
	if err != nil {
		return nil, err
	}

	// Convert to service data results
	results := make([]AuthClientApiServiceDataResult, len(authClientApis))
	for i, authClientApi := range authClientApis {
		// Convert API to service data
		apiResult := APIServiceDataResult{
			APIUUID:     authClientApi.API.APIUUID,
			Name:        authClientApi.API.Name,
			DisplayName: authClientApi.API.DisplayName,
			Description: authClientApi.API.Description,
			Status:      authClientApi.API.Status,
			IsDefault:   authClientApi.API.IsDefault,
			IsSystem:    authClientApi.API.IsSystem,
			CreatedAt:   authClientApi.API.CreatedAt,
			UpdatedAt:   authClientApi.API.UpdatedAt,
		}

		// Convert permissions to service data
		permissions := make([]PermissionServiceDataResult, len(authClientApi.Permissions))
		for j, authClientPermission := range authClientApi.Permissions {
			if authClientPermission.Permission != nil {
				permissions[j] = PermissionServiceDataResult{
					PermissionUUID: authClientPermission.Permission.PermissionUUID,
					Name:           authClientPermission.Permission.Name,
					Description:    authClientPermission.Permission.Description,
					Status:         authClientPermission.Permission.Status,
					IsDefault:      authClientPermission.Permission.IsDefault,
					IsSystem:       authClientPermission.Permission.IsSystem,
					CreatedAt:      authClientPermission.Permission.CreatedAt,
					UpdatedAt:      authClientPermission.Permission.UpdatedAt,
				}
			}
		}

		results[i] = AuthClientApiServiceDataResult{
			AuthClientApiUUID: authClientApi.AuthClientApiUUID,
			Api:               apiResult,
			Permissions:       permissions,
			CreatedAt:         authClientApi.CreatedAt,
		}
	}

	return results, nil
}

// Add APIs to auth client
func (s *authClientService) AddAuthClientApis(tenantID int64, authClientUUID uuid.UUID, apiUUIDs []uuid.UUID) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		authClientRepo := s.authClientRepo.WithTx(tx)
		authClientApiRepo := s.authClientApiRepo.WithTx(tx)
		apiRepo := s.apiRepo.WithTx(tx)

		// Get auth client with identity provider
		authClient, err := authClientRepo.FindByUUID(authClientUUID, "IdentityProvider")
		if err != nil {
			return err
		}
		if authClient == nil {
			return errors.New("auth client not found")
		}

		// Validate tenant access
		if authClient.IdentityProvider == nil || authClient.IdentityProvider.TenantID != tenantID {
			return errors.New("unauthorized access to auth client")
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
			existing, err := authClientApiRepo.FindByAuthClientAndApi(authClient.AuthClientID, api.APIID)
			if err != nil {
				return err
			}
			if existing != nil {
				return errors.New("API already assigned to auth client: " + apiUUID.String())
			}

			// Create new auth client API relationship
			authClientApi := &model.AuthClientApi{
				AuthClientApiUUID: uuid.New(),
				AuthClientID:      authClient.AuthClientID,
				APIID:             api.APIID,
			}

			_, err = authClientApiRepo.Create(authClientApi)
			if err != nil {
				// Check if it's a unique constraint violation
				if strings.Contains(err.Error(), "uq_auth_client_apis_client_api") {
					return errors.New("API already assigned to auth client: " + apiUUID.String())
				}
				return err
			}
		}

		return nil
	})
}

// Remove API from auth client
func (s *authClientService) RemoveAuthClientApi(tenantID int64, authClientUUID uuid.UUID, apiUUID uuid.UUID) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		authClientRepo := s.authClientRepo.WithTx(tx)
		authClientApiRepo := s.authClientApiRepo.WithTx(tx)

		// Get auth client with identity provider
		authClient, err := authClientRepo.FindByUUID(authClientUUID, "IdentityProvider")
		if err != nil {
			return err
		}
		if authClient == nil {
			return errors.New("auth client not found")
		}

		// Validate tenant access
		if authClient.IdentityProvider == nil || authClient.IdentityProvider.TenantID != tenantID {
			return errors.New("unauthorized access to auth client")
		}

		// Remove the API relationship (this will cascade delete permissions)
		err = authClientApiRepo.RemoveByAuthClientUUIDAndApiUUID(authClientUUID, apiUUID)
		if err != nil {
			return err
		}

		return nil
	})
}

// Get permissions for a specific API assigned to auth client
func (s *authClientService) GetAuthClientApiPermissions(tenantID int64, authClientUUID uuid.UUID, apiUUID uuid.UUID) ([]PermissionServiceDataResult, error) {
	// Get auth client API relationship
	authClientApi, err := s.authClientApiRepo.FindByAuthClientUUIDAndApiUUID(authClientUUID, apiUUID)
	if err != nil {
		return nil, err
	}
	if authClientApi == nil {
		return nil, errors.New("auth client API relationship not found")
	}

	// Validate tenant access
	authClient, err := s.authClientRepo.FindByUUID(authClientUUID, "IdentityProvider")
	if err != nil {
		return nil, err
	}
	if authClient == nil {
		return nil, errors.New("auth client not found")
	}
	if authClient.IdentityProvider == nil || authClient.IdentityProvider.TenantID != tenantID {
		return nil, errors.New("unauthorized access to auth client")
	}

	// Get permissions for this auth client API
	authClientPermissions, err := s.authClientPermissionRepo.FindByAuthClientApiID(authClientApi.AuthClientApiID)
	if err != nil {
		return nil, err
	}

	// Convert to service data results
	results := make([]PermissionServiceDataResult, len(authClientPermissions))
	for i, authClientPermission := range authClientPermissions {
		results[i] = PermissionServiceDataResult{
			PermissionUUID: authClientPermission.Permission.PermissionUUID,
			Name:           authClientPermission.Permission.Name,
			Description:    authClientPermission.Permission.Description,
			Status:         authClientPermission.Permission.Status,
			IsDefault:      authClientPermission.Permission.IsDefault,
			IsSystem:       authClientPermission.Permission.IsSystem,
			CreatedAt:      authClientPermission.Permission.CreatedAt,
			UpdatedAt:      authClientPermission.Permission.UpdatedAt,
		}
	}

	return results, nil
}

// Add permissions to a specific API for auth client
func (s *authClientService) AddAuthClientApiPermissions(tenantID int64, authClientUUID uuid.UUID, apiUUID uuid.UUID, permissionUUIDs []uuid.UUID) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		authClientRepo := s.authClientRepo.WithTx(tx)
		authClientApiRepo := s.authClientApiRepo.WithTx(tx)
		authClientPermissionRepo := s.authClientPermissionRepo.WithTx(tx)
		permissionRepo := s.permissionRepo.WithTx(tx)

		// Get auth client with identity provider
		authClient, err := authClientRepo.FindByUUID(authClientUUID, "IdentityProvider")
		if err != nil {
			return err
		}
		if authClient == nil {
			return errors.New("auth client not found")
		}

		// Validate tenant access
		if authClient.IdentityProvider == nil || authClient.IdentityProvider.TenantID != tenantID {
			return errors.New("unauthorized access to auth client")
		}

		// Get auth client API relationship
		authClientApi, err := authClientApiRepo.FindByAuthClientUUIDAndApiUUID(authClientUUID, apiUUID)
		if err != nil {
			return err
		}
		if authClientApi == nil {
			return errors.New("auth client API relationship not found")
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
			existing, err := authClientPermissionRepo.FindByAuthClientApiAndPermission(authClientApi.AuthClientApiID, permission.PermissionID)
			if err != nil {
				return err
			}
			if existing != nil {
				return errors.New("permission already assigned to auth client API: " + permissionUUID.String())
			}

			// Create new auth client permission relationship
			authClientPermission := &model.AuthClientPermission{
				AuthClientPermissionUUID: uuid.New(),
				AuthClientApiID:          authClientApi.AuthClientApiID,
				PermissionID:             permission.PermissionID,
			}

			_, err = authClientPermissionRepo.Create(authClientPermission)
			if err != nil {
				// Check if it's a unique constraint violation
				if strings.Contains(err.Error(), "uq_auth_client_permissions_api_permission") {
					return errors.New("permission already assigned to auth client API: " + permissionUUID.String())
				}
				return err
			}
		}

		return nil
	})
}

// Remove permission from a specific API for auth client
func (s *authClientService) RemoveAuthClientApiPermission(tenantID int64, authClientUUID uuid.UUID, apiUUID uuid.UUID, permissionUUID uuid.UUID) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		authClientRepo := s.authClientRepo.WithTx(tx)
		authClientApiRepo := s.authClientApiRepo.WithTx(tx)
		authClientPermissionRepo := s.authClientPermissionRepo.WithTx(tx)
		permissionRepo := s.permissionRepo.WithTx(tx)

		// Get auth client with identity provider
		authClient, err := authClientRepo.FindByUUID(authClientUUID, "IdentityProvider")
		if err != nil {
			return err
		}
		if authClient == nil {
			return errors.New("auth client not found")
		}

		// Validate tenant access
		if authClient.IdentityProvider == nil || authClient.IdentityProvider.TenantID != tenantID {
			return errors.New("unauthorized access to auth client")
		}

		// Get auth client API relationship
		authClientApi, err := authClientApiRepo.FindByAuthClientUUIDAndApiUUID(authClientUUID, apiUUID)
		if err != nil {
			return err
		}
		if authClientApi == nil {
			return errors.New("auth client API relationship not found")
		}

		// Get permission
		permission, err := permissionRepo.FindByUUID(permissionUUID)
		if err != nil {
			return err
		}
		if permission == nil {
			return errors.New("permission not found")
		}

		// Remove the permission relationship
		err = authClientPermissionRepo.RemoveByAuthClientApiAndPermission(authClientApi.AuthClientApiID, permission.PermissionID)
		if err != nil {
			return err
		}

		return nil
	})
}
