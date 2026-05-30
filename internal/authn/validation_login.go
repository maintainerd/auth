package authn

import (
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/maintainerd/auth/internal/platform/security"
	"github.com/maintainerd/auth/internal/platform/signedurl"
	"net/url"
)

func (r *LoginRequestDTO) Validate() error {
	// Sanitize inputs first
	r.Username = security.SanitizeInput(r.Username)
	r.Password = security.SanitizeInput(r.Password)

	return validation.ValidateStruct(r,
		validation.Field(&r.Username,
			validation.Required.Error("Username is required"),
			validation.Length(1, 255).Error("Username must not exceed 255 characters"),
		),
		validation.Field(&r.Password,
			validation.Required.Error("Password is required"),
			validation.Length(1, 128).Error("Password must not exceed 128 characters"),
		),
	)
}

// Login query parameters structure
type LoginQueryDTO struct {
	ClientID   string `json:"client_id"`
	ProviderID string `json:"provider_id"`
}

func (q *LoginQueryDTO) Validate() error {
	// Sanitize inputs first
	q.ClientID = security.SanitizeInput(q.ClientID)
	q.ProviderID = security.SanitizeInput(q.ProviderID)

	return validation.ValidateStruct(q,
		validation.Field(&q.ClientID,
			validation.Required.Error("Client ID is required"),
			validation.Length(1, 255).Error("Client ID must not exceed 255 characters"),
		),
		validation.Field(&q.ProviderID,
			validation.Required.Error("Provider ID is required"),
			validation.Length(1, 255).Error("Provider ID must not exceed 255 characters"),
		),
	)
}

// ValidateSignedURL validates signed URL parameters for login
func (q *LoginQueryDTO) ValidateSignedURL(values url.Values) error {
	// Extract and validate signed URL parameters
	if _, err := signedurl.ValidateSignedURL(values); err != nil {
		return err
	}
	return nil
}
