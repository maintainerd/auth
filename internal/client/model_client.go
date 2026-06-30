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

type Branding struct {
	BrandingID   int64     `gorm:"column:branding_id;primaryKey"`
	BrandingUUID uuid.UUID `gorm:"column:branding_uuid"`
	TenantID     int64     `gorm:"column:tenant_id"`
	Name         string    `gorm:"column:name"`
}

func (Branding) TableName() string { return "branding" }

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

// Client represents an OAuth2/OIDC downstream relying-party application (SPA,
// traditional web, mobile/native, or M2M) registered under a tenant. The OAuth
// columns describe how the application authenticates to and obtains tokens from
// this authorization server. External provider credentials are not stored here;
// they live in dedicated identity_providers columns (provider_client_id,
// provider_client_secret_encrypted) and are enabled per client through
// client_identity_providers.
//
// Field groups mirror the clients migration: identity & ownership, descriptive,
// secret storage, config & lifecycle, OAuth core, token lifetime, security
// overrides, advanced client auth, claims, and audit.
type Client struct {
	// Identity & ownership
	ClientID   int64     `gorm:"column:client_id;primaryKey"`
	ClientUUID uuid.UUID `gorm:"column:client_uuid"`
	TenantID   int64     `gorm:"column:tenant_id;not null"`
	ServiceID  *int64    `gorm:"column:service_id"`

	// Legacy read projection for callers that have not migrated to
	// ConnectedProviders yet. This is populated from client_identity_providers,
	// not persisted on clients.
	IdentityProviderID int64             `gorm:"-"`
	IdentityProvider   *IdentityProvider `gorm:"-"`

	// Descriptive
	Name        string  `gorm:"column:name"`
	DisplayName string  `gorm:"column:display_name"`
	ClientType  string  `gorm:"column:client_type"`
	Domain      *string `gorm:"column:domain"`
	Identifier  *string `gorm:"column:identifier"`

	// Secret storage (nullable — public clients carry no secret)
	SecretHash              *string    `gorm:"column:secret_hash"`
	SecretEncrypted         *string    `gorm:"column:secret_encrypted"`
	PreviousSecretHash      *string    `gorm:"column:previous_secret_hash"`
	PreviousSecretEncrypted *string    `gorm:"column:previous_secret_encrypted"`
	PreviousSecretExpiresAt *time.Time `gorm:"column:previous_secret_expires_at"`

	// Free-form config blob + lifecycle
	Config            datatypes.JSON `gorm:"column:config"`
	Status            string         `gorm:"column:status;default:'inactive'"`
	IsDefault         bool           `gorm:"column:is_default;default:false"`
	IsSystem          bool           `gorm:"column:is_system;default:false"`
	BrandingID        *int64         `gorm:"column:branding_id" json:"branding_id,omitempty"`
	AllowRegistration bool           `gorm:"column:allow_registration;not null;default:true" json:"allow_registration"`

	// OAuth 2.0 core
	TokenEndpointAuthMethod string         `gorm:"column:token_endpoint_auth_method;default:'client_secret_basic'"`
	GrantTypes              pq.StringArray `gorm:"column:grant_types;type:text[];default:'{authorization_code}'"`
	ResponseTypes           pq.StringArray `gorm:"column:response_types;type:text[];default:'{code}'"`
	RequireConsent          bool           `gorm:"column:require_consent;default:true"`
	// RequirePKCE: pointer so an explicit false is distinguishable from "unset"
	// (unset → DB default TRUE). PKCE is mandatory for public clients.
	RequirePKCE   *bool          `gorm:"column:require_pkce;default:true"`
	AllowedScopes pq.StringArray `gorm:"column:allowed_scopes;type:text[];default:'{}'"`

	// Token lifetime overrides (seconds). nil = inherit tenant token_config.
	AccessTokenTTL  *int `gorm:"column:access_token_ttl"`
	RefreshTokenTTL *int `gorm:"column:refresh_token_ttl"`

	// Security overrides — runtime enforcement only, tighten-only.
	// nil = inherit the tenant security_settings default.
	RequiredACR            *string `gorm:"column:required_acr"`             // MFA/step-up: "1" pwd, "2" step-up
	SessionIdleTimeout     *int    `gorm:"column:session_idle_timeout"`     // sliding idle window (seconds)
	SessionAbsoluteTimeout *int    `gorm:"column:session_absolute_timeout"` // hard session cap (seconds)

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

	// Audit
	CreatedBy *int64         `gorm:"column:created_by"`
	UpdatedBy *int64         `gorm:"column:updated_by"`
	CreatedAt time.Time      `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time      `gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at;index"`

	// Relationships
	Tenant             *Tenant                   `gorm:"foreignKey:TenantID;references:TenantID"`
	Branding           *Branding                 `gorm:"foreignKey:BrandingID;references:BrandingID"`
	ConnectedProviders *[]ClientIdentityProvider `gorm:"foreignKey:ClientID;references:ClientID"`
	ClientURIs         *[]ClientURI              `gorm:"foreignKey:ClientID;references:ClientID"`
	ClientAPIs         *[]ClientAPI              `gorm:"foreignKey:ClientID;references:ClientID"`
}

func (Client) TableName() string {
	return "clients"
}

func (ac *Client) BeforeCreate(tx *gorm.DB) (err error) {
	if ac.ClientUUID == uuid.Nil {
		ac.ClientUUID = uuid.New()
	}
	if ac.GrantTypes == nil {
		ac.GrantTypes = pq.StringArray{"authorization_code"}
	}
	if ac.ResponseTypes == nil {
		ac.ResponseTypes = pq.StringArray{"code"}
	}
	if ac.AllowedScopes == nil {
		ac.AllowedScopes = pq.StringArray{}
	}
	return
}
