package dto

import (
	"regexp"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
)

// TenantMetadataDto represents the metadata structure for tenant configuration
type TenantMetadataDto struct {
	ApplicationLogoURL *string `json:"application_logo_url,omitempty"`
	FaviconURL         *string `json:"favicon_url,omitempty"`
	Language           *string `json:"language,omitempty"`
	Timezone           *string `json:"timezone,omitempty"`
	DateFormat         *string `json:"date_format,omitempty"`
	TimeFormat         *string `json:"time_format,omitempty"`
	PrivacyPolicyURL   *string `json:"privacy_policy_url,omitempty"`
	TermOfServiceURL   *string `json:"term_of_service_url,omitempty"`
}

func (dto TenantMetadataDto) Validate() error {
	return validation.ValidateStruct(&dto,
		validation.Field(&dto.ApplicationLogoURL,
			validation.When(dto.ApplicationLogoURL != nil,
				is.URL.Error("Application logo URL must be a valid URL"),
				validation.Length(0, 500).Error("Application logo URL must not exceed 500 characters"),
			),
		),
		validation.Field(&dto.FaviconURL,
			validation.When(dto.FaviconURL != nil,
				is.URL.Error("Favicon URL must be a valid URL"),
				validation.Length(0, 500).Error("Favicon URL must not exceed 500 characters"),
			),
		),
		validation.Field(&dto.Language,
			validation.When(dto.Language != nil,
				validation.Length(2, 10).Error("Language must be between 2 and 10 characters"),
				validation.Match(regexp.MustCompile(`^[a-zA-Z]{2}(-[a-zA-Z]{2})?$`)).Error("Language must be in format 'en' or 'en-US'"),
			),
		),
		validation.Field(&dto.Timezone,
			validation.When(dto.Timezone != nil,
				validation.Length(0, 50).Error("Timezone must not exceed 50 characters"),
			),
		),
		validation.Field(&dto.DateFormat,
			validation.When(dto.DateFormat != nil,
				validation.Length(0, 20).Error("Date format must not exceed 20 characters"),
			),
		),
		validation.Field(&dto.TimeFormat,
			validation.When(dto.TimeFormat != nil,
				validation.Length(0, 20).Error("Time format must not exceed 20 characters"),
			),
		),
		validation.Field(&dto.PrivacyPolicyURL,
			validation.When(dto.PrivacyPolicyURL != nil,
				is.URL.Error("Privacy policy URL must be a valid URL"),
				validation.Length(0, 500).Error("Privacy policy URL must not exceed 500 characters"),
			),
		),
		validation.Field(&dto.TermOfServiceURL,
			validation.When(dto.TermOfServiceURL != nil,
				is.URL.Error("Terms of service URL must be a valid URL"),
				validation.Length(0, 500).Error("Terms of service URL must not exceed 500 characters"),
			),
		),
	)
}

// CreateTenantRequestDto for initial tenant setup
type CreateTenantRequestDto struct {
	Name        string             `json:"name"`
	Description *string            `json:"description,omitempty"`
	Metadata    *TenantMetadataDto `json:"metadata,omitempty"`
}

func (dto CreateTenantRequestDto) Validate() error {
	return validation.ValidateStruct(&dto,
		validation.Field(&dto.Name,
			validation.Required.Error("Tenant name is required"),
			validation.Length(2, 100).Error("Tenant name must be between 2 and 100 characters"),
			validation.Match(regexp.MustCompile(`^[a-zA-Z0-9\s\-_\.]+$`)).Error("Tenant name contains invalid characters"),
		),
		validation.Field(&dto.Description,
			validation.When(dto.Description != nil,
				validation.Length(0, 500).Error("Description must not exceed 500 characters"),
			),
		),
		validation.Field(&dto.Metadata,
			validation.When(dto.Metadata != nil,
				validation.By(func(value any) error {
					if metadata, ok := value.(*TenantMetadataDto); ok && metadata != nil {
						return metadata.Validate()
					}
					return nil
				}),
			),
		),
	)
}

// CreateAdminRequestDto for initial admin user setup
type CreateAdminRequestDto struct {
	Username string `json:"username"`
	Fullname string `json:"fullname"`
	Password string `json:"password"`
	Email    string `json:"email"`
}

func (dto CreateAdminRequestDto) Validate() error {
	return validation.ValidateStruct(&dto,
		validation.Field(&dto.Username,
			validation.Required.Error("Username is required"),
			validation.Length(3, 50).Error("Username must be between 3 and 50 characters"),
			validation.Match(regexp.MustCompile(`^[a-zA-Z0-9_\-\.@]+$`)).Error("Username contains invalid characters"),
		),
		validation.Field(&dto.Fullname,
			validation.Required.Error("Fullname is required"),
			validation.Length(1, 255).Error("Fullname must be between 1 and 255 characters"),
		),
		validation.Field(&dto.Password,
			validation.Required.Error("Password is required"),
			validation.Length(8, 100).Error("Password must be between 8 and 100 characters"),
		),
		validation.Field(&dto.Email,
			validation.Required.Error("Email is required"),
			is.Email.Error("Invalid email format"),
			validation.Length(0, 100).Error("Email must not exceed 100 characters"),
		),
	)
}

// SetupStatusResponseDto for checking setup status
type SetupStatusResponseDto struct {
	IsTenantSetup   bool `json:"is_tenant_setup"`
	IsAdminSetup    bool `json:"is_admin_setup"`
	IsProfileSetup  bool `json:"is_profile_setup"`
	IsSetupComplete bool `json:"is_setup_complete"`
}

// CreateTenantResponseDto for tenant creation response
type CreateTenantResponseDto struct {
	Tenant            TenantResponseDto `json:"tenant"`
	DefaultClientID   string            `json:"default_client_id,omitempty"`
	DefaultProviderID string            `json:"default_provider_id,omitempty"`
}

// CreateAdminResponseDto for admin creation response
type CreateAdminResponseDto struct {
	User          UserResponseDto   `json:"user"`
	TokenResponse *LoginResponseDto `json:"token_response,omitempty"`
}

// CreateProfileRequestDto for initial profile setup
type CreateProfileRequestDto struct {
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
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

func (dto CreateProfileRequestDto) Validate() error {
	return validation.ValidateStruct(&dto,
		// Basic Identity Information
		validation.Field(&dto.FirstName,
			validation.Required.Error("First name is required"),
			validation.RuneLength(1, 100).Error("First name must be 1-100 characters"),
		),
		validation.Field(&dto.MiddleName,
			validation.NilOrNotEmpty,
			validation.RuneLength(0, 100).Error("Middle name must be at most 100 characters"),
		),
		validation.Field(&dto.LastName,
			validation.NilOrNotEmpty,
			validation.RuneLength(0, 100).Error("Last name must be at most 100 characters"),
		),
		validation.Field(&dto.Suffix,
			validation.NilOrNotEmpty,
			validation.RuneLength(0, 50).Error("Suffix must be at most 50 characters"),
		),
		validation.Field(&dto.DisplayName,
			validation.NilOrNotEmpty,
			validation.RuneLength(0, 100).Error("Display name must be at most 100 characters"),
		),

		// Personal Information
		validation.Field(&dto.Birthdate,
			validation.NilOrNotEmpty,
			validation.By(validateDateFormat),
		),
		validation.Field(&dto.Gender,
			validation.NilOrNotEmpty,
			validation.In("male", "female", "other", "prefer_not_to_say").Error("Gender must be male, female, other, or prefer_not_to_say"),
		),
		validation.Field(&dto.Bio,
			validation.NilOrNotEmpty,
			validation.RuneLength(0, 1000).Error("Bio must be at most 1000 characters"),
		),

		// Contact Information
		validation.Field(&dto.Phone,
			validation.NilOrNotEmpty,
			validation.RuneLength(0, 20).Error("Phone must be at most 20 characters"),
		),
		validation.Field(&dto.Email,
			validation.NilOrNotEmpty,
			is.Email.Error("Invalid email format"),
			validation.RuneLength(0, 255).Error("Email must be at most 255 characters"),
		),
		validation.Field(&dto.Address,
			validation.NilOrNotEmpty,
			validation.RuneLength(0, 500).Error("Address must be at most 500 characters"),
		),

		// Location Information
		validation.Field(&dto.City,
			validation.NilOrNotEmpty,
			validation.RuneLength(0, 100).Error("City must be at most 100 characters"),
		),
		validation.Field(&dto.Country,
			validation.NilOrNotEmpty,
			validation.RuneLength(2, 2).Error("Country must be a 2-character ISO code (e.g., US, PH, CA)"),
		),

		// Preference
		validation.Field(&dto.Timezone,
			validation.NilOrNotEmpty,
			validation.RuneLength(0, 50).Error("Timezone must be at most 50 characters"),
		),
		validation.Field(&dto.Language,
			validation.NilOrNotEmpty,
			validation.RuneLength(0, 10).Error("Language must be at most 10 characters"),
		),

		// Media & Assets
		validation.Field(&dto.ProfileURL,
			validation.NilOrNotEmpty,
			is.URL.Error("Invalid profile URL format"),
			validation.RuneLength(0, 1000).Error("Profile URL must be at most 1000 characters"),
		),
	)
}

// CreateProfileResponseDto for profile creation response
type CreateProfileResponseDto struct {
	Profile ProfileResponse `json:"profile"`
}
