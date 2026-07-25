package user

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// ChangeEmailRequestDTO is the request to initiate an email address change.
type ChangeEmailRequestDTO struct {
	NewEmail        string `json:"new_email"`
	CurrentPassword string `json:"current_password"`
}

// VerifyEmailChangeDTO is the request to confirm an email change via OTP.
type VerifyEmailChangeDTO struct {
	OTP string `json:"otp"`
}

// ChangeUsernameDTO is the request to change a username.
type ChangeUsernameDTO struct {
	NewUsername     string `json:"new_username"`
	CurrentPassword string `json:"current_password"`
}

// ChangePasswordDTO is the request to rotate the authenticated user's own
// password. Both fields are required; no length or composition rules are
// declared here on purpose — those belong to the tenant's password policy, and
// duplicating them at the DTO layer is how the two drift apart.
type ChangePasswordDTO struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// ChangePasswordResponseDTO reports what happened to the user's other sessions,
// so the client can tell them rather than leaving them to discover it.
type ChangePasswordResponseDTO struct {
	OtherSessionsRevoked bool `json:"other_sessions_revoked"`
	// ReauthenticationRequired is true when the caller's own session could not
	// be identified and everything was revoked as the safe fallback.
	ReauthenticationRequired bool `json:"reauthentication_required"`
}

// AccountDeleteDTO is the request to permanently delete an account.
type AccountDeleteDTO struct {
	CurrentPassword string `json:"current_password"`
}

// AccountExportDTO is the response payload for account data export.
type AccountExportDTO struct {
	UserUUID  string      `json:"user_uuid"`
	Username  string      `json:"username"`
	Email     string      `json:"email"`
	Phone     string      `json:"phone"`
	CreatedAt time.Time   `json:"created_at"`
	Profile   interface{} `json:"profile,omitempty"`
	Roles     []string    `json:"roles"`
	Settings  interface{} `json:"settings,omitempty"`
}

// GenerateBackupCodesResponseDTO holds the plaintext backup codes shown once.
type GenerateBackupCodesResponseDTO struct {
	Codes []string `json:"codes"`
}

// SendPhoneVerificationDTO is the request to send an SMS OTP to verify a phone number.
type SendPhoneVerificationDTO struct {
	Phone string `json:"phone"`
}

// VerifyPhoneDTO is the request to verify a phone number with an SMS OTP.
type VerifyPhoneDTO struct {
	Phone string `json:"phone"`
	Code  string `json:"code"`
}

// VerifyBackupCodeDTO is the request to recover an account via a backup code.
type VerifyBackupCodeDTO struct {
	Email      string `json:"email"`
	Code       string `json:"code"`
	ClientID   string `json:"client_id"`
	ProviderID string `json:"provider_id"`
}

type ProfileRequestDTO struct {
	// Basic Identity Information
	FirstName   string  `json:"first_name"`
	MiddleName  *string `json:"middle_name,omitempty"`
	LastName    *string `json:"last_name,omitempty"`
	DisplayName *string `json:"display_name,omitempty"`

	// Personal Information
	Birthdate *string `json:"birthdate,omitempty"` // YYYY-MM-DD format
	Gender    *string `json:"gender,omitempty"`

	// Contact Information (transient — not stored on profile)
	Email *string `json:"email,omitempty"`

	// Preference
	Timezone *string `json:"timezone,omitempty"`
	Language *string `json:"language,omitempty"`

	// Media & Assets
	ProfileURL *string `json:"profile_url,omitempty"`

	// Extended data (custom fields — use metadata.address for OIDC address claim)
	Metadata map[string]any `json:"metadata,omitempty"`
}

// validateDateFormat ensures the date is in "YYYY-MM-DD" format.
type ProfileResponseDTO struct {
	ProfileUUID string `json:"profile_id"`

	// Basic Identity Information
	FirstName   string  `json:"first_name"`
	MiddleName  *string `json:"middle_name,omitempty"`
	LastName    *string `json:"last_name,omitempty"`
	DisplayName *string `json:"display_name,omitempty"`

	// Personal Information
	Birthdate *time.Time `json:"birthdate,omitempty"`
	Gender    *string    `json:"gender,omitempty"`

	// Contact Information (transient)
	Email *string `json:"email,omitempty"`

	// Preference
	Timezone *string `json:"timezone,omitempty"`
	Language *string `json:"language,omitempty"`

	// Media & Assets (auth-centric)
	ProfileURL *string `json:"profile_url,omitempty"`

	// Extended data (includes OIDC address claim as metadata.address)
	Metadata map[string]any `json:"metadata"`

	// Profile state
	IsDefault bool `json:"is_default"`

	// System Fields
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ProfileFilterDTO for filtering and paginating profiles
type ProfileFilterDTO struct {
	FirstName *string `json:"first_name,omitempty"`
	LastName  *string `json:"last_name,omitempty"`
	Email     *string `json:"email,omitempty"`
	PaginationRequestDTO
}

// User output structure
type UserResponseDTO struct {
	UserUUID        uuid.UUID          `json:"user_id"`
	Username        string             `json:"username"`
	Fullname        string             `json:"fullname"`
	Email           string             `json:"email"`
	Phone           string             `json:"phone"`
	IsEmailVerified bool               `json:"is_email_verified"`
	IsPhoneVerified bool               `json:"is_phone_verified"`
	PhoneVerifiedAt *time.Time         `json:"phone_verified_at,omitempty"`
	Status          string             `json:"status"`
	Metadata        datatypes.JSON     `json:"metadata"`
	LastLoginAt     *time.Time         `json:"last_login_at,omitempty"`
	LoginCount      int                `json:"login_count,omitempty"`
	EmailVerifiedAt *time.Time         `json:"email_verified_at,omitempty"`
	ExternalID      *string            `json:"external_id,omitempty"`
	Tenant          *TenantResponseDTO `json:"tenant,omitempty"`
	CreatedAt       time.Time          `json:"created_at"`
	UpdatedAt       time.Time          `json:"updated_at"`
}

type UserIdentityResponseDTO struct {
	UserIdentityUUID uuid.UUID          `json:"user_identity_id"`
	Provider         string             `json:"provider"`
	Sub              string             `json:"sub"`
	Metadata         datatypes.JSON     `json:"metadata"`
	Client           *ClientResponseDTO `json:"client,omitempty"`
	CreatedAt        time.Time          `json:"created_at"`
	UpdatedAt        time.Time          `json:"updated_at"`
}

type UserMFAResponseDTO struct {
	IsTOTPEnabled      bool                    `json:"is_totp_enabled"`
	IsWebAuthnEnabled  bool                    `json:"is_webauthn_enabled"`
	IsSMSEnabled       bool                    `json:"is_sms_enabled"`
	BackupCodesCount   int                     `json:"backup_codes_count"`
	WebAuthnKeys       []UserMFAWebAuthnKeyDTO `json:"webauthn_keys,omitempty"`
	FirstMFAEnrolledAt *string                 `json:"mfa_enabled_at,omitempty"`
}

type UserMFAWebAuthnKeyDTO struct {
	CredentialUUID string  `json:"credential_uuid"`
	Name           string  `json:"name"`
	Transport      string  `json:"transport,omitempty"`
	LastUsedAt     *string `json:"last_used_at,omitempty"`
	CreatedAt      string  `json:"created_at"`
}

// User input structures
type UserCreateRequestDTO struct {
	Username string         `json:"username"`
	Email    *string        `json:"email,omitempty"`
	Phone    *string        `json:"phone,omitempty"`
	Password string         `json:"password"`
	Status   string         `json:"status"`
	Metadata datatypes.JSON `json:"metadata,omitempty"`
}

type UserUpdateRequestDTO struct {
	Username string         `json:"username"`
	Email    *string        `json:"email,omitempty"`
	Phone    *string        `json:"phone,omitempty"`
	Status   string         `json:"status"`
	Metadata datatypes.JSON `json:"metadata,omitempty"`
}

type UserSetStatusRequestDTO struct {
	Status string `json:"status"`
}

type UserAssignRolesRequestDTO struct {
	RoleUUIDs []uuid.UUID `json:"role_ids"`
}

// User filter structure
type UserFilterDTO struct {
	Search     *string  `json:"search,omitempty"`
	Username   *string  `json:"username,omitempty"`
	Email      *string  `json:"email,omitempty"`
	Phone      *string  `json:"phone,omitempty"`
	Fullname   *string  `json:"fullname,omitempty"`
	Status     []string `json:"status,omitempty"`
	TenantUUID *string  `json:"tenant_id,omitempty"`
	RoleUUID   *string  `json:"role_id,omitempty"`
	ClientUUID *string  `json:"client_id,omitempty"`

	// Pagination and sorting
	PaginationRequestDTO
}

// Validate validates the user filter DTO.
// User role filter structure
type UserRoleFilterDTO struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	Status      *string `json:"status,omitempty"`

	// Pagination and sorting
	PaginationRequestDTO
}

// User identity filter structure
type UserIdentityFilterDTO struct {
	Provider *string `json:"provider,omitempty"`

	// Pagination and sorting
	PaginationRequestDTO
}

type UserSettingRequestDTO struct {
	Timezone          *string `json:"timezone,omitempty"`
	PreferredLanguage *string `json:"preferred_language,omitempty"`
	Locale            *string `json:"locale,omitempty"`
}

type UserSettingResponseDTO struct {
	UserSettingUUID   string    `json:"user_setting_id"`
	Timezone          *string   `json:"timezone,omitempty"`
	PreferredLanguage *string   `json:"preferred_language,omitempty"`
	Locale            *string   `json:"locale,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
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

type SessionDataResult struct {
	SessionID         string     `json:"session_id"`
	IPAddress         *string    `json:"ip_address,omitempty"`
	UserAgent         *string    `json:"user_agent,omitempty"`
	LastUsedAt        *time.Time `json:"last_used_at,omitempty"`
	ExpiresAt         *time.Time `json:"expires_at,omitempty"`
	AbsoluteExpiresAt *time.Time `json:"absolute_expires_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
}

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

type ClientResponseDTO struct {
	ClientUUID  uuid.UUID `json:"client_id"`
	Name        string    `json:"name"`
	DisplayName string    `json:"display_name"`
	ClientType  string    `json:"client_type"`
	Domain      *string   `json:"domain,omitempty"`
	Status      string    `json:"status"`
	IsDefault   bool      `json:"is_default"`
	IsSystem    bool      `json:"is_system"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
