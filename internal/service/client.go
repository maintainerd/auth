package service

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/repository"
	"github.com/maintainerd/auth/internal/generator"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type ClientSecretServiceDataResult struct {
	ClientID     string
	ClientSecret *string
}

type ClientURIServiceDataResult struct {
	ClientURIUUID uuid.UUID
	URI           string
	Type          string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type ClientServiceDataResult struct {
	ClientUUID       uuid.UUID
	Name             string
	DisplayName      string
	ClientType       string
	Domain           *string
	ClientURIs       *[]ClientURIServiceDataResult
	IdentityProvider *IdentityProviderServiceDataResult
	Permissions      *[]PermissionServiceDataResult
	Status           string
	IsDefault        bool
	IsSystem         bool
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type ClientAPIServiceDataResult struct {
	ClientAPIUUID uuid.UUID
	Api           APIServiceDataResult
	Permissions   []PermissionServiceDataResult
	CreatedAt     time.Time
}

type ClientServiceGetFilter struct {
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

type ClientServiceGetResult struct {
	Data       []ClientServiceDataResult
	Total      int64
	Page       int
	Limit      int
	TotalPages int
}

type ClientService interface {
	Get(filter ClientServiceGetFilter) (*ClientServiceGetResult, error)
	GetByUUID(ClientUUID uuid.UUID, tenantID int64) (*ClientServiceDataResult, error)
	GetSecretByUUID(ClientUUID uuid.UUID, tenantID int64) (*ClientSecretServiceDataResult, error)
	GetConfigByUUID(ClientUUID uuid.UUID, tenantID int64) (datatypes.JSON, error)
	Create(tenantID int64, name string, displayName string, clientType string, domain string, config datatypes.JSON, status string, isDefault bool, identityProviderUUID string, actorUserUUID uuid.UUID) (*ClientServiceDataResult, error)
	Update(ClientUUID uuid.UUID, tenantID int64, name string, displayName string, clientType string, domain string, config datatypes.JSON, status string, isDefault bool, actorUserUUID uuid.UUID) (*ClientServiceDataResult, error)
	SetStatusByUUID(ClientUUID uuid.UUID, tenantID int64, status string, actorUserUUID uuid.UUID) (*ClientServiceDataResult, error)
	DeleteByUUID(ClientUUID uuid.UUID, tenantID int64, actorUserUUID uuid.UUID) (*ClientServiceDataResult, error)
	CreateURI(ClientUUID uuid.UUID, tenantID int64, uri string, uriType string, actorUserUUID uuid.UUID) (*ClientServiceDataResult, error)
	UpdateURI(ClientUUID uuid.UUID, tenantID int64, ClientURIUUID uuid.UUID, uri string, uriType string, actorUserUUID uuid.UUID) (*ClientServiceDataResult, error)
	DeleteURI(ClientUUID uuid.UUID, tenantID int64, ClientURIUUID uuid.UUID, actorUserUUID uuid.UUID) (*ClientServiceDataResult, error)

	// Auth Client API methods
	GetClientAPIs(tenantID int64, ClientUUID uuid.UUID) ([]ClientAPIServiceDataResult, error)
	AddClientAPIs(tenantID int64, ClientUUID uuid.UUID, apiUUIDs []uuid.UUID) error
	RemoveClientAPI(tenantID int64, ClientUUID uuid.UUID, apiUUID uuid.UUID) error

	// Auth Client API Permission methods
	GetClientAPIPermissions(tenantID int64, ClientUUID uuid.UUID, apiUUID uuid.UUID) ([]PermissionServiceDataResult, error)
	AddClientAPIPermissions(tenantID int64, ClientUUID uuid.UUID, apiUUID uuid.UUID, permissionUUIDs []uuid.UUID) error
	RemoveClientAPIPermission(tenantID int64, ClientUUID uuid.UUID, apiUUID uuid.UUID, permissionUUID uuid.UUID) error
}

type clientService struct {
	db                   *gorm.DB
	clientRepo           repository.ClientRepository
	clientURIRepo        repository.ClientURIRepository
	idpRepo              repository.IdentityProviderRepository
	permissionRepo       repository.PermissionRepository
	clientPermissionRepo repository.ClientPermissionRepository
	clientAPIRepo        repository.ClientAPIRepository
	apiRepo              repository.APIRepository
	userRepo             repository.UserRepository
	tenantRepo           repository.TenantRepository
}

func NewClientService(
	db *gorm.DB,
	clientRepo repository.ClientRepository,
	clientURIRepo repository.ClientURIRepository,
	idpRepo repository.IdentityProviderRepository,
	permissionRepo repository.PermissionRepository,
	clientPermissionRepo repository.ClientPermissionRepository,
	clientAPIRepo repository.ClientAPIRepository,
	apiRepo repository.APIRepository,
	userRepo repository.UserRepository,
	tenantRepo repository.TenantRepository,
) ClientService {
	return &clientService{
		db:                   db,
		clientRepo:           clientRepo,
		clientURIRepo:        clientURIRepo,
		idpRepo:              idpRepo,
		permissionRepo:       permissionRepo,
		clientPermissionRepo: clientPermissionRepo,
		clientAPIRepo:        clientAPIRepo,
		apiRepo:              apiRepo,
		userRepo:             userRepo,
		tenantRepo:           tenantRepo,
	}
}

func (s *clientService) Get(filter ClientServiceGetFilter) (*ClientServiceGetResult, error) {
	var idpID *int64

	// Get identity provider
	if filter.IdentityProviderUUID != nil {
		idp, err := s.idpRepo.FindByUUID(*filter.IdentityProviderUUID)
		if err != nil || idp == nil {
			// Return empty result instead of error when identity provider not found
			return &ClientServiceGetResult{
				Data:       []ClientServiceDataResult{},
				Total:      0,
				Page:       filter.Page,
				Limit:      filter.Limit,
				TotalPages: 0,
			}, nil
		}
		idpID = &idp.IdentityProviderID
	}

	// Build query filter
	queryFilter := repository.ClientRepositoryGetFilter{
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

	result, err := s.clientRepo.FindPaginated(queryFilter)
	if err != nil {
		return nil, err
	}

	// Build response data
	resData := make([]ClientServiceDataResult, len(result.Data))
	for i, rdata := range result.Data {
		resData[i] = *ToClientServiceDataResult(&rdata)
	}

	return &ClientServiceGetResult{
		Data:       resData,
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}, nil
}

func (s *clientService) GetByUUID(ClientUUID uuid.UUID, tenantID int64) (*ClientServiceDataResult, error) {
	Client, err := s.clientRepo.FindByUUIDAndTenantID(ClientUUID, tenantID)
	if err != nil {
		return nil, err
	}
	if Client == nil {
		return nil, errors.New("auth client not found or access denied")
	}

	return ToClientServiceDataResult(Client), nil
}

func (s *clientService) GetSecretByUUID(ClientUUID uuid.UUID, tenantID int64) (*ClientSecretServiceDataResult, error) {
	Client, err := s.clientRepo.FindByUUIDAndTenantID(ClientUUID, tenantID)
	if err != nil {
		return nil, err
	}
	if Client == nil {
		return nil, errors.New("auth client not found or access denied")
	}

	return &ClientSecretServiceDataResult{
		ClientID:     *Client.Identifier,
		ClientSecret: Client.Secret,
	}, nil
}

func (s *clientService) GetConfigByUUID(ClientUUID uuid.UUID, tenantID int64) (datatypes.JSON, error) {
	Client, err := s.clientRepo.FindByUUIDAndTenantID(ClientUUID, tenantID)
	if err != nil {
		return nil, err
	}
	if Client == nil {
		return nil, errors.New("auth client not found or access denied")
	}

	return Client.Config, nil
}

func (s *clientService) Create(tenantID int64, name string, displayName string, clientType string, domain string, config datatypes.JSON, status string, isDefault bool, identityProviderUUID string, actorUserUUID uuid.UUID) (*ClientServiceDataResult, error) {
	var createdClient *model.Client

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txClientRepo := s.clientRepo.WithTx(tx)
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
		actorUser, err := txUserRepo.FindByUUID(actorUserUUID, "UserIdentities.Tenant")
		if err != nil || actorUser == nil {
			return errors.New("actor user not found")
		}

		// Validate tenant access permissions
		if err := ValidateTenantAccess(actorUser, identityProvider.Tenant); err != nil {
			return err
		}

		// Check if auth client already exists
		existingClient, err := txClientRepo.FindByNameAndIdentityProvider(name, identityProvider.IdentityProviderID, tenantID)
		if err != nil {
			return err
		}
		if existingClient != nil {
			return errors.New(name + " auth client already exists")
		}

		// Generate identifier
		clientId := generator.GenerateIdentifier(12)
		clientSecret := generator.GenerateIdentifier(64)

		// Create auth client
		newClient := &model.Client{
			Name:               name,
			DisplayName:        displayName,
			ClientType:         clientType,
			Domain:             &domain,
			Identifier:         &clientId,
			Secret:             &clientSecret,
			Config:             config,
			TenantID:           tenantID,
			IdentityProviderID: identityProvider.IdentityProviderID,
			Status:             status,
			IsDefault:          isDefault,
			IsSystem:           false,
		}

		_, err = txClientRepo.CreateOrUpdate(newClient)
		if err != nil {
			return err
		}

		// Fetch Client with Service preloaded
		createdClient, err = txClientRepo.FindByUUID(newClient.ClientUUID, "IdentityProvider", "ClientURIs")
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return ToClientServiceDataResult(createdClient), nil
}

func (s *clientService) Update(ClientUUID uuid.UUID, tenantID int64, name string, displayName string, clientType string, domain string, config datatypes.JSON, status string, isDefault bool, actorUserUUID uuid.UUID) (*ClientServiceDataResult, error) {
	var updatedClient *model.Client

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txClientRepo := s.clientRepo.WithTx(tx)
		txUserRepo := s.userRepo.WithTx(tx)

		// Get auth client
		Client, err := txClientRepo.FindByUUID(ClientUUID, "IdentityProvider.Tenant", "ClientURIs")
		if err != nil || Client == nil {
			return errors.New("auth client not found")
		}

		// Get actor user with tenant info
		actorUser, err := txUserRepo.FindByUUID(actorUserUUID, "UserIdentities.Tenant")
		if err != nil || actorUser == nil {
			return errors.New("actor user not found")
		}

		// Validate tenant access permissions
		if err := ValidateTenantAccess(actorUser, Client.IdentityProvider.Tenant); err != nil {
			return err
		}

		// Check if default
		if Client.IsDefault {
			return errors.New("default auth client cannot cannot be updated")
		}

		// Check if auth client already exist
		if Client.Name != name {
			existingClient, err := txClientRepo.FindByNameAndIdentityProvider(name, Client.IdentityProviderID, tenantID)
			if err != nil {
				return err
			}
			if existingClient != nil && existingClient.ClientUUID != ClientUUID {
				return errors.New(name + " auth client already exists")
			}
		}

		// Set values
		Client.Name = name
		Client.DisplayName = displayName
		Client.ClientType = clientType
		Client.Domain = &domain
		Client.Config = config
		Client.Status = status
		Client.IsDefault = isDefault

		// Update
		_, err = txClientRepo.CreateOrUpdate(Client)
		if err != nil {
			return err
		}

		updatedClient = Client

		return nil
	})

	if err != nil {
		return nil, err
	}

	return ToClientServiceDataResult(updatedClient), nil
}

func (s *clientService) SetStatusByUUID(ClientUUID uuid.UUID, tenantID int64, status string, actorUserUUID uuid.UUID) (*ClientServiceDataResult, error) {
	var updatedClient *model.Client

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txClientRepo := s.clientRepo.WithTx(tx)
		txUserRepo := s.userRepo.WithTx(tx)

		// Get auth client
		Client, err := txClientRepo.FindByUUID(ClientUUID, "IdentityProvider.Tenant", "ClientURIs")
		if err != nil || Client == nil {
			return errors.New("auth client not found")
		}

		// Get actor user with tenant info
		actorUser, err := txUserRepo.FindByUUID(actorUserUUID, "UserIdentities.Tenant")
		if err != nil || actorUser == nil {
			return errors.New("actor user not found")
		}

		// Validate tenant access permissions
		if err := ValidateTenantAccess(actorUser, Client.IdentityProvider.Tenant); err != nil {
			return err
		}

		// Check if default or system
		if Client.IsDefault {
			return errors.New("default auth client cannot be updated")
		}
		if Client.IsSystem {
			return errors.New("system auth client cannot be updated")
		}

		// Set values
		Client.Status = status

		// Update
		_, err = txClientRepo.CreateOrUpdate(Client)
		if err != nil {
			return err
		}

		updatedClient = Client

		return nil
	})

	if err != nil {
		return nil, err
	}

	return ToClientServiceDataResult(updatedClient), nil
}

func (s *clientService) DeleteByUUID(ClientUUID uuid.UUID, tenantID int64, actorUserUUID uuid.UUID) (*ClientServiceDataResult, error) {
	var deletedClient *model.Client

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txClientRepo := s.clientRepo.WithTx(tx)
		txUserRepo := s.userRepo.WithTx(tx)

		// Get auth client
		Client, err := txClientRepo.FindByUUID(ClientUUID, "IdentityProvider.Tenant", "ClientURIs")
		if err != nil || Client == nil {
			return errors.New("auth client not found")
		}

		// Get actor user with tenant info
		actorUser, err := txUserRepo.FindByUUID(actorUserUUID, "UserIdentities.Tenant")
		if err != nil || actorUser == nil {
			return errors.New("actor user not found")
		}

		// Validate tenant access permissions
		if err := ValidateTenantAccess(actorUser, Client.IdentityProvider.Tenant); err != nil {
			return err
		}

		// Check if default
		if Client.IsDefault {
			return errors.New("default auth client cannot be deleted")
		}

		// Delete
		if err := txClientRepo.DeleteByUUID(ClientUUID); err != nil {
			return err
		}

		deletedClient = Client

		return nil
	})

	if err != nil {
		return nil, err
	}

	return ToClientServiceDataResult(deletedClient), nil
}

func (s *clientService) CreateURI(ClientUUID uuid.UUID, tenantID int64, uri string, uriType string, actorUserUUID uuid.UUID) (*ClientServiceDataResult, error) {
	var createdClient *model.Client

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txClientRepo := s.clientRepo.WithTx(tx)
		txURIRepo := s.clientURIRepo.WithTx(tx)
		txUserRepo := s.userRepo.WithTx(tx)

		// Find the auth client by UUID and tenant
		Client, err := txClientRepo.FindByUUIDAndTenantID(ClientUUID, tenantID)
		if err != nil || Client == nil {
			return errors.New("auth client not found or access denied")
		}

		// Get actor user with tenant info
		actorUser, err := txUserRepo.FindByUUID(actorUserUUID, "UserIdentities.Tenant")
		if err != nil || actorUser == nil {
			return errors.New("actor user not found")
		}

		// Validate tenant ownership
		if Client.TenantID != tenantID {
			return errors.New("auth client not found or access denied")
		}

		// Create the URI entry
		newURI := &model.ClientURI{
			TenantID: tenantID,
			ClientID: Client.ClientID,
			URI:      uri,
			Type:     uriType,
		}

		_, err = txURIRepo.CreateOrUpdate(newURI)
		if err != nil {
			return err
		}

		// Find the auth client updated with the new URI
		ClientCreated, err := txClientRepo.FindByUUIDAndTenantID(ClientUUID, tenantID)
		if err != nil || ClientCreated == nil {
			return errors.New("auth client not found")
		}

		createdClient = ClientCreated

		return nil
	})

	if err != nil {
		return nil, err
	}

	return ToClientServiceDataResult(createdClient), nil
}

func (s *clientService) UpdateURI(ClientUUID uuid.UUID, tenantID int64, ClientURIUUID uuid.UUID, uri string, uriType string, actorUserUUID uuid.UUID) (*ClientServiceDataResult, error) {
	var updatedClient *model.Client

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txClientRepo := s.clientRepo.WithTx(tx)
		txURIRepo := s.clientURIRepo.WithTx(tx)
		txUserRepo := s.userRepo.WithTx(tx)

		// Find the auth client by UUID and tenant
		Client, err := txClientRepo.FindByUUIDAndTenantID(ClientUUID, tenantID)
		if err != nil || Client == nil {
			return errors.New("auth client not found or access denied")
		}

		// Get actor user with tenant info
		actorUser, err := txUserRepo.FindByUUID(actorUserUUID, "UserIdentities.Tenant")
		if err != nil || actorUser == nil {
			return errors.New("actor user not found")
		}

		// Validate tenant ownership
		if Client.TenantID != tenantID {
			return errors.New("auth client not found or access denied")
		}

		// Find the URI entry by UUID and tenant
		existingURI, err := txURIRepo.FindByUUIDAndTenantID(ClientURIUUID.String(), tenantID)
		if err != nil || existingURI == nil {
			return errors.New("URI not found or access denied")
		}

		// Check if the URI belongs to the auth client
		if existingURI.ClientID != Client.ClientID {
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
		ClientUpdated, err := txClientRepo.FindByUUIDAndTenantID(ClientUUID, tenantID)
		if err != nil || ClientUpdated == nil {
			return errors.New("auth client not found")
		}

		updatedClient = ClientUpdated

		return nil
	})

	if err != nil {
		return nil, err
	}

	return ToClientServiceDataResult(updatedClient), nil
}

func (s *clientService) DeleteURI(ClientUUID uuid.UUID, tenantID int64, ClientURIUUID uuid.UUID, actorUserUUID uuid.UUID) (*ClientServiceDataResult, error) {
	var deletedClient *model.Client

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txClientRepo := s.clientRepo.WithTx(tx)
		txURIRepo := s.clientURIRepo.WithTx(tx)
		txUserRepo := s.userRepo.WithTx(tx)

		// Find the auth client by UUID and tenant
		Client, err := txClientRepo.FindByUUIDAndTenantID(ClientUUID, tenantID)
		if err != nil || Client == nil {
			return errors.New("auth client not found or access denied")
		}

		// Get actor user with tenant info
		actorUser, err := txUserRepo.FindByUUID(actorUserUUID, "UserIdentities.Tenant")
		if err != nil || actorUser == nil {
			return errors.New("actor user not found")
		}

		// Validate tenant ownership
		if Client.TenantID != tenantID {
			return errors.New("auth client not found or access denied")
		}

		// Find the URI entry by UUID and tenant
		existingURI, err := txURIRepo.FindByUUIDAndTenantID(ClientURIUUID.String(), tenantID)
		if err != nil || existingURI == nil {
			return errors.New("URI not found or access denied")
		}

		// Check if the URI belongs to the auth client
		if existingURI.ClientID != Client.ClientID {
			return errors.New("URI does not belong to the specified auth client")
		}

		// Delete the entry
		if err := txURIRepo.DeleteByUUIDAndTenantID(ClientURIUUID.String(), tenantID); err != nil {
			return err
		}

		deletedClient = Client

		return nil
	})

	if err != nil {
		return nil, err
	}

	return ToClientServiceDataResult(deletedClient), nil
}

// Response builder - made public for use in other services
func ToClientServiceDataResult(Client *model.Client) *ClientServiceDataResult {
	if Client == nil {
		return nil
	}

	result := &ClientServiceDataResult{
		ClientUUID:  Client.ClientUUID,
		Name:        Client.Name,
		DisplayName: Client.DisplayName,
		ClientType:  Client.ClientType,
		Domain:      Client.Domain,
		Status:      Client.Status,
		IsDefault:   Client.IsDefault,
		IsSystem:    Client.IsSystem,
		CreatedAt:   Client.CreatedAt,
		UpdatedAt:   Client.UpdatedAt,
	}

	if Client.IdentityProvider != nil {
		result.IdentityProvider = &IdentityProviderServiceDataResult{
			IdentityProviderUUID: Client.IdentityProvider.IdentityProviderUUID,
			Name:                 Client.IdentityProvider.Name,
			DisplayName:          Client.IdentityProvider.DisplayName,
			Provider:             Client.IdentityProvider.Provider,
			ProviderType:         Client.IdentityProvider.ProviderType,
			Identifier:           Client.IdentityProvider.Identifier,
			Status:               Client.IdentityProvider.Status,
			IsDefault:            Client.IdentityProvider.IsDefault,
			IsSystem:             Client.IdentityProvider.IsSystem,
			CreatedAt:            Client.IdentityProvider.CreatedAt,
			UpdatedAt:            Client.IdentityProvider.UpdatedAt,
		}
	}

	// Map URIs
	if Client.ClientURIs != nil && len(*Client.ClientURIs) > 0 {
		uris := make([]ClientURIServiceDataResult, len(*Client.ClientURIs))
		for i, u := range *Client.ClientURIs {
			uris[i] = ClientURIServiceDataResult{
				ClientURIUUID: u.ClientURIUUID,
				URI:           u.URI,
				Type:          u.Type,
				CreatedAt:     u.CreatedAt,
				UpdatedAt:     u.UpdatedAt,
			}
		}
		result.ClientURIs = &uris
	}

	// TODO: Permissions are now accessed via ClientAPIs relationship
	// This will be implemented when we complete the new API structure

	return result
}

// Get APIs assigned to auth client
func (s *clientService) GetClientAPIs(tenantID int64, ClientUUID uuid.UUID) ([]ClientAPIServiceDataResult, error) {
	// Get auth client APIs from repository
	ClientAPIs, err := s.clientAPIRepo.FindByClientUUID(ClientUUID)
	if err != nil {
		return nil, err
	}

	// Convert to service data results
	results := make([]ClientAPIServiceDataResult, len(ClientAPIs))
	for i, ClientAPI := range ClientAPIs {
		// Convert API to service data
		apiResult := APIServiceDataResult{
			APIUUID:     ClientAPI.API.APIUUID,
			Name:        ClientAPI.API.Name,
			DisplayName: ClientAPI.API.DisplayName,
			Description: ClientAPI.API.Description,
			Status:      ClientAPI.API.Status,
			IsSystem:    ClientAPI.API.IsSystem,
			CreatedAt:   ClientAPI.API.CreatedAt,
			UpdatedAt:   ClientAPI.API.UpdatedAt,
		}

		// Convert permissions to service data
		permissions := make([]PermissionServiceDataResult, len(ClientAPI.Permissions))
		for j, ClientPermission := range ClientAPI.Permissions {
			if ClientPermission.Permission != nil {
				permissions[j] = PermissionServiceDataResult{
					PermissionUUID: ClientPermission.Permission.PermissionUUID,
					Name:           ClientPermission.Permission.Name,
					Description:    ClientPermission.Permission.Description,
					Status:         ClientPermission.Permission.Status,
					IsDefault:      ClientPermission.Permission.IsDefault,
					IsSystem:       ClientPermission.Permission.IsSystem,
					CreatedAt:      ClientPermission.Permission.CreatedAt,
					UpdatedAt:      ClientPermission.Permission.UpdatedAt,
				}
			}
		}

		results[i] = ClientAPIServiceDataResult{
			ClientAPIUUID: ClientAPI.ClientAPIUUID,
			Api:           apiResult,
			Permissions:   permissions,
			CreatedAt:     ClientAPI.CreatedAt,
		}
	}

	return results, nil
}

// Add APIs to auth client
func (s *clientService) AddClientAPIs(tenantID int64, ClientUUID uuid.UUID, apiUUIDs []uuid.UUID) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		txClientRepo := s.clientRepo.WithTx(tx)
		txClientAPIRepo := s.clientAPIRepo.WithTx(tx)
		apiRepo := s.apiRepo.WithTx(tx)

		// Get auth client with identity provider
		Client, err := txClientRepo.FindByUUID(ClientUUID, "IdentityProvider")
		if err != nil {
			return err
		}
		if Client == nil {
			return errors.New("auth client not found")
		}

		// Validate tenant access
		if Client.IdentityProvider == nil || Client.IdentityProvider.TenantID != tenantID {
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
			existing, err := txClientAPIRepo.FindByClientAndAPI(Client.ClientID, api.APIID)
			if err != nil {
				return err
			}
			if existing != nil {
				return errors.New("API already assigned to auth client: " + apiUUID.String())
			}

			// Create new auth client API relationship
			ClientAPI := &model.ClientAPI{
				ClientAPIUUID: uuid.New(),
				ClientID:      Client.ClientID,
				APIID:         api.APIID,
			}

			_, err = txClientAPIRepo.Create(ClientAPI)
			if err != nil {
				// Check if it's a unique constraint violation
				if strings.Contains(err.Error(), "uq_client_apis_client_api") {
					return errors.New("API already assigned to auth client: " + apiUUID.String())
				}
				return err
			}
		}

		return nil
	})
}

// Remove API from auth client
func (s *clientService) RemoveClientAPI(tenantID int64, ClientUUID uuid.UUID, apiUUID uuid.UUID) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		txClientRepo := s.clientRepo.WithTx(tx)
		txClientAPIRepo := s.clientAPIRepo.WithTx(tx)

		// Get auth client with identity provider
		Client, err := txClientRepo.FindByUUID(ClientUUID, "IdentityProvider")
		if err != nil {
			return err
		}
		if Client == nil {
			return errors.New("auth client not found")
		}

		// Validate tenant access
		if Client.IdentityProvider == nil || Client.IdentityProvider.TenantID != tenantID {
			return errors.New("unauthorized access to auth client")
		}

		// Remove the API relationship (this will cascade delete permissions)
		err = txClientAPIRepo.RemoveByClientUUIDAndAPIUUID(ClientUUID, apiUUID)
		if err != nil {
			return err
		}

		return nil
	})
}

// Get permissions for a specific API assigned to auth client
func (s *clientService) GetClientAPIPermissions(tenantID int64, ClientUUID uuid.UUID, apiUUID uuid.UUID) ([]PermissionServiceDataResult, error) {
	// Get auth client API relationship
	ClientAPI, err := s.clientAPIRepo.FindByClientUUIDAndAPIUUID(ClientUUID, apiUUID)
	if err != nil {
		return nil, err
	}
	if ClientAPI == nil {
		return nil, errors.New("auth client API relationship not found")
	}

	// Validate tenant access
	Client, err := s.clientRepo.FindByUUID(ClientUUID, "IdentityProvider")
	if err != nil {
		return nil, err
	}
	if Client == nil {
		return nil, errors.New("auth client not found")
	}
	if Client.IdentityProvider == nil || Client.IdentityProvider.TenantID != tenantID {
		return nil, errors.New("unauthorized access to auth client")
	}

	// Get permissions for this auth client API
	ClientPermissions, err := s.clientPermissionRepo.FindByClientAPIID(ClientAPI.ClientAPIID)
	if err != nil {
		return nil, err
	}

	// Convert to service data results
	results := make([]PermissionServiceDataResult, len(ClientPermissions))
	for i, ClientPermission := range ClientPermissions {
		results[i] = PermissionServiceDataResult{
			PermissionUUID: ClientPermission.Permission.PermissionUUID,
			Name:           ClientPermission.Permission.Name,
			Description:    ClientPermission.Permission.Description,
			Status:         ClientPermission.Permission.Status,
			IsDefault:      ClientPermission.Permission.IsDefault,
			IsSystem:       ClientPermission.Permission.IsSystem,
			CreatedAt:      ClientPermission.Permission.CreatedAt,
			UpdatedAt:      ClientPermission.Permission.UpdatedAt,
		}
	}

	return results, nil
}

// Add permissions to a specific API for auth client
func (s *clientService) AddClientAPIPermissions(tenantID int64, ClientUUID uuid.UUID, apiUUID uuid.UUID, permissionUUIDs []uuid.UUID) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		txClientRepo := s.clientRepo.WithTx(tx)
		txClientAPIRepo := s.clientAPIRepo.WithTx(tx)
		txClientPermissionRepo := s.clientPermissionRepo.WithTx(tx)
		permissionRepo := s.permissionRepo.WithTx(tx)

		// Get auth client with identity provider
		Client, err := txClientRepo.FindByUUID(ClientUUID, "IdentityProvider")
		if err != nil {
			return err
		}
		if Client == nil {
			return errors.New("auth client not found")
		}

		// Validate tenant access
		if Client.IdentityProvider == nil || Client.IdentityProvider.TenantID != tenantID {
			return errors.New("unauthorized access to auth client")
		}

		// Get auth client API relationship
		ClientAPI, err := txClientAPIRepo.FindByClientUUIDAndAPIUUID(ClientUUID, apiUUID)
		if err != nil {
			return err
		}
		if ClientAPI == nil {
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
			existing, err := txClientPermissionRepo.FindByClientAPIAndPermission(ClientAPI.ClientAPIID, permission.PermissionID)
			if err != nil {
				return err
			}
			if existing != nil {
				return errors.New("permission already assigned to auth client API: " + permissionUUID.String())
			}

			// Create new auth client permission relationship
			ClientPermission := &model.ClientPermission{
				ClientPermissionUUID: uuid.New(),
				ClientAPIID:          ClientAPI.ClientAPIID,
				PermissionID:         permission.PermissionID,
			}

			_, err = txClientPermissionRepo.Create(ClientPermission)
			if err != nil {
				// Check if it's a unique constraint violation
				if strings.Contains(err.Error(), "uq_client_permissions_client_permission") {
					return errors.New("permission already assigned to auth client API: " + permissionUUID.String())
				}
				return err
			}
		}

		return nil
	})
}

// Remove permission from a specific API for auth client
func (s *clientService) RemoveClientAPIPermission(tenantID int64, ClientUUID uuid.UUID, apiUUID uuid.UUID, permissionUUID uuid.UUID) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		txClientRepo := s.clientRepo.WithTx(tx)
		txClientAPIRepo := s.clientAPIRepo.WithTx(tx)
		txClientPermissionRepo := s.clientPermissionRepo.WithTx(tx)
		permissionRepo := s.permissionRepo.WithTx(tx)

		// Get auth client with identity provider
		Client, err := txClientRepo.FindByUUID(ClientUUID, "IdentityProvider")
		if err != nil {
			return err
		}
		if Client == nil {
			return errors.New("auth client not found")
		}

		// Validate tenant access
		if Client.IdentityProvider == nil || Client.IdentityProvider.TenantID != tenantID {
			return errors.New("unauthorized access to auth client")
		}

		// Get auth client API relationship
		ClientAPI, err := txClientAPIRepo.FindByClientUUIDAndAPIUUID(ClientUUID, apiUUID)
		if err != nil {
			return err
		}
		if ClientAPI == nil {
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
		err = txClientPermissionRepo.RemoveByClientAPIAndPermission(ClientAPI.ClientAPIID, permission.PermissionID)
		if err != nil {
			return err
		}

		return nil
	})
}
