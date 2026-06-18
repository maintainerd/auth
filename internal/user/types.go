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
	Suffix      *string `json:"suffix,omitempty"`
	DisplayName *string `json:"display_name,omitempty"`

	// Personal Information
	Birthdate *string `json:"birthdate,omitempty"` // YYYY-MM-DD format
	Gender    *string `json:"gender,omitempty"`
	Bio       *string `json:"bio,omitempty"`

	// Contact Information
	Phone   *string `json:"phone,omitempty"`
	Email   *string `json:"email,omitempty"`
	Address *string `json:"address,omitempty"`

	// Location Information
	City    *string `json:"city,omitempty"`
	Country *string `json:"country,omitempty"`

	// Preference
	Timezone *string `json:"timezone,omitempty"`
	Language *string `json:"language,omitempty"`

	// Media & Assets
	ProfileURL *string `json:"profile_url,omitempty"`

	// Extended data (custom fields)
	Metadata map[string]any `json:"metadata,omitempty"`
}

// validateDateFormat ensures the date is in "YYYY-MM-DD" format.
type ProfileResponseDTO struct {
	ProfileUUID string `json:"profile_id"`

	// Basic Identity Information
	FirstName   string  `json:"first_name"`
	MiddleName  *string `json:"middle_name,omitempty"`
	LastName    *string `json:"last_name,omitempty"`
	Suffix      *string `json:"suffix,omitempty"`
	DisplayName *string `json:"display_name,omitempty"`
	Bio         *string `json:"bio,omitempty"`

	// Personal Information
	Birthdate *time.Time `json:"birthdate,omitempty"`
	Gender    *string    `json:"gender,omitempty"`

	// Contact Information
	Phone   *string `json:"phone,omitempty"`
	Email   *string `json:"email,omitempty"`
	Address *string `json:"address,omitempty"`

	// Location Information
	City    *string `json:"city,omitempty"`    // Current city
	Country *string `json:"country,omitempty"` // ISO 3166-1 alpha-2 code

	// Preference
	Timezone *string `json:"timezone,omitempty"` // User timezone
	Language *string `json:"language,omitempty"` // ISO 639-1 language code

	// Media & Assets (auth-centric)
	ProfileURL *string `json:"profile_url,omitempty"` // User profile picture

	// Profile Flags
	IsDefault bool `json:"is_default"`

	// Extended data
	Metadata map[string]any `json:"metadata"`

	// System Fields
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ProfileFilterDTO for filtering and paginating profiles
type ProfileFilterDTO struct {
	FirstName *string `json:"first_name,omitempty"`
	LastName  *string `json:"last_name,omitempty"`
	Email     *string `json:"email,omitempty"`
	Phone     *string `json:"phone,omitempty"`
	City      *string `json:"city,omitempty"`
	Country   *string `json:"country,omitempty"`
	IsDefault *bool   `json:"is_default,omitempty"`
	PaginationRequestDTO
}

// User output structure
type UserResponseDTO struct {
	UserUUID           uuid.UUID          `json:"user_id"`
	Username           string             `json:"username"`
	Fullname           string             `json:"fullname"`
	Email              string             `json:"email"`
	Phone              string             `json:"phone"`
	IsEmailVerified    bool               `json:"is_email_verified"`
	IsPhoneVerified    bool               `json:"is_phone_verified"`
	IsProfileCompleted bool               `json:"is_profile_completed"`
	IsAccountCompleted bool               `json:"is_account_completed"`
	Status             string             `json:"status"`
	Metadata           datatypes.JSON     `json:"metadata"`
	Tenant             *TenantResponseDTO `json:"tenant,omitempty"`
	CreatedAt          time.Time          `json:"created_at"`
	UpdatedAt          time.Time          `json:"updated_at"`
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
	IsTOTPEnabled     bool                    `json:"is_totp_enabled"`
	IsWebAuthnEnabled bool                    `json:"is_webauthn_enabled"`
	IsSMSEnabled      bool                    `json:"is_sms_enabled"`
	BackupCodesCount  int                     `json:"backup_codes_count"`
	WebAuthnKeys      []UserMFAWebAuthnKeyDTO `json:"webauthn_keys,omitempty"`
	MFAEnabledAt      *string                 `json:"mfa_enabled_at,omitempty"`
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
	// Internationalization
	Timezone          *string `json:"timezone,omitempty"`
	PreferredLanguage *string `json:"preferred_language,omitempty"`
	Locale            *string `json:"locale,omitempty"`

	// Social Media & External Links
	SocialLinks map[string]string `json:"social_links,omitempty"`

	// Communication Preferences
	PreferredContactMethod   *string `json:"preferred_contact_method,omitempty"`
	MarketingEmailConsent    *bool   `json:"marketing_email_consent,omitempty"`
	SMSNotificationsConsent  *bool   `json:"sms_notifications_consent,omitempty"`
	PushNotificationsConsent *bool   `json:"push_notifications_consent,omitempty"`

	// Privacy & Compliance
	ProfileVisibility     *string `json:"profile_visibility,omitempty"`
	DataProcessingConsent *bool   `json:"data_processing_consent,omitempty"`

	// Emergency Contact
	EmergencyContactName     *string `json:"emergency_contact_name,omitempty"`
	EmergencyContactPhone    *string `json:"emergency_contact_phone,omitempty"`
	EmergencyContactEmail    *string `json:"emergency_contact_email,omitempty"`
	EmergencyContactRelation *string `json:"emergency_contact_relation,omitempty"`
}

type UserSettingResponseDTO struct {
	UserSettingUUID string `json:"user_setting_id"`

	// Internationalization
	Timezone          *string `json:"timezone,omitempty"`
	PreferredLanguage *string `json:"preferred_language,omitempty"`
	Locale            *string `json:"locale,omitempty"`

	// Social Media & External Links
	SocialLinks map[string]any `json:"social_links,omitempty"`

	// Communication Preferences
	PreferredContactMethod   *string `json:"preferred_contact_method,omitempty"`
	MarketingEmailConsent    bool    `json:"marketing_email_consent"`
	SMSNotificationsConsent  bool    `json:"sms_notifications_consent"`
	PushNotificationsConsent bool    `json:"push_notifications_consent"`

	// Privacy & Compliance
	ProfileVisibility       *string    `json:"profile_visibility,omitempty"`
	DataProcessingConsent   bool       `json:"data_processing_consent"`
	TermsAcceptedAt         *time.Time `json:"terms_accepted_at,omitempty"`
	PrivacyPolicyAcceptedAt *time.Time `json:"privacy_policy_accepted_at,omitempty"`

	// Emergency Contact
	EmergencyContactName     *string `json:"emergency_contact_name,omitempty"`
	EmergencyContactPhone    *string `json:"emergency_contact_phone,omitempty"`
	EmergencyContactEmail    *string `json:"emergency_contact_email,omitempty"`
	EmergencyContactRelation *string `json:"emergency_contact_relation,omitempty"`

	// System Fields
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
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
