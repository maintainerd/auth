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
	Issuer               string    `json:"issuer,omitempty"`
	ProviderClientID     string    `json:"provider_client_id,omitempty"`
	AllowJITProvisioning bool      `json:"allow_jit_provisioning"`
	EmailDomains         []string  `json:"email_domains"`
	Status               string    `json:"status"`
	IsDefault            bool      `json:"is_default"`
	IsSystem             bool      `json:"is_system"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

// OIDCProviderConfig is the runtime view of an external provider's OIDC/OAuth2
// settings stored in the IdentityProvider.Config JSONB column. It holds ONLY the
// polymorphic fields below (endpoints / scopes / attribute mapping). The issuer,
// provider client_id, client secret and allow_jit_provisioning live in dedicated
// columns and are read directly off the model (idp.IssuerOrEmpty(),
// idp.ProviderClientIDOrEmpty(), idp.DecryptedProviderClientSecret(),
// idp.AllowJITProvisioning).
type OIDCProviderConfig struct {
	Scopes           []string          `json:"scopes,omitempty"`
	AttributeMapping map[string]string `json:"attribute_mapping,omitempty"`
	UserinfoEndpoint string            `json:"userinfo_endpoint,omitempty"`
	// Explicit OAuth2/OIDC endpoints. When unset they are derived from the issuer
	// column via OIDC discovery (or, for the token endpoint, the issuer-based
	// default). Set these for providers like Google/Facebook whose endpoints are
	// not at the issuer root.
	AuthorizationEndpoint string `json:"authorization_endpoint,omitempty"`
	TokenEndpoint         string `json:"token_endpoint,omitempty"`
}

// BrokerProviderInfo holds the upstream OAuth2 authorize parameters resolved for
// a brokered identity provider: its authorization endpoint, the upstream
// client_id, and the requested scopes. Secrets are never included.
type BrokerProviderInfo struct {
	AuthorizationEndpoint string
	ClientID              string
	Scopes                []string
}

// BrokerResolvedUser is the maintainerd user resolved after exchanging an
// upstream provider's authorization code and provisioning the identity.
type BrokerResolvedUser struct {
	UserID      int64
	UserUUID    uuid.UUID
	IdentitySub string
	SessionID   string
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
	Issuer               string             `json:"issuer,omitempty"`
	ProviderClientID     string             `json:"provider_client_id,omitempty"`
	AllowJITProvisioning bool               `json:"allow_jit_provisioning"`
	AllowRegistration    bool               `json:"allow_registration"`
	EmailDomains         []string           `json:"email_domains"`
	Config               *datatypes.JSON    `json:"config,omitempty"`
	Tenant               *TenantResponseDTO `json:"tenant,omitempty"`
	Status               string             `json:"status"`
	IsDefault            bool               `json:"is_default"`
	IsSystem             bool               `json:"is_system"`
	CreatedAt            time.Time          `json:"created_at"`
	UpdatedAt            time.Time          `json:"updated_at"`
}

// Create identity provider request DTO. Security-critical/queried fields are
// now first-class top-level inputs (promoted out of the config JSONB blob):
// issuer, provider_client_id, provider_client_secret (write-only), allow_jit_provisioning and
// email_domains. Config carries only the remaining JSONB fields (endpoints /
// scopes / attribute_mapping / system settings).
type IdentityProviderCreateRequestDTO struct {
	Name                 string         `json:"name"`
	DisplayName          string         `json:"display_name"`
	Provider             string         `json:"provider"`
	ProviderType         string         `json:"provider_type"`
	Issuer               string         `json:"issuer"`
	ProviderClientID     string         `json:"provider_client_id"`
	ProviderClientSecret string         `json:"provider_client_secret"`
	AllowJITProvisioning bool           `json:"allow_jit_provisioning"`
	AllowRegistration    bool           `json:"allow_registration"`
	EmailDomains         []string       `json:"email_domains"`
	Config               datatypes.JSON `json:"config"`
	Status               string         `json:"status"`
}

// Update identity provider request dto
type IdentityProviderUpdateRequestDTO struct {
	Name                 string         `json:"name"`
	DisplayName          string         `json:"display_name"`
	Provider             string         `json:"provider"`
	ProviderType         string         `json:"provider_type"`
	Issuer               string         `json:"issuer"`
	ProviderClientID     string         `json:"provider_client_id"`
	ProviderClientSecret string         `json:"provider_client_secret"`
	AllowJITProvisioning bool           `json:"allow_jit_provisioning"`
	AllowRegistration    bool           `json:"allow_registration"`
	EmailDomains         []string       `json:"email_domains"`
	Config               datatypes.JSON `json:"config"`
	Status               string         `json:"status"`
}

// Identity provider status update DTO
type IdentityProviderStatusUpdateDTO struct {
	Status string `json:"status"`
}

// Identity provider listing / filter DTO
type IdentityProviderFilterDTO struct {
	Search       *string  `json:"search"`
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

// Auth flow output structure
type AuthFlowResponseDTO struct {
	AuthFlowUUID string    `json:"auth_flow_id"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	Identifier   string    `json:"identifier"`
	Destination  string    `json:"destination"`
	Status       string    `json:"status"`
	ClientUUID   string    `json:"client_id,omitempty"`
	BrandingUUID string    `json:"branding_id,omitempty"`
	AllowRegistration    bool   `json:"allow_registration"`
	VerificationRequired bool   `json:"verification_required"`
	RequiredFields       string `json:"required_fields"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Create auth flow request dto
type AuthFlowCreateRequestDTO struct {
	Name                  string   `json:"name"`
	Description           string   `json:"description"`
	Destination           string   `json:"destination"`
	Status                *string  `json:"status,omitempty"`
	ClientUUID            string   `json:"client_id"`
	BrandingUUID          *string  `json:"branding_id,omitempty"`
	RoleIDs               []string `json:"role_ids,omitempty"`
	ClientURIIDs          []string `json:"client_uri_ids,omitempty"`
	AllowRegistration     *bool    `json:"allow_registration,omitempty"`
	VerificationRequired  *bool    `json:"verification_required,omitempty"`
	RequiredFields        *string  `json:"required_fields,omitempty"`
}

// Update auth flow request dto. RoleIDs / ClientURIIDs, when present, replace the
// flow's role / callback-URI membership to exactly the provided set (an empty
// array clears it; omitting the field leaves it untouched).
type AuthFlowUpdateRequestDTO struct {
	Name                  string   `json:"name"`
	Description           string   `json:"description"`
	Status                *string  `json:"status,omitempty"`
	BrandingUUID          *string  `json:"branding_id,omitempty"`
	RoleIDs               []string `json:"role_ids,omitempty"`
	ClientURIIDs          []string `json:"client_uri_ids,omitempty"`
	AllowRegistration     *bool    `json:"allow_registration,omitempty"`
	VerificationRequired  *bool    `json:"verification_required,omitempty"`
	RequiredFields        *string  `json:"required_fields,omitempty"`
}

// Update signup flow status request dto
type AuthFlowUpdateStatusRequestDTO struct {
	Status string `json:"status"`
}

// Signup flow listing request dto
type AuthFlowFilterDTO struct {
	Name       *string  `json:"name"`
	Identifier *string  `json:"identifier"`
	Status     []string `json:"status"`
	ClientUUID *string  `json:"client_id"`

	// Pagination and sorting
	PaginationRequestDTO
}

// Validate validates the signup flow filter DTO.

// Signup flow role output structure
type AuthFlowRoleResponseDTO struct {
	AuthFlowRoleUUID string    `json:"auth_flow_role_id"`
	AuthFlowUUID     string    `json:"auth_flow_id"`
	RoleUUID         string    `json:"role_id"`
	RoleName         string    `json:"role_name,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
}

// Assign roles to signup flow request dto
type AuthFlowAssignRolesRequestDTO struct {
	RoleUUIDs []string `json:"role_uuids"`
}

// Auth flow callback URI output structure
type AuthFlowCallbackURIResponseDTO struct {
	AuthFlowCallbackURIUUID string    `json:"auth_flow_callback_uri_id"`
	AuthFlowUUID            string    `json:"auth_flow_id"`
	ClientURIUUID           string    `json:"client_uri_id"`
	URI                     string    `json:"uri"`
	CreatedAt               time.Time `json:"created_at"`
}

// Assign callback URIs to an auth flow request dto. Each UUID must reference one
// of the flow's client's registered URIs.
type AuthFlowAssignCallbackURIsRequestDTO struct {
	ClientURIUUIDs []string `json:"client_uri_uuids"`
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

// ── IdP Test Connection ──────────────────────────────────────────────────────

// TestConnectionRequestDTO is the JSON body for POST /identity_providers/test.
// It mirrors the unsaved fields of an IdentityProvider so the admin can validate
// a config before persisting it.
type TestConnectionRequestDTO struct {
	Provider     string `json:"provider"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	DiscoveryURL string `json:"discovery_url"`
}

// TestConnectionResultDTO is the JSON response for POST /identity_providers/test.
// Each check that passed is listed; broken checks carry an error.
type TestConnectionResultDTO struct {
	Success bool             `json:"success"`
	Checks  []TestCheckDTO   `json:"checks"`
}

// TestCheckDTO describes the result of a single validation step during an IdP
// test-connection probe.
type TestCheckDTO struct {
	Step  string `json:"step"`
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
	URL   string `json:"url,omitempty"`
}
