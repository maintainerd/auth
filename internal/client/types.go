package client

import (
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/branding"
	"gorm.io/datatypes"
)

type ClientSecretResponseDTO struct {
	ClientID     string  `json:"client_id"`
	ClientSecret *string `json:"client_secret"`
}

// ClientCreateSecretResponseDTO is returned exactly once at client creation.
// The plaintext secret is never stored and cannot be retrieved again.
type ClientCreateSecretResponseDTO struct {
	ClientUUID   string `json:"client_uuid"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

// RotateSecretRequestDTO controls secret rotation behaviour.
type RotateSecretRequestDTO struct {
	// GracePeriodHours keeps the old secret valid for this many hours (0 = revoke immediately).
	GracePeriodHours int `json:"grace_period_hours"`
}

// RotateSecretResponseDTO is returned exactly once after rotation.
type RotateSecretResponseDTO struct {
	ClientID                string  `json:"client_id"`
	ClientSecret            string  `json:"client_secret"`
	PreviousSecretExpiresAt *string `json:"previous_secret_expires_at,omitempty"`
}

type ClientURIResponseDTO struct {
	ClientURIUUID uuid.UUID `json:"uri_id"`
	URI           string    `json:"uri"`
	Type          string    `json:"type"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type ClientURIsResponseDTO struct {
	URIs []ClientURIResponseDTO `json:"uris"`
}

type ClientAPIsResponseDTO struct {
	APIs []ClientAPIResponseDTO `json:"apis"`
}

type ClientAPIPermissionsResponseDTO struct {
	Permissions []PermissionResponseDTO `json:"permissions"`
}

// Auth client output structure
type ClientResponseDTO struct {
	// ClientUUID is the management handle used in console URLs. Note the json name
	// is "client_id" for backward compatibility even though the OAuth client_id is
	// Identifier below — do not confuse the two.
	ClientUUID uuid.UUID `json:"client_id"`
	// Identifier is the OAuth 2.0 client_id: the value an application actually
	// puts in an authorize or token request, and what /oauth/connections resolves
	// a client by. Without it in the response an operator cannot configure their
	// app, so it is exposed here even though it is server-generated.
	Identifier *string `json:"identifier,omitempty"`
	// ServiceUUID is the service this client authenticates as, when bound.
	ServiceUUID *string `json:"service_id,omitempty"`

	Name              string                       `json:"name"`
	DisplayName       string                       `json:"display_name"`
	ClientType        string                       `json:"client_type"`
	Domain            *string                      `json:"domain,omitempty"`
	URIs              []ClientURIResponseDTO       `json:"uris,omitempty"`
	IdentityProvider  *IdentityProviderResponseDTO `json:"identity_provider,omitempty"`
	Connections       []ClientIdentityProviderDTO  `json:"connections,omitempty"`
	Permissions       *[]PermissionResponseDTO     `json:"permissions,omitempty"`
	Status            string                       `json:"status"`
	IsDefault         bool                         `json:"is_default"`
	IsSystem          bool                         `json:"is_system"`
	BrandingUUID      *string                      `json:"branding_id,omitempty"`
	AllowRegistration bool                         `json:"allow_registration"`
	AllowMagicLink    bool                         `json:"allow_magic_link"`

	// OIDC Session Management
	BackchannelLogoutURI             *string `json:"backchannel_logout_uri,omitempty"`
	FrontchannelLogoutURI            *string `json:"frontchannel_logout_uri,omitempty"`
	BackchannelLogoutSessionRequired bool    `json:"backchannel_logout_session_required"`
	DPoPRequired                     bool    `json:"dpop_required"`

	// OAuth metadata as ENFORCED by the runtime. These live in real columns; the
	// config blob is only mirrored into them on write, so reading the blob shows
	// values the server may have rejected.
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method,omitempty"`
	GrantTypes              []string `json:"grant_types,omitempty"`
	ResponseTypes           []string `json:"response_types,omitempty"`
	AllowedScopes           []string `json:"allowed_scopes"`
	RequireConsent          *bool    `json:"require_consent,omitempty"`
	AccessTokenTTL          *int     `json:"access_token_ttl,omitempty"`
	RefreshTokenTTL         *int     `json:"refresh_token_ttl,omitempty"`

	// Security posture / per-client overrides. The override fields are null when
	// the client inherits the tenant security_settings default.
	RequirePKCE            *bool   `json:"require_pkce,omitempty"`
	RequiredACR            *string `json:"required_acr,omitempty"`
	SessionIdleTimeout     *int    `json:"session_idle_timeout,omitempty"`
	SessionAbsoluteTimeout *int    `json:"session_absolute_timeout,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ClientPublicResponseDTO struct {
	ClientID         string                           `json:"client_id"`
	Name             string                           `json:"name"`
	DisplayName      string                           `json:"display_name"`
	ClientType       string                           `json:"client_type"`
	Domain           *string                          `json:"domain,omitempty"`
	TenantIdentifier string                           `json:"tenant_id"`
	Branding         *branding.ClientBrandingResponse `json:"branding,omitempty"`
}

type ClientIdentityProviderDTO struct {
	ClientIdentityProviderUUID uuid.UUID                   `json:"client_identity_provider_id"`
	IdentityProvider           IdentityProviderResponseDTO `json:"identity_provider"`
	IsDefault                  bool                        `json:"is_default"`
	Enabled                    bool                        `json:"enabled"`
	DisplayOrder               int                         `json:"display_order"`
	CreatedAt                  time.Time                   `json:"created_at"`
	UpdatedAt                  time.Time                   `json:"updated_at"`
}

// List of identity provider connections enabled on a client
type ClientIdentityProvidersResponseDTO struct {
	Connections []ClientIdentityProviderDTO `json:"connections"`
}

// Connect an identity provider to a client request DTO
type AddClientIdentityProviderRequestDTO struct {
	IdentityProviderUUID string `json:"identity_provider_id"`
	IsDefault            bool   `json:"is_default"`
	Enabled              *bool  `json:"enabled"`
	DisplayOrder         int    `json:"display_order"`
}

// Update an identity provider connection request DTO
// UpdateClientIdentityProviderRequestDTO carries omitted-means-unchanged
// semantics on every field.
//
// These were non-pointers, so a partial payload silently rewrote the fields it
// did not mention: toggling `enabled` alone cleared `is_default` (demoting the
// client's default identity provider), and changing `display_order` alone did the
// same. The console sends exactly those partial payloads.
type UpdateClientIdentityProviderRequestDTO struct {
	IsDefault    *bool `json:"is_default"`
	Enabled      *bool `json:"enabled"`
	DisplayOrder *int  `json:"display_order"`
}

// Create auth client request DTO
type ClientCreateRequestDTO struct {
	Name                 string         `json:"name"`
	DisplayName          string         `json:"display_name"`
	ClientType           string         `json:"client_type"`
	Domain               string         `json:"domain"`
	Config               datatypes.JSON `json:"config"`
	Status               string         `json:"status"`
	IdentityProviderUUID string         `json:"identity_provider_id"`
	BrandingUUID         *string        `json:"branding_id,omitempty"`
	AllowRegistration    *bool          `json:"allow_registration,omitempty"`
	AllowMagicLink       *bool          `json:"allow_magic_link,omitempty"`

	// ServiceUUID binds this client to a service, making it that service's
	// credential. A token issued to a bound client carries the `svc` claim, which is
	// what the policy bundle and the gRPC authorizer resolve a principal from — so
	// this field is what makes service-to-service authorization reachable at all.
	// Only an m2m client may be bound; see validateClientServiceBinding.
	ServiceUUID *string `json:"service_id,omitempty"`

	BackchannelLogoutURI             *string `json:"backchannel_logout_uri,omitempty"`
	FrontchannelLogoutURI            *string `json:"frontchannel_logout_uri,omitempty"`
	BackchannelLogoutSessionRequired *bool   `json:"backchannel_logout_session_required,omitempty"`
	DPoPRequired                     *bool   `json:"dpop_required,omitempty"`
}

// Validation

// Update auth client request DTO
type ClientUpdateRequestDTO struct {
	Name              string         `json:"name"`
	DisplayName       string         `json:"display_name"`
	ClientType        string         `json:"client_type"`
	Domain            string         `json:"domain"`
	Config            datatypes.JSON `json:"config"`
	Status            string         `json:"status"`
	BrandingUUID      *string        `json:"branding_id,omitempty"`
	AllowRegistration *bool          `json:"allow_registration,omitempty"`
	AllowMagicLink    *bool          `json:"allow_magic_link,omitempty"`

	// ServiceUUID binds this client to a service (see the create DTO). Sending an
	// empty string unbinds it; omitting the field leaves the binding unchanged.
	ServiceUUID *string `json:"service_id,omitempty"`

	BackchannelLogoutURI             *string `json:"backchannel_logout_uri,omitempty"`
	FrontchannelLogoutURI            *string `json:"frontchannel_logout_uri,omitempty"`
	BackchannelLogoutSessionRequired *bool   `json:"backchannel_logout_session_required,omitempty"`
	DPoPRequired                     *bool   `json:"dpop_required,omitempty"`

	// ExpectedUpdatedAt is the optimistic-concurrency token: the `updated_at` the
	// caller loaded. An update replaces the whole client, so without it two
	// operators editing at once silently overwrite each other. Omit it to opt out
	// (service-to-service callers that are not editing a form).
	ExpectedUpdatedAt *time.Time `json:"expected_updated_at,omitempty"`
}

// Validation

// Create or update auth client URI request DTO
type ClientURICreateOrUpdateRequestDTO struct {
	URI  string `json:"uri"`
	Type string `json:"type"`
}

// Validation

// Auth client listing / filter DTO
type ClientFilterDTO struct {
	Name                 *string  `json:"name"`
	DisplayName          *string  `json:"display_name"`
	ClientType           []string `json:"client_type"`
	IdentityProviderUUID *string  `json:"identity_provider_id"`
	Status               []string `json:"status"`
	IsDefault            *bool    `json:"is_default"`
	IsSystem             *bool    `json:"is_system"`

	// Pagination and sorting
	PaginationRequestDTO
}

// Validate validates the client filter DTO.

// Add permissions to auth client request dto
type ClientAddPermissionsRequestDTO struct {
	Permissions []uuid.UUID `json:"permissions"`
}

// Auth Client API DTOs
type ClientAPIResponseDTO struct {
	ClientAPIUUID uuid.UUID               `json:"client_api_id"`
	API           APIResponseDTO          `json:"api"`
	Permissions   []PermissionResponseDTO `json:"permissions,omitempty"`
	CreatedAt     time.Time               `json:"created_at"`
}

// Add APIs to auth client request dto
type AddClientAPIsRequestDTO struct {
	APIUUIDs []uuid.UUID `json:"api_uuids"`
}

// Add permissions to auth client API request dto
type AddClientAPIPermissionsRequestDTO struct {
	PermissionUUIDs []uuid.UUID `json:"permission_uuids"`
}

type APIResponseDTO struct {
	APIUUID     uuid.UUID `json:"api_id"`
	Name        string    `json:"name"`
	DisplayName string    `json:"display_name"`
	Description string    `json:"description"`
	Identifier  string    `json:"identifier"`
	Status      string    `json:"status"`
	IsSystem    bool      `json:"is_system"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type PermissionResponseDTO struct {
	PermissionUUID uuid.UUID `json:"permission_id"`
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	Status         string    `json:"status"`
	IsSystem       bool      `json:"is_system"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

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

// ClientSetStatusRequestDTO is the request body for PATCH
// /clients/{client_uuid}/status.
//
// The handler used to ignore the body entirely and blind-toggle from whatever
// was currently in the DB. The console always sends an explicit target status,
// so under any staleness — a double-click, another admin flipping it first, a
// stale cache — the toggle could land on the opposite of what the operator
// picked while still reporting success, silently enabling or disabling an
// OAuth client.
type ClientSetStatusRequestDTO struct {
	Status string `json:"status"`
}
