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
	"github.com/maintainerd/maintainerd-auth/internal/platform/cache"
	"github.com/maintainerd/maintainerd-auth/internal/platform/config"
	"github.com/maintainerd/maintainerd-auth/internal/platform/crypto"
	"github.com/maintainerd/maintainerd-auth/internal/platform/jwt"
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

// ClientSecretServiceDataResult existed only as the return type of the removed
// GetSecretByUUID, which could never populate it. Creation and rotation return
// their one-time plaintext through ClientCreateServiceResult and RotateSecret's
// string return instead.

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
	ClientUUID uuid.UUID
	// Identifier is the OAuth client_id an application presents. Operators need
	// it to configure their app.
	Identifier *string
	// ServiceUUID is the service this client authenticates as, when bound.
	ServiceUUID *string

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
	AllowMagicLink    bool

	// OIDC Session Management
	BackchannelLogoutURI             *string
	FrontchannelLogoutURI            *string
	BackchannelLogoutSessionRequired bool
	DPoPRequired                     bool

	// OAuth metadata as enforced by the runtime (real columns, not the config blob)
	TokenEndpointAuthMethod string
	GrantTypes              []string
	ResponseTypes           []string
	AllowedScopes           []string
	RequireConsent          *bool
	AccessTokenTTL          *int
	RefreshTokenTTL         *int

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
	// IsFirstPartyClient reports whether the client is one this deployment owns,
	// as opposed to a third-party application a tenant registered.
	IsFirstPartyClient(ctx context.Context, clientIdentifier string) bool
	// BoundCertThumbprint returns the certificate this client's tokens are bound
	// to (RFC 8705), or "" when the client is not certificate-bound.
	BoundCertThumbprint(ctx context.Context, clientIdentifier string) string
	GetByUUID(ctx context.Context, ClientUUID uuid.UUID, tenantID int64) (*ClientServiceDataResult, error)
	// There is deliberately no secret read: secrets are bcrypt hashed at rest and
	// cannot be recovered. RotateSecret is the only way to obtain one after
	// creation.
	GetConfigByUUID(ctx context.Context, ClientUUID uuid.UUID, tenantID int64) (datatypes.JSON, error)
	Create(ctx context.Context, tenantID int64, name string, displayName string, clientType string, domain string, config datatypes.JSON, status string, identityProviderUUID string, brandingUUID *uuid.UUID, allowRegistration bool, backchannelLogoutURI *string, frontchannelLogoutURI *string, backchannelLogoutSessionRequired *bool, dPoPRequired *bool, actorUserUUID uuid.UUID, serviceUUID *string) (*ClientCreateServiceResult, error)
	Update(ctx context.Context, ClientUUID uuid.UUID, tenantID int64, name string, displayName string, clientType string, domain string, config datatypes.JSON, status string, brandingUUID *uuid.UUID, allowRegistration *bool, allowMagicLink *bool, backchannelLogoutURI *string, frontchannelLogoutURI *string, backchannelLogoutSessionRequired *bool, dPoPRequired *bool, actorUserUUID uuid.UUID, expectedUpdatedAt *time.Time, serviceUUID *string) (*ClientServiceDataResult, error)
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
	UpdateConnection(ctx context.Context, ClientUUID uuid.UUID, tenantID int64, connectionUUID uuid.UUID, isDefault *bool, enabled *bool, displayOrder *int, actorUserUUID uuid.UUID) (*ClientServiceDataResult, error)
	RemoveConnection(ctx context.Context, ClientUUID uuid.UUID, tenantID int64, connectionUUID uuid.UUID, actorUserUUID uuid.UUID) (*ClientServiceDataResult, error)

	// Auth Client API methods
	GetClientAPIs(ctx context.Context, tenantID int64, ClientUUID uuid.UUID) ([]ClientAPIServiceDataResult, error)
	AddClientAPIs(ctx context.Context, tenantID int64, ClientUUID uuid.UUID, apiUUIDs []uuid.UUID, actorUserUUID uuid.UUID) error
	RemoveClientAPI(ctx context.Context, tenantID int64, ClientUUID uuid.UUID, apiUUID uuid.UUID, actorUserUUID uuid.UUID) error

	// Auth Client API Permission methods
	GetClientAPIPermissions(ctx context.Context, tenantID int64, ClientUUID uuid.UUID, apiUUID uuid.UUID) ([]PermissionServiceDataResult, error)
	AddClientAPIPermissions(ctx context.Context, tenantID int64, ClientUUID uuid.UUID, apiUUID uuid.UUID, permissionUUIDs []uuid.UUID, actorUserUUID uuid.UUID) error
	RemoveClientAPIPermission(ctx context.Context, tenantID int64, ClientUUID uuid.UUID, apiUUID uuid.UUID, permissionUUID uuid.UUID, actorUserUUID uuid.UUID) error

	// Client role assignment
	AssignClientRole(ctx context.Context, ClientUUID uuid.UUID, roleUUID uuid.UUID, tenantID int64, actorUserUUID uuid.UUID) (*ClientRole, error)
	RemoveClientRole(ctx context.Context, ClientUUID uuid.UUID, roleUUID uuid.UUID, tenantID int64, actorUserUUID uuid.UUID) error
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

// IsFirstPartyClient reports whether clientIdentifier resolves to a client this
// deployment owns — a seeded system client (the admin console or the hosted
// login app).
//
// The end-user self-service API (/account, /profiles, /mfa, trusted devices,
// data erasure, identity linking) is reachable with any valid access token for
// the subject. Without this, an access token minted for a THIRD-PARTY OAuth
// client — one the user consented to for, say, `openid profile` — could read
// and mutate that user's entire account: change their email, rotate their
// password, enumerate and revoke their sessions, or strip their MFA. Consenting
// to sign in with an app must never hand it the keys to the account itself.
//
// First-party is decided by DOMAIN, not by a flag on the row: a client is
// first-party when its registered domain shares a registrable domain (eTLD+1)
// with this deployment's own public hostname.
//
// That is not a naming convention, it is the actual security boundary being
// defended. This guard exists because a browser will attach the auth session
// cookie to a same-site app; an app on a different registrable domain cannot
// receive that cookie and has to come through OAuth consent like any other
// third party. So "can this client be trusted with the account-management
// surface" and "will the browser hand this client our cookie" are the same
// question, and answering it from the domain keeps them from ever diverging.
//
// It deliberately does NOT read c.IsSystem. A boolean on the row is a second,
// hand-maintained answer to a question the domain already answers, and the two
// drift: mark a client on someone else's domain as system and it silently gains
// the account surface. The seeded console and hosted-login apps are first-party
// here because they are DEPLOYED on this domain, which is the reason they are
// trusted — not because a column says so.
//
// This is a WEB boundary. Native mobile apps hold no cookies and authenticate
// with their own tokens, so they are third-party by this rule and reach the
// account surface through the same consented OAuth path as anyone else.
//
// Never derived from anything the caller supplies: the domain is read from the
// stored client record, and the deployment's own hostname from verified config.
func (s *clientService) IsFirstPartyClient(_ context.Context, clientIdentifier string) bool {
	id := strings.TrimSpace(clientIdentifier)
	if id == "" {
		return false
	}
	c, err := s.clientRepo.FindByIdentifier(id)
	if err != nil || c == nil {
		// Fail closed: an unresolvable client is not first-party.
		return false
	}
	if c.Domain == nil {
		// A client with no registered domain cannot be shown to be same-site.
		return false
	}
	return shared.SameRegistrableDomain(*c.Domain, config.AppPublicHostname)
}

var (
	generateClientIdentifier = crypto.GenerateIdentifier
	hashClientSecret         = security.HashClientSecret
	encryptClientSecret      = crypto.EncryptAtRest
)

const (
	// clientIdentifierLength is the length of a generated OAuth client_id. 12
	// symbols from crypto.GenerateIdentifier's 62-symbol alphabet is ~71 bits.
	clientIdentifierLength = 12
	// clientIdentifierAttempts bounds the collision retry below.
	clientIdentifierAttempts = 5
)

// generateUniqueClientIdentifier mints an OAuth client_id that no client — active,
// inactive or soft-deleted — already holds.
//
// Generation was unchecked, and `identifier` is what every token and authorize
// request resolves a client by, with a global UNIQUE index behind it
// (uq_clients_identifier). An unchecked collision therefore surfaced as a raw
// constraint violation (a 500 on client creation) instead of a retry. Soft-deleted
// rows are included because the index counts only live rows but a resurrected row
// would collide, and because reusing a retired client_id would silently re-point
// anything still holding it.
func generateUniqueClientIdentifier(repo ClientRepository) (string, error) {
	for attempt := 0; attempt < clientIdentifierAttempts; attempt++ {
		candidate, err := generateClientIdentifier(clientIdentifierLength)
		if err != nil {
			return "", err
		}
		taken, err := repo.ExistsByIdentifier(candidate)
		if err != nil {
			return "", err
		}
		if !taken {
			return candidate, nil
		}
	}
	// Exhausting the retries points at a broken generator, not bad luck — at ~71
	// bits the odds of five collisions are not worth writing down. Fail rather
	// than hand out an identifier that was never checked.
	return "", apperror.NewInternal("could not allocate a unique client identifier", nil)
}

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
	// grantAuthorityRepo backs assertClientGrantWithinActorAuthority. A client's
	// roles and API permissions become an M2M access token's permissions, so
	// every grant path has to be able to ask what the actor already holds.
	grantAuthorityRepo GrantAuthorityRepository
	authEventService   authevent.AuthEventService
	eventService       event.EventService
	// cacheInvalidator clears the middleware's user-context cache. Reachability
	// is resolved through client_identity_providers, so a connection change is
	// an authorization change — without this, a disabled connection keeps
	// authenticating from cache for the full TTL.
	cacheInvalidator cache.Invalidator
}

// invalidateUserContexts drops cached user contexts after an authorization
// change. Entries are keyed by (sub, client), and one connection can cover many
// subjects, so this clears all of them rather than guessing the affected set.
func (s *clientService) invalidateUserContexts(ctx context.Context) {
	if s.cacheInvalidator != nil {
		s.cacheInvalidator.InvalidateAllUsers(ctx)
	}
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
	// Variadic so the many existing call sites need no change; pass one to make
	// connection changes take effect immediately rather than after the cache TTL.
	cacheInvalidator ...cache.Invalidator,
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
		// Same reasoning for the grant-authority queries: they exist only to serve
		// the escalation guard in this file.
		grantAuthorityRepo:   NewGrantAuthorityRepository(db),
		idpRepo:              idpRepo,
		permissionRepo:       permissionRepo,
		clientPermissionRepo: clientPermissionRepo,
		clientAPIRepo:        clientAPIRepo,
		clientRoleRepo:       clientRoleRepo,
		roleRepo:             roleRepo,
		apiRepo:              apiRepo,
		userRepo:             userRepo,
		tenantRepo:           tenantRepo,
		cacheInvalidator:     firstCacheInvalidator(cacheInvalidator),
		authEventService:     coalesceAuthEventService(authEventService),
		eventService:         eventService,
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
	// Report the EFFECTIVE config (blob overlaid with the authoritative columns).
	// Serving the raw blob let the console show stale values and round-trip them
	// back on save.
	return effectiveClientConfig(Client), nil
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

func (s *clientService) Create(ctx context.Context, tenantID int64, name string, displayName string, clientType string, domain string, config datatypes.JSON, status string, identityProviderUUID string, brandingUUID *uuid.UUID, allowRegistration bool, backchannelLogoutURI *string, frontchannelLogoutURI *string, backchannelLogoutSessionRequired *bool, dPoPRequired *bool, actorUserUUID uuid.UUID, serviceUUID *string) (*ClientCreateServiceResult, error) {
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

		// Binding this client to a service makes its tokens carry the `svc` claim,
		// which is the principal the policy bundle and the gRPC authorizer resolve.
		// It is a privilege grant, so it is validated here rather than trusted.
		boundServiceID, err := resolveServiceBinding(tx, tenantID, clientType, serviceUUID)
		if err != nil {
			return err
		}

		existingClient, err := txClientRepo.FindByNameAndTenantID(name, tenantID)
		if err != nil {
			return err
		}
		if existingClient != nil {
			return apperror.NewConflict(name + " auth client already exists")
		}

		clientID, err := generateUniqueClientIdentifier(txClientRepo)
		if err != nil {
			return err
		}
		// Only a confidential client gets a secret. A public client (spa, mobile)
		// cannot keep one — it ships in code the user can read — so issuing it
		// would be a false sense of security, and it would leave the client on the
		// column default of client_secret_basic instead of "none".
		var secretHashPtr *string
		if !IsPublicClientType(clientType) {
			rawSecret, err := generateClientIdentifier(64)
			if err != nil {
				return err
			}
			secretHash, err := hashClientSecret(ctx, rawSecret)
			if err != nil {
				return err
			}
			secretHashPtr = &secretHash
			plaintextSecret = rawSecret
		}

		// A brand-new client starts minting tokens with this domain as `iss`
		// immediately, so the allowlist has to know about it before that happens.
		registerIssuer(&domain)

		newClient := &Client{
			ServiceID:   boundServiceID,
			Name:        name,
			DisplayName: displayName,
			ClientType:  clientType,
			Domain:      &domain,
			Identifier:  &clientID,
			SecretHash:  secretHashPtr,
			Config:      config,

			TenantID: tenantID,
			Status:   status,
			// is_default is platform-owned, not tenant-admin input: it is set once by
			// the seeder on the bootstrap client, the table enforces one per tenant
			// (uq_clients_tenant_default), and Update/SetStatus/Delete all refuse a
			// default client. Letting a caller set it at create therefore minted a
			// client that could never be edited, deactivated or deleted again — the
			// same reason is_system is hard-coded below.
			IsDefault:         false,
			IsSystem:          false,
			AllowRegistration: &allowRegistration,
			// Passwordless email sign-in is opt-in: a new client starts with it
			// off and an operator turns it on deliberately.
			AllowMagicLink: &magicLinkDisabledByDefault,

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

		// A public client defaults to "none": the column default is
		// client_secret_basic, which is wrong for a client with no secret.
		if IsPublicClientType(newClient.ClientType) && newClient.TokenEndpointAuthMethod == "" {
			newClient.TokenEndpointAuthMethod = TokenAuthMethodNone
		}

		// secret_encrypted is reversible (AES under one app-wide key), so it is
		// recoverable plaintext at rest. Only client_secret_jwt needs it — that
		// method HMACs the assertion with the secret, so the server must hold the
		// plaintext (see clientSecretJWTSecrets). client_secret_basic/_post verify
		// against the bcrypt hash and never read it, so storing it for them was a
		// credential store with no consumer. It is written after
		// applyConfigToClientColumns because that is what resolves the auth method.
		if plaintextSecret != "" && newClient.TokenEndpointAuthMethod == TokenAuthMethodClientSecretJWT {
			secretEncrypted, err := encryptClientSecret(plaintextSecret)
			if err != nil {
				return err
			}
			newClient.SecretEncrypted = &secretEncrypted
		}

		if err := ValidateClientOAuthMatrix(
			newClient.ClientType,
			newClient.TokenEndpointAuthMethod,
			newClient.GrantTypes,
			newClient.AllowedScopes,
			newClient.SecretHash != nil,
			newClient.JWKS != nil || newClient.JWKSUri != nil,
		); err != nil {
			return err
		}

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
		Enabled:            &enabled,
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
		existing.Enabled = &enabled
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

	// The cap lived only in the HTTP DTO, and the gRPC handler validates nothing —
	// so a rotation with grace_period_hours = 876000 kept the compromised previous
	// secret accepted by the token endpoint for a century while the tenant saw a
	// successful rotation and a client.secret_rotated event and believed the
	// credential was revoked. Enforcing it here covers every transport.
	if gracePeriodHours < 0 || gracePeriodHours > maxSecretGracePeriodHours {
		span.SetStatus(codes.Error, "grace period out of range")
		return "", apperror.NewValidation(fmt.Sprintf(
			"grace_period_hours must be between 0 and %d (7 days)", maxSecretGracePeriodHours))
	}

	var plaintextSecret string
	var capturedActorID int64
	var rotatedClientName string

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txClientRepo := s.clientRepo.WithTx(tx)
		txUserRepo := s.userRepo.WithTx(tx)

		client, err := txClientRepo.FindByUUIDAndTenantID(clientUUID, tenantID)
		if err != nil {
			return err
		}
		if client == nil {
			return apperror.NewNotFoundWithReason("auth client not found or access denied")
		}

		// Rotating a credential is the most sensitive mutation in this domain, so
		// the actor is resolved and checked against the tenant like every other one.
		actorUser, err := txUserRepo.FindByUUID(actorUserUUID, "UserIdentities.Tenant")
		if err != nil || actorUser == nil {
			return apperror.NewNotFoundWithReason("actor user not found")
		}
		if err := ValidateTenantAccess(actorUser, &Tenant{TenantID: tenantID}); err != nil {
			return err
		}
		capturedActorID = actorUser.UserID
		rotatedClientName = client.Name

		// Rotating a system client's secret would break the console or login UI,
		// which are configured with the seeded value.
		if client.IsSystem {
			return apperror.NewValidation("system auth client secret cannot be rotated")
		}

		// A public client has no secret to rotate, and minting one would imply a
		// credential it cannot protect.
		if IsPublicClientType(client.ClientType) {
			return apperror.NewValidation("a " + client.ClientType + " client has no client secret to rotate")
		}

		rawSecret, err := generateClientIdentifier(64)
		if err != nil {
			return err
		}
		newHash, err := hashClientSecret(ctx, rawSecret)
		if err != nil {
			return err
		}
		// Only client_secret_jwt needs a reversible copy — it HMACs the assertion
		// with the secret, so the server must hold the plaintext. Every other method
		// verifies against the bcrypt hash, so keeping an AES-recoverable copy for
		// them was a second, weaker credential store with no consumer. Rotating a
		// client that has moved OFF client_secret_jwt clears the stale copy.
		var newEncrypted *string
		if client.TokenEndpointAuthMethod == TokenAuthMethodClientSecretJWT {
			enc, err := encryptClientSecret(rawSecret)
			if err != nil {
				return err
			}
			newEncrypted = &enc
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
		client.SecretEncrypted = newEncrypted

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
	// A credential rotation with no audit record would be invisible after the
	// fact — the one mutation where that matters most.
	s.authEventService.Log(ctx, authevent.AuthEventInput{
		TenantID:    tenantID,
		ActorUserID: &capturedActorID,
		IPAddress:   middleware.ClientIPFromContext(ctx),
		UserAgent:   ptr.PtrOrNil(middleware.UserAgentFromContext(ctx)),
		Category:    authevent.AuthEventCategoryAuthz,
		EventType:   authevent.AuthEventTypeAuthzAdmin,
		Severity:    authevent.AuthEventSeverityWarn,
		Result:      authevent.AuthEventResultSuccess,
		Description: ptr.Ptr(fmt.Sprintf("Client secret rotated: %s (grace %dh)", rotatedClientName, gracePeriodHours)),
	})
	return plaintextSecret, nil
}

// assertClientSecretJWTHasKeyMaterial refuses an update that moves a client onto
// client_secret_jwt when the server holds no reversible copy of its secret.
//
// client_secret_jwt verifies the assertion by HMAC-ing it with the secret, so the
// server needs the plaintext (oauth/authentication.go clientSecretJWTSecrets reads
// secret_encrypted). That column is written only by Create and RotateSecret — the
// two moments the server actually holds the plaintext — and only for clients already
// on this method. A client created as client_secret_basic therefore has it NULL, and
// before this check the switch succeeded and left a client whose every token request
// failed "client has no registered secret", with nothing in its config to explain it.
//
// Refusing is chosen over minting a secret here: an auto-mint would silently replace
// a credential the operator's deployed client is still using, and they would learn
// about it from the outage rather than from the API. The switch is therefore a
// create-time decision — the error says so rather than pointing at rotate-secret,
// which cannot help: rotate keys the encrypted copy off the CURRENTLY stored method,
// so rotating a client_secret_basic client still writes nothing.
//
// Switching AWAY from client_secret_jwt deliberately leaves secret_encrypted in
// place (only a rotation clears it), which is what makes the reverse switch back
// onto the method safe — hence the check is on the stored copy, not on the previous
// method alone.
func assertClientSecretJWTHasKeyMaterial(c *Client, previousAuthMethod string) error {
	if c.TokenEndpointAuthMethod != TokenAuthMethodClientSecretJWT {
		return nil
	}
	if previousAuthMethod == TokenAuthMethodClientSecretJWT || c.SecretEncrypted != nil {
		return nil
	}
	return apperror.NewValidation(
		"token_endpoint_auth_method cannot be changed to client_secret_jwt: this client's secret was issued " +
			"under a method that keeps only a one-way hash, so the server cannot sign or verify its assertions. " +
			"Create a client with token_endpoint_auth_method=client_secret_jwt instead")
}

// magicLinkDisabledByDefault backs the AllowMagicLink pointer on newly created
// clients. A pointer is required so an explicit false is distinguishable from
// "unset" (see the field comment on the model).
var magicLinkDisabledByDefault = false

func (s *clientService) Update(ctx context.Context, ClientUUID uuid.UUID, tenantID int64, name string, displayName string, clientType string, domain string, config datatypes.JSON, status string, brandingUUID *uuid.UUID, allowRegistration *bool, allowMagicLink *bool, backchannelLogoutURI *string, frontchannelLogoutURI *string, backchannelLogoutSessionRequired *bool, dPoPRequired *bool, actorUserUUID uuid.UUID, expectedUpdatedAt *time.Time, serviceUUID *string) (*ClientServiceDataResult, error) {
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

		// Optimistic concurrency. An update replaces the whole client (config
		// included), so two operators editing the same client silently overwrote each
		// other and the loser's change vanished behind a 200. When the caller states
		// which version it loaded, a newer one is a conflict rather than a clobber.
		if err := assertClientUnchangedSince(Client, expectedUpdatedAt); err != nil {
			return err
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

		// System clients back the console and the hosted login UI and are resolved
		// by name, so renaming, retyping or deactivating one breaks the tenant.
		// Guarded before is_default because auth-identity is is_system WITHOUT
		// being is_default, so the is_default check alone let it through.
		if Client.IsSystem {
			return apperror.NewValidation("system auth client cannot be updated")
		}
		if Client.IsDefault {
			return apperror.NewValidation("default auth client cannot be updated")
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
		registerIssuer(Client.Domain)
		// A nil config means "unchanged". config JSONB is NOT NULL, so assigning nil
		// unconditionally would violate the column on any caller that omits it.
		if config != nil {
			Client.Config = config
		}
		Client.Status = status
		// is_default is deliberately NOT assigned here. It is platform-owned (set by
		// the seeder, one per tenant via uq_clients_tenant_default) and every mutating
		// path above refuses a client that carries it, so promoting one through this
		// endpoint would turn an ordinary client into one nobody can edit,
		// deactivate or delete again.
		// nil means "unchanged"; an explicit empty string unbinds. The client type
		// used for the check is the one the update is applying, so a client cannot be
		// converted away from m2m while keeping a service binding.
		if serviceUUID != nil {
			boundServiceID, bindErr := resolveServiceBinding(tx, tenantID, clientType, serviceUUID)
			if bindErr != nil {
				return bindErr
			}
			Client.ServiceID = boundServiceID
		}
		if allowRegistration != nil {
			Client.AllowRegistration = allowRegistration
		}
		if allowMagicLink != nil {
			Client.AllowMagicLink = allowMagicLink
		}
		// A nil pointer means "unchanged"; an EMPTY string means "clear". Without the
		// empty-string case a logout URI could be set but never removed: JSON null
		// decodes to the same nil pointer as an omitted key, so the caller had no way
		// to express removal at all.
		Client.BackchannelLogoutURI = resolveOptionalString(Client.BackchannelLogoutURI, backchannelLogoutURI)
		Client.FrontchannelLogoutURI = resolveOptionalString(Client.FrontchannelLogoutURI, frontchannelLogoutURI)
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
		previousAuthMethod := Client.TokenEndpointAuthMethod
		applyConfigToClientColumns(Client, config)

		if err := assertClientSecretJWTHasKeyMaterial(Client, previousAuthMethod); err != nil {
			return err
		}

		if err := ValidateClientOAuthMatrix(
			Client.ClientType,
			Client.TokenEndpointAuthMethod,
			Client.GrantTypes,
			Client.AllowedScopes,
			Client.SecretHash != nil,
			Client.JWKS != nil || Client.JWKSUri != nil,
		); err != nil {
			return err
		}

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
	s.invalidateUserContexts(ctx)
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
	s.invalidateUserContexts(ctx)
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

		// Deleting a system client is unrecoverable without a re-seed and takes the
		// tenant's console or login UI with it.
		if Client.IsSystem {
			return apperror.NewValidation("system auth client cannot be deleted")
		}
		if Client.IsDefault {
			return apperror.NewValidation("default auth client cannot be deleted")
		}

		// Clear the client's children BEFORE deleting it.
		//
		// clients is soft-deleted, so the ON DELETE CASCADE on these tables never
		// fires. Without this, client_uris and client_identity_providers keep
		// deleted_at IS NULL, and client_apis / client_permissions / client_roles —
		// which have no deleted_at at all — become permanent orphans that still
		// resolve: ResolvePermissions would keep granting permissions for a deleted
		// client, and its APIs would keep listing.
		//
		// GORM picks the right semantics per model: soft delete where the model has
		// a DeletedAt field, hard delete where it does not.
		for _, child := range []any{
			&ClientURI{},
			&ClientIdentityProvider{},
			&ClientPermission{},
			&ClientAPI{},
			&ClientRole{},
		} {
			if err := tx.Where("client_id = ?", Client.ClientID).Delete(child).Error; err != nil {
				return err
			}
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
	s.invalidateUserContexts(ctx)
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
		// The actor must hold an identity in this tenant. Loading the actor only to
		// stamp an audit id left the middleware-supplied tenant as the sole trust
		// boundary on these mutations.
		if err := ValidateTenantAccess(actorUser, &Tenant{TenantID: tenantID}); err != nil {
			return err
		}
		capturedActorID = actorUser.UserID

		// Validate tenant ownership
		if Client.TenantID != tenantID {
			return apperror.NewNotFoundWithReason("auth client not found or access denied")
		}

		// Registration-time validation is stricter than the runtime denylist: it
		// requires an absolute https URI (or http loopback, or a private-use scheme
		// for a native client) with no fragment and no embedded credentials.
		// Client type matters — only a mobile client may register com.example.app:/…
		if uriType == shared.ClientURITypeRedirect || uriType == shared.ClientURITypeLogout || uriType == shared.ClientURITypeLogin {
			if err := ValidateRegisteredRedirectURI(Client.ClientType, uri); err != nil {
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
		// The actor must hold an identity in this tenant. Loading the actor only to
		// stamp an audit id left the middleware-supplied tenant as the sole trust
		// boundary on these mutations.
		if err := ValidateTenantAccess(actorUser, &Tenant{TenantID: tenantID}); err != nil {
			return err
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
		// Registration-time validation is stricter than the runtime denylist: it
		// requires an absolute https URI (or http loopback, or a private-use scheme
		// for a native client) with no fragment and no embedded credentials.
		// Client type matters — only a mobile client may register com.example.app:/…
		if uriType == shared.ClientURITypeRedirect || uriType == shared.ClientURITypeLogout || uriType == shared.ClientURITypeLogin {
			if err := ValidateRegisteredRedirectURI(Client.ClientType, uri); err != nil {
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
		// The actor must hold an identity in this tenant. Loading the actor only to
		// stamp an audit id left the middleware-supplied tenant as the sole trust
		// boundary on these mutations.
		if err := ValidateTenantAccess(actorUser, &Tenant{TenantID: tenantID}); err != nil {
			return err
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
		// The actor must hold an identity in this tenant. Loading the actor only to
		// stamp an audit id left the middleware-supplied tenant as the sole trust
		// boundary on these mutations.
		if err := ValidateTenantAccess(actorUser, &Tenant{TenantID: tenantID}); err != nil {
			return err
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
			Enabled:            &enabled,
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
func (s *clientService) UpdateConnection(ctx context.Context, ClientUUID uuid.UUID, tenantID int64, connectionUUID uuid.UUID, isDefault *bool, enabled *bool, displayOrder *int, actorUserUUID uuid.UUID) (*ClientServiceDataResult, error) {
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
		// The actor must hold an identity in this tenant. Loading the actor only to
		// stamp an audit id left the middleware-supplied tenant as the sole trust
		// boundary on these mutations.
		if err := ValidateTenantAccess(actorUser, &Tenant{TenantID: tenantID}); err != nil {
			return err
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

		// Omitted means unchanged: fall back to the connection's current values so a
		// partial payload cannot silently demote the default or reset the ordering.
		wantEnabled := connection.Enabled == nil || *connection.Enabled
		if enabled != nil {
			wantEnabled = *enabled
		}
		wantDefault := connection.IsDefault
		if isDefault != nil {
			wantDefault = *isDefault
		}
		wantOrder := connection.DisplayOrder
		if displayOrder != nil {
			wantOrder = *displayOrder
		}

		if err := s.assertConnectionMutationKeepsClientUsable(
			tx, Client.ClientID, connection, wantEnabled, wantDefault, false,
		); err != nil {
			return err
		}

		// Clear any other default before promoting this connection.
		if wantDefault {
			if err := txConnectionRepo.UnsetDefaultForClient(Client.ClientID, connection.ClientIdentityProviderID); err != nil {
				return err
			}
		}

		connection.IsDefault = wantDefault
		connection.Enabled = &wantEnabled
		connection.DisplayOrder = wantOrder
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
	s.invalidateUserContexts(ctx)
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
		// The actor must hold an identity in this tenant. Loading the actor only to
		// stamp an audit id left the middleware-supplied tenant as the sole trust
		// boundary on these mutations.
		if err := ValidateTenantAccess(actorUser, &Tenant{TenantID: tenantID}); err != nil {
			return err
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
		if err := s.assertConnectionMutationKeepsClientUsable(
			tx, Client.ClientID, connection, false, false, true,
		); err != nil {
			return err
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
	s.invalidateUserContexts(ctx)
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
		Identifier:                       Client.Identifier,
		ServiceUUID:                      serviceUUIDForClient(Client),
		TokenEndpointAuthMethod:          Client.TokenEndpointAuthMethod,
		GrantTypes:                       []string(Client.GrantTypes),
		ResponseTypes:                    []string(Client.ResponseTypes),
		AllowedScopes:                    []string(Client.AllowedScopes),
		RequireConsent:                   Client.RequireConsent,
		AccessTokenTTL:                   Client.AccessTokenTTL,
		RefreshTokenTTL:                  Client.RefreshTokenTTL,
		AllowRegistration:                boolValue(Client.AllowRegistration, true),
		AllowMagicLink:                   boolValue(Client.AllowMagicLink, false),
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
				Enabled:                    boolValue(connection.Enabled, true),
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
func (s *clientService) AddClientAPIs(ctx context.Context, tenantID int64, ClientUUID uuid.UUID, apiUUIDs []uuid.UUID, actorUserUUID uuid.UUID) error {
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

		if _, err := s.requireActorTenantAccess(tx, actorUserUUID, tenantID); err != nil {
			return err
		}

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
func (s *clientService) RemoveClientAPI(ctx context.Context, tenantID int64, ClientUUID uuid.UUID, apiUUID uuid.UUID, actorUserUUID uuid.UUID) error {
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

		if _, err := s.requireActorTenantAccess(tx, actorUserUUID, tenantID); err != nil {
			return err
		}

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
func (s *clientService) AddClientAPIPermissions(ctx context.Context, tenantID int64, ClientUUID uuid.UUID, apiUUID uuid.UUID, permissionUUIDs []uuid.UUID, actorUserUUID uuid.UUID) error {
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

		actorID, err := s.requireActorTenantAccess(tx, actorUserUUID, tenantID)
		if err != nil {
			return err
		}

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

		// Resolve and validate the whole batch BEFORE creating any of it: the
		// escalation guard has to see every name that is about to be granted, and
		// a partially applied batch would leave the client holding some of them.
		resolved := make([]*Permission, 0, len(permissionUUIDs))
		granting := make([]string, 0, len(permissionUUIDs))
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
			// The permission must belong to the API being attached.
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

			resolved = append(resolved, permission)
			granting = append(granting, permission.Name)
		}

		if err := s.assertClientGrantWithinActorAuthority(tx, actorID, tenantID, granting); err != nil {
			return err
		}

		for i, permission := range resolved {
			// Create new auth client permission relationship
			ClientPermission := &ClientPermission{
				ClientPermissionUUID: uuid.New(),
				ClientAPIID:          ClientAPI.ClientAPIID,
				PermissionID:         permission.PermissionID,
			}

			if _, err := txClientPermissionRepo.Create(ClientPermission); err != nil {
				// Check if it's a unique constraint violation
				if strings.Contains(err.Error(), "uq_client_permissions_client_permission") {
					return apperror.NewConflict("permission already assigned to auth client API: " + permissionUUIDs[i].String())
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
func (s *clientService) RemoveClientAPIPermission(ctx context.Context, tenantID int64, ClientUUID uuid.UUID, apiUUID uuid.UUID, permissionUUID uuid.UUID, actorUserUUID uuid.UUID) error {
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

		if _, err := s.requireActorTenantAccess(tx, actorUserUUID, tenantID); err != nil {
			return err
		}

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

// requireActorTenantAccess resolves the acting user and asserts it holds an identity
// in the target tenant, returning the actor's id for audit stamping.
//
// The tenant on these mutations comes from the middleware, which made it the sole
// trust boundary; the actor check is the second one, and it is what makes an
// unattributed call (an empty actor UUID from the gRPC surface) fail rather than
// mutate a client's API and role grants anonymously.
//
// tx may be nil for a method that does not run in a transaction.
// assertClientUnchangedSince implements the optimistic-concurrency check for a
// client update.
//
// A nil expectation opts out, so service-to-service callers and the seeder are
// unaffected; the console always sends the updated_at it loaded. Timestamps are
// compared at microsecond precision because that is what Postgres stores — an
// RFC3339 round trip through JSON can carry more digits than the column does, and a
// naive equality check would then never match.
// resolveOptionalString applies the "nil = unchanged, empty = clear" convention
// used by the optional string columns on a client update.
func resolveOptionalString(current, incoming *string) *string {
	if incoming == nil {
		return current
	}
	if strings.TrimSpace(*incoming) == "" {
		return nil
	}
	trimmed := strings.TrimSpace(*incoming)
	return &trimmed
}

func assertClientUnchangedSince(client *Client, expectedUpdatedAt *time.Time) error {
	if expectedUpdatedAt == nil {
		return nil
	}
	stored := client.UpdatedAt.UTC().Truncate(time.Microsecond)
	expected := expectedUpdatedAt.UTC().Truncate(time.Microsecond)
	if stored.Equal(expected) {
		return nil
	}
	return apperror.NewConflict(
		"this client was modified by someone else after you loaded it; reload it and reapply your changes")
}

func (s *clientService) requireActorTenantAccess(tx *gorm.DB, actorUserUUID uuid.UUID, tenantID int64) (int64, error) {
	userRepo := s.userRepo
	if tx != nil {
		userRepo = userRepo.WithTx(tx)
	}

	actorUser, err := userRepo.FindByUUID(actorUserUUID, "UserIdentities.Tenant")
	if err != nil || actorUser == nil {
		return 0, apperror.NewNotFoundWithReason("actor user not found")
	}
	if err := ValidateTenantAccess(actorUser, &Tenant{TenantID: tenantID}); err != nil {
		return 0, err
	}
	return actorUser.UserID, nil
}

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

// assertClientGrantWithinActorAuthority refuses to grant a client a permission
// the acting user does not already hold themselves.
//
// A client's permissions ARE an access token's permissions: the token service
// resolves client_roles → role_permissions plus the direct client_permissions
// rows into the `permissions` claim of every client_credentials token
// (oauth.service_token clientCredentials path). Without this guard "may I
// configure a client?" silently meant "may I mint a token holding anything?" —
// an admin with only client:role:create or client:api:permission:create could
// attach tenant:delete to a client they control, run one client_credentials
// exchange, and call the management API with a permission they were never
// granted. Route guards match on the permission NAME and never ask who attached
// it.
//
// This is the same containment rule iam.assertNoPrivilegeEscalation applies to
// role→permission grants and the user domain applies to user→role grants: a
// super-admin is seeded with every administrative permission, so it does not
// restrict them and needs no special case.
//
// Only elevated (management-plane) permissions are gated: account:…:self and
// public:… confer nothing beyond the holder's own account.
func (s *clientService) assertClientGrantWithinActorAuthority(tx *gorm.DB, actorUserID, tenantID int64, granting []string) error {
	if shared.FirstElevatedPermission(granting) == "" {
		return nil
	}

	if s.grantAuthorityRepo == nil {
		// Fail CLOSED: with no way to read the actor's permissions there is no way
		// to tell an escalation from a legitimate grant.
		return apperror.NewInternal("could not resolve the acting user's permissions", nil)
	}
	held, err := s.grantAuthorityRepo.WithTx(tx).ActorPermissionNames(actorUserID, tenantID)
	if err != nil {
		// Fail CLOSED: an unreadable actor permission set must not be read as
		// "holds everything".
		return apperror.NewInternal("could not resolve the acting user's permissions", err)
	}
	if unheld := shared.FirstUnheldElevatedPermission(granting, held); unheld != "" {
		return apperror.NewForbidden(fmt.Sprintf(
			"you cannot grant %q to a client because you do not hold it", unheld))
	}
	return nil
}

// ──────────────────────────────────────────────────────────────────────────────
// Client role assignment
// ──────────────────────────────────────────────────────────────────────────────

func (s *clientService) AssignClientRole(ctx context.Context, ClientUUID uuid.UUID, roleUUID uuid.UUID, tenantID int64, actorUserUUID uuid.UUID) (*ClientRole, error) {
	// The actor id is also the created_by stamp: the handler used to pass nil, so
	// every client-role grant was recorded with no author.
	actorID, err := s.requireActorTenantAccess(nil, actorUserUUID, tenantID)
	if err != nil {
		return nil, err
	}

	// 404, not 403: a distinct "belongs to another tenant" answer confirms the UUID
	// exists, which turns these endpoints into an existence oracle across tenants.
	// Every other client mutation already answers "not found or access denied".
	client, err := s.clientRepo.FindByUUIDAndTenantID(ClientUUID, tenantID)
	if err != nil || client == nil {
		return nil, apperror.NewNotFoundWithReason("client not found or access denied")
	}

	role, err := s.roleRepo.FindByUUID(roleUUID)
	if err != nil || role == nil || role.TenantID != tenantID {
		return nil, apperror.NewNotFoundWithReason("role not found or access denied")
	}

	// Granting a role grants everything in it, so the role's contents — not the
	// role itself — are what the actor must already hold.
	if s.grantAuthorityRepo == nil {
		return nil, apperror.NewInternal("could not resolve the permissions this role grants", nil)
	}
	conferred, err := s.grantAuthorityRepo.RolePermissionNames(role.RoleID)
	if err != nil {
		// Fail CLOSED: an unreadable role must not be treated as an empty one.
		return nil, apperror.NewInternal("could not resolve the permissions this role grants", err)
	}
	if err := s.assertClientGrantWithinActorAuthority(nil, actorID, tenantID, conferred); err != nil {
		return nil, err
	}

	return s.clientRoleRepo.AssignRole(client.ClientID, role.RoleID, &actorID)
}

func (s *clientService) RemoveClientRole(ctx context.Context, ClientUUID uuid.UUID, roleUUID uuid.UUID, tenantID int64, actorUserUUID uuid.UUID) error {
	if _, err := s.requireActorTenantAccess(nil, actorUserUUID, tenantID); err != nil {
		return err
	}

	client, err := s.clientRepo.FindByUUIDAndTenantID(ClientUUID, tenantID)
	if err != nil || client == nil {
		return apperror.NewNotFoundWithReason("client not found or access denied")
	}

	// The tenant check was missing entirely here: a role UUID from another tenant
	// reached RemoveRole, and the differing response revealed that it exists.
	role, err := s.roleRepo.FindByUUID(roleUUID)
	if err != nil || role == nil || role.TenantID != tenantID {
		return apperror.NewNotFoundWithReason("role not found or access denied")
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

// firstCacheInvalidator unwraps the optional variadic invalidator.
func firstCacheInvalidator(in []cache.Invalidator) cache.Invalidator {
	if len(in) > 0 {
		return in[0]
	}
	return nil
}

// registerIssuer adds a client's domain to the JWT issuer allowlist.
//
// The `iss` claim on every issued token is the client's domain, and the JWT
// validator checks it against an allowlist seeded from the registered clients
// at startup. Without this, a client created — or re-domained — after boot
// would immediately mint tokens the validator rejects, until someone restarted
// the process.
func registerIssuer(domain *string) {
	if domain != nil {
		jwt.AddAcceptedIssuer(*domain)
	}
}

// BoundCertThumbprint returns the base64url SHA-256 thumbprint of the
// certificate this client's tokens are bound to, or "" when it has none.
//
// An empty result means "not certificate-bound" and callers treat the token as
// an ordinary bearer token. It deliberately does NOT distinguish "no binding"
// from "client not found": an unresolvable client is not bound to anything, and
// the caller's own authentication already established that the token is valid.
func (s *clientService) BoundCertThumbprint(_ context.Context, clientIdentifier string) string {
	id := strings.TrimSpace(clientIdentifier)
	if id == "" {
		return ""
	}
	c, err := s.clientRepo.FindByIdentifier(id)
	if err != nil || c == nil || c.MTLSBoundCertThumbprint == nil {
		return ""
	}
	return strings.TrimSpace(*c.MTLSBoundCertThumbprint)
}
