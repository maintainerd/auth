package branding

import (
	"time"
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

// Update email template request DTO
type EmailTemplateUpdateRequestDTO struct {
	Name      string  `json:"name"`
	Subject   string  `json:"subject"`
	BodyHTML  string  `json:"body_html"`
	BodyPlain *string `json:"body_plain,omitempty"`
	Status    *string `json:"status,omitempty"`
}

// Update email template status request DTO
type EmailTemplateUpdateStatusRequestDTO struct {
	Status string `json:"status"`
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

// Update login template request DTO
type LoginTemplateUpdateRequestDTO struct {
	Name        string         `json:"name"`
	Description *string        `json:"description,omitempty"`
	Template    string         `json:"template"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	Status      *string        `json:"status,omitempty"`
}

// Update login template status request DTO
type LoginTemplateUpdateStatusRequestDTO struct {
	Status string `json:"status"`
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

// Update SMS template request DTO
type SMSTemplateUpdateRequestDTO struct {
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	Message     string  `json:"message"`
	SenderID    *string `json:"sender_id,omitempty"`
	Status      *string `json:"status,omitempty"`
}

// Update SMS template status request DTO
type SMSTemplateUpdateStatusRequestDTO struct {
	Status string `json:"status"`
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
