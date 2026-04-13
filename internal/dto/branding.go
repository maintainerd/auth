package dto

import (
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
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
