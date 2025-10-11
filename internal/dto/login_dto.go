package dto

import (
	"net/url"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/maintainerd/auth/internal/util"
)

// Login request payload structure
type LoginRequestDto struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (r *LoginRequestDto) Validate() error {
	// Sanitize inputs first
	r.Username = util.SanitizeInput(r.Username)
	r.Password = util.SanitizeInput(r.Password)

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
type LoginQueryDto struct {
	ClientID   string `json:"client_id"`
	ProviderID string `json:"provider_id"`
}

func (q *LoginQueryDto) Validate() error {
	// Sanitize inputs first
	q.ClientID = util.SanitizeInput(q.ClientID)
	q.ProviderID = util.SanitizeInput(q.ProviderID)

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
func (q *LoginQueryDto) ValidateSignedURL(values url.Values) error {
	// Extract and validate signed URL parameters
	if _, err := util.ValidateSignedURL(values); err != nil {
		return err
	}
	return nil
}

// LoginResponseDto is the response structure for login operations
type LoginResponseDto struct {
	AccessToken  string `json:"access_token"`
	IDToken      string `json:"id_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	ExpiresIn    int64  `json:"expires_in"`
	TokenType    string `json:"token_type"`
	IssuedAt     int64  `json:"issued_at"`
}
