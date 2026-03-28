package dto

import (
	"net/url"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/maintainerd/auth/internal/util"
)

// ResetPasswordRequestDto represents the request to reset a password
// Token is always extracted from the signed URL, not from request body
type ResetPasswordRequestDto struct {
	NewPassword string `json:"new_password" example:"NewSecurePassword123!"`
}

// Validate validates the reset password request
func (dto ResetPasswordRequestDto) Validate() error {
	return validation.ValidateStruct(&dto,
		validation.Field(&dto.NewPassword, validation.Required.Error("New password is required")),
		// Token is optional in request body - can come from signed URL instead
	)
}

// ResetPasswordResponseDto represents the response after password reset
type ResetPasswordResponseDto struct {
	Message string `json:"message" example:"Password has been reset successfully"`
	Success bool   `json:"success" example:"true"`
}

// ResetPasswordQueryDto represents query parameters for signed URL validation
type ResetPasswordQueryDto struct {
	Token      string `json:"token"`
	ClientID   string `json:"client_id"`
	ProviderID string `json:"provider_id"`
	Expires    string `json:"expires"`
	Sig        string `json:"sig"`
}

// Validate validates the reset password query parameters
func (q ResetPasswordQueryDto) Validate() error {
	return validation.ValidateStruct(&q,
		validation.Field(&q.Token,
			validation.Required.Error("Token is required"),
			validation.Length(1, 500).Error("Token must not exceed 500 characters"),
		),
		validation.Field(&q.ClientID,
			validation.Required.Error("Client ID is required"),
			validation.Length(1, 100).Error("Client ID must not exceed 100 characters"),
		),
		validation.Field(&q.ProviderID,
			validation.Required.Error("Provider ID is required"),
			validation.Length(1, 100).Error("Provider ID must not exceed 100 characters"),
		),
		validation.Field(&q.Expires,
			validation.Required.Error("Expires is required"),
			validation.Length(1, 50).Error("Expires must not exceed 50 characters"),
		),
		validation.Field(&q.Sig,
			validation.Required.Error("Signature is required"),
			validation.Length(1, 500).Error("Signature must not exceed 500 characters"),
		),
	)
}

// ValidateSignedURL validates signed URL parameters for reset password
func (q *ResetPasswordQueryDto) ValidateSignedURL(values url.Values) error {
	// Extract and validate signed URL parameters
	if _, err := util.ValidateSignedURL(values); err != nil {
		return err
	}
	return nil
}
