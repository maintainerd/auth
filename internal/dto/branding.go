package dto

import (
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
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
	)
}
