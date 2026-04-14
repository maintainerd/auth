package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// TokenEndpointAuthMethod constants for the token_endpoint_auth_method column.
const (
	TokenAuthMethodSecretBasic = "client_secret_basic"
	TokenAuthMethodSecretPost  = "client_secret_post"
	TokenAuthMethodNone        = "none"
)

// OAuth grant type constants.
const (
	GrantTypeAuthorizationCode = "authorization_code"
	GrantTypeClientCredentials = "client_credentials"
	GrantTypeRefreshToken      = "refresh_token"
)

// OAuth response type constants.
const (
	ResponseTypeCode = "code"
)

// Client represents an OAuth2/OIDC client application registered with an
// identity provider under a tenant.
type Client struct {
	ClientID           int64          `gorm:"column:client_id;primaryKey"`
	ClientUUID         uuid.UUID      `gorm:"column:client_uuid"`
	TenantID           int64          `gorm:"column:tenant_id;not null"`
	IdentityProviderID int64          `gorm:"column:identity_provider_id"`
	Name               string         `gorm:"column:name"`
	DisplayName        string         `gorm:"column:display_name"`
	ClientType         string         `gorm:"column:client_type"`
	Domain             *string        `gorm:"column:domain"`
	Identifier         *string        `gorm:"column:identifier"`
	Secret             *string        `gorm:"column:secret"`
	Config             datatypes.JSON `gorm:"column:config"`
	Status             string         `gorm:"column:status;default:'inactive'"`
	IsDefault          bool           `gorm:"column:is_default;default:false"`
	IsSystem           bool           `gorm:"column:is_system;default:false"`
	CreatedAt          time.Time      `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt          time.Time      `gorm:"column:updated_at;autoUpdateTime"`

	// OAuth 2.0 fields
	TokenEndpointAuthMethod string         `gorm:"column:token_endpoint_auth_method;default:'client_secret_basic'"`
	GrantTypes              pq.StringArray `gorm:"column:grant_types;type:text[]"`
	ResponseTypes           pq.StringArray `gorm:"column:response_types;type:text[]"`
	AccessTokenTTL          *int           `gorm:"column:access_token_ttl"`
	RefreshTokenTTL         *int           `gorm:"column:refresh_token_ttl"`
	RequireConsent          bool           `gorm:"column:require_consent;default:true"`

	// Relationships
	IdentityProvider *IdentityProvider `gorm:"foreignKey:IdentityProviderID;references:IdentityProviderID"`
	ClientURIs       *[]ClientURI      `gorm:"foreignKey:ClientID;references:ClientID"`
	ClientAPIs       *[]ClientAPI      `gorm:"foreignKey:ClientID;references:ClientID"`
}

func (Client) TableName() string {
	return "clients"
}

func (ac *Client) BeforeCreate(tx *gorm.DB) (err error) {
	if ac.ClientUUID == uuid.Nil {
		ac.ClientUUID = uuid.New()
	}
	return
}
