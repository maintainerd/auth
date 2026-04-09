package service

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/apperror"
	"github.com/maintainerd/auth/internal/crypto"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/repository"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
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
	Get(ctx context.Context, filter ClientServiceGetFilter) (*ClientServiceGetResult, error)
	GetByUUID(ctx context.Context, ClientUUID uuid.UUID, tenantID int64) (*ClientServiceDataResult, error)
	GetSecretByUUID(ctx context.Context, ClientUUID uuid.UUID, tenantID int64) (*ClientSecretServiceDataResult, error)
	GetConfigByUUID(ctx context.Context, ClientUUID uuid.UUID, tenantID int64) (datatypes.JSON, error)
	Create(ctx context.Context, tenantID int64, name string, displayName string, clientType string, domain string, config datatypes.JSON, status string, isDefault bool, identityProviderUUID string, actorUserUUID uuid.UUID) (*ClientServiceDataResult, error)
	Update(ctx context.Context, ClientUUID uuid.UUID, tenantID int64, name string, displayName string, clientType string, domain string, config datatypes.JSON, status string, isDefault bool, actorUserUUID uuid.UUID) (*ClientServiceDataResult, error)
	SetStatusByUUID(ctx context.Context, ClientUUID uuid.UUID, tenantID int64, status string, actorUserUUID uuid.UUID) (*ClientServiceDataResult, error)
	DeleteByUUID(ctx context.Context, ClientUUID uuid.UUID, tenantID int64, actorUserUUID uuid.UUID) (*ClientServiceDataResult, error)
	CreateURI(ctx context.Context, ClientUUID uuid.UUID, tenantID int64, uri string, uriType string, actorUserUUID uuid.UUID) (*ClientServiceDataResult, error)
	UpdateURI(ctx context.Context, ClientUUID uuid.UUID, tenantID int64, ClientURIUUID uuid.UUID, uri string, uriType string, actorUserUUID uuid.UUID) (*ClientServiceDataResult, error)
	DeleteURI(ctx context.Context, ClientUUID uuid.UUID, tenantID int64, ClientURIUUID uuid.UUID, actorUserUUID uuid.UUID) (*ClientServiceDataResult, error)

	// Auth Client API methods
	GetClientAPIs(ctx context.Context, tenantID int64, ClientUUID uuid.UUID) ([]ClientAPIServiceDataResult, error)
	AddClientAPIs(ctx context.Context, tenantID int64, ClientUUID uuid.UUID, apiUUIDs []uuid.UUID) error
	RemoveClientAPI(ctx context.Context, tenantID int64, ClientUUID uuid.UUID, apiUUID uuid.UUID) error

	// Auth Client API Permission methods
	GetClientAPIPermissions(ctx context.Context, tenantID int64, ClientUUID uuid.UUID, apiUUID uuid.UUID) ([]PermissionServiceDataResult, error)
	AddClientAPIPermissions(ctx context.Context, tenantID int64, ClientUUID uuid.UUID, apiUUID uuid.UUID, permissionUUIDs []uuid.UUID) error
	RemoveClientAPIPermission(ctx context.Context, tenantID int64, ClientUUID uuid.UUID, apiUUID uuid.UUID, permissionUUID uuid.UUID) error
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

func (s *clientService) Get(ctx context.Context, filter ClientServiceGetFilter) (*ClientServiceGetResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "client.list")
	defer span.End()
	span.SetAttributes(attribute.Int64("tenant.id", filter.TenantID))

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
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to fetch clients")
		return nil, err
	}

	// Build response data
	resData := make([]ClientServiceDataResult, len(result.Data))
	for i, rdata := range result.Data {
		resData[i] = *ToClientServiceDataResult(&rdata)
	}

	span.SetStatus(codes.Ok, "")
	return &ClientServiceGetResult{
		Data:       resData,
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}, nil
}

func (s *clientService) GetByUUID(ctx context.Context, ClientUUID uuid.UUID, tenantID int64) (*ClientServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "client.get")
	defer span.End()
	span.SetAttributes(
		attribute.String("client.uuid", ClientUUID.String()),
		attribute.Int64("tenant.id", tenantID),
	)

	Client, err := s.clientRepo.FindByUUIDAndTenantID(ClientUUID, tenantID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to fetch client")
		return nil, err
	}
	if Client == nil {
		span.SetStatus(codes.Error, "auth client not found or access denied")
		return nil, apperror.NewNotFoundWithReason("auth client not found or access denied")
	}

	span.SetStatus(codes.Ok, "")
	return ToClientServiceDataResult(Client), nil
}

func (s *clientService) GetSecretByUUID(ctx context.Context, ClientUUID uuid.UUID, tenantID int64) (*ClientSecretServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "client.getSecret")
	defer span.End()
	span.SetAttributes(
		attribute.String("client.uuid", ClientUUID.String()),
		attribute.Int64("tenant.id", tenantID),
	)

	Client, err := s.clientRepo.FindByUUIDAndTenantID(ClientUUID, tenantID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to fetch client")
		return nil, err
	}
	if Client == nil {
		span.SetStatus(codes.Error, "auth client not found or access denied")
		return nil, apperror.NewNotFoundWithReason("auth client not found or access denied")
	}

	span.SetStatus(codes.Ok, "")
	return &ClientSecretServiceDataResult{
		ClientID:     *Client.Identifier,
		ClientSecret: Client.Secret,
	}, nil
}

func (s *clientService) GetConfigByUUID(ctx context.Context, ClientUUID uuid.UUID, tenantID int64) (datatypes.JSON, error) {
	_, span := otel.Tracer("service").Start(ctx, "client.getConfig")
	defer span.End()
	span.SetAttributes(
		attribute.String("client.uuid", ClientUUID.String()),
		attribute.Int64("tenant.id", tenantID),
	)

	Client, err := s.clientRepo.FindByUUIDAndTenantID(ClientUUID, tenantID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to fetch client")
		return nil, err
	}
	if Client == nil {
		span.SetStatus(codes.Error, "auth client not found or access denied")
		return nil, apperror.NewNotFoundWithReason("auth client not found or access denied")
	}

	span.SetStatus(codes.Ok, "")
	return Client.Config, nil
}

func (s *clientService) Create(ctx context.Context, tenantID int64, name string, displayName string, clientType string, domain string, config datatypes.JSON, status string, isDefault bool, identityProviderUUID string, actorUserUUID uuid.UUID) (*ClientServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "client.create")
	defer span.End()
	span.SetAttributes(
		attribute.Int64("tenant.id", tenantID),
		attribute.String("client.name", name),
	)

	var createdClient *model.Client

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txClientRepo := s.clientRepo.WithTx(tx)
		txIdpRepo := s.idpRepo.WithTx(tx)
		txUserRepo := s.userRepo.WithTx(tx)

		// Parse identity provider UUID
		idpUUIDParsed, err := uuid.Parse(identityProviderUUID)
		if err != nil {
			return apperror.NewValidation("invalid identity provider UUID")
		}

		// Check if identity provider exists
		identityProvider, err := txIdpRepo.FindByUUID(idpUUIDParsed, "Tenant")
		if err != nil || identityProvider == nil {
			return apperror.NewNotFoundWithReason("identity provider not found")
		}

		// Get actor user with tenant info
		actorUser, err := txUserRepo.FindByUUID(actorUserUUID, "UserIdentities.Tenant")
		if err != nil || actorUser == nil {
			return apperror.NewNotFoundWithReason("actor user not found")
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
			return apperror.NewConflict(name + " auth client already exists")
		}

		// Generate identifier
		clientId, err := crypto.GenerateIdentifier(12)
		if err != nil {
			return err
		}
		clientSecret, err := crypto.GenerateIdentifier(64)
		if err != nil {
			return err
		}

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
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to create client")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return ToClientServiceDataResult(createdClient), nil
}

func (s *clientService) Update(ctx context.Context, ClientUUID uuid.UUID, tenantID int64, name string, displayName string, clientType string, domain string, config datatypes.JSON, status string, isDefault bool, actorUserUUID uuid.UUID) (*ClientServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "client.update")
	defer span.End()
	span.SetAttributes(
		attribute.String("client.uuid", ClientUUID.String()),
		attribute.Int64("tenant.id", tenantID),
	)

	var updatedClient *model.Client

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txClientRepo := s.clientRepo.WithTx(tx)
		txUserRepo := s.userRepo.WithTx(tx)

		// Get auth client
		Client, err := txClientRepo.FindByUUID(ClientUUID, "IdentityProvider.Tenant", "ClientURIs")
		if err != nil || Client == nil {
			return apperror.NewNotFoundWithReason("auth client not found")
		}

		// Get actor user with tenant info
		actorUser, err := txUserRepo.FindByUUID(actorUserUUID, "UserIdentities.Tenant")
		if err != nil || actorUser == nil {
			return apperror.NewNotFoundWithReason("actor user not found")
		}

		// Validate tenant access permissions
		if err := ValidateTenantAccess(actorUser, Client.IdentityProvider.Tenant); err != nil {
			return err
		}

		// Check if default
		if Client.IsDefault {
			return apperror.NewValidation("default auth client cannot cannot be updated")
		}

		// Check if auth client already exist
		if Client.Name != name {
			existingClient, err := txClientRepo.FindByNameAndIdentityProvider(name, Client.IdentityProviderID, tenantID)
			if err != nil {
				return err
			}
			if existingClient != nil && existingClient.ClientUUID != ClientUUID {
				return apperror.NewConflict(name + " auth client already exists")
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
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to update client")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return ToClientServiceDataResult(updatedClient), nil
}

func (s *clientService) SetStatusByUUID(ctx context.Context, ClientUUID uuid.UUID, tenantID int64, status string, actorUserUUID uuid.UUID) (*ClientServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "client.setStatus")
	defer span.End()
	span.SetAttributes(
		attribute.String("client.uuid", ClientUUID.String()),
		attribute.Int64("tenant.id", tenantID),
		attribute.String("client.status", status),
	)

	var updatedClient *model.Client

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txClientRepo := s.clientRepo.WithTx(tx)
		txUserRepo := s.userRepo.WithTx(tx)

		// Get auth client
		Client, err := txClientRepo.FindByUUID(ClientUUID, "IdentityProvider.Tenant", "ClientURIs")
		if err != nil || Client == nil {
			return apperror.NewNotFoundWithReason("auth client not found")
		}

		// Get actor user with tenant info
		actorUser, err := txUserRepo.FindByUUID(actorUserUUID, "UserIdentities.Tenant")
		if err != nil || actorUser == nil {
			return apperror.NewNotFoundWithReason("actor user not found")
		}

		// Validate tenant access permissions
		if err := ValidateTenantAccess(actorUser, Client.IdentityProvider.Tenant); err != nil {
			return err
		}

		// Check if default or system
		if Client.IsDefault {
			return apperror.NewValidation("default auth client cannot be updated")
		}
		if Client.IsSystem {
			return apperror.NewValidation("system auth client cannot be updated")
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
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to update client status")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return ToClientServiceDataResult(updatedClient), nil
}

func (s *clientService) DeleteByUUID(ctx context.Context, ClientUUID uuid.UUID, tenantID int64, actorUserUUID uuid.UUID) (*ClientServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "client.delete")
	defer span.End()
	span.SetAttributes(
		attribute.String("client.uuid", ClientUUID.String()),
		attribute.Int64("tenant.id", tenantID),
	)

	var deletedClient *model.Client

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txClientRepo := s.clientRepo.WithTx(tx)
		txUserRepo := s.userRepo.WithTx(tx)

		// Get auth client
		Client, err := txClientRepo.FindByUUID(ClientUUID, "IdentityProvider.Tenant", "ClientURIs")
		if err != nil || Client == nil {
			return apperror.NewNotFoundWithReason("auth client not found")
		}

		// Get actor user with tenant info
		actorUser, err := txUserRepo.FindByUUID(actorUserUUID, "UserIdentities.Tenant")
		if err != nil || actorUser == nil {
			return apperror.NewNotFoundWithReason("actor user not found")
		}

		// Validate tenant access permissions
		if err := ValidateTenantAccess(actorUser, Client.IdentityProvider.Tenant); err != nil {
			return err
		}

		// Check if default
		if Client.IsDefault {
			return apperror.NewValidation("default auth client cannot be deleted")
		}

		// Delete
		if err := txClientRepo.DeleteByUUID(ClientUUID); err != nil {
			return err
		}

		deletedClient = Client

		return nil
	})

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to delete client")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return ToClientServiceDataResult(deletedClient), nil
}

func (s *clientService) CreateURI(ctx context.Context, ClientUUID uuid.UUID, tenantID int64, uri string, uriType string, actorUserUUID uuid.UUID) (*ClientServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "client.createURI")
	defer span.End()
	span.SetAttributes(
		attribute.String("client.uuid", ClientUUID.String()),
		attribute.Int64("tenant.id", tenantID),
	)

	var createdClient *model.Client

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txClientRepo := s.clientRepo.WithTx(tx)
		txURIRepo := s.clientURIRepo.WithTx(tx)
		txUserRepo := s.userRepo.WithTx(tx)

		// Find the auth client by UUID and tenant
		Client, err := txClientRepo.FindByUUIDAndTenantID(ClientUUID, tenantID)
		if err != nil || Client == nil {
			return apperror.NewNotFoundWithReason("auth client not found or access denied")
		}

		// Get actor user with tenant info
		actorUser, err := txUserRepo.FindByUUID(actorUserUUID, "UserIdentities.Tenant")
		if err != nil || actorUser == nil {
			return apperror.NewNotFoundWithReason("actor user not found")
		}

		// Validate tenant ownership
		if Client.TenantID != tenantID {
			return apperror.NewNotFoundWithReason("auth client not found or access denied")
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
			return apperror.NewNotFoundWithReason("auth client not found")
		}

		createdClient = ClientCreated

		return nil
	})

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to create client uri")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return ToClientServiceDataResult(createdClient), nil
}

func (s *clientService) UpdateURI(ctx context.Context, ClientUUID uuid.UUID, tenantID int64, ClientURIUUID uuid.UUID, uri string, uriType string, actorUserUUID uuid.UUID) (*ClientServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "client.updateURI")
	defer span.End()
	span.SetAttributes(
		attribute.String("client.uuid", ClientUUID.String()),
		attribute.Int64("tenant.id", tenantID),
	)

	var updatedClient *model.Client

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txClientRepo := s.clientRepo.WithTx(tx)
		txURIRepo := s.clientURIRepo.WithTx(tx)
		txUserRepo := s.userRepo.WithTx(tx)

		// Find the auth client by UUID and tenant
		Client, err := txClientRepo.FindByUUIDAndTenantID(ClientUUID, tenantID)
		if err != nil || Client == nil {
			return apperror.NewNotFoundWithReason("auth client not found or access denied")
		}

		// Get actor user with tenant info
		actorUser, err := txUserRepo.FindByUUID(actorUserUUID, "UserIdentities.Tenant")
		if err != nil || actorUser == nil {
			return apperror.NewNotFoundWithReason("actor user not found")
		}

		// Validate tenant ownership
		if Client.TenantID != tenantID {
			return apperror.NewNotFoundWithReason("auth client not found or access denied")
		}

		// Find the URI entry by UUID and tenant
		existingURI, err := txURIRepo.FindByUUIDAndTenantID(ClientURIUUID.String(), tenantID)
		if err != nil || existingURI == nil {
			return apperror.NewNotFoundWithReason("URI not found or access denied")
		}

		// Check if the URI belongs to the auth client
		if existingURI.ClientID != Client.ClientID {
			return apperror.NewValidation("URI does not belong to the specified auth client")
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
			return apperror.NewNotFoundWithReason("auth client not found")
		}

		updatedClient = ClientUpdated

		return nil
	})

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to update client uri")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return ToClientServiceDataResult(updatedClient), nil
}

func (s *clientService) DeleteURI(ctx context.Context, ClientUUID uuid.UUID, tenantID int64, ClientURIUUID uuid.UUID, actorUserUUID uuid.UUID) (*ClientServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "client.deleteURI")
	defer span.End()
	span.SetAttributes(
		attribute.String("client.uuid", ClientUUID.String()),
		attribute.Int64("tenant.id", tenantID),
	)

	var deletedClient *model.Client

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txClientRepo := s.clientRepo.WithTx(tx)
		txURIRepo := s.clientURIRepo.WithTx(tx)
		txUserRepo := s.userRepo.WithTx(tx)

		// Find the auth client by UUID and tenant
		Client, err := txClientRepo.FindByUUIDAndTenantID(ClientUUID, tenantID)
		if err != nil || Client == nil {
			return apperror.NewNotFoundWithReason("auth client not found or access denied")
		}

		// Get actor user with tenant info
		actorUser, err := txUserRepo.FindByUUID(actorUserUUID, "UserIdentities.Tenant")
		if err != nil || actorUser == nil {
			return apperror.NewNotFoundWithReason("actor user not found")
		}

		// Validate tenant ownership
		if Client.TenantID != tenantID {
			return apperror.NewNotFoundWithReason("auth client not found or access denied")
		}

		// Find the URI entry by UUID and tenant
		existingURI, err := txURIRepo.FindByUUIDAndTenantID(ClientURIUUID.String(), tenantID)
		if err != nil || existingURI == nil {
			return apperror.NewNotFoundWithReason("URI not found or access denied")
		}

		// Check if the URI belongs to the auth client
		if existingURI.ClientID != Client.ClientID {
			return apperror.NewValidation("URI does not belong to the specified auth client")
		}

		// Delete the entry
		if err := txURIRepo.DeleteByUUIDAndTenantID(ClientURIUUID.String(), tenantID); err != nil {
			return err
		}

		deletedClient = Client

		return nil
	})

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to delete client uri")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
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

	return result
}

// Get APIs assigned to auth client
func (s *clientService) GetClientAPIs(ctx context.Context, tenantID int64, ClientUUID uuid.UUID) ([]ClientAPIServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "client.getAPIs")
	defer span.End()
	span.SetAttributes(
		attribute.String("client.uuid", ClientUUID.String()),
		attribute.Int64("tenant.id", tenantID),
	)

	// Get auth client APIs from repository
	ClientAPIs, err := s.clientAPIRepo.FindByClientUUID(ClientUUID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to fetch client apis")
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

	span.SetStatus(codes.Ok, "")
	return results, nil
}

// Add APIs to auth client
func (s *clientService) AddClientAPIs(ctx context.Context, tenantID int64, ClientUUID uuid.UUID, apiUUIDs []uuid.UUID) error {
	_, span := otel.Tracer("service").Start(ctx, "client.addAPIs")
	defer span.End()
	span.SetAttributes(
		attribute.String("client.uuid", ClientUUID.String()),
		attribute.Int64("tenant.id", tenantID),
	)

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txClientRepo := s.clientRepo.WithTx(tx)
		txClientAPIRepo := s.clientAPIRepo.WithTx(tx)
		apiRepo := s.apiRepo.WithTx(tx)

		// Get auth client with identity provider
		Client, err := txClientRepo.FindByUUID(ClientUUID, "IdentityProvider")
		if err != nil {
			return err
		}
		if Client == nil {
			return apperror.NewNotFoundWithReason("auth client not found")
		}

		// Validate tenant access
		if Client.IdentityProvider == nil || Client.IdentityProvider.TenantID != tenantID {
			return apperror.NewForbidden("unauthorized access to auth client")
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
			existing, err := txClientAPIRepo.FindByClientAndAPI(Client.ClientID, api.APIID)
			if err != nil {
				return err
			}
			if existing != nil {
				return apperror.NewConflict("API already assigned to auth client: " + apiUUID.String())
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
					return apperror.NewConflict("API already assigned to auth client: " + apiUUID.String())
				}
				return err
			}
		}

		return nil
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to add apis to client")
		return err
	}
	span.SetStatus(codes.Ok, "")
	return nil
}

// Remove API from auth client
func (s *clientService) RemoveClientAPI(ctx context.Context, tenantID int64, ClientUUID uuid.UUID, apiUUID uuid.UUID) error {
	_, span := otel.Tracer("service").Start(ctx, "client.removeAPI")
	defer span.End()
	span.SetAttributes(
		attribute.String("client.uuid", ClientUUID.String()),
		attribute.Int64("tenant.id", tenantID),
		attribute.String("api.uuid", apiUUID.String()),
	)

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txClientRepo := s.clientRepo.WithTx(tx)
		txClientAPIRepo := s.clientAPIRepo.WithTx(tx)

		// Get auth client with identity provider
		Client, err := txClientRepo.FindByUUID(ClientUUID, "IdentityProvider")
		if err != nil {
			return err
		}
		if Client == nil {
			return apperror.NewNotFoundWithReason("auth client not found")
		}

		// Validate tenant access
		if Client.IdentityProvider == nil || Client.IdentityProvider.TenantID != tenantID {
			return apperror.NewForbidden("unauthorized access to auth client")
		}

		// Remove the API relationship (this will cascade delete permissions)
		err = txClientAPIRepo.RemoveByClientUUIDAndAPIUUID(ClientUUID, apiUUID)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to remove api from client")
		return err
	}
	span.SetStatus(codes.Ok, "")
	return nil
}

// Get permissions for a specific API assigned to auth client
func (s *clientService) GetClientAPIPermissions(ctx context.Context, tenantID int64, ClientUUID uuid.UUID, apiUUID uuid.UUID) ([]PermissionServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "client.getAPIPermissions")
	defer span.End()
	span.SetAttributes(
		attribute.String("client.uuid", ClientUUID.String()),
		attribute.Int64("tenant.id", tenantID),
		attribute.String("api.uuid", apiUUID.String()),
	)

	// Get auth client API relationship
	ClientAPI, err := s.clientAPIRepo.FindByClientUUIDAndAPIUUID(ClientUUID, apiUUID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to fetch client api relationship")
		return nil, err
	}
	if ClientAPI == nil {
		span.SetStatus(codes.Error, "auth client API relationship not found")
		return nil, apperror.NewNotFoundWithReason("auth client API relationship not found")
	}

	// Validate tenant access
	Client, err := s.clientRepo.FindByUUID(ClientUUID, "IdentityProvider")
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to fetch client")
		return nil, err
	}
	if Client == nil {
		span.SetStatus(codes.Error, "auth client not found")
		return nil, apperror.NewNotFoundWithReason("auth client not found")
	}
	if Client.IdentityProvider == nil || Client.IdentityProvider.TenantID != tenantID {
		span.SetStatus(codes.Error, "unauthorized access to auth client")
		return nil, apperror.NewForbidden("unauthorized access to auth client")
	}

	// Get permissions for this auth client API
	ClientPermissions, err := s.clientPermissionRepo.FindByClientAPIID(ClientAPI.ClientAPIID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to fetch permissions")
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

	span.SetStatus(codes.Ok, "")
	return results, nil
}

// Add permissions to a specific API for auth client
func (s *clientService) AddClientAPIPermissions(ctx context.Context, tenantID int64, ClientUUID uuid.UUID, apiUUID uuid.UUID, permissionUUIDs []uuid.UUID) error {
	_, span := otel.Tracer("service").Start(ctx, "client.addAPIPermissions")
	defer span.End()
	span.SetAttributes(
		attribute.String("client.uuid", ClientUUID.String()),
		attribute.Int64("tenant.id", tenantID),
		attribute.String("api.uuid", apiUUID.String()),
	)

	err := s.db.Transaction(func(tx *gorm.DB) error {
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
			return apperror.NewNotFoundWithReason("auth client not found")
		}

		// Validate tenant access
		if Client.IdentityProvider == nil || Client.IdentityProvider.TenantID != tenantID {
			return apperror.NewForbidden("unauthorized access to auth client")
		}

		// Get auth client API relationship
		ClientAPI, err := txClientAPIRepo.FindByClientUUIDAndAPIUUID(ClientUUID, apiUUID)
		if err != nil {
			return err
		}
		if ClientAPI == nil {
			return apperror.NewNotFoundWithReason("auth client API relationship not found")
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

			// Check if relationship already exists
			existing, err := txClientPermissionRepo.FindByClientAPIAndPermission(ClientAPI.ClientAPIID, permission.PermissionID)
			if err != nil {
				return err
			}
			if existing != nil {
				return apperror.NewConflict("permission already assigned to auth client API: " + permissionUUID.String())
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
					return apperror.NewConflict("permission already assigned to auth client API: " + permissionUUID.String())
				}
				return err
			}
		}

		return nil
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to add permissions to client api")
		return err
	}
	span.SetStatus(codes.Ok, "")
	return nil
}

// Remove permission from a specific API for auth client
func (s *clientService) RemoveClientAPIPermission(ctx context.Context, tenantID int64, ClientUUID uuid.UUID, apiUUID uuid.UUID, permissionUUID uuid.UUID) error {
	_, span := otel.Tracer("service").Start(ctx, "client.removeAPIPermission")
	defer span.End()
	span.SetAttributes(
		attribute.String("client.uuid", ClientUUID.String()),
		attribute.Int64("tenant.id", tenantID),
		attribute.String("api.uuid", apiUUID.String()),
		attribute.String("permission.uuid", permissionUUID.String()),
	)

	err := s.db.Transaction(func(tx *gorm.DB) error {
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
			return apperror.NewNotFoundWithReason("auth client not found")
		}

		// Validate tenant access
		if Client.IdentityProvider == nil || Client.IdentityProvider.TenantID != tenantID {
			return apperror.NewForbidden("unauthorized access to auth client")
		}

		// Get auth client API relationship
		ClientAPI, err := txClientAPIRepo.FindByClientUUIDAndAPIUUID(ClientUUID, apiUUID)
		if err != nil {
			return err
		}
		if ClientAPI == nil {
			return apperror.NewNotFoundWithReason("auth client API relationship not found")
		}

		// Get permission
		permission, err := permissionRepo.FindByUUID(permissionUUID)
		if err != nil {
			return err
		}
		if permission == nil {
			return apperror.NewNotFound("permission not found")
		}

		// Remove the permission relationship
		err = txClientPermissionRepo.RemoveByClientAPIAndPermission(ClientAPI.ClientAPIID, permission.PermissionID)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to remove permission from client api")
		return err
	}
	span.SetStatus(codes.Ok, "")
	return nil
}
