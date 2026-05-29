package user

import (
	"encoding/json"
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/platform/security"
	"github.com/maintainerd/auth/internal/shared"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// ChangeEmailRequestDTO is the request to initiate an email address change.
type ChangeEmailRequestDTO struct {
	NewEmail        string `json:"new_email"`
	CurrentPassword string `json:"current_password"`
}

func (r *ChangeEmailRequestDTO) Validate() error {
	r.NewEmail = security.SanitizeInput(r.NewEmail)
	return validation.ValidateStruct(r,
		validation.Field(&r.NewEmail, validation.Required, is.Email),
		validation.Field(&r.CurrentPassword, validation.Required),
	)
}

// VerifyEmailChangeDTO is the request to confirm an email change via OTP.
type VerifyEmailChangeDTO struct {
	OTP string `json:"otp"`
}

func (r *VerifyEmailChangeDTO) Validate() error {
	return validation.ValidateStruct(r,
		validation.Field(&r.OTP, validation.Required, validation.Length(6, 6)),
	)
}

// ChangeUsernameDTO is the request to change a username.
type ChangeUsernameDTO struct {
	NewUsername     string `json:"new_username"`
	CurrentPassword string `json:"current_password"`
}

func (r *ChangeUsernameDTO) Validate() error {
	r.NewUsername = security.SanitizeInput(r.NewUsername)
	return validation.ValidateStruct(r,
		validation.Field(&r.NewUsername, validation.Required, validation.Length(3, 50)),
		validation.Field(&r.CurrentPassword, validation.Required),
	)
}

// AccountDeleteDTO is the request to permanently delete an account.
type AccountDeleteDTO struct {
	CurrentPassword string `json:"current_password"`
}

func (r *AccountDeleteDTO) Validate() error {
	return validation.ValidateStruct(r,
		validation.Field(&r.CurrentPassword, validation.Required),
	)
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

func (r *VerifyBackupCodeDTO) Validate() error {
	r.Email = security.SanitizeInput(r.Email)
	return validation.ValidateStruct(r,
		validation.Field(&r.Email, validation.Required, is.Email),
		validation.Field(&r.Code, validation.Required),
		validation.Field(&r.ClientID, validation.Required),
		validation.Field(&r.ProviderID, validation.Required),
	)
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

func (r ProfileRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		// Basic Identity Information
		validation.Field(&r.FirstName,
			validation.Required.Error("First name is required"),
			validation.RuneLength(1, 100).Error("First name must be 1-100 characters"),
		),
		validation.Field(&r.MiddleName,
			validation.NilOrNotEmpty,
			validation.RuneLength(0, 100).Error("Middle name must be at most 100 characters"),
		),
		validation.Field(&r.LastName,
			validation.NilOrNotEmpty,
			validation.RuneLength(0, 100).Error("Last name must be at most 100 characters"),
		),
		validation.Field(&r.Suffix,
			validation.NilOrNotEmpty,
			validation.RuneLength(0, 50).Error("Suffix must be at most 50 characters"),
		),
		validation.Field(&r.DisplayName,
			validation.NilOrNotEmpty,
			validation.RuneLength(0, 100).Error("Display name must be at most 100 characters"),
		),

		// Personal Information
		validation.Field(&r.Birthdate,
			validation.NilOrNotEmpty,
			validation.By(validateDateFormat),
		),
		validation.Field(&r.Gender,
			validation.NilOrNotEmpty,
			validation.In(shared.GenderMale, shared.GenderFemale, shared.GenderOther, shared.GenderPreferNotToSay).Error("Gender must be male, female, other, or prefer_not_to_say"),
		),
		validation.Field(&r.Bio,
			validation.NilOrNotEmpty,
			validation.RuneLength(0, 1000).Error("Bio must be at most 1000 characters"),
		),

		// Contact Information
		validation.Field(&r.Phone,
			validation.NilOrNotEmpty,
			validation.RuneLength(0, 20).Error("Phone must be at most 20 characters"),
		),
		validation.Field(&r.Email,
			validation.NilOrNotEmpty,
			is.Email.Error("Invalid email format"),
			validation.RuneLength(0, 255).Error("Email must be at most 255 characters"),
		),
		validation.Field(&r.Address,
			validation.NilOrNotEmpty,
			validation.RuneLength(0, 500).Error("Address must be at most 500 characters"),
		),

		// Location Information (minimal)
		validation.Field(&r.City,
			validation.NilOrNotEmpty,
			validation.RuneLength(0, 100).Error("City must be at most 100 characters"),
		),
		validation.Field(&r.Country,
			validation.NilOrNotEmpty,
			validation.RuneLength(2, 2).Error("Country must be a 2-character ISO code (e.g., US, PH, CA)"),
		),

		// Preference
		validation.Field(&r.Timezone,
			validation.NilOrNotEmpty,
			validation.RuneLength(0, 50).Error("Timezone must be at most 50 characters"),
		),
		validation.Field(&r.Language,
			validation.NilOrNotEmpty,
			validation.RuneLength(0, 10).Error("Language must be at most 10 characters"),
		),

		// Media & Assets
		validation.Field(&r.ProfileURL,
			validation.NilOrNotEmpty,
			is.URL.Error("Invalid profile URL format"),
			validation.RuneLength(0, 1000).Error("Profile URL must be at most 1000 characters"),
		),
	)
}

// validateDateFormat ensures the date is in "YYYY-MM-DD" format.
func validateDateFormat(value any) error {
	if str, ok := value.(*string); ok && str != nil {
		if _, err := time.Parse("2006-01-02", *str); err != nil {
			return validation.NewError("validation_invalid_date", "Birthdate must be in YYYY-MM-DD format (e.g., 1990-01-25)")
		}
	}
	return nil
}

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

func NewProfileResponseDTO(p *Profile) *ProfileResponseDTO {
	return &ProfileResponseDTO{
		ProfileUUID: p.ProfileUUID.String(),

		// Basic Identity Information
		FirstName:   p.FirstName,
		MiddleName:  p.MiddleName,
		LastName:    p.LastName,
		Suffix:      p.Suffix,
		DisplayName: p.DisplayName,
		Bio:         p.Bio,

		// Personal Information
		Birthdate: p.Birthdate,
		Gender:    p.Gender,

		// Contact Information
		Phone:   p.Phone,
		Email:   p.Email,
		Address: p.Address,

		// Location Information
		City:    p.City,
		Country: p.Country,

		// Preference
		Timezone: p.Timezone,
		Language: p.Language,

		// Media & Assets (auth-centric)
		ProfileURL: p.ProfileURL,

		// Profile Flags
		IsDefault: p.IsDefault,

		// Extended data
		Metadata: convertJSONBToMap(p.Metadata),

		// System Fields
		CreatedAt: p.CreatedAt,
		UpdatedAt: p.UpdatedAt,
	}
}

// Helper function to convert JSONB to map
func convertJSONBToMap(jsonb datatypes.JSON) map[string]any {
	if len(jsonb) == 0 {
		return make(map[string]any)
	}

	var result map[string]any
	if err := json.Unmarshal(jsonb, &result); err != nil {
		return make(map[string]any)
	}
	return result
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

func (f ProfileFilterDTO) Validate() error {
	return f.PaginationRequestDTO.Validate()
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

// User input structures
type UserCreateRequestDTO struct {
	Username   string         `json:"username"`
	Fullname   string         `json:"fullname"`
	Email      *string        `json:"email,omitempty"`
	Phone      *string        `json:"phone,omitempty"`
	Password   string         `json:"password"`
	Status     string         `json:"status"`
	Metadata   datatypes.JSON `json:"metadata,omitempty"`
	TenantUUID string         `json:"tenant_id"`
}

func (dto UserCreateRequestDTO) Validate() error {
	return validation.ValidateStruct(&dto,
		validation.Field(&dto.Username, validation.Required, validation.Length(3, 50)),
		validation.Field(&dto.Fullname, validation.Required, validation.Length(1, 255)),
		validation.Field(&dto.Email, validation.When(dto.Email != nil, is.Email)),
		validation.Field(&dto.Phone, validation.When(dto.Phone != nil, validation.Length(10, 20))),
		validation.Field(&dto.Password, validation.Required, validation.Length(8, 100)),
		validation.Field(&dto.Status, validation.Required, validation.In(shared.StatusActive, shared.StatusInactive, shared.StatusPending, shared.StatusSuspended)),
		validation.Field(&dto.TenantUUID, validation.Required, is.UUID),
	)
}

type UserUpdateRequestDTO struct {
	Username string         `json:"username"`
	Fullname string         `json:"fullname"`
	Email    *string        `json:"email,omitempty"`
	Phone    *string        `json:"phone,omitempty"`
	Status   string         `json:"status"`
	Metadata datatypes.JSON `json:"metadata,omitempty"`
}

func (dto UserUpdateRequestDTO) Validate() error {
	return validation.ValidateStruct(&dto,
		validation.Field(&dto.Username, validation.Required, validation.Length(3, 50)),
		validation.Field(&dto.Fullname, validation.Required, validation.Length(1, 255)),
		validation.Field(&dto.Email, validation.When(dto.Email != nil, is.Email)),
		validation.Field(&dto.Phone, validation.When(dto.Phone != nil, validation.Length(10, 20))),
		validation.Field(&dto.Status, validation.Required, validation.In(shared.StatusActive, shared.StatusInactive, shared.StatusPending, shared.StatusSuspended)),
	)
}

type UserSetStatusRequestDTO struct {
	Status string `json:"status"`
}

func (dto UserSetStatusRequestDTO) Validate() error {
	return validation.ValidateStruct(&dto,
		validation.Field(&dto.Status, validation.Required, validation.In(shared.StatusActive, shared.StatusInactive, shared.StatusPending, shared.StatusSuspended)),
	)
}

type UserAssignRolesRequestDTO struct {
	RoleUUIDs []uuid.UUID `json:"role_ids"`
}

func (dto UserAssignRolesRequestDTO) Validate() error {
	return validation.ValidateStruct(&dto,
		validation.Field(&dto.RoleUUIDs, validation.Required, validation.Length(1, 10)),
	)
}

// User filter structure
type UserFilterDTO struct {
	Username     *string  `json:"username,omitempty"`
	Email        *string  `json:"email,omitempty"`
	Phone        *string  `json:"phone,omitempty"`
	Status       []string `json:"status,omitempty"`
	TenantUUID   *string  `json:"tenant_id,omitempty"`
	RoleUUID     *string  `json:"role_id,omitempty"`
	UserPoolUUID *string  `json:"user_pool_id,omitempty"`
	ClientUUID   *string  `json:"client_id,omitempty"`

	// Pagination and sorting
	PaginationRequestDTO
}

// Validate validates the user filter DTO.
func (f UserFilterDTO) Validate() error {
	return validation.ValidateStruct(&f,
		validation.Field(&f.Status,
			validation.When(len(f.Status) > 0,
				validation.Each(validation.In(shared.StatusActive, shared.StatusInactive).Error("Status must be 'active' or 'inactive'")),
			),
		),
		validation.Field(&f.TenantUUID,
			validation.When(f.TenantUUID != nil,
				is.UUID.Error("Tenant ID must be a valid UUID"),
			),
		),
		validation.Field(&f.RoleUUID,
			validation.When(f.RoleUUID != nil,
				is.UUID.Error("Role ID must be a valid UUID"),
			),
		),
		validation.Field(&f.UserPoolUUID,
			validation.When(f.UserPoolUUID != nil,
				is.UUID.Error("User pool ID must be a valid UUID"),
			),
		),
		validation.Field(&f.ClientUUID,
			validation.When(f.ClientUUID != nil,
				is.UUID.Error("Client ID must be a valid UUID"),
			),
		),
		validation.Field(&f.PaginationRequestDTO),
	)
}

// User role filter structure
type UserRoleFilterDTO struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	Status      *string `json:"status,omitempty"`

	// Pagination and sorting
	PaginationRequestDTO
}

func (r UserRoleFilterDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.PaginationRequestDTO),
	)
}

// User identity filter structure
type UserIdentityFilterDTO struct {
	Provider *string `json:"provider,omitempty"`

	// Pagination and sorting
	PaginationRequestDTO
}

func (r UserIdentityFilterDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.PaginationRequestDTO),
	)
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

func (r UserSettingRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		// Internationalization
		validation.Field(&r.Timezone,
			validation.NilOrNotEmpty,
			validation.RuneLength(0, 50).Error("Timezone must be at most 50 characters"),
		),
		validation.Field(&r.PreferredLanguage,
			validation.NilOrNotEmpty,
			validation.RuneLength(2, 10).Error("Preferred language must be 2-10 characters"),
		),
		validation.Field(&r.Locale,
			validation.NilOrNotEmpty,
			validation.RuneLength(2, 10).Error("Locale must be 2-10 characters"),
		),

		// Communication Preferences
		validation.Field(&r.PreferredContactMethod,
			validation.NilOrNotEmpty,
			validation.In(shared.ContactMethodEmail, shared.ContactMethodPhone, shared.ContactMethodSMS).Error("Preferred contact method must be email, phone, or sms"),
		),

		// Privacy & Compliance
		validation.Field(&r.ProfileVisibility,
			validation.NilOrNotEmpty,
			validation.In(shared.VisibilityPublic, shared.VisibilityPrivate, shared.VisibilityFriends).Error("Profile visibility must be public, private, or friends"),
		),

		// Emergency Contact
		validation.Field(&r.EmergencyContactName,
			validation.NilOrNotEmpty,
			validation.RuneLength(0, 200).Error("Emergency contact name must be at most 200 characters"),
		),
		validation.Field(&r.EmergencyContactPhone,
			validation.NilOrNotEmpty,
			validation.RuneLength(0, 20).Error("Emergency contact phone must be at most 20 characters"),
		),
		validation.Field(&r.EmergencyContactEmail,
			validation.NilOrNotEmpty,
			is.Email.Error("Invalid emergency contact email format"),
			validation.RuneLength(0, 255).Error("Emergency contact email must be at most 255 characters"),
		),
		validation.Field(&r.EmergencyContactRelation,
			validation.NilOrNotEmpty,
			validation.RuneLength(0, 50).Error("Emergency contact relation must be at most 50 characters"),
		),
	)
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

func NewUserSettingResponseDTO(us *UserSetting) *UserSettingResponseDTO {
	// Convert GORM JSON to map for social links
	var socialLinks map[string]any
	if len(us.SocialLinks) > 0 {
		if err := json.Unmarshal(us.SocialLinks, &socialLinks); err != nil {
			socialLinks = nil
		}
	}

	return &UserSettingResponseDTO{
		UserSettingUUID: us.UserSettingUUID.String(),

		// Internationalization
		Timezone:          us.Timezone,
		PreferredLanguage: us.PreferredLanguage,
		Locale:            us.Locale,

		// Social Media & External Links
		SocialLinks: socialLinks,

		// Communication Preferences
		PreferredContactMethod:   us.PreferredContactMethod,
		MarketingEmailConsent:    us.MarketingEmailConsent,
		SMSNotificationsConsent:  us.SMSNotificationsConsent,
		PushNotificationsConsent: us.PushNotificationsConsent,

		// Privacy & Compliance
		ProfileVisibility:       us.ProfileVisibility,
		DataProcessingConsent:   us.DataProcessingConsent,
		TermsAcceptedAt:         us.TermsAcceptedAt,
		PrivacyPolicyAcceptedAt: us.PrivacyPolicyAcceptedAt,

		// Emergency Contact
		EmergencyContactName:     us.EmergencyContactName,
		EmergencyContactPhone:    us.EmergencyContactPhone,
		EmergencyContactEmail:    us.EmergencyContactEmail,
		EmergencyContactRelation: us.EmergencyContactRelation,

		// System Fields
		CreatedAt: us.CreatedAt,
		UpdatedAt: us.UpdatedAt,
	}
}

type UserSetting struct {
	UserSettingID   int64     `gorm:"column:user_setting_id;primaryKey"`
	UserSettingUUID uuid.UUID `gorm:"column:user_setting_uuid;unique;not null"`
	UserID          int64     `gorm:"column:user_id;not null;unique"`

	// Internationalization (BCP-47 locale is the single source of truth)
	Timezone *string `gorm:"column:timezone"`
	Locale   *string `gorm:"column:locale"` // BCP-47 locale code (en-US, es-ES, fr-FR)
	// PreferredLanguage was removed from the DB; kept as transient field for API compatibility.
	// Mirror it to/from Locale in the service layer.
	PreferredLanguage *string `gorm:"-"`

	// Social Media & External Links
	SocialLinks datatypes.JSON `gorm:"column:social_links"` // JSON object for flexible social media links

	// Communication Preferences
	PreferredContactMethod   *string `gorm:"column:preferred_contact_method"` // 'email', 'phone', 'sms'
	MarketingEmailConsent    bool    `gorm:"column:marketing_email_consent;default:false"`
	SMSNotificationsConsent  bool    `gorm:"column:sms_notifications_consent;default:false"`
	PushNotificationsConsent bool    `gorm:"column:push_notifications_consent;default:false"`

	// Privacy & Compliance
	ProfileVisibility       *string    `gorm:"column:profile_visibility;default:'private'"` // 'public', 'private', 'friends'
	DataProcessingConsent   bool       `gorm:"column:data_processing_consent;default:false"`
	TermsAcceptedAt         *time.Time `gorm:"column:terms_accepted_at"`
	PrivacyPolicyAcceptedAt *time.Time `gorm:"column:privacy_policy_accepted_at"`

	// Emergency Contact
	EmergencyContactName     *string `gorm:"column:emergency_contact_name"`
	EmergencyContactPhone    *string `gorm:"column:emergency_contact_phone"`
	EmergencyContactEmail    *string `gorm:"column:emergency_contact_email"`
	EmergencyContactRelation *string `gorm:"column:emergency_contact_relation"`

	// System Fields
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime"`

	// Relationships
	User *User `gorm:"foreignKey:UserID;references:UserID"`
}

func (UserSetting) TableName() string {
	return "user_settings"
}

func (us *UserSetting) BeforeCreate(tx *gorm.DB) (err error) {
	if us.UserSettingUUID == uuid.Nil {
		us.UserSettingUUID = uuid.New()
	}
	return
}
