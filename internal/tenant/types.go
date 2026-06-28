package tenant

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// Tenant output structure
type TenantResponseDTO struct {
	TenantUUID  uuid.UUID      `json:"tenant_id"`
	Name        string         `json:"name"`
	DisplayName string         `json:"display_name"`
	Description string         `json:"description"`
	Identifier  string         `json:"identifier"`
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
	Identifier         string                    `json:"identifier"`
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

type PasswordConfigPublic struct {
	MinLength        int  `json:"min_length"`
	MaxLength        int  `json:"max_length"`
	RequireUppercase bool `json:"require_uppercase"`
	RequireLowercase bool `json:"require_lowercase"`
	RequireNumber    bool `json:"require_number"`
	RequireSymbol    bool `json:"require_symbol"`
}

type RegistrationConfigPublic struct {
	SelfRegistrationEnabled  bool `json:"self_registration_enabled"`
	RequireEmailVerification bool `json:"require_email_verification"`
	CaptchaOnSignup          bool `json:"captcha_on_signup"`
}

type BrandingPublic struct {
	Layout            string         `json:"layout"`
	CompanyName       string         `json:"company_name"`
	LogoURL           string         `json:"logo_url"`
	FaviconURL        string         `json:"favicon_url"`
	SupportURL        string         `json:"support_url"`
	PrivacyPolicyURL  string         `json:"privacy_policy_url"`
	TermsOfServiceURL string         `json:"terms_of_service_url"`
	Metadata          datatypes.JSON `json:"metadata"`
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
	Identifier  *string  `json:"identifier"`
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
	UserUUID           uuid.UUID      `json:"user_id"`
	Username           string         `json:"username"`
	Fullname           string         `json:"fullname"`
	Email              string         `json:"email"`
	Phone              string         `json:"phone"`
	IsEmailVerified    bool           `json:"is_email_verified"`
	IsPhoneVerified    bool           `json:"is_phone_verified"`
	IsProfileCompleted bool           `json:"is_profile_completed"`
	IsAccountCompleted bool           `json:"is_account_completed"`
	Status             string         `json:"status"`
	Metadata           datatypes.JSON `json:"metadata"`
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
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
