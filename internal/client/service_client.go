package client

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/authevent"
	"github.com/maintainerd/maintainerd-auth/internal/event"
	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	"github.com/maintainerd/maintainerd-auth/internal/platform/crypto"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
	"github.com/maintainerd/maintainerd-auth/internal/platform/ptr"
	"github.com/maintainerd/maintainerd-auth/internal/platform/security"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type ClientSecretServiceDataResult struct {
	ClientID     string
	ClientSecret *string // populated only on creation or rotation, never retrieved from DB
}

// ClientCreateServiceResult wraps the new client data together with the one-time
// plaintext secret. The secret is returned exactly once and cannot be retrieved later.
type ClientCreateServiceResult struct {
	Client           *ClientServiceDataResult
	ClientIdentifier string // OAuth client_id (the identifier string)
	PlaintextSecret  string
}

type ClientURIServiceDataResult struct {
	ClientURIUUID uuid.UUID
	URI           string
	Type          string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type ClientServiceDataResult struct {
	ClientUUID        uuid.UUID
	Name              string
	DisplayName       string
	ClientType        string
	Domain            *string
	ClientURIs        *[]ClientURIServiceDataResult
	IdentityProvider  *IdentityProviderServiceDataResult
	Connections       *[]ClientIdentityProviderServiceDataResult
	Permissions       *[]PermissionServiceDataResult
	Status            string
	IsDefault         bool
	IsSystem          bool
	BrandingUUID      *uuid.UUID
	AllowRegistration bool

	// OIDC Session Management
	BackchannelLogoutURI             *string
	FrontchannelLogoutURI            *string
	BackchannelLogoutSessionRequired bool
	DPoPRequired                     bool

	// Security posture / per-client overrides (nil = inherit tenant default)
	RequirePKCE            *bool
	RequiredACR            *string
	SessionIdleTimeout     *int
	SessionAbsoluteTimeout *int

	CreatedAt time.Time
	UpdatedAt time.Time
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
	// IsManagementClient reports whether the client identified by
	// clientIdentifier is a first-party management client (the seeded
	// auth-console system client) permitted to call the internal management API.
	IsManagementClient(ctx context.Context, clientIdentifier string) bool
	GetByUUID(ctx context.Context, ClientUUID uuid.UUID, tenantID int64) (*ClientServiceDataResult, error)
	// GetSecretByUUID always returns an error — secrets cannot be retrieved after creation.
	// Use RotateSecret to obtain a new secret.
	GetSecretByUUID(ctx context.Context, ClientUUID uuid.UUID, tenantID int64) (*ClientSecretServiceDataResult, error)
	GetConfigByUUID(ctx context.Context, ClientUUID uuid.UUID, tenantID int64) (datatypes.JSON, error)
	Create(ctx context.Context, tenantID int64, name string, displayName string, clientType string, domain string, config datatypes.JSON, status string, isDefault bool, identityProviderUUID string, brandingUUID *uuid.UUID, allowRegistration bool, backchannelLogoutURI *string, frontchannelLogoutURI *string, backchannelLogoutSessionRequired *bool, dPoPRequired *bool, actorUserUUID uuid.UUID) (*ClientCreateServiceResult, error)
	Update(ctx context.Context, ClientUUID uuid.UUID, tenantID int64, name string, displayName string, clientType string, domain string, config datatypes.JSON, status string, isDefault bool, brandingUUID *uuid.UUID, allowRegistration *bool, backchannelLogoutURI *string, frontchannelLogoutURI *string, backchannelLogoutSessionRequired *bool, dPoPRequired *bool, actorUserUUID uuid.UUID) (*ClientServiceDataResult, error)
	// RotateSecret generates a new secret, hashes and persists it, and keeps the old
	// hash valid for the specified grace period (gracePeriodHours=0 revokes immediately).
	// Returns the new plaintext secret once — it cannot be retrieved again.
	RotateSecret(ctx context.Context, clientUUID uuid.UUID, tenantID int64, actorUserUUID uuid.UUID, gracePeriodHours int) (string, error)
	SetStatusByUUID(ctx context.Context, ClientUUID uuid.UUID, tenantID int64, status string, actorUserUUID uuid.UUID) (*ClientServiceDataResult, error)
	DeleteByUUID(ctx context.Context, ClientUUID uuid.UUID, tenantID int64, actorUserUUID uuid.UUID) (*ClientServiceDataResult, error)
	CreateURI(ctx context.Context, ClientUUID uuid.UUID, tenantID int64, uri string, uriType string, actorUserUUID uuid.UUID) (*ClientServiceDataResult, error)
	UpdateURI(ctx context.Context, ClientUUID uuid.UUID, tenantID int64, ClientURIUUID uuid.UUID, uri string, uriType string, actorUserUUID uuid.UUID) (*ClientServiceDataResult, error)
	DeleteURI(ctx context.Context, ClientUUID uuid.UUID, tenantID int64, ClientURIUUID uuid.UUID, actorUserUUID uuid.UUID) (*ClientServiceDataResult, error)

	// Auth Client identity provider connection methods
	GetConnections(ctx context.Context, ClientUUID uuid.UUID, tenantID int64) ([]ClientIdentityProviderServiceDataResult, error)
	AddConnection(ctx context.Context, ClientUUID uuid.UUID, tenantID int64, identityProviderUUID uuid.UUID, isDefault bool, enabled bool, displayOrder int, actorUserUUID uuid.UUID) (*ClientServiceDataResult, error)
	UpdateConnection(ctx context.Context, ClientUUID uuid.UUID, tenantID int64, connectionUUID uuid.UUID, isDefault bool, enabled bool, displayOrder int, actorUserUUID uuid.UUID) (*ClientServiceDataResult, error)
	RemoveConnection(ctx context.Context, ClientUUID uuid.UUID, tenantID int64, connectionUUID uuid.UUID, actorUserUUID uuid.UUID) (*ClientServiceDataResult, error)

	// Auth Client API methods
	GetClientAPIs(ctx context.Context, tenantID int64, ClientUUID uuid.UUID) ([]ClientAPIServiceDataResult, error)
	AddClientAPIs(ctx context.Context, tenantID int64, ClientUUID uuid.UUID, apiUUIDs []uuid.UUID) error
	RemoveClientAPI(ctx context.Context, tenantID int64, ClientUUID uuid.UUID, apiUUID uuid.UUID) error

	// Auth Client API Permission methods
	GetClientAPIPermissions(ctx context.Context, tenantID int64, ClientUUID uuid.UUID, apiUUID uuid.UUID) ([]PermissionServiceDataResult, error)
	AddClientAPIPermissions(ctx context.Context, tenantID int64, ClientUUID uuid.UUID, apiUUID uuid.UUID, permissionUUIDs []uuid.UUID) error
	RemoveClientAPIPermission(ctx context.Context, tenantID int64, ClientUUID uuid.UUID, apiUUID uuid.UUID, permissionUUID uuid.UUID) error

	// Client role assignment
	AssignClientRole(ctx context.Context, ClientUUID uuid.UUID, roleUUID uuid.UUID, tenantID int64, createdBy *int64) (*ClientRole, error)
	RemoveClientRole(ctx context.Context, ClientUUID uuid.UUID, roleUUID uuid.UUID, tenantID int64) error
	ListClientRoles(ctx context.Context, ClientUUID uuid.UUID, tenantID int64) ([]ClientRole, error)
}

type ClientPublicServiceDataResult struct {
	ClientID         string
	Name             string
	DisplayName      string
	ClientType       string
	Domain           *string
	TenantIdentifier string
}

func (s *clientService) GetPublicByIdentifier(ctx context.Context, identifier string) (*ClientPublicServiceDataResult, error) {
	client, err := s.clientRepo.FindByIdentifier(strings.TrimSpace(identifier))
	if err != nil {
		return nil, err
	}
	if client == nil || client.Status != shared.StatusActive || client.Identifier == nil ||
		client.Tenant == nil {
		return nil, apperror.NewNotFound("auth client not found")
	}
	return &ClientPublicServiceDataResult{
		ClientID:         *client.Identifier,
		Name:             client.Name,
		DisplayName:      client.DisplayName,
		ClientType:       client.ClientType,
		Domain:           client.Domain,
		TenantIdentifier: client.Tenant.Name,
	}, nil
}

func (s *clientService) GetPublicConsoleByTenantIdentifier(ctx context.Context, tenantIdentifier string) (*ClientPublicServiceDataResult, error) {
	client, err := s.clientRepo.FindSystemByTenantIdentifierAndName(
		strings.TrimSpace(tenantIdentifier),
		shared.SystemClientNameAuthConsole,
	)
	if err != nil {
		return nil, err
	}
	if client == nil || client.Status != shared.StatusActive || client.Identifier == nil ||
		client.Tenant == nil {
		return nil, apperror.NewNotFound("console client not found")
	}
	return &ClientPublicServiceDataResult{
		ClientID:         *client.Identifier,
		Name:             client.Name,
		DisplayName:      client.DisplayName,
		ClientType:       client.ClientType,
		Domain:           client.Domain,
		TenantIdentifier: client.Tenant.Name,
	}, nil
}

// GetPublicIdentityByTenantIdentifier returns the tenant's seeded identity
// system client (auth-identity). It mirrors GetPublicConsoleByTenantIdentifier
// so the domain-bootstrap endpoint can advertise the correct per-surface client:
// console → auth-console, identity → auth-identity.
func (s *clientService) GetPublicIdentityByTenantIdentifier(ctx context.Context, tenantIdentifier string) (*ClientPublicServiceDataResult, error) {
	client, err := s.clientRepo.FindSystemByTenantIdentifierAndName(
		strings.TrimSpace(tenantIdentifier),
		shared.SystemClientNameAuthIdentity,
	)
	if err != nil {
		return nil, err
	}
	if client == nil || client.Status != shared.StatusActive || client.Identifier == nil ||
		client.Tenant == nil {
		return nil, apperror.NewNotFound("identity client not found")
	}
	return &ClientPublicServiceDataResult{
		ClientID:         *client.Identifier,
		Name:             client.Name,
		DisplayName:      client.DisplayName,
		ClientType:       client.ClientType,
		Domain:           client.Domain,
		TenantIdentifier: client.Tenant.Name,
	}, nil
}

// IsManagementClient reports whether clientIdentifier resolves to the seeded
// first-party management client (the auth-console system client). The internal
// management API uses this to reject tokens minted for any other client, even
// when their subject holds the required permissions.
func (s *clientService) IsManagementClient(_ context.Context, clientIdentifier string) bool {
	id := strings.TrimSpace(clientIdentifier)
	if id == "" {
		return false
	}
	c, err := s.clientRepo.FindByIdentifier(id)
	if err != nil || c == nil {
		return false
	}
	return c.IsSystem && c.Name == shared.SystemClientNameAuthConsole
}

var (
	generateClientIdentifier = crypto.GenerateIdentifier
	hashClientSecret         = security.HashClientSecret
	encryptClientSecret      = crypto.EncryptAtRest
)

type clientService struct {
	db                         *gorm.DB
	clientRepo                 ClientRepository
	clientURIRepo              ClientURIRepository
	clientIdentityProviderRepo ClientIdentityProviderRepository
	idpRepo                    IdentityProviderRepository
	permissionRepo             PermissionRepository
	clientPermissionRepo       ClientPermissionRepository
	clientAPIRepo              ClientAPIRepository
	clientRoleRepo             ClientRoleRepository
	apiRepo                    APIRepository
	roleRepo                   RoleRepository
	userRepo                   UserRepository
	tenantRepo                 TenantRepository
	authEventService           authevent.AuthEventService
	eventService               event.EventService
}

func NewClientService(
	db *gorm.DB,
	clientRepo ClientRepository,
	clientURIRepo ClientURIRepository,
	idpRepo IdentityProviderRepository,
	permissionRepo PermissionRepository,
	clientPermissionRepo ClientPermissionRepository,
	clientAPIRepo ClientAPIRepository,
	clientRoleRepo ClientRoleRepository,
	roleRepo RoleRepository,
	apiRepo APIRepository,
	userRepo UserRepository,
	tenantRepo TenantRepository,
	authEventService authevent.AuthEventService,
	eventService event.EventService,
) ClientService {
	return &clientService{
		db:            db,
		clientRepo:    clientRepo,
		clientURIRepo: clientURIRepo,
		// The connection repo is an internal detail of the client service: the
		// client_identity_providers table is owned here, so it is constructed over
		// the same db rather than injected (which would ripple through every
		// NewClientService call site for no added value).
		clientIdentityProviderRepo: NewClientIdentityProviderRepository(db),
		idpRepo:                    idpRepo,
		permissionRepo:             permissionRepo,
		clientPermissionRepo:       clientPermissionRepo,
		clientAPIRepo:              clientAPIRepo,
		clientRoleRepo:             clientRoleRepo,
		roleRepo:                   roleRepo,
		apiRepo:                    apiRepo,
		userRepo:                   userRepo,
		tenantRepo:                 tenantRepo,
		authEventService:           coalesceAuthEventService(authEventService),
		eventService:               eventService,
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
	queryFilter := ClientRepositoryGetFilter{
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
	// Secrets are hashed at rest and cannot be retrieved. Callers must rotate.
	span.SetStatus(codes.Error, "secret retrieval not supported")
	return nil, apperror.NewValidation("client secret cannot be retrieved after creation; use POST /{client_uuid}/rotate-secret to issue a new one")
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

func (s *clientService) resolveBrandingID(tx *gorm.DB, tenantID int64, brandingUUID uuid.UUID) (*int64, error) {
	var b Branding
	err := tx.Where("branding_uuid = ? AND tenant_id = ?", brandingUUID, tenantID).First(&b).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NewNotFoundWithReason("branding not found")
		}
		return nil, err
	}
	return &b.BrandingID, nil
}

func (s *clientService) Create(ctx context.Context, tenantID int64, name string, displayName string, clientType string, domain string, config datatypes.JSON, status string, isDefault bool, identityProviderUUID string, brandingUUID *uuid.UUID, allowRegistration bool, backchannelLogoutURI *string, frontchannelLogoutURI *string, backchannelLogoutSessionRequired *bool, dPoPRequired *bool, actorUserUUID uuid.UUID) (*ClientCreateServiceResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "client.create")
	defer span.End()
	span.SetAttributes(
		attribute.Int64("tenant.id", tenantID),
		attribute.String("client.name", name),
	)

	var createdClient *Client
	var plaintextSecret string
	var capturedActorID int64

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txClientRepo := s.clientRepo.WithTx(tx)
		txIdpRepo := s.idpRepo.WithTx(tx)
		txUserRepo := s.userRepo.WithTx(tx)

		identityProvider, err := s.resolveInitialIdentityProvider(tx, txIdpRepo, tenantID, identityProviderUUID)
		if err != nil {
			return err
		}

		actorUser, err := txUserRepo.FindByUUID(actorUserUUID, "UserIdentities.Tenant")
		if err != nil || actorUser == nil {
			return apperror.NewNotFoundWithReason("actor user not found")
		}

		if err := ValidateTenantAccess(actorUser, &Tenant{TenantID: tenantID}); err != nil {
			return err
		}
		capturedActorID = actorUser.UserID

		existingClient, err := txClientRepo.FindByNameAndTenantID(name, tenantID)
		if err != nil {
			return err
		}
		if existingClient != nil {
			return apperror.NewConflict(name + " auth client already exists")
		}

		clientID, err := generateClientIdentifier(12)
		if err != nil {
			return err
		}
		rawSecret, err := generateClientIdentifier(64)
		if err != nil {
			return err
		}
		secretHash, err := hashClientSecret(ctx, rawSecret)
		if err != nil {
			return err
		}
		secretEncrypted, err := encryptClientSecret(rawSecret)
		if err != nil {
			return err
		}
		plaintextSecret = rawSecret

		newClient := &Client{
			Name:            name,
			DisplayName:     displayName,
			ClientType:      clientType,
			Domain:          &domain,
			Identifier:      &clientID,
			SecretHash:      &secretHash,
			SecretEncrypted: &secretEncrypted,
			Config:          config,

			TenantID:          tenantID,
			Status:            status,
			IsDefault:         isDefault,
			IsSystem:          false,
			AllowRegistration: allowRegistration,

			BackchannelLogoutURI:  backchannelLogoutURI,
			FrontchannelLogoutURI: frontchannelLogoutURI,
		}
		if backchannelLogoutSessionRequired != nil {
			newClient.BackchannelLogoutSessionRequired = *backchannelLogoutSessionRequired
		}
		if dPoPRequired != nil {
			newClient.DPoPRequired = *dPoPRequired
		}

		if brandingUUID != nil {
			brandingID, err := s.resolveBrandingID(tx, tenantID, *brandingUUID)
			if err != nil {
				return err
			}
			newClient.BrandingID = brandingID
		}

		// Mirror OAuth settings from config into the first-class columns the
		// authorization and token-issuance paths read at runtime.
		applyConfigToClientColumns(newClient, config)

		_, err = txClientRepo.CreateOrUpdate(newClient)
		if err != nil {
			return err
		}
		if err := s.ensureClientIdentityProviderConnection(tx, newClient, identityProvider, true, true, 0, capturedActorID); err != nil {
			return err
		}

		createdClient, err = txClientRepo.FindByUUID(newClient.ClientUUID, "Tenant", "Branding", "ConnectedProviders.IdentityProvider", "ClientURIs")
		if err != nil {
			return err
		}
		if s.eventService != nil {
			if _, emitErr := s.eventService.Emit(ctx, tx, event.NewIntegrationEvent(
				event.EventTypeClientCreated, 1, tenantID,
			).SetActor(&capturedActorID).SetSubject(&createdClient.ClientUUID, "client")); emitErr != nil {
				return emitErr
			}
		}
		return nil
	})

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to create client")
		return nil, err
	}

	identifier := ""
	if createdClient.Identifier != nil {
		identifier = *createdClient.Identifier
	}

	span.SetStatus(codes.Ok, "")
	s.authEventService.Log(ctx, authevent.AuthEventInput{
		TenantID:    tenantID,
		ActorUserID: &capturedActorID,
		IPAddress:   middleware.ClientIPFromContext(ctx),
		UserAgent:   ptr.PtrOrNil(middleware.UserAgentFromContext(ctx)),
		Category:    authevent.AuthEventCategoryAuthz,
		EventType:   authevent.AuthEventTypeAuthzAdmin,
		Severity:    authevent.AuthEventSeverityInfo,
		Result:      authevent.AuthEventResultSuccess,
		Description: ptr.Ptr(fmt.Sprintf("Client created: %s", createdClient.Name)),
	})
	return &ClientCreateServiceResult{
		Client:           ToClientServiceDataResult(createdClient),
		ClientIdentifier: identifier,
		PlaintextSecret:  plaintextSecret,
	}, nil
}

func (s *clientService) resolveInitialIdentityProvider(tx *gorm.DB, repo IdentityProviderRepository, tenantID int64, identityProviderUUID string) (*IdentityProvider, error) {
	identityProviderUUID = strings.TrimSpace(identityProviderUUID)
	if identityProviderUUID != "" {
		idpUUIDParsed, err := uuid.Parse(identityProviderUUID)
		if err != nil {
			return nil, apperror.NewValidation("invalid identity provider UUID")
		}
		identityProvider, err := repo.FindByUUID(idpUUIDParsed, "Tenant")
		if err != nil || identityProvider == nil {
			return nil, apperror.NewNotFoundWithReason("identity provider not found")
		}
		providerTenantID := identityProvider.TenantID
		if providerTenantID == 0 && identityProvider.Tenant != nil {
			providerTenantID = identityProvider.Tenant.TenantID
		}
		if providerTenantID != tenantID {
			return nil, apperror.NewForbidden("identity provider does not belong to tenant")
		}
		return identityProvider, nil
	}

	var identityProvider IdentityProvider
	err := tx.
		Where("tenant_id = ? AND provider = ? AND is_system = ? AND status = ?", tenantID, shared.IDPProviderMaintainerd, true, shared.StatusActive).
		First(&identityProvider).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NewNotFoundWithReason("system identity provider not found")
		}
		return nil, err
	}
	return &identityProvider, nil
}

func (s *clientService) ensureClientIdentityProviderConnection(tx *gorm.DB, client *Client, identityProvider *IdentityProvider, isDefault bool, enabled bool, displayOrder int, actorUserID int64) error {
	if client == nil || identityProvider == nil {
		return apperror.NewValidation("client and identity provider are required")
	}
	connection := &ClientIdentityProvider{
		TenantID:           client.TenantID,
		ClientID:           client.ClientID,
		IdentityProviderID: identityProvider.IdentityProviderID,
		IsDefault:          isDefault,
		Enabled:            enabled,
		DisplayOrder:       displayOrder,
		CreatedBy:          &actorUserID,
		UpdatedBy:          &actorUserID,
	}
	var existing ClientIdentityProvider
	err := tx.
		Where("client_id = ? AND identity_provider_id = ? AND deleted_at IS NULL", client.ClientID, identityProvider.IdentityProviderID).
		First(&existing).Error
	if err == nil {
		existing.IsDefault = isDefault
		existing.Enabled = enabled
		existing.DisplayOrder = displayOrder
		existing.UpdatedBy = &actorUserID
		return tx.Save(&existing).Error
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	return tx.Create(connection).Error
}

// RotateSecret generates a new client secret, hashes it, and keeps the previous
// hash valid for gracePeriodHours (0 = revoke immediately). Returns the new
// plaintext secret exactly once — it cannot be retrieved again.
func (s *clientService) RotateSecret(ctx context.Context, clientUUID uuid.UUID, tenantID int64, actorUserUUID uuid.UUID, gracePeriodHours int) (string, error) {
	_, span := otel.Tracer("service").Start(ctx, "client.rotateSecret")
	defer span.End()
	span.SetAttributes(
		attribute.String("client.uuid", clientUUID.String()),
		attribute.Int64("tenant.id", tenantID),
		attribute.Int("grace_period_hours", gracePeriodHours),
	)

	var plaintextSecret string

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txClientRepo := s.clientRepo.WithTx(tx)

		client, err := txClientRepo.FindByUUIDAndTenantID(clientUUID, tenantID)
		if err != nil {
			return err
		}
		if client == nil {
			return apperror.NewNotFoundWithReason("auth client not found or access denied")
		}

		rawSecret, err := generateClientIdentifier(64)
		if err != nil {
			return err
		}
		newHash, err := hashClientSecret(ctx, rawSecret)
		if err != nil {
			return err
		}
		newEncrypted, err := encryptClientSecret(rawSecret)
		if err != nil {
			return err
		}
		plaintextSecret = rawSecret

		// Move current hash to previous for the grace window.
		client.PreviousSecretHash = client.SecretHash
		client.PreviousSecretEncrypted = client.SecretEncrypted
		if gracePeriodHours > 0 {
			exp := time.Now().Add(time.Duration(gracePeriodHours) * time.Hour)
			client.PreviousSecretExpiresAt = &exp
		} else {
			client.PreviousSecretHash = nil
			client.PreviousSecretEncrypted = nil
			client.PreviousSecretExpiresAt = nil
		}
		client.SecretHash = &newHash
		client.SecretEncrypted = &newEncrypted

		_, err = txClientRepo.CreateOrUpdate(client)
		if err != nil {
			return err
		}
		// Emit client.secret_rotated inside the transaction
		if s.eventService != nil {
			if _, emitErr := s.eventService.Emit(ctx, tx, event.NewIntegrationEvent(
				event.EventTypeClientSecretRotated, 1, tenantID,
			).SetSubject(&client.ClientUUID, "client")); emitErr != nil {
				return emitErr
			}
		}
		return nil
	})

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to rotate client secret")
		return "", err
	}

	span.SetStatus(codes.Ok, "")
	return plaintextSecret, nil
}

func (s *clientService) Update(ctx context.Context, ClientUUID uuid.UUID, tenantID int64, name string, displayName string, clientType string, domain string, config datatypes.JSON, status string, isDefault bool, brandingUUID *uuid.UUID, allowRegistration *bool, backchannelLogoutURI *string, frontchannelLogoutURI *string, backchannelLogoutSessionRequired *bool, dPoPRequired *bool, actorUserUUID uuid.UUID) (*ClientServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "client.update")
	defer span.End()
	span.SetAttributes(
		attribute.String("client.uuid", ClientUUID.String()),
		attribute.Int64("tenant.id", tenantID),
	)

	var updatedClient *Client
	var capturedActorID int64

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txClientRepo := s.clientRepo.WithTx(tx)
		txUserRepo := s.userRepo.WithTx(tx)

		// Get auth client
		Client, err := txClientRepo.FindByUUIDAndTenantID(ClientUUID, tenantID)
		if err != nil || Client == nil {
			return apperror.NewNotFoundWithReason("auth client not found")
		}

		// Get actor user with tenant info
		actorUser, err := txUserRepo.FindByUUID(actorUserUUID, "UserIdentities.Tenant")
		if err != nil || actorUser == nil {
			return apperror.NewNotFoundWithReason("actor user not found")
		}

		// Validate tenant access permissions
		if err := ValidateTenantAccess(actorUser, &Tenant{TenantID: tenantID}); err != nil {
			return err
		}
		capturedActorID = actorUser.UserID

		// Check if default
		if Client.IsDefault {
			return apperror.NewValidation("default auth client cannot cannot be updated")
		}

		// Check if auth client already exist
		if Client.Name != name {
			existingClient, err := txClientRepo.FindByNameAndTenantID(name, tenantID)
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
		if allowRegistration != nil {
			Client.AllowRegistration = *allowRegistration
		}
		if backchannelLogoutURI != nil {
			Client.BackchannelLogoutURI = backchannelLogoutURI
		}
		if frontchannelLogoutURI != nil {
			Client.FrontchannelLogoutURI = frontchannelLogoutURI
		}
		if backchannelLogoutSessionRequired != nil {
			Client.BackchannelLogoutSessionRequired = *backchannelLogoutSessionRequired
		}
		if dPoPRequired != nil {
			Client.DPoPRequired = *dPoPRequired
		}

		if brandingUUID == nil {
			Client.BrandingID = nil
		} else {
			brandingID, err := s.resolveBrandingID(tx, tenantID, *brandingUUID)
			if err != nil {
				return err
			}
			Client.BrandingID = brandingID
		}

		// Mirror OAuth settings from config into the first-class columns the
		// authorization and token-issuance paths read at runtime.
		applyConfigToClientColumns(Client, config)

		// Update
		_, err = txClientRepo.CreateOrUpdate(Client)
		if err != nil {
			return err
		}

		updatedClient = Client

		if s.eventService != nil {
			if _, emitErr := s.eventService.Emit(ctx, tx, event.NewIntegrationEvent(
				event.EventTypeClientUpdated, 1, tenantID,
			).SetActor(&capturedActorID).SetSubject(&updatedClient.ClientUUID, "client")); emitErr != nil {
				return emitErr
			}
		}
		return nil
	})

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to update client")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	s.authEventService.Log(ctx, authevent.AuthEventInput{
		TenantID:    tenantID,
		ActorUserID: &capturedActorID,
		IPAddress:   middleware.ClientIPFromContext(ctx),
		UserAgent:   ptr.PtrOrNil(middleware.UserAgentFromContext(ctx)),
		Category:    authevent.AuthEventCategoryAuthz,
		EventType:   authevent.AuthEventTypeAuthzAdmin,
		Severity:    authevent.AuthEventSeverityInfo,
		Result:      authevent.AuthEventResultSuccess,
		Description: ptr.Ptr(fmt.Sprintf("Client updated: %s", updatedClient.Name)),
	})
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

	var updatedClient *Client
	var capturedActorID int64

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txClientRepo := s.clientRepo.WithTx(tx)
		txUserRepo := s.userRepo.WithTx(tx)

		// Get auth client
		Client, err := txClientRepo.FindByUUIDAndTenantID(ClientUUID, tenantID)
		if err != nil || Client == nil {
			return apperror.NewNotFoundWithReason("auth client not found")
		}

		// Get actor user with tenant info
		actorUser, err := txUserRepo.FindByUUID(actorUserUUID, "UserIdentities.Tenant")
		if err != nil || actorUser == nil {
			return apperror.NewNotFoundWithReason("actor user not found")
		}

		// Validate tenant access permissions
		if err := ValidateTenantAccess(actorUser, &Tenant{TenantID: tenantID}); err != nil {
			return err
		}
		capturedActorID = actorUser.UserID

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

		if s.eventService != nil {
			if _, emitErr := s.eventService.Emit(ctx, tx, event.NewIntegrationEvent(
				event.EventTypeClientStatusChanged, 1, tenantID,
			).SetActor(&capturedActorID).SetSubject(&updatedClient.ClientUUID, "client").
				SetChangedFields("status")); emitErr != nil {
				return emitErr
			}
		}
		return nil
	})

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to update client status")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	s.authEventService.Log(ctx, authevent.AuthEventInput{
		TenantID:    tenantID,
		ActorUserID: &capturedActorID,
		IPAddress:   middleware.ClientIPFromContext(ctx),
		UserAgent:   ptr.PtrOrNil(middleware.UserAgentFromContext(ctx)),
		Category:    authevent.AuthEventCategoryAuthz,
		EventType:   authevent.AuthEventTypeAuthzAdmin,
		Severity:    authevent.AuthEventSeverityInfo,
		Result:      authevent.AuthEventResultSuccess,
		Description: ptr.Ptr(fmt.Sprintf("Client status set to %s: %s", status, updatedClient.Name)),
	})
	return ToClientServiceDataResult(updatedClient), nil
}

func (s *clientService) DeleteByUUID(ctx context.Context, ClientUUID uuid.UUID, tenantID int64, actorUserUUID uuid.UUID) (*ClientServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "client.delete")
	defer span.End()
	span.SetAttributes(
		attribute.String("client.uuid", ClientUUID.String()),
		attribute.Int64("tenant.id", tenantID),
	)

	var deletedClient *Client
	var capturedActorID int64

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txClientRepo := s.clientRepo.WithTx(tx)
		txUserRepo := s.userRepo.WithTx(tx)

		// Get auth client
		Client, err := txClientRepo.FindByUUIDAndTenantID(ClientUUID, tenantID)
		if err != nil || Client == nil {
			return apperror.NewNotFoundWithReason("auth client not found")
		}

		// Get actor user with tenant info
		actorUser, err := txUserRepo.FindByUUID(actorUserUUID, "UserIdentities.Tenant")
		if err != nil || actorUser == nil {
			return apperror.NewNotFoundWithReason("actor user not found")
		}

		// Validate tenant access permissions
		if err := ValidateTenantAccess(actorUser, &Tenant{TenantID: tenantID}); err != nil {
			return err
		}
		capturedActorID = actorUser.UserID

		// Check if default
		if Client.IsDefault {
			return apperror.NewValidation("default auth client cannot be deleted")
		}

		// Delete
		if err := txClientRepo.DeleteByUUID(ClientUUID); err != nil {
			return err
		}

		deletedClient = Client

		if s.eventService != nil {
			if _, emitErr := s.eventService.Emit(ctx, tx, event.NewIntegrationEvent(
				event.EventTypeClientDeleted, 1, tenantID,
			).SetSubject(&Client.ClientUUID, "client")); emitErr != nil {
				return emitErr
			}
		}
		return nil
	})

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to delete client")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	s.authEventService.Log(ctx, authevent.AuthEventInput{
		TenantID:    tenantID,
		ActorUserID: &capturedActorID,
		IPAddress:   middleware.ClientIPFromContext(ctx),
		UserAgent:   ptr.PtrOrNil(middleware.UserAgentFromContext(ctx)),
		Category:    authevent.AuthEventCategoryAuthz,
		EventType:   authevent.AuthEventTypeAuthzAdmin,
		Severity:    authevent.AuthEventSeverityWarn,
		Result:      authevent.AuthEventResultSuccess,
		Description: ptr.Ptr(fmt.Sprintf("Client deleted: %s", deletedClient.Name)),
	})
	return ToClientServiceDataResult(deletedClient), nil
}

func (s *clientService) CreateURI(ctx context.Context, ClientUUID uuid.UUID, tenantID int64, uri string, uriType string, actorUserUUID uuid.UUID) (*ClientServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "client.createURI")
	defer span.End()
	span.SetAttributes(
		attribute.String("client.uuid", ClientUUID.String()),
		attribute.Int64("tenant.id", tenantID),
	)

	var createdClient *Client
	var capturedActorID int64

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
		capturedActorID = actorUser.UserID

		// Validate tenant ownership
		if Client.TenantID != tenantID {
			return apperror.NewNotFoundWithReason("auth client not found or access denied")
		}

		if uriType == shared.ClientURITypeRedirect || uriType == shared.ClientURITypeLogout || uriType == shared.ClientURITypeLogin {
			if err := security.ValidateRedirectURI(uri); err != nil {
				return apperror.NewValidation(err.Error())
			}
		}

		// Create the URI entry
		newURI := &ClientURI{
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
	s.authEventService.Log(ctx, authevent.AuthEventInput{
		TenantID:    tenantID,
		ActorUserID: &capturedActorID,
		IPAddress:   middleware.ClientIPFromContext(ctx),
		UserAgent:   ptr.PtrOrNil(middleware.UserAgentFromContext(ctx)),
		Category:    authevent.AuthEventCategoryAuthz,
		EventType:   authevent.AuthEventTypeAuthzAdmin,
		Severity:    authevent.AuthEventSeverityInfo,
		Result:      authevent.AuthEventResultSuccess,
		Description: ptr.Ptr(fmt.Sprintf("URI added to client: %s (%s)", uri, uriType)),
	})
	return ToClientServiceDataResult(createdClient), nil
}

func (s *clientService) UpdateURI(ctx context.Context, ClientUUID uuid.UUID, tenantID int64, ClientURIUUID uuid.UUID, uri string, uriType string, actorUserUUID uuid.UUID) (*ClientServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "client.updateURI")
	defer span.End()
	span.SetAttributes(
		attribute.String("client.uuid", ClientUUID.String()),
		attribute.Int64("tenant.id", tenantID),
	)

	var updatedClient *Client
	var capturedActorID int64

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
		capturedActorID = actorUser.UserID

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

		// Reject dangerous schemes on redirect/logout/login URIs
		if uriType == shared.ClientURITypeRedirect || uriType == shared.ClientURITypeLogout || uriType == shared.ClientURITypeLogin {
			if err := security.ValidateRedirectURI(uri); err != nil {
				return apperror.NewValidation(err.Error())
			}
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
	s.authEventService.Log(ctx, authevent.AuthEventInput{
		TenantID:    tenantID,
		ActorUserID: &capturedActorID,
		IPAddress:   middleware.ClientIPFromContext(ctx),
		UserAgent:   ptr.PtrOrNil(middleware.UserAgentFromContext(ctx)),
		Category:    authevent.AuthEventCategoryAuthz,
		EventType:   authevent.AuthEventTypeAuthzAdmin,
		Severity:    authevent.AuthEventSeverityInfo,
		Result:      authevent.AuthEventResultSuccess,
		Description: ptr.Ptr(fmt.Sprintf("Client URI updated: %s (%s)", uri, uriType)),
	})
	return ToClientServiceDataResult(updatedClient), nil
}

func (s *clientService) DeleteURI(ctx context.Context, ClientUUID uuid.UUID, tenantID int64, ClientURIUUID uuid.UUID, actorUserUUID uuid.UUID) (*ClientServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "client.deleteURI")
	defer span.End()
	span.SetAttributes(
		attribute.String("client.uuid", ClientUUID.String()),
		attribute.Int64("tenant.id", tenantID),
	)

	var deletedClient *Client
	var capturedActorID int64

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
		capturedActorID = actorUser.UserID

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
	s.authEventService.Log(ctx, authevent.AuthEventInput{
		TenantID:    tenantID,
		ActorUserID: &capturedActorID,
		IPAddress:   middleware.ClientIPFromContext(ctx),
		UserAgent:   ptr.PtrOrNil(middleware.UserAgentFromContext(ctx)),
		Category:    authevent.AuthEventCategoryAuthz,
		EventType:   authevent.AuthEventTypeAuthzAdmin,
		Severity:    authevent.AuthEventSeverityWarn,
		Result:      authevent.AuthEventResultSuccess,
		Description: ptr.Ptr(fmt.Sprintf("Client URI deleted from client: %s", deletedClient.Name)),
	})
	return ToClientServiceDataResult(deletedClient), nil
}

// GetConnections returns the identity provider connections enabled on a client.
func (s *clientService) GetConnections(ctx context.Context, ClientUUID uuid.UUID, tenantID int64) ([]ClientIdentityProviderServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "client.getConnections")
	defer span.End()
	span.SetAttributes(
		attribute.String("client.uuid", ClientUUID.String()),
		attribute.Int64("tenant.id", tenantID),
	)

	Client, err := s.clientRepo.FindByUUIDAndTenantID(ClientUUID, tenantID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to load auth client")
		return nil, err
	}
	if Client == nil {
		return nil, apperror.NewNotFoundWithReason("auth client not found or access denied")
	}

	span.SetStatus(codes.Ok, "")
	result := ToClientServiceDataResult(Client)
	if result.Connections == nil {
		return []ClientIdentityProviderServiceDataResult{}, nil
	}
	return *result.Connections, nil
}

// AddConnection connects an identity provider to a client. The provider must
// belong to the same tenant, and a provider may only be connected once per
// client. Promoting the connection to default clears any existing default first
// to satisfy the single-default-per-client constraint.
func (s *clientService) AddConnection(ctx context.Context, ClientUUID uuid.UUID, tenantID int64, identityProviderUUID uuid.UUID, isDefault bool, enabled bool, displayOrder int, actorUserUUID uuid.UUID) (*ClientServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "client.addConnection")
	defer span.End()
	span.SetAttributes(
		attribute.String("client.uuid", ClientUUID.String()),
		attribute.Int64("tenant.id", tenantID),
	)

	var updatedClient *Client
	var capturedActorID int64
	var providerName string

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txClientRepo := s.clientRepo.WithTx(tx)
		txConnectionRepo := s.clientIdentityProviderRepo.WithTx(tx)
		txIDPRepo := s.idpRepo.WithTx(tx)
		txUserRepo := s.userRepo.WithTx(tx)

		Client, err := txClientRepo.FindByUUIDAndTenantID(ClientUUID, tenantID)
		if err != nil || Client == nil {
			return apperror.NewNotFoundWithReason("auth client not found or access denied")
		}

		actorUser, err := txUserRepo.FindByUUID(actorUserUUID, "UserIdentities.Tenant")
		if err != nil || actorUser == nil {
			return apperror.NewNotFoundWithReason("actor user not found")
		}
		capturedActorID = actorUser.UserID

		if Client.TenantID != tenantID {
			return apperror.NewNotFoundWithReason("auth client not found or access denied")
		}

		identityProvider, err := txIDPRepo.FindByUUID(identityProviderUUID, "Tenant")
		if err != nil || identityProvider == nil {
			return apperror.NewNotFoundWithReason("identity provider not found")
		}
		if identityProvider.TenantID != tenantID {
			return apperror.NewForbidden("identity provider does not belong to tenant")
		}
		providerName = identityProvider.Name

		existing, err := txConnectionRepo.FindByClientAndProvider(Client.ClientID, identityProvider.IdentityProviderID)
		if err != nil {
			return err
		}
		if existing != nil {
			return apperror.NewValidation("identity provider is already connected to this client")
		}

		if isDefault {
			if err := txConnectionRepo.UnsetDefaultForClient(Client.ClientID, 0); err != nil {
				return err
			}
		}

		connection := &ClientIdentityProvider{
			TenantID:           tenantID,
			ClientID:           Client.ClientID,
			IdentityProviderID: identityProvider.IdentityProviderID,
			IsDefault:          isDefault,
			Enabled:            enabled,
			DisplayOrder:       displayOrder,
			CreatedBy:          &capturedActorID,
			UpdatedBy:          &capturedActorID,
		}
		if _, err := txConnectionRepo.Create(connection); err != nil {
			return err
		}

		ClientUpdated, err := txClientRepo.FindByUUIDAndTenantID(ClientUUID, tenantID)
		if err != nil || ClientUpdated == nil {
			return apperror.NewNotFoundWithReason("auth client not found")
		}
		updatedClient = ClientUpdated
		return nil
	})

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to add identity provider connection")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	s.authEventService.Log(ctx, authevent.AuthEventInput{
		TenantID:    tenantID,
		ActorUserID: &capturedActorID,
		IPAddress:   middleware.ClientIPFromContext(ctx),
		UserAgent:   ptr.PtrOrNil(middleware.UserAgentFromContext(ctx)),
		Category:    authevent.AuthEventCategoryAuthz,
		EventType:   authevent.AuthEventTypeAuthzAdmin,
		Severity:    authevent.AuthEventSeverityInfo,
		Result:      authevent.AuthEventResultSuccess,
		Description: ptr.Ptr(fmt.Sprintf("Identity provider connected to client: %s", providerName)),
	})
	return ToClientServiceDataResult(updatedClient), nil
}

// UpdateConnection changes the enabled/default/order fields of an existing
// client→identity-provider connection.
func (s *clientService) UpdateConnection(ctx context.Context, ClientUUID uuid.UUID, tenantID int64, connectionUUID uuid.UUID, isDefault bool, enabled bool, displayOrder int, actorUserUUID uuid.UUID) (*ClientServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "client.updateConnection")
	defer span.End()
	span.SetAttributes(
		attribute.String("client.uuid", ClientUUID.String()),
		attribute.Int64("tenant.id", tenantID),
	)

	var updatedClient *Client
	var capturedActorID int64

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txClientRepo := s.clientRepo.WithTx(tx)
		txConnectionRepo := s.clientIdentityProviderRepo.WithTx(tx)
		txUserRepo := s.userRepo.WithTx(tx)

		Client, err := txClientRepo.FindByUUIDAndTenantID(ClientUUID, tenantID)
		if err != nil || Client == nil {
			return apperror.NewNotFoundWithReason("auth client not found or access denied")
		}

		actorUser, err := txUserRepo.FindByUUID(actorUserUUID, "UserIdentities.Tenant")
		if err != nil || actorUser == nil {
			return apperror.NewNotFoundWithReason("actor user not found")
		}
		capturedActorID = actorUser.UserID

		if Client.TenantID != tenantID {
			return apperror.NewNotFoundWithReason("auth client not found or access denied")
		}

		connection, err := txConnectionRepo.FindByUUIDAndTenantID(connectionUUID.String(), tenantID)
		if err != nil || connection == nil {
			return apperror.NewNotFoundWithReason("identity provider connection not found or access denied")
		}
		if connection.ClientID != Client.ClientID {
			return apperror.NewValidation("identity provider connection does not belong to the specified auth client")
		}

		// Clear any other default before promoting this connection.
		if isDefault {
			if err := txConnectionRepo.UnsetDefaultForClient(Client.ClientID, connection.ClientIdentityProviderID); err != nil {
				return err
			}
		}

		connection.IsDefault = isDefault
		connection.Enabled = enabled
		connection.DisplayOrder = displayOrder
		connection.UpdatedBy = &capturedActorID
		if _, err := txConnectionRepo.CreateOrUpdate(connection); err != nil {
			return err
		}

		ClientUpdated, err := txClientRepo.FindByUUIDAndTenantID(ClientUUID, tenantID)
		if err != nil || ClientUpdated == nil {
			return apperror.NewNotFoundWithReason("auth client not found")
		}
		updatedClient = ClientUpdated
		return nil
	})

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to update identity provider connection")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	s.authEventService.Log(ctx, authevent.AuthEventInput{
		TenantID:    tenantID,
		ActorUserID: &capturedActorID,
		IPAddress:   middleware.ClientIPFromContext(ctx),
		UserAgent:   ptr.PtrOrNil(middleware.UserAgentFromContext(ctx)),
		Category:    authevent.AuthEventCategoryAuthz,
		EventType:   authevent.AuthEventTypeAuthzAdmin,
		Severity:    authevent.AuthEventSeverityInfo,
		Result:      authevent.AuthEventResultSuccess,
		Description: ptr.Ptr(fmt.Sprintf("Identity provider connection updated for client: %s", updatedClient.Name)),
	})
	return ToClientServiceDataResult(updatedClient), nil
}

// RemoveConnection detaches an identity provider from a client. The built-in
// (system) provider connection cannot be removed so the in-house
// username/password login always remains available.
func (s *clientService) RemoveConnection(ctx context.Context, ClientUUID uuid.UUID, tenantID int64, connectionUUID uuid.UUID, actorUserUUID uuid.UUID) (*ClientServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "client.removeConnection")
	defer span.End()
	span.SetAttributes(
		attribute.String("client.uuid", ClientUUID.String()),
		attribute.Int64("tenant.id", tenantID),
	)

	var updatedClient *Client
	var capturedActorID int64

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txClientRepo := s.clientRepo.WithTx(tx)
		txConnectionRepo := s.clientIdentityProviderRepo.WithTx(tx)
		txUserRepo := s.userRepo.WithTx(tx)

		Client, err := txClientRepo.FindByUUIDAndTenantID(ClientUUID, tenantID)
		if err != nil || Client == nil {
			return apperror.NewNotFoundWithReason("auth client not found or access denied")
		}

		actorUser, err := txUserRepo.FindByUUID(actorUserUUID, "UserIdentities.Tenant")
		if err != nil || actorUser == nil {
			return apperror.NewNotFoundWithReason("actor user not found")
		}
		capturedActorID = actorUser.UserID

		if Client.TenantID != tenantID {
			return apperror.NewNotFoundWithReason("auth client not found or access denied")
		}

		connection, err := txConnectionRepo.FindByUUIDAndTenantID(connectionUUID.String(), tenantID)
		if err != nil || connection == nil {
			return apperror.NewNotFoundWithReason("identity provider connection not found or access denied")
		}
		if connection.ClientID != Client.ClientID {
			return apperror.NewValidation("identity provider connection does not belong to the specified auth client")
		}
		if connection.IdentityProvider != nil && connection.IdentityProvider.IsSystem {
			return apperror.NewValidation("the built-in identity provider connection cannot be removed")
		}

		if err := txConnectionRepo.DeleteByUUIDAndTenantID(connectionUUID.String(), tenantID); err != nil {
			return err
		}

		ClientUpdated, err := txClientRepo.FindByUUIDAndTenantID(ClientUUID, tenantID)
		if err != nil || ClientUpdated == nil {
			return apperror.NewNotFoundWithReason("auth client not found")
		}
		updatedClient = ClientUpdated
		return nil
	})

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to remove identity provider connection")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	s.authEventService.Log(ctx, authevent.AuthEventInput{
		TenantID:    tenantID,
		ActorUserID: &capturedActorID,
		IPAddress:   middleware.ClientIPFromContext(ctx),
		UserAgent:   ptr.PtrOrNil(middleware.UserAgentFromContext(ctx)),
		Category:    authevent.AuthEventCategoryAuthz,
		EventType:   authevent.AuthEventTypeAuthzAdmin,
		Severity:    authevent.AuthEventSeverityWarn,
		Result:      authevent.AuthEventResultSuccess,
		Description: ptr.Ptr(fmt.Sprintf("Identity provider connection removed from client: %s", updatedClient.Name)),
	})
	return ToClientServiceDataResult(updatedClient), nil
}

// Response builder - made public for use in other services
func ToClientServiceDataResult(Client *Client) *ClientServiceDataResult {
	if Client == nil {
		return nil
	}

	result := &ClientServiceDataResult{
		ClientUUID:                       Client.ClientUUID,
		Name:                             Client.Name,
		DisplayName:                      Client.DisplayName,
		ClientType:                       Client.ClientType,
		Domain:                           Client.Domain,
		Status:                           Client.Status,
		IsDefault:                        Client.IsDefault,
		IsSystem:                         Client.IsSystem,
		AllowRegistration:                Client.AllowRegistration,
		BackchannelLogoutURI:             Client.BackchannelLogoutURI,
		FrontchannelLogoutURI:            Client.FrontchannelLogoutURI,
		BackchannelLogoutSessionRequired: Client.BackchannelLogoutSessionRequired,
		DPoPRequired:                     Client.DPoPRequired,
		RequirePKCE:                      Client.RequirePKCE,
		RequiredACR:                      Client.RequiredACR,
		SessionIdleTimeout:               Client.SessionIdleTimeout,
		SessionAbsoluteTimeout:           Client.SessionAbsoluteTimeout,
		CreatedAt:                        Client.CreatedAt,
		UpdatedAt:                        Client.UpdatedAt,
	}

	if Client.Branding != nil {
		result.BrandingUUID = &Client.Branding.BrandingUUID
	}

	if Client.ConnectedProviders != nil && len(*Client.ConnectedProviders) > 0 {
		connections := make([]ClientIdentityProviderServiceDataResult, 0, len(*Client.ConnectedProviders))
		for _, connection := range *Client.ConnectedProviders {
			if connection.IdentityProvider == nil {
				continue
			}
			idp := IdentityProviderServiceDataResult{
				IdentityProviderUUID: connection.IdentityProvider.IdentityProviderUUID,
				Name:                 connection.IdentityProvider.Name,
				DisplayName:          connection.IdentityProvider.DisplayName,
				Provider:             connection.IdentityProvider.Provider,
				ProviderType:         connection.IdentityProvider.ProviderType,
				Identifier:           connection.IdentityProvider.Identifier,
				Status:               connection.IdentityProvider.Status,
				IsDefault:            connection.IdentityProvider.IsDefault,
				IsSystem:             connection.IdentityProvider.IsSystem,
				CreatedAt:            connection.IdentityProvider.CreatedAt,
				UpdatedAt:            connection.IdentityProvider.UpdatedAt,
			}
			if connection.IsDefault && result.IdentityProvider == nil {
				legacy := idp
				result.IdentityProvider = &legacy
			}
			connections = append(connections, ClientIdentityProviderServiceDataResult{
				ClientIdentityProviderUUID: connection.ClientIdentityProviderUUID,
				IdentityProvider:           idp,
				IsDefault:                  connection.IsDefault,
				Enabled:                    connection.Enabled,
				DisplayOrder:               connection.DisplayOrder,
				CreatedAt:                  connection.CreatedAt,
				UpdatedAt:                  connection.UpdatedAt,
			})
		}
		result.Connections = &connections
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

	client, err := s.clientRepo.FindByUUIDAndTenantID(ClientUUID, tenantID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to fetch client")
		return nil, err
	}
	if client == nil {
		span.SetStatus(codes.Error, "client not found or access denied")
		return nil, apperror.NewNotFoundWithReason("client not found or access denied")
	}

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
				permissions[j] = toPermissionServiceDataResult(ClientPermission.Permission)
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

		Client, err := txClientRepo.FindByUUIDAndTenantID(ClientUUID, tenantID)
		if err != nil {
			return err
		}
		if Client == nil {
			return apperror.NewNotFoundWithReason("auth client not found")
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
			if api.TenantID != tenantID {
				return apperror.NewNotFoundWithReason("API not found or access denied: " + apiUUID.String())
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
			ClientAPI := &ClientAPI{
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

		Client, err := txClientRepo.FindByUUIDAndTenantID(ClientUUID, tenantID)
		if err != nil {
			return err
		}
		if Client == nil {
			return apperror.NewNotFoundWithReason("auth client not found")
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
	Client, err := s.clientRepo.FindByUUIDAndTenantID(ClientUUID, tenantID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to fetch client")
		return nil, err
	}
	if Client == nil {
		span.SetStatus(codes.Error, "auth client not found")
		return nil, apperror.NewNotFoundWithReason("auth client not found")
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
		results[i] = toPermissionServiceDataResult(ClientPermission.Permission)
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

		Client, err := txClientRepo.FindByUUIDAndTenantID(ClientUUID, tenantID)
		if err != nil {
			return err
		}
		if Client == nil {
			return apperror.NewNotFoundWithReason("auth client not found")
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
			if permission.TenantID != tenantID {
				return apperror.NewNotFoundWithReason("permission not found or access denied: " + permissionUUID.String())
			}
			// The permission must belong to the API being attached (mirrors the
			// api-key reference path).
			if permission.APIID != ClientAPI.APIID {
				return apperror.NewValidation("permission does not belong to the specified API: " + permissionUUID.String())
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
			ClientPermission := &ClientPermission{
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

		Client, err := txClientRepo.FindByUUIDAndTenantID(ClientUUID, tenantID)
		if err != nil {
			return err
		}
		if Client == nil {
			return apperror.NewNotFoundWithReason("auth client not found")
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
		if permission.TenantID != tenantID {
			return apperror.NewNotFoundWithReason("permission not found or access denied")
		}
		if permission.APIID != ClientAPI.APIID {
			return apperror.NewValidation("permission does not belong to the specified API: " + permissionUUID.String())
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

// TokenEndpointAuthMethod constants for the token_endpoint_auth_method column.

func ValidateTenantAccess(actor *User, target *Tenant) error {
	if actor == nil {
		return apperror.NewUnauthorized("actor user not found")
	}
	if target == nil {
		return apperror.NewNotFoundWithReason("tenant not found")
	}
	// Tenant isolation: access is granted only to the actor's own tenant(s).
	// System-tenant identities do NOT get a cross-tenant override here — that
	// override is confined to the tenant package (tenant-management ops only).
	for _, identity := range actor.UserIdentities {
		if identity.TenantID == target.TenantID {
			return nil
		}
	}
	return apperror.NewForbidden("tenant access denied")
}

// ──────────────────────────────────────────────────────────────────────────────
// Client role assignment
// ──────────────────────────────────────────────────────────────────────────────

func (s *clientService) AssignClientRole(ctx context.Context, ClientUUID uuid.UUID, roleUUID uuid.UUID, tenantID int64, createdBy *int64) (*ClientRole, error) {
	client, err := s.clientRepo.FindByUUID(ClientUUID)
	if err != nil || client == nil {
		return nil, apperror.NewNotFound("client not found")
	}
	if client.TenantID != tenantID {
		return nil, apperror.NewForbidden("client does not belong to your tenant")
	}

	role, err := s.roleRepo.FindByUUID(roleUUID)
	if err != nil || role == nil {
		return nil, apperror.NewNotFound("role not found")
	}
	if role.TenantID != tenantID {
		return nil, apperror.NewForbidden("role does not belong to your tenant")
	}

	return s.clientRoleRepo.AssignRole(client.ClientID, role.RoleID, createdBy)
}

func (s *clientService) RemoveClientRole(ctx context.Context, ClientUUID uuid.UUID, roleUUID uuid.UUID, tenantID int64) error {
	client, err := s.clientRepo.FindByUUID(ClientUUID)
	if err != nil || client == nil {
		return apperror.NewNotFound("client not found")
	}
	if client.TenantID != tenantID {
		return apperror.NewForbidden("client does not belong to your tenant")
	}

	role, err := s.roleRepo.FindByUUID(roleUUID)
	if err != nil || role == nil {
		return apperror.NewNotFound("role not found")
	}

	return s.clientRoleRepo.RemoveRole(client.ClientID, role.RoleID)
}

func (s *clientService) ListClientRoles(ctx context.Context, ClientUUID uuid.UUID, tenantID int64) ([]ClientRole, error) {
	client, err := s.clientRepo.FindByUUID(ClientUUID)
	if err != nil || client == nil {
		return nil, apperror.NewNotFound("client not found")
	}
	if client.TenantID != tenantID {
		return nil, apperror.NewForbidden("client does not belong to your tenant")
	}

	return s.clientRoleRepo.ListRoles(client.ClientID)
}
