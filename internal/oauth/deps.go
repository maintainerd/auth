package oauth

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// ---------------------------------------------------------------------------
// Upstream-domain projections
//
// These structs are local projections of types owned by other domains (tenant,
// idp, client, user). They carry no json tags because they never touch the
// wire; the oauth package declares them so it does not import those domains
// directly. The composition root wires adapters that satisfy the consumer
// repository interfaces below.
// ---------------------------------------------------------------------------

type Tenant struct {
	TenantID    int64
	TenantUUID  uuid.UUID
	Name        string
	DisplayName string
	Description string
	Identifier  string
	Status      string
	IsSystem    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (Tenant) TableName() string { return "tenants" }

type IdentityProvider struct {
	IdentityProviderID   int64
	IdentityProviderUUID uuid.UUID
	TenantID             int64
	Name                 string
	DisplayName          string
	Provider             string
	ProviderType         string
	Identifier           string
	Status               string
	IsDefault            bool
	IsSystem             bool
	AllowRegistration    bool
	Tenant               *Tenant `gorm:"foreignKey:TenantID;references:TenantID"`
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

func (IdentityProvider) TableName() string { return "identity_providers" }

// ClientIdentityProvider projects the client_identity_providers join so the
// connections endpoint can list a client's enabled login providers. It carries
// no provider config/secrets.
type ClientIdentityProvider struct {
	ClientIdentityProviderID   int64             `gorm:"column:client_identity_provider_id;primaryKey"`
	ClientIdentityProviderUUID uuid.UUID         `gorm:"column:client_identity_provider_uuid"`
	TenantID                   int64             `gorm:"column:tenant_id"`
	ClientID                   int64             `gorm:"column:client_id"`
	IdentityProviderID         int64             `gorm:"column:identity_provider_id"`
	IsDefault                  bool              `gorm:"column:is_default"`
	Enabled                    bool              `gorm:"column:enabled"`
	DisplayOrder               int               `gorm:"column:display_order"`
	IdentityProvider           *IdentityProvider `gorm:"foreignKey:IdentityProviderID;references:IdentityProviderID"`
	DeletedAt                  gorm.DeletedAt    `gorm:"column:deleted_at;index"`
}

func (ClientIdentityProvider) TableName() string { return "client_identity_providers" }

type Client struct {
	ClientID                int64             `gorm:"column:client_id;primaryKey"`
	ClientUUID              uuid.UUID         `gorm:"column:client_uuid"`
	TenantID                int64             `gorm:"column:tenant_id"`
	ServiceID               *int64            `gorm:"column:service_id"`
	IdentityProviderID      int64             `gorm:"-"`
	IdentityProvider        *IdentityProvider `gorm:"-"`
	Name                    string            `gorm:"column:name"`
	DisplayName             string            `gorm:"column:display_name"`
	ClientType              string            `gorm:"column:client_type"`
	Domain                  *string           `gorm:"column:domain"`
	Identifier              *string           `gorm:"column:identifier"`
	SecretHash              *string           `gorm:"column:secret_hash"`
	SecretEncrypted         *string           `gorm:"column:secret_encrypted"`
	PreviousSecretHash      *string           `gorm:"column:previous_secret_hash"`
	PreviousSecretEncrypted *string           `gorm:"column:previous_secret_encrypted"`
	PreviousSecretExpiresAt *time.Time        `gorm:"column:previous_secret_expires_at"`
	Status                  string            `gorm:"column:status"`
	IsDefault               bool              `gorm:"column:is_default"`
	IsSystem                bool              `gorm:"column:is_system"`
	TokenEndpointAuthMethod string            `gorm:"column:token_endpoint_auth_method"`
	GrantTypes              pq.StringArray    `gorm:"column:grant_types;type:text[]"`
	ResponseTypes           pq.StringArray    `gorm:"column:response_types;type:text[]"`
	RequirePKCE             *bool             `gorm:"column:require_pkce"`
	AccessTokenTTL          *int              `gorm:"column:access_token_ttl"`
	RefreshTokenTTL         *int              `gorm:"column:refresh_token_ttl"`
	RequiredACR             *string           `gorm:"column:required_acr"`
	SessionIdleTimeout      *int              `gorm:"column:session_idle_timeout"`
	SessionAbsoluteTimeout  *int              `gorm:"column:session_absolute_timeout"`
	BrandingID              *int64            `gorm:"column:branding_id"`
	AllowRegistration       bool              `gorm:"column:allow_registration;not null;default:true"`
	RequireConsent          bool              `gorm:"column:require_consent"`
	AllowedScopes           pq.StringArray    `gorm:"column:allowed_scopes;type:text[]"`
	ClientURIs              *[]ClientURI      `gorm:"foreignKey:ClientID;references:ClientID"`
	Tenant                  *Tenant           `gorm:"foreignKey:TenantID;references:TenantID"`
	Service                 *Service          `gorm:"foreignKey:ServiceID;references:ServiceID"`

	JWKS    datatypes.JSON `gorm:"column:jwks;type:jsonb"`
	JWKSUri *string        `gorm:"column:jwks_uri"`

	ScopeClaimMappings datatypes.JSON `gorm:"column:scope_claim_mappings;type:jsonb"`
	ClaimMappers       datatypes.JSON `gorm:"column:claim_mappers;type:jsonb"`

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (Client) TableName() string { return "clients" }

type Service struct {
	ServiceID int64
	Name      string
	Status    string
}

func (Service) TableName() string { return "services" }

type ClientURI struct {
	ClientURIID   int64     `gorm:"column:client_uri_id;primaryKey"`
	ClientURIUUID uuid.UUID `gorm:"column:client_uri_uuid"`
	TenantID      int64     `gorm:"column:tenant_id"`
	ClientID      int64     `gorm:"column:client_id"`
	URI           string    `gorm:"column:uri"`
	Type          string    `gorm:"column:type"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (ClientURI) TableName() string { return "client_uris" }

type UserIdentity struct {
	UserIdentityID   int64          `gorm:"column:user_identity_id;primaryKey"`
	UserIdentityUUID uuid.UUID      `gorm:"column:user_identity_uuid"`
	TenantID         int64          `gorm:"column:tenant_id"`
	UserID           int64          `gorm:"column:user_id"`
	ClientID         int64          `gorm:"column:client_id"`
	Provider         string         `gorm:"column:provider"`
	Sub              string         `gorm:"column:sub"`
	Metadata         datatypes.JSON `gorm:"column:metadata"`
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (UserIdentity) TableName() string { return "user_identities" }

// ---------------------------------------------------------------------------
// Consumer repository interfaces
// ---------------------------------------------------------------------------

// BrokerProvider holds the upstream OAuth2 authorize parameters for a brokered
// identity provider, resolved (and secret-free) by a BrokerProviderResolver.
type BrokerProvider struct {
	AuthorizationEndpoint string
	ClientID              string
	Scopes                []string
}

// BrokerProviderResolver resolves the upstream OAuth2 authorize parameters for an
// identity provider by identifier. It is satisfied by an idp-package adapter
// wired at startup via SetBrokerProviderResolver, so the oauth package never
// imports the idp domain or handles provider secrets.
type BrokerProviderResolver interface {
	ResolveBrokerProvider(ctx context.Context, idpIdentifier string) (*BrokerProvider, error)
}

// BrokerResolvedUser is the maintainerd user resolved after exchanging an
// upstream provider's authorization code.
type BrokerResolvedUser struct {
	UserID      int64
	UserUUID    uuid.UUID
	IdentitySub string
	SessionID   string
}

// BrokerCallbackResolver resolves the maintainerd user for the broker callback
// by exchanging the upstream provider's authorization code (with PKCE),
// validating the id_token, and provisioning the user identity.
type BrokerCallbackResolver interface {
	ResolveBrokerUser(ctx context.Context, idpID int64, code, pkceVerifier, nonce, redirectURI string, clientID int64) (*BrokerResolvedUser, error)
}

type ClientRepository interface {
	BaseRepositoryMethods[Client]
	WithTx(tx *gorm.DB) ClientRepository
	FindByID(id any, preloads ...string) (*Client, error)
	FindSystem() (*Client, error)
	FindByIdentifier(identifier string) (*Client, error)
	FindSystemByTenantIdentifierAndName(tenantIdentifier, name string) (*Client, error)
	FindByClientIDAndIdentityProvider(clientID, identityProviderIdentifier string) (*Client, error)
}

type ClientURIRepository interface {
	BaseRepositoryMethods[ClientURI]
	WithTx(tx *gorm.DB) ClientURIRepository
}

type TenantRepository interface {
	BaseRepositoryMethods[Tenant]
	WithTx(tx *gorm.DB) TenantRepository
	FindSystem() (*Tenant, error)
}

type UserRepository interface {
	BaseRepositoryMethods[User]
	FindByID(id any, preloads ...string) (*User, error)
	WithTx(tx *gorm.DB) UserRepository
	// FindByEmailAndTenantID scopes the lookup to a tenant; users are isolated
	// per tenant and no unscoped email lookup is exposed.
	FindByEmailAndTenantID(email string, tenantID int64) (*User, error)
	FindBySubAndClientID(sub, clientID string) (*User, error)
}

type UserIdentityRepository interface {
	BaseRepositoryMethods[UserIdentity]
	WithTx(tx *gorm.DB) UserIdentityRepository
	FindByUserIDAndClientID(userID, clientID int64) (*UserIdentity, error)
}

// ClientPermissionResolver resolves the set of permissions for a system client
// by merging both direct client_permissions and role-inherited client_roles →
// role_permissions. Returns deduplicated permission names.
type ClientPermissionResolver interface {
	ResolvePermissions(ctx context.Context, clientID int64) ([]string, error)
}

// ---------------------------------------------------------------------------
// OAuth-package-owned repository interfaces
//
// These were previously declared next to their implementations. They live here
// so that other files in the oauth package can reference them without importing
// the concrete repository files, following the same consumer-interface pattern
// used for cross-domain repos above.
// ---------------------------------------------------------------------------

// SigningKeyRepository manages asymmetric key pairs used for token signing.
type SigningKeyRepository interface {
	FindActiveByTenantID(tenantID int64) ([]SigningKey, error)
	FindByKID(kid string) (*SigningKey, error)
	Create(key *SigningKey) error
	RetireByKID(kid string) error
	MarkCompromised(kid string) error
}

// OAuthTokenRevocationRepository stores revoked token JTIs for introspection checks.
type OAuthTokenRevocationRepository interface {
	Revoke(revocation *OAuthTokenRevocation) error
	IsRevoked(tenantID int64, jti string) (bool, error)
	// IsRevokedByJTI checks revocation by JTI alone, without a tenant filter.
	// Used by the JWT bearer middleware where tenant context is not yet available.
	IsRevokedByJTI(jti string) (bool, error)
	DeleteExpired() (int64, error)
}

// OAuthTokenExchangeRepository records RFC 8693 token exchange events for audit.
type OAuthTokenExchangeRepository interface {
	Record(exchange *OAuthTokenExchange) error
}

// OAuthDPoPNonceRepository persists server-issued DPoP nonces (RFC 9449 §8).
type OAuthDPoPNonceRepository interface {
	// SaveNonce persists a freshly issued nonce for a client with a TTL.
	SaveNonce(tenantID, clientID int64, nonce string, expiresAt time.Time) error
	// ConsumeNonce atomically validates and marks a nonce used. Returns ok=false
	// when the nonce is unknown, already used, or expired.
	ConsumeNonce(nonce string) (ok bool, err error)
	// DeleteExpired removes expired nonces (cleanup worker). Returns rows deleted.
	DeleteExpired() (int64, error)
}

// DPoPRequirement is the resolved DPoP configuration for a token-endpoint client.
type DPoPRequirement struct {
	Required         bool
	TenantID         int64
	InternalClientID int64
}

// DPoPRequirementResolver resolves whether a client (by presented client_id /
// identifier) requires DPoP, for the token endpoint's nonce gate (RFC 9449 §8).
// It is satisfied by an app adapter over the clients table so the oauth handler
// does not resolve the client inline. ok=false means the client could not be
// resolved — the normal token flow then reports the auth error.
type DPoPRequirementResolver interface {
	ResolveDPoPRequirement(ctx context.Context, clientID string) (requirement DPoPRequirement, ok bool)
}

// WorkloadTokenExchangeInput carries the parameters for a workload identity
// federation token exchange, parsed from the /oauth/token form body.
type WorkloadTokenExchangeInput struct {
	SubjectToken string
	Scope        string
	Audience     string
	Resource     string
	IPAddress    string
}

// WorkloadTokenExchangeResult is the issued token for a successful workload
// identity federation exchange.
type WorkloadTokenExchangeResult struct {
	AccessToken     string
	IssuedTokenType string
	TokenType       string
	ExpiresIn       int
	Scope           string
}

// WorkloadIdentityExchanger exchanges an external OIDC/JWT subject token for a
// platform access token via a configured workload identity federation (section
// 3.21). It is satisfied by a federation-package adapter wired at startup, so
// the oauth package never imports the federation domain.
//
// ExchangeWorkloadToken returns (nil, nil) when no federation trusts the
// token's issuer, signalling the token endpoint to fall back to the standard
// RFC 8693 token-exchange path.
type WorkloadIdentityExchanger interface {
	ExchangeWorkloadToken(ctx context.Context, in WorkloadTokenExchangeInput) (*WorkloadTokenExchangeResult, *apperror.OAuthError)
}
