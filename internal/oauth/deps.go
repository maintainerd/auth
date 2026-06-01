package oauth

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
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
	IsPublic    bool
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
	Provider             string
	ProviderType         string
	Identifier           string
	Status               string
	IsDefault            bool
	IsSystem             bool
	Tenant               *Tenant `gorm:"foreignKey:TenantID;references:TenantID"`
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

func (IdentityProvider) TableName() string { return "identity_providers" }

type Client struct {
	ClientID                int64             `gorm:"column:client_id;primaryKey"`
	ClientUUID              uuid.UUID         `gorm:"column:client_uuid"`
	TenantID                int64             `gorm:"column:tenant_id"`
	IdentityProviderID      int64             `gorm:"column:identity_provider_id"`
	Name                    string            `gorm:"column:name"`
	DisplayName             string            `gorm:"column:display_name"`
	ClientType              string            `gorm:"column:client_type"`
	Domain                  *string           `gorm:"column:domain"`
	Identifier              *string           `gorm:"column:identifier"`
	SecretHash              *string           `gorm:"column:secret_hash"`
	PreviousSecretHash      *string           `gorm:"column:previous_secret_hash"`
	PreviousSecretExpiresAt *time.Time        `gorm:"column:previous_secret_expires_at"`
	Status                  string            `gorm:"column:status"`
	IsDefault               bool              `gorm:"column:is_default"`
	IsSystem                bool              `gorm:"column:is_system"`
	TokenEndpointAuthMethod string            `gorm:"column:token_endpoint_auth_method"`
	GrantTypes              pq.StringArray    `gorm:"column:grant_types;type:text[]"`
	ResponseTypes           pq.StringArray    `gorm:"column:response_types;type:text[]"`
	AccessTokenTTL          *int              `gorm:"column:access_token_ttl"`
	RefreshTokenTTL         *int              `gorm:"column:refresh_token_ttl"`
	RequireConsent          bool              `gorm:"column:require_consent"`
	AllowedScopes           pq.StringArray    `gorm:"column:allowed_scopes;type:text[]"`
	ClientURIs              *[]ClientURI      `gorm:"foreignKey:ClientID;references:ClientID"`
	IdentityProvider        *IdentityProvider `gorm:"foreignKey:IdentityProviderID;references:IdentityProviderID"`

	JWKS    datatypes.JSON `gorm:"column:jwks;type:jsonb"`
	JWKSUri *string        `gorm:"column:jwks_uri"`

	CreatedAt time.Time
	UpdatedAt               time.Time
	DeletedAt               gorm.DeletedAt `gorm:"index"`
}

func (Client) TableName() string { return "clients" }

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

type ClientRepository interface {
	BaseRepositoryMethods[Client]
	WithTx(tx *gorm.DB) ClientRepository
	FindSystem() (*Client, error)
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
	WithTx(tx *gorm.DB) UserRepository
	FindByEmail(email string) (*User, error)
	FindBySubAndClientID(sub, clientID string) (*User, error)
}

type UserIdentityRepository interface {
	BaseRepositoryMethods[UserIdentity]
	WithTx(tx *gorm.DB) UserIdentityRepository
	FindByUserIDAndClientID(userID, clientID int64) (*UserIdentity, error)
}
