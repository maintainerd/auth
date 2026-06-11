package branding

import (
	"net/url"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
)

var httpURL = validation.By(func(value any) error {
	raw, _ := value.(string)
	if raw == "" {
		return nil
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return validation.NewError("validation_url", "must be a valid URL")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return validation.NewError("validation_http_url", "must use http or https")
	}
	return nil
})

// Validate validates the branding update request.
func (r BrandingUpdateRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.CompanyName,
			validation.Length(0, 255).Error("Company name must not exceed 255 characters"),
		),
		validation.Field(&r.LogoURL,
			validation.Length(0, 2048).Error("Logo URL must not exceed 2048 characters"),
			validation.When(r.LogoURL != "", is.URL.Error("Logo URL must be a valid URL"), httpURL),
		),
		validation.Field(&r.FaviconURL,
			validation.Length(0, 2048).Error("Favicon URL must not exceed 2048 characters"),
			validation.When(r.FaviconURL != "", is.URL.Error("Favicon URL must be a valid URL"), httpURL),
		),
		validation.Field(&r.Name,
			validation.Length(0, 100).Error("Name must not exceed 100 characters"),
		),
		validation.Field(&r.SupportURL,
			validation.Length(0, 2048).Error("Support URL must not exceed 2048 characters"),
			validation.When(r.SupportURL != "", is.URL.Error("Support URL must be a valid URL"), httpURL),
		),
		validation.Field(&r.PrivacyPolicyURL,
			validation.Length(0, 2048).Error("Privacy policy URL must not exceed 2048 characters"),
			validation.When(r.PrivacyPolicyURL != "", is.URL.Error("Privacy policy URL must be a valid URL"), httpURL),
		),
		validation.Field(&r.TermsOfServiceURL,
			validation.Length(0, 2048).Error("Terms of service URL must not exceed 2048 characters"),
			validation.When(r.TermsOfServiceURL != "", is.URL.Error("Terms of service URL must be a valid URL"), httpURL),
		),
	)
}
