package branding

import (
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/maintainerd/auth/internal/shared"
)

// BrandingResponseDTO is the JSON representation of a branding record.
type BrandingResponseDTO struct {
	BrandingID        string    `json:"branding_id"`
	CompanyName       string    `json:"company_name"`
	LogoURL           string    `json:"logo_url"`
	FaviconURL        string    `json:"favicon_url"`
	PrimaryColor      string    `json:"primary_color"`
	SecondaryColor    string    `json:"secondary_color"`
	AccentColor       string    `json:"accent_color"`
	FontFamily        string    `json:"font_family"`
	CustomCSS         string    `json:"custom_css"`
	SupportURL        string    `json:"support_url"`
	PrivacyPolicyURL  string    `json:"privacy_policy_url"`
	TermsOfServiceURL string    `json:"terms_of_service_url"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// BrandingUpdateRequestDTO is the request body for updating branding.
type BrandingUpdateRequestDTO struct {
	CompanyName       string `json:"company_name"`
	LogoURL           string `json:"logo_url"`
	FaviconURL        string `json:"favicon_url"`
	PrimaryColor      string `json:"primary_color"`
	SecondaryColor    string `json:"secondary_color"`
	AccentColor       string `json:"accent_color"`
	FontFamily        string `json:"font_family"`
	CustomCSS         string `json:"custom_css"`
	SupportURL        string `json:"support_url"`
	PrivacyPolicyURL  string `json:"privacy_policy_url"`
	TermsOfServiceURL string `json:"terms_of_service_url"`
}

// Validate validates the branding update request.
func (r BrandingUpdateRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.CompanyName,
			validation.Length(0, 255).Error("Company name must not exceed 255 characters"),
		),
		validation.Field(&r.LogoURL,
			validation.Length(0, 2048).Error("Logo URL must not exceed 2048 characters"),
			validation.When(r.LogoURL != "", is.URL.Error("Logo URL must be a valid URL")),
		),
		validation.Field(&r.FaviconURL,
			validation.Length(0, 2048).Error("Favicon URL must not exceed 2048 characters"),
			validation.When(r.FaviconURL != "", is.URL.Error("Favicon URL must be a valid URL")),
		),
		validation.Field(&r.PrimaryColor,
			validation.Length(0, 20).Error("Primary color must not exceed 20 characters"),
		),
		validation.Field(&r.SecondaryColor,
			validation.Length(0, 20).Error("Secondary color must not exceed 20 characters"),
		),
		validation.Field(&r.AccentColor,
			validation.Length(0, 20).Error("Accent color must not exceed 20 characters"),
		),
		validation.Field(&r.FontFamily,
			validation.Length(0, 100).Error("Font family must not exceed 100 characters"),
		),
		validation.Field(&r.CustomCSS,
			validation.Length(0, 50000).Error("Custom CSS must not exceed 50000 characters"),
		),
		validation.Field(&r.SupportURL,
			validation.Length(0, 2048).Error("Support URL must not exceed 2048 characters"),
			validation.When(r.SupportURL != "", is.URL.Error("Support URL must be a valid URL")),
		),
		validation.Field(&r.PrivacyPolicyURL,
			validation.Length(0, 2048).Error("Privacy policy URL must not exceed 2048 characters"),
			validation.When(r.PrivacyPolicyURL != "", is.URL.Error("Privacy policy URL must be a valid URL")),
		),
		validation.Field(&r.TermsOfServiceURL,
			validation.Length(0, 2048).Error("Terms of service URL must not exceed 2048 characters"),
			validation.When(r.TermsOfServiceURL != "", is.URL.Error("Terms of service URL must be a valid URL")),
		),
	)
}

// Email template list response DTO (without body content)
type EmailTemplateListResponseDTO struct {
	EmailTemplateID string    `json:"email_template_id"`
	Name            string    `json:"name"`
	Subject         string    `json:"subject"`
	Status          string    `json:"status"`
	IsDefault       bool      `json:"is_default"`
	IsSystem        bool      `json:"is_system"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// Email template response DTO (full details with body content)
type EmailTemplateResponseDTO struct {
	EmailTemplateID string    `json:"email_template_id"`
	Name            string    `json:"name"`
	Subject         string    `json:"subject"`
	BodyHTML        string    `json:"body_html"`
	BodyPlain       *string   `json:"body_plain"`
	Status          string    `json:"status"`
	IsDefault       bool      `json:"is_default"`
	IsSystem        bool      `json:"is_system"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// Create email template request DTO
type EmailTemplateCreateRequestDTO struct {
	Name      string  `json:"name"`
	Subject   string  `json:"subject"`
	BodyHTML  string  `json:"body_html"`
	BodyPlain *string `json:"body_plain,omitempty"`
	Status    *string `json:"status,omitempty"`
}

func (r EmailTemplateCreateRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, 100).Error("Name must be between 1 and 100 characters"),
		),
		validation.Field(&r.Subject,
			validation.Required.Error("Subject is required"),
			validation.Length(1, 255).Error("Subject must be between 1 and 255 characters"),
		),
		validation.Field(&r.BodyHTML,
			validation.Required.Error("Body HTML is required"),
		),
		validation.Field(&r.Status,
			validation.In(shared.StatusActive, shared.StatusInactive).Error("Status must be 'active' or 'inactive'"),
		),
	)
}

// Update email template request DTO
type EmailTemplateUpdateRequestDTO struct {
	Name      string  `json:"name"`
	Subject   string  `json:"subject"`
	BodyHTML  string  `json:"body_html"`
	BodyPlain *string `json:"body_plain,omitempty"`
	Status    *string `json:"status,omitempty"`
}

func (r EmailTemplateUpdateRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, 100).Error("Name must be between 1 and 100 characters"),
		),
		validation.Field(&r.Subject,
			validation.Required.Error("Subject is required"),
			validation.Length(1, 255).Error("Subject must be between 1 and 255 characters"),
		),
		validation.Field(&r.BodyHTML,
			validation.Required.Error("Body HTML is required"),
		),
		validation.Field(&r.Status,
			validation.In(shared.StatusActive, shared.StatusInactive).Error("Status must be 'active' or 'inactive'"),
		),
	)
}

// Update email template status request DTO
type EmailTemplateUpdateStatusRequestDTO struct {
	Status string `json:"status"`
}

func (r EmailTemplateUpdateStatusRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			validation.In(shared.StatusActive, shared.StatusInactive).Error("Status must be 'active' or 'inactive'"),
		),
	)
}

// Email template filter DTO
type EmailTemplateFilterDTO struct {
	Name      *string  `json:"name"`
	Status    []string `json:"status"`
	IsDefault *bool    `json:"is_default"`
	IsSystem  *bool    `json:"is_system"`

	// Pagination and sorting
	PaginationRequestDTO
}

// Validate validates the email template filter DTO.
func (f EmailTemplateFilterDTO) Validate() error {
	return validation.ValidateStruct(&f,
		validation.Field(&f.Status,
			validation.When(len(f.Status) > 0,
				validation.Each(validation.In(shared.StatusActive, shared.StatusInactive).Error("Status must be 'active' or 'inactive'")),
			),
		),
		validation.Field(&f.PaginationRequestDTO),
	)
}

// Login template list response DTO (without metadata)
type LoginTemplateListResponseDTO struct {
	LoginTemplateID string    `json:"login_template_id"`
	Name            string    `json:"name"`
	Description     *string   `json:"description"`
	Template        string    `json:"template"`
	Status          string    `json:"status"`
	IsDefault       bool      `json:"is_default"`
	IsSystem        bool      `json:"is_system"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// Login template response DTO (full details with metadata)
type LoginTemplateResponseDTO struct {
	LoginTemplateID string         `json:"login_template_id"`
	Name            string         `json:"name"`
	Description     *string        `json:"description"`
	Template        string         `json:"template"`
	Status          string         `json:"status"`
	Metadata        map[string]any `json:"metadata"`
	IsDefault       bool           `json:"is_default"`
	IsSystem        bool           `json:"is_system"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}

// Create login template request DTO
type LoginTemplateCreateRequestDTO struct {
	Name        string         `json:"name"`
	Description *string        `json:"description,omitempty"`
	Template    string         `json:"template"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	Status      *string        `json:"status,omitempty"`
}

func (r LoginTemplateCreateRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, 100).Error("Name must be between 1 and 100 characters"),
		),
		validation.Field(&r.Template,
			validation.Required.Error("Template is required"),
			validation.In(shared.LoginTemplateModern, shared.LoginTemplateClassic, shared.LoginTemplateMinimal, shared.LoginTemplateCorporate, shared.LoginTemplateCreative, shared.LoginTemplateCustom).Error("Template must be one of: modern, classic, minimal, corporate, creative, custom"),
		),
		validation.Field(&r.Status,
			validation.In(shared.StatusActive, shared.StatusInactive).Error("Status must be 'active' or 'inactive'"),
		),
	)
}

// Update login template request DTO
type LoginTemplateUpdateRequestDTO struct {
	Name        string         `json:"name"`
	Description *string        `json:"description,omitempty"`
	Template    string         `json:"template"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	Status      *string        `json:"status,omitempty"`
}

func (r LoginTemplateUpdateRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, 100).Error("Name must be between 1 and 100 characters"),
		),
		validation.Field(&r.Template,
			validation.Required.Error("Template is required"),
			validation.In(shared.LoginTemplateModern, shared.LoginTemplateClassic, shared.LoginTemplateMinimal, shared.LoginTemplateCorporate, shared.LoginTemplateCreative, shared.LoginTemplateCustom).Error("Template must be one of: modern, classic, minimal, corporate, creative, custom"),
		),
		validation.Field(&r.Status,
			validation.In(shared.StatusActive, shared.StatusInactive).Error("Status must be 'active' or 'inactive'"),
		),
	)
}

// Update login template status request DTO
type LoginTemplateUpdateStatusRequestDTO struct {
	Status string `json:"status"`
}

func (r LoginTemplateUpdateStatusRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			validation.In(shared.StatusActive, shared.StatusInactive).Error("Status must be 'active' or 'inactive'"),
		),
	)
}

// Login template filter DTO
type LoginTemplateFilterDTO struct {
	Name      *string  `json:"name"`
	Status    []string `json:"status"`
	Template  *string  `json:"template"`
	IsDefault *bool    `json:"is_default"`
	IsSystem  *bool    `json:"is_system"`

	// Pagination and sorting
	PaginationRequestDTO
}

// Validate validates the login template filter DTO.
func (f LoginTemplateFilterDTO) Validate() error {
	return validation.ValidateStruct(&f,
		validation.Field(&f.Template,
			validation.When(f.Template != nil,
				validation.In(shared.LoginTemplateModern, shared.LoginTemplateClassic, shared.LoginTemplateMinimal, shared.LoginTemplateCorporate, shared.LoginTemplateCreative, shared.LoginTemplateCustom).Error("Template must be one of: modern, classic, minimal, corporate, creative, custom"),
			),
		),
		validation.Field(&f.Status,
			validation.When(len(f.Status) > 0,
				validation.Each(validation.In(shared.StatusActive, shared.StatusInactive).Error("Status must be 'active' or 'inactive'")),
			),
		),
		validation.Field(&f.PaginationRequestDTO),
	)
}

// SMS template list response DTO (without message content)
type SMSTemplateListResponseDTO struct {
	SMSTemplateID string    `json:"sms_template_id"`
	Name          string    `json:"name"`
	Description   *string   `json:"description"`
	SenderID      *string   `json:"sender_id"`
	Status        string    `json:"status"`
	IsDefault     bool      `json:"is_default"`
	IsSystem      bool      `json:"is_system"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// SMS template response DTO (full details with message content)
type SMSTemplateResponseDTO struct {
	SMSTemplateID string    `json:"sms_template_id"`
	Name          string    `json:"name"`
	Description   *string   `json:"description"`
	Message       string    `json:"message"`
	SenderID      *string   `json:"sender_id"`
	Status        string    `json:"status"`
	IsDefault     bool      `json:"is_default"`
	IsSystem      bool      `json:"is_system"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// Create SMS template request DTO
type SMSTemplateCreateRequestDTO struct {
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	Message     string  `json:"message"`
	SenderID    *string `json:"sender_id,omitempty"`
	Status      *string `json:"status,omitempty"`
}

func (r SMSTemplateCreateRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, 100).Error("Name must be between 1 and 100 characters"),
		),
		validation.Field(&r.Message,
			validation.Required.Error("Message is required"),
		),
		validation.Field(&r.SenderID,
			validation.Length(0, 20).Error("Sender ID must not exceed 20 characters"),
		),
		validation.Field(&r.Status,
			validation.In(shared.StatusActive, shared.StatusInactive).Error("Status must be 'active' or 'inactive'"),
		),
	)
}

// Update SMS template request DTO
type SMSTemplateUpdateRequestDTO struct {
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	Message     string  `json:"message"`
	SenderID    *string `json:"sender_id,omitempty"`
	Status      *string `json:"status,omitempty"`
}

func (r SMSTemplateUpdateRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, 100).Error("Name must be between 1 and 100 characters"),
		),
		validation.Field(&r.Message,
			validation.Required.Error("Message is required"),
		),
		validation.Field(&r.SenderID,
			validation.Length(0, 20).Error("Sender ID must not exceed 20 characters"),
		),
		validation.Field(&r.Status,
			validation.In(shared.StatusActive, shared.StatusInactive).Error("Status must be 'active' or 'inactive'"),
		),
	)
}

// Update SMS template status request DTO
type SMSTemplateUpdateStatusRequestDTO struct {
	Status string `json:"status"`
}

func (r SMSTemplateUpdateStatusRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			validation.In(shared.StatusActive, shared.StatusInactive).Error("Status must be 'active' or 'inactive'"),
		),
	)
}

// SMS template filter DTO
type SMSTemplateFilterDTO struct {
	Name      *string  `json:"name"`
	Status    []string `json:"status"`
	IsDefault *bool    `json:"is_default"`
	IsSystem  *bool    `json:"is_system"`

	// Pagination and sorting
	PaginationRequestDTO
}

// Validate validates the SMS template filter DTO.
func (f SMSTemplateFilterDTO) Validate() error {
	return validation.ValidateStruct(&f,
		validation.Field(&f.Status,
			validation.When(len(f.Status) > 0,
				validation.Each(validation.In(shared.StatusActive, shared.StatusInactive).Error("Status must be 'active' or 'inactive'")),
			),
		),
		validation.Field(&f.PaginationRequestDTO),
	)
}
