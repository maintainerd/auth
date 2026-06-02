package idp

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type FederationTokenRequestDTO struct {
	// ProviderIdentifier is the identifier of the configured IdentityProvider
	// record (e.g. "idp-abc123xyz").
	ProviderIdentifier string `json:"provider_identifier"`
	// ExternalToken is the raw OIDC ID token (JWT) from the upstream provider.
	ExternalToken string `json:"external_token"`
	// ClientID is our OAuth2 client identifier used to scope the issued tokens.
	ClientID string `json:"client_id"`
}

// FederationOAuth2CallbackDTO is the body for POST /federation/oauth2/callback.
// Clients send the authorization code received from the upstream OAuth2 provider
// after the user has authorized and been redirected back.
type FederationOAuth2CallbackDTO struct {
	ProviderIdentifier string `json:"provider_identifier"`
	Code               string `json:"code"`
	RedirectURI        string `json:"redirect_uri"`
	ClientID           string `json:"client_id"`
}

// Identity link / unlink

// LinkIdentityRequestDTO is the body for POST /account/identities/link.
type LinkIdentityRequestDTO struct {
	ProviderIdentifier string `json:"provider_identifier"`
	ExternalToken      string `json:"external_token"`
}

// IdentityDTO is the public view of a UserIdentity record.
type IdentityDTO struct {
	IdentityUUID string  `json:"identity_uuid"`
	Provider     string  `json:"provider"`
	Sub          string  `json:"sub"`
	IsDefault    bool    `json:"is_default"`
	CreatedAt    string  `json:"created_at"`
	Email        *string `json:"email,omitempty"`
	Name         *string `json:"name,omitempty"`
	Picture      *string `json:"picture,omitempty"`
}

// Home Realm Discovery

// HRDResponseDTO tells the frontend which provider handles the given email.
type HRDResponseDTO struct {
	ProviderIdentifier string `json:"provider_identifier"`
	Provider           string `json:"provider"`
	DisplayName        string `json:"display_name"`
}

// Identity provider list response structure (without config and tenant)
type IdentityProviderResponseDTO struct {
	IdentityProviderUUID uuid.UUID `json:"identity_provider_id"`
	Name                 string    `json:"name"`
	DisplayName          string    `json:"display_name"`
	Provider             string    `json:"provider"`
	ProviderType         string    `json:"provider_type"`
	Identifier           string    `json:"identifier"`
	Status               string    `json:"status"`
	IsDefault            bool      `json:"is_default"`
	IsSystem             bool      `json:"is_system"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

// OIDCProviderConfig is stored as JSONB in IdentityProvider.Config for
// providers with provider_type = "social" (external OIDC/OAuth2 upstreams).
type OIDCProviderConfig struct {
	Issuer               string            `json:"issuer"`
	ClientID             string            `json:"client_id"`
	ClientSecret         string            `json:"client_secret,omitempty"`
	Scopes               []string          `json:"scopes,omitempty"`
	AllowJITProvisioning bool              `json:"allow_jit_provisioning"`
	AttributeMapping     map[string]string `json:"attribute_mapping,omitempty"`
	EmailDomains         []string          `json:"email_domains,omitempty"`
	UserinfoEndpoint     string            `json:"userinfo_endpoint,omitempty"`
}

// IdentityMetadata is stored as JSONB in UserIdentity.Metadata for external
// provider identities.
type IdentityMetadata struct {
	Email         string `json:"email,omitempty"`
	EmailVerified bool   `json:"email_verified,omitempty"`
	Name          string `json:"name,omitempty"`
	GivenName     string `json:"given_name,omitempty"`
	FamilyName    string `json:"family_name,omitempty"`
	Picture       string `json:"picture,omitempty"`
	Locale        string `json:"locale,omitempty"`
}

// Identity provider detail response structure (with config and tenant)
type IdentityProviderDetailResponseDTO struct {
	IdentityProviderUUID uuid.UUID          `json:"identity_provider_id"`
	Name                 string             `json:"name"`
	DisplayName          string             `json:"display_name"`
	Provider             string             `json:"provider"`
	ProviderType         string             `json:"provider_type"`
	Identifier           string             `json:"identifier"`
	Config               *datatypes.JSON    `json:"config,omitempty"`
	Tenant               *TenantResponseDTO `json:"tenant,omitempty"`
	Status               string             `json:"status"`
	IsDefault            bool               `json:"is_default"`
	IsSystem             bool               `json:"is_system"`
	CreatedAt            time.Time          `json:"created_at"`
	UpdatedAt            time.Time          `json:"updated_at"`
}

// Create identity provider request DTO
type IdentityProviderCreateRequestDTO struct {
	Name         string         `json:"name"`
	DisplayName  string         `json:"display_name"`
	Provider     string         `json:"provider"`
	ProviderType string         `json:"provider_type"`
	Config       datatypes.JSON `json:"config"`
	Status       string         `json:"status"`
	TenantUUID   string         `json:"tenant_id"`
}

// Update identity provider request DTO (without tenant_id)
type IdentityProviderUpdateRequestDTO struct {
	Name         string         `json:"name"`
	DisplayName  string         `json:"display_name"`
	Provider     string         `json:"provider"`
	ProviderType string         `json:"provider_type"`
	Config       datatypes.JSON `json:"config"`
	Status       string         `json:"status"`
}

// Identity provider status update DTO
type IdentityProviderStatusUpdateDTO struct {
	Status string `json:"status"`
}

// Identity provider listing / filter DTO
type IdentityProviderFilterDTO struct {
	Name         *string  `json:"name"`
	DisplayName  *string  `json:"display_name"`
	Provider     []string `json:"provider"`
	ProviderType *string  `json:"provider_type"`
	Identifier   *string  `json:"identifier"`
	Status       []string `json:"status"`
	IsDefault    *bool    `json:"is_default"`
	IsSystem     *bool    `json:"is_system"`

	// Pagination and sorting
	PaginationRequestDTO
}

// Signup flow output structure
type SignupFlowResponseDTO struct {
	SignupFlowUUID string         `json:"signup_flow_id"`
	Name           string         `json:"name"`
	Description    string         `json:"description"`
	Identifier     string         `json:"identifier"`
	Config         map[string]any `json:"config"`
	Status         string         `json:"status"`
	ClientUUID     string         `json:"client_id,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

// Create signup flow request dto
type SignupFlowCreateRequestDTO struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Config      map[string]any `json:"config,omitempty"`
	Status      *string        `json:"status,omitempty"`
	ClientUUID  string         `json:"client_id"`
}

// Update signup flow request dto
type SignupFlowUpdateRequestDTO struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Config      map[string]any `json:"config,omitempty"`
	Status      *string        `json:"status,omitempty"`
}

// Update signup flow status request dto
type SignupFlowUpdateStatusRequestDTO struct {
	Status string `json:"status"`
}

// Signup flow listing request dto
type SignupFlowFilterDTO struct {
	Name       *string  `json:"name"`
	Identifier *string  `json:"identifier"`
	Status     []string `json:"status"`
	ClientUUID *string  `json:"client_id"`

	// Pagination and sorting
	PaginationRequestDTO
}

// Validate validates the signup flow filter DTO.

// Signup flow role output structure
type SignupFlowRoleResponseDTO struct {
	SignupFlowRoleUUID string    `json:"signup_flow_role_id"`
	SignupFlowUUID     string    `json:"signup_flow_id"`
	RoleUUID           string    `json:"role_id"`
	RoleName           string    `json:"role_name,omitempty"`
	CreatedAt          time.Time `json:"created_at"`
}

// Assign roles to signup flow request dto
type SignupFlowAssignRolesRequestDTO struct {
	RoleUUIDs []string `json:"role_uuids"`
}

type LoginResponseDTO struct {
	AccessToken           string  `json:"access_token"`
	IDToken               string  `json:"id_token"`
	RefreshToken          string  `json:"refresh_token,omitempty"`
	ExpiresIn             int64   `json:"expires_in"`
	TokenType             string  `json:"token_type"`
	IssuedAt              int64   `json:"issued_at"`
	RequirePasswordChange bool    `json:"require_password_change,omitempty"`
	SessionID             *string `json:"session_id,omitempty"`
}

type TenantResponseDTO struct {
	TenantUUID  uuid.UUID `json:"tenant_id"`
	Name        string    `json:"name"`
	DisplayName string    `json:"display_name,omitempty"`
	Description string    `json:"description"`
	Identifier  string    `json:"identifier"`
	Status      string    `json:"status"`
	IsPublic    bool      `json:"is_public"`
	IsSystem    bool      `json:"is_system,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type RoleResponseDTO struct {
	RoleUUID    uuid.UUID `json:"role_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	IsDefault   bool      `json:"is_default"`
	IsSystem    bool      `json:"is_system"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
