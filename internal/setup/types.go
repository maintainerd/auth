package setup

// TenantMetadataDTO represents the metadata structure for tenant configuration
type TenantMetadataDTO struct {
	ApplicationLogoURL *string `json:"application_logo_url,omitempty"`
	FaviconURL         *string `json:"favicon_url,omitempty"`
	Language           *string `json:"language,omitempty"`
	Timezone           *string `json:"timezone,omitempty"`
	DateFormat         *string `json:"date_format,omitempty"`
	TimeFormat         *string `json:"time_format,omitempty"`
	PrivacyPolicyURL   *string `json:"privacy_policy_url,omitempty"`
	TermOfServiceURL   *string `json:"term_of_service_url,omitempty"`
}

// CreateTenantRequestDTO for initial tenant setup
type CreateTenantRequestDTO struct {
	Name        string             `json:"name"`
	DisplayName string             `json:"display_name"`
	Description *string            `json:"description,omitempty"`
	Metadata    *TenantMetadataDTO `json:"metadata,omitempty"`
}

// CreateAdminRequestDTO for initial admin user setup
type CreateAdminRequestDTO struct {
	Username string  `json:"username"`
	Email    string  `json:"email"`
	Password string  `json:"password"`
	Fullname *string `json:"fullname,omitempty"`
}

// SetupStatusResponseDTO for checking setup status
type SetupStatusResponseDTO struct {
	IsTenantSetup   bool `json:"is_tenant_setup"`
	IsAdminSetup    bool `json:"is_admin_setup"`
	IsProfileSetup  bool `json:"is_profile_setup"`
	IsSetupComplete bool `json:"is_setup_complete"`
}

type CompleteSetupResponseDTO struct {
	IsSetupComplete bool `json:"is_setup_complete"`
}

type RegisterControlServiceRequestDTO struct {
	// AllowedActions is the control policy, supplied by the caller. Empty means
	// the documented default set (see seeder.DefaultControlActions), which excludes
	// user:* and account:*:self.
	AllowedActions []string
	// PolicyName names this orchestrator's control policy. Empty means the
	// default. Distinct names let an instance serve more than one orchestrator
	// without them silently sharing a grant.
	PolicyName  string
	Name        string  `json:"name"`
	DisplayName string  `json:"display_name"`
	Description *string `json:"description,omitempty"`
	Version     string  `json:"version,omitempty"`
}

type RegisterControlServiceResponseDTO struct {
	ServiceUUID       string `json:"service_uuid"`
	Name              string `json:"name"`
	DisplayName       string `json:"display_name"`
	PolicyUUID        string `json:"policy_uuid"`
	PolicyName        string `json:"policy_name"`
	AlreadyExisted    bool   `json:"already_existed"`
	PolicyWasAttached bool   `json:"policy_was_attached"`
}

// CreateTenantResponseDTO for tenant creation response
type CreateTenantResponseDTO struct {
	Tenant            TenantResponseDTO `json:"tenant"`
	DefaultClientID   string            `json:"default_client_id,omitempty"`
	DefaultProviderID string            `json:"default_provider_id,omitempty"`
}

// CreateAdminResponseDTO for admin creation response
type CreateAdminResponseDTO struct {
	User          UserResponseDTO   `json:"user"`
	TokenResponse *LoginResponseDTO `json:"token_response,omitempty"`
}

// CreateProfileRequestDTO for initial profile setup
type CreateProfileRequestDTO struct {
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

// CreateProfileResponseDTO for profile creation response
type CreateProfileResponseDTO struct {
	Profile ProfileResponseDTO `json:"profile"`
}
