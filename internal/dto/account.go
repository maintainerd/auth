package dto

import (
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/maintainerd/auth/internal/security"
)

// ChangeEmailRequestDTO is the request to initiate an email address change.
type ChangeEmailRequestDTO struct {
	NewEmail        string `json:"new_email"`
	CurrentPassword string `json:"current_password"`
}

func (r *ChangeEmailRequestDTO) Validate() error {
	r.NewEmail = security.SanitizeInput(r.NewEmail)
	return validation.ValidateStruct(r,
		validation.Field(&r.NewEmail, validation.Required, is.Email),
		validation.Field(&r.CurrentPassword, validation.Required),
	)
}

// VerifyEmailChangeDTO is the request to confirm an email change via OTP.
type VerifyEmailChangeDTO struct {
	OTP string `json:"otp"`
}

func (r *VerifyEmailChangeDTO) Validate() error {
	return validation.ValidateStruct(r,
		validation.Field(&r.OTP, validation.Required, validation.Length(6, 6)),
	)
}

// ChangeUsernameDTO is the request to change a username.
type ChangeUsernameDTO struct {
	NewUsername     string `json:"new_username"`
	CurrentPassword string `json:"current_password"`
}

func (r *ChangeUsernameDTO) Validate() error {
	r.NewUsername = security.SanitizeInput(r.NewUsername)
	return validation.ValidateStruct(r,
		validation.Field(&r.NewUsername, validation.Required, validation.Length(3, 50)),
		validation.Field(&r.CurrentPassword, validation.Required),
	)
}

// AccountDeleteDTO is the request to permanently delete an account.
type AccountDeleteDTO struct {
	CurrentPassword string `json:"current_password"`
}

func (r *AccountDeleteDTO) Validate() error {
	return validation.ValidateStruct(r,
		validation.Field(&r.CurrentPassword, validation.Required),
	)
}

// AccountExportDTO is the response payload for account data export.
type AccountExportDTO struct {
	UserUUID  string      `json:"user_uuid"`
	Username  string      `json:"username"`
	Email     string      `json:"email"`
	Phone     string      `json:"phone"`
	CreatedAt time.Time   `json:"created_at"`
	Profile   interface{} `json:"profile,omitempty"`
	Roles     []string    `json:"roles"`
	Settings  interface{} `json:"settings,omitempty"`
}

// GenerateBackupCodesResponseDTO holds the plaintext backup codes shown once.
type GenerateBackupCodesResponseDTO struct {
	Codes []string `json:"codes"`
}

// VerifyBackupCodeDTO is the request to recover an account via a backup code.
type VerifyBackupCodeDTO struct {
	Email      string `json:"email"`
	Code       string `json:"code"`
	ClientID   string `json:"client_id"`
	ProviderID string `json:"provider_id"`
}

func (r *VerifyBackupCodeDTO) Validate() error {
	r.Email = security.SanitizeInput(r.Email)
	return validation.ValidateStruct(r,
		validation.Field(&r.Email, validation.Required, is.Email),
		validation.Field(&r.Code, validation.Required),
		validation.Field(&r.ClientID, validation.Required),
		validation.Field(&r.ProviderID, validation.Required),
	)
}
