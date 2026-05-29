package oauth

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/maintainerd/auth/internal/platform/cache"
	"github.com/maintainerd/auth/internal/platform/database"
	"github.com/maintainerd/auth/internal/platform/pagination"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type BaseRepository[T any] = database.BaseRepository[T]
type BaseRepositoryMethods[T any] = database.BaseRepositoryMethods[T]
type PaginationResult[T any] = database.PaginationResult[T]

type PaginationRequestDTO = pagination.PaginationRequestDTO
type PaginatedResponseDTO[T any] = pagination.PaginatedResponseDTO[T]
type SuccessResponseDTO = pagination.SuccessResponseDTO

const (
	SortOrderAsc  = pagination.SortOrderAsc
	SortOrderDesc = pagination.SortOrderDesc
)

func NewBaseRepository[T any](db any, uuidFieldName, idFieldName string) *database.BaseRepository[T] {
	return database.NewBaseRepository[T](db.(*gorm.DB), uuidFieldName, idFieldName)
}

func parsePaginationQuery(r *http.Request) pagination.PaginationRequestDTO {
	return pagination.ParseQuery(r)
}

// ---------------------------------------------------------------------------
// Type aliases — cache auth types stored in middleware.AuthContext
// ---------------------------------------------------------------------------

// User and Profile are aliases for the cache auth types so that handlers can
// inject rich user data into the auth context and read it back.
type User = cache.AuthUser
type Profile = cache.AuthProfile

// ---------------------------------------------------------------------------
// OAuth constants (defined locally to avoid importing the client package)
// ---------------------------------------------------------------------------

const (
	GrantTypeAuthorizationCode = "authorization_code"
	GrantTypeClientCredentials = "client_credentials"
	GrantTypeRefreshToken      = "refresh_token"

	TokenAuthMethodSecretBasic = "client_secret_basic"
	TokenAuthMethodSecretPost  = "client_secret_post"
	TokenAuthMethodNone        = "none"

	ResponseTypeCode = "code"
)

// ---------------------------------------------------------------------------
// Local aggregate structs
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
	CreatedAt               time.Time
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
