package client

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	TokenAuthMethodSecretBasic             = "client_secret_basic"
	TokenAuthMethodSecretPost              = "client_secret_post"
	TokenAuthMethodNone                    = "none"
	TokenAuthMethodPrivateKeyJWT           = "private_key_jwt"
	TokenAuthMethodClientSecretJWT         = "client_secret_jwt"
	TokenAuthMethodTLSClientAuth           = "tls_client_auth"
	TokenAuthMethodSelfSignedTLSClientAuth = "self_signed_tls_client_auth"
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
	ClientID                int64          `gorm:"column:client_id;primaryKey"`
	ClientUUID              uuid.UUID      `gorm:"column:client_uuid"`
	TenantID                int64          `gorm:"column:tenant_id;not null"`
	IdentityProviderID      int64          `gorm:"column:identity_provider_id"`
	Name                    string         `gorm:"column:name"`
	DisplayName             string         `gorm:"column:display_name"`
	ClientType              string         `gorm:"column:client_type"`
	Domain                  *string        `gorm:"column:domain"`
	Identifier              *string        `gorm:"column:identifier"`
	SecretHash              *string        `gorm:"column:secret_hash"`
	PreviousSecretHash      *string        `gorm:"column:previous_secret_hash"`
	PreviousSecretExpiresAt *time.Time     `gorm:"column:previous_secret_expires_at"`
	Config                  datatypes.JSON `gorm:"column:config"`
	Status                  string         `gorm:"column:status;default:'inactive'"`
	IsDefault               bool           `gorm:"column:is_default;default:false"`
	IsSystem                bool           `gorm:"column:is_system;default:false"`
	CreatedBy               *int64         `gorm:"column:created_by"`
	UpdatedBy               *int64         `gorm:"column:updated_by"`
	CreatedAt               time.Time      `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt               time.Time      `gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt               gorm.DeletedAt `gorm:"column:deleted_at;index"`

	// OAuth 2.0 fields
	TokenEndpointAuthMethod string         `gorm:"column:token_endpoint_auth_method;default:'client_secret_basic'"`
	GrantTypes              pq.StringArray `gorm:"column:grant_types;type:text[]"`
	ResponseTypes           pq.StringArray `gorm:"column:response_types;type:text[]"`
	AccessTokenTTL          *int           `gorm:"column:access_token_ttl"`
	RefreshTokenTTL         *int           `gorm:"column:refresh_token_ttl"`
	RequireConsent          bool           `gorm:"column:require_consent;default:true"`
	AllowedScopes           pq.StringArray `gorm:"column:allowed_scopes;type:text[]"`

	// JWT client auth (RFC 7523)
	JWKS    datatypes.JSON `gorm:"column:jwks;type:jsonb"`
	JWKSUri *string        `gorm:"column:jwks_uri"`

	// mTLS client auth (RFC 8705) — SHA-256 fingerprint of the expected certificate
	MTLSBoundCertThumbprint *string `gorm:"column:mtls_bound_cert_thumbprint"`

	// Scope-to-claim mapping: overrides default OIDC scope → claims table per client.
	// Stored as {"scope": ["claim1", "claim2"]}.
	ScopeClaimMappings datatypes.JSON `gorm:"column:scope_claim_mappings;type:jsonb"`

	// ClaimMappers: static or metadata-derived extra claims injected into tokens.
	// Stored as {"claim_name": "static_value"}.
	ClaimMappers datatypes.JSON `gorm:"column:claim_mappers;type:jsonb"`

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
