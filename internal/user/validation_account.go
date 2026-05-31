package user

import (
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/maintainerd/auth/internal/platform/security"
)

func (r *ChangeEmailRequestDTO) Validate() error {
	r.NewEmail = security.SanitizeInput(r.NewEmail)
	return validation.ValidateStruct(r,
		validation.Field(&r.NewEmail, validation.Required, is.Email),
		validation.Field(&r.CurrentPassword, validation.Required),
	)
}

func (r *VerifyEmailChangeDTO) Validate() error {
	return validation.ValidateStruct(r,
		validation.Field(&r.OTP, validation.Required, validation.Length(6, 6)),
	)
}

func (r *ChangeUsernameDTO) Validate() error {
	r.NewUsername = security.SanitizeInput(r.NewUsername)
	return validation.ValidateStruct(r,
		validation.Field(&r.NewUsername, validation.Required, validation.Length(3, 50)),
		validation.Field(&r.CurrentPassword, validation.Required),
	)
}

func (r *AccountDeleteDTO) Validate() error {
	return validation.ValidateStruct(r,
		validation.Field(&r.CurrentPassword, validation.Required),
	)
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
