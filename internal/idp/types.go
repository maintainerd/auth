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
	// CallbackURL is the OAuth2/OIDC redirect URI to register in the upstream
	// provider (e.g. Cognito "Allowed callback URLs"). Empty for SAML providers.
	CallbackURL          string    `json:"callback_url,omitempty"`
	Issuer               string    `json:"issuer,omitempty"`
	ProviderClientID     string    `json:"provider_client_id,omitempty"`
	AllowJITProvisioning bool      `json:"allow_jit_provisioning"`
	AllowTokenFederation bool      `json:"allow_token_federation"`
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

// SAMLProviderConfig is the runtime view of a SAML IdP's settings stored in the
// IdentityProvider.Config JSONB column. All IdP-specific SAML fields live here;
// common columns (name, display_name, allow_jit_provisioning, status, etc.) are
// stored directly on the model row.
type SAMLProviderConfig struct {
	// EntityID is the SAML EntityID (Issuer) of the IdP — required.
	EntityID string `json:"entity_id"`
	// SSOURL is the IdP's HTTP-POST or HTTP-Redirect SSO endpoint — required.
	SSOURL string `json:"sso_url"`
	// SLOURL is the IdP's Single Logout endpoint — optional.
	SLOURL string `json:"slo_url,omitempty"`
	// Certificate is the IdP's signing certificate in PEM format — required when active.
	// It is used to verify assertion signatures and to populate certificate_expires_at.
	Certificate string `json:"certificate"`
	// NameIDFormat controls the NameIDPolicy sent in AuthnRequests (e.g.
	// "urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress"). Defaults to
	// "urn:oasis:names:tc:SAML:2.0:nameid-format:persistent" when blank.
	NameIDFormat string `json:"name_id_format,omitempty"`
	// AttributeMapping maps IdP SAML attribute names to our IdentityMetadata
	// fields (e.g. "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress" → "email").
	AttributeMapping map[string]string `json:"attribute_mapping,omitempty"`
}

// SAMLInitiateInput is the service-layer input for beginning a SAML SSO flow.
type SAMLInitiateInput struct {
	ProviderIdentifier string // matches IdentityProvider.Identifier
	ClientID           string // OAuth client_id (our client, not the SAML IdP's)
	RedirectURI        string // where the frontend wants to land after ACS
	TenantID           int64
}

// SAMLInitiateResult carries the IdP redirect URL the browser should navigate to.
type SAMLInitiateResult struct {
	RedirectURL string
}

// SAMLCallbackResult is returned by HandleSAMLResponse after a successful ACS POST.
type SAMLCallbackResult struct {
	// RedirectURI is the URL the ACS handler should redirect the browser to.
	// It already contains the Code as a query parameter.
	RedirectURI string
	// Code is a short-lived, single-use exchange code (5 min TTL) stored in
	// cache. The frontend POSTs it to /federation/saml/exchange to obtain tokens.
	Code  string
	IsNew bool
}

// SAMLLogoutInitiateInput is the service-layer input for SP-initiated SAML
// Single Logout. IDTokenHint is the only credential the endpoint has (the SLO
// surface is public), so it identifies the subject whose sessions end.
type SAMLLogoutInitiateInput struct {
	ProviderIdentifier    string
	ClientID              string // our OAuth client_id, needed to validate PostLogoutRedirectURI
	IDTokenHint           string
	PostLogoutRedirectURI string
}

// SAMLLogoutInitiateResult carries the IdP SLO URL the browser should follow.
type SAMLLogoutInitiateResult struct {
	RedirectURL string
}

// SAMLSingleLogoutResult is returned by the SLO endpoint for both directions of
// the exchange.
type SAMLSingleLogoutResult struct {
	// RedirectURL is where the browser goes next: back to the IdP carrying our
	// LogoutResponse (IdP-initiated), or the post-logout landing page validated
	// at initiate time (SP-initiated). Empty means "nothing left to visit".
	RedirectURL string
	// LoggedOut reports whether local sessions were terminated for this subject.
	LoggedOut bool
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
	// Populated when a verified-email collision requires explicit confirmation.
	AccountLinkToken    string
	AccountLinkProvider string
	AccountLinkEmail    string
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
	IdentityProviderUUID uuid.UUID `json:"identity_provider_id"`
	Name                 string    `json:"name"`
	DisplayName          string    `json:"display_name"`
	Provider             string    `json:"provider"`
	ProviderType         string    `json:"provider_type"`
	Identifier           string    `json:"identifier"`
	// CallbackURL is the OAuth2/OIDC redirect URI to register in the upstream
	// provider (e.g. Cognito "Allowed callback URLs"). Empty for SAML providers.
	CallbackURL          string             `json:"callback_url,omitempty"`
	Issuer               string             `json:"issuer,omitempty"`
	ProviderClientID     string             `json:"provider_client_id,omitempty"`
	AllowJITProvisioning bool               `json:"allow_jit_provisioning"`
	AllowRegistration    bool               `json:"allow_registration"`
	AllowTokenFederation bool               `json:"allow_token_federation"`
	AllowedAudiences     []string           `json:"allowed_audiences"`
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
// issuer, provider_client_id, provider_client_secret (write-only), allow_jit_provisioning,
// allow_token_federation, allowed_audiences and email_domains. Config carries only the
// remaining JSONB fields (endpoints / scopes / attribute_mapping / system settings).
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
	AllowTokenFederation bool           `json:"allow_token_federation"`
	AllowedAudiences     []string       `json:"allowed_audiences"`
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
	AllowTokenFederation bool           `json:"allow_token_federation"`
	AllowedAudiences     []string       `json:"allowed_audiences"`
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

// RegistrationFlowClientSummaryDTO is the nested client projection on the
// registration flow detail response. A registration link is only valid for its
// client, so the detail view resolves the client to a human-readable name
// rather than making the operator decode a bare UUID.
type RegistrationFlowClientSummaryDTO struct {
	ClientUUID  string `json:"client_id"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name,omitempty"`
	Identifier  string `json:"identifier"`
	Status      string `json:"status"`
}

// Registration flow list output structure (lean — one row per listing entry).
type RegistrationFlowResponseDTO struct {
	RegistrationFlowUUID string    `json:"registration_flow_id"`
	Name                 string    `json:"name"`
	Description          string    `json:"description"`
	Status               string    `json:"status"`
	ClientUUID           *string   `json:"client_id,omitempty"`
	VerificationRequired bool      `json:"verification_required"`
	IsSystem             bool      `json:"is_system"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

// Registration flow detail output structure (adds required_fields + the
// resolved client summary on top of the list shape).
type RegistrationFlowDetailResponseDTO struct {
	RegistrationFlowUUID string                            `json:"registration_flow_id"`
	Name                 string                            `json:"name"`
	Description          string                            `json:"description"`
	Status               string                            `json:"status"`
	ClientUUID           *string                           `json:"client_id,omitempty"`
	Client               *RegistrationFlowClientSummaryDTO `json:"client,omitempty"`
	VerificationRequired bool                              `json:"verification_required"`
	RequiredFields       datatypes.JSON                    `json:"required_fields"`
	IsSystem             bool                              `json:"is_system"`
	CreatedAt            time.Time                         `json:"created_at"`
	UpdatedAt            time.Time                         `json:"updated_at"`
}

// Create registration flow request dto.
//
// Name is the public value that appears in registration links
// (?registration_flow=<name>), so it is validated as a slug and must be unique
// within the tenant. There is no separate identifier.
type RegistrationFlowCreateRequestDTO struct {
	Name                 string    `json:"name"`
	Description          string    `json:"description"`
	Status               *string   `json:"status,omitempty"`
	ClientUUID           string    `json:"client_id"`
	RoleIDs              []string  `json:"role_ids,omitempty"`
	VerificationRequired *bool     `json:"verification_required,omitempty"`
	RequiredFields       *[]string `json:"required_fields"`
}

// Update registration flow request dto.
//
// Every optional field carries omitted-means-unchanged semantics: a nil pointer
// leaves the stored value alone. This matters because Status,
// VerificationRequired and RequiredFields are security controls — a partial PUT
// must never silently re-activate a disabled flow, turn off verification, or
// wipe the required-field set.
//
// RoleIDs, when present, replaces the flow's role membership with exactly the
// provided set (an empty array clears it; omitting the field leaves it untouched).
//
// Renaming a flow changes its public registration link, so any link an external
// app has already published stops resolving. Callers should surface that.
type RegistrationFlowUpdateRequestDTO struct {
	Name                 *string   `json:"name,omitempty"`
	Description          *string   `json:"description,omitempty"`
	Status               *string   `json:"status,omitempty"`
	RoleIDs              []string  `json:"role_ids,omitempty"`
	VerificationRequired *bool     `json:"verification_required,omitempty"`
	RequiredFields       *[]string `json:"required_fields,omitempty"`
}

// Update registration flow status request dto
type RegistrationFlowUpdateStatusRequestDTO struct {
	Status string `json:"status"`
}

// Registration flow listing request dto
type RegistrationFlowFilterDTO struct {
	Name       *string  `json:"name"`
	Search     *string  `json:"search"`
	Status     []string `json:"status"`
	ClientUUID *string  `json:"client_id"`
	IsSystem   *bool    `json:"is_system"`

	// Pagination and sorting
	PaginationRequestDTO
}

// Assign roles to registration flow request dto
type RegistrationFlowAssignRolesRequestDTO struct {
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
	Success bool           `json:"success"`
	Checks  []TestCheckDTO `json:"checks"`
}

// TestCheckDTO describes the result of a single validation step during an IdP
// test-connection probe.
type TestCheckDTO struct {
	Step  string `json:"step"`
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
	URL   string `json:"url,omitempty"`
}
