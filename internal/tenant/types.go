package tenant

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/branding"
	"gorm.io/datatypes"
)

// Tenant output structure
type TenantResponseDTO struct {
	TenantUUID  uuid.UUID      `json:"tenant_id"`
	Name        string         `json:"name"`
	DisplayName string         `json:"display_name"`
	Description string         `json:"description"`
	Status      string         `json:"status"`
	IsSystem    bool           `json:"is_system"`
	Metadata    datatypes.JSON `json:"metadata"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

// TenantPublicResponseDTO is the public-facing tenant response (no auth required).
// It includes password policy, registration config, and active branding needed by
// the login/register/reset-password pages for client-side validation and theming.
type TenantPublicResponseDTO struct {
	Name               string                    `json:"name"`
	DisplayName        string                    `json:"display_name"`
	Description        string                    `json:"description"`
	Status             string                    `json:"status"`
	IsSystem           bool                      `json:"is_system"`
	IsDefault          bool                      `json:"is_default"`
	PasswordConfig     *PasswordConfigPublic     `json:"password_config,omitempty"`
	RegistrationConfig *RegistrationConfigPublic `json:"registration_config,omitempty"`
	Branding           *BrandingPublic           `json:"branding,omitempty"`
}

// TenantBootstrapResponseDTO is the domain-resolved bootstrap payload returned
// by GET /tenant?domain=<host>. It tells the frontend which tenant and surface a
// host maps to, the canonical per-surface URLs, public branding, and the seeded
// system client for the resolved surface. It is a public (unauthenticated)
// response and carries only public fields.
type TenantBootstrapResponseDTO struct {
	Tenant             TenantBootstrapTenantDTO  `json:"tenant"`
	Surface            string                    `json:"surface"`
	IdentityURL        string                    `json:"identity_url"`
	ConsoleURL         string                    `json:"console_url"`
	PasswordConfig     *PasswordConfigPublic     `json:"password_config,omitempty"`
	RegistrationConfig *RegistrationConfigPublic `json:"registration_config,omitempty"`
	Branding           *BrandingPublic           `json:"branding,omitempty"`
	Client             *TenantBootstrapClientDTO `json:"client,omitempty"`
	// Connections are the federated login options (identity providers) enabled
	// on the resolved surface client, ordered by display order. They ride along
	// with initialization so the hosted login page can render its provider
	// buttons immediately, without a second round trip and without needing an
	// in-flight OAuth authorize request to justify the lookup. Always non-nil:
	// an empty array means "no federated providers", never "not yet known".
	Connections []TenantBootstrapConnectionDTO `json:"connections"`
	// MagicLinkEnabled reports whether this surface offers passwordless email
	// sign-in. Off unless an operator enables it on the client.
	MagicLinkEnabled bool `json:"magic_link_enabled"`
}

// TenantBootstrapConnectionDTO is one federated login option on the resolved
// surface client. It mirrors the /oauth/connections projection field-for-field
// so both entry points hand the login page the same shape. Provider secrets and
// upstream config are never included.
type TenantBootstrapConnectionDTO struct {
	Identifier   string `json:"identifier"`
	DisplayName  string `json:"display_name"`
	Provider     string `json:"provider"`
	ProviderType string `json:"provider_type"`
	IsDefault    bool   `json:"is_default"`
	DisplayOrder int    `json:"display_order"`
}

// TenantBootstrapTenantDTO is the public tenant projection embedded in the
// bootstrap response — only non-sensitive identifying fields.
type TenantBootstrapTenantDTO struct {
	TenantUUID  uuid.UUID `json:"tenant_id"`
	Name        string    `json:"name"`
	DisplayName string    `json:"display_name"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	IsSystem    bool      `json:"is_system"`
}

// TenantBootstrapClientDTO is the public projection of the tenant's seeded
// system client for the resolved surface. No secrets or internal IDs.
type TenantBootstrapClientDTO struct {
	ClientID    string `json:"client_id"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	ClientType  string `json:"client_type"`
}

// PublicClientBrandingReader resolves a client-attached theme for a public
// tenant bootstrap response. Implemented outside the tenant package so this
// handler does not need to know how clients store branding relationships.
type PublicClientBrandingReader interface {
	GetPublicClientBranding(ctx context.Context, tenantID int64, clientIdentifier string) (*branding.BrandingServiceDataResult, error)
}

type PasswordConfigPublic struct {
	MinLength        int  `json:"min_length"`
	MaxLength        int  `json:"max_length"`
	RequireUppercase bool `json:"require_uppercase"`
	RequireLowercase bool `json:"require_lowercase"`
	RequireNumber    bool `json:"require_number"`
	RequireSymbol    bool `json:"require_symbol"`
	// Backend-authoritative checks, surfaced so the frontend can show them as
	// requirements/hints. Common-password and HIBP screening run only on the
	// backend (the client can't do them reliably); the frontend displays the
	// backend's rejection message. MinStrengthScore 0 disables the strength gate.
	MinStrengthScore      int  `json:"min_strength_score"`
	RejectCommonPasswords bool `json:"reject_common_passwords"`
	CheckHibp             bool `json:"check_hibp"`
}

type RegistrationConfigPublic struct {
	SelfRegistrationEnabled  bool `json:"self_registration_enabled"`
	RequireEmailVerification bool `json:"require_email_verification"`
	CaptchaOnSignup          bool `json:"captcha_on_signup"`
}

type BrandingPublic struct {
	Layout                string         `json:"layout"`
	CompanyName           string         `json:"company_name"`
	LogoLabel             string         `json:"logo_label"`
	LogoDetail            string         `json:"logo_detail"`
	ShowLogoLabel         bool           `json:"show_logo_label"`
	IdentityLogoLabel     string         `json:"identity_logo_label"`
	IdentityShowLogoLabel bool           `json:"identity_show_logo_label"`
	LogoURL               string         `json:"logo_url"`
	FaviconURL            string         `json:"favicon_url"`
	SupportURL            string         `json:"support_url"`
	PrivacyPolicyURL      string         `json:"privacy_policy_url"`
	TermsOfServiceURL     string         `json:"terms_of_service_url"`
	Metadata              datatypes.JSON `json:"metadata"`
}

// Create Tenant request DTO
type TenantCreateRequestDTO struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
	Status      string `json:"status"`
}

// Update Tenant request DTO
type TenantUpdateRequestDTO struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
	Status      string `json:"status"`
}

// TenantSetStatusRequestDTO is the request body for changing tenant status.
type TenantSetStatusRequestDTO struct {
	Status string `json:"status"`
}

// API listing / filter DTO
type TenantFilterDTO struct {
	Name        *string  `json:"name"`
	DisplayName *string  `json:"display_name"`
	Description *string  `json:"description"`
	Status      []string `json:"status"`
	IsSystem    *bool    `json:"is_system"`

	// Pagination and sorting
	PaginationRequestDTO
}

// TenantMemberResponseDTO for tenant member output
type TenantMemberResponseDTO struct {
	TenantMemberUUID uuid.UUID              `json:"tenant_member_id"`
	Role             string                 `json:"role"`
	User             *MemberUserResponseDTO `json:"user"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
}

// MemberUserResponseDTO is tenant's view of a user in member responses.
// It mirrors the user domain's user response shape to preserve the API.
type MemberUserResponseDTO struct {
	UserUUID        uuid.UUID      `json:"user_id"`
	Username        string         `json:"username"`
	Fullname        string         `json:"fullname"`
	Email           string         `json:"email"`
	Phone           string         `json:"phone"`
	IsEmailVerified bool           `json:"is_email_verified"`
	IsPhoneVerified bool           `json:"is_phone_verified"`
	Status          string         `json:"status"`
	Metadata        datatypes.JSON `json:"metadata"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}

// TenantMemberAddMemberRequestDTO for adding member to tenant
type TenantMemberAddMemberRequestDTO struct {
	UserUUID uuid.UUID `json:"user_id"`
	Role     string    `json:"role"`
}

// TenantMemberUpdateRoleRequestDTO for updating member role
type TenantMemberUpdateRoleRequestDTO struct {
	Role string `json:"role"`
}

// TenantMemberFilterDTO for filtering tenant members
type TenantMemberFilterDTO struct {
	Role *string `json:"role"`
	PaginationRequestDTO
}

// TenantSettingConfigResponseDTO returns a single JSONB config as a map.
type TenantSettingConfigResponseDTO map[string]any

// TenantSettingUpdateConfigRequestDTO is the request body for updating a
// tenant setting config section.
type TenantSettingUpdateConfigRequestDTO map[string]any
