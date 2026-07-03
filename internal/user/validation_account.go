package user

import (
	"errors"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/maintainerd/maintainerd-auth/internal/platform/security"
	"github.com/maintainerd/maintainerd-auth/internal/platform/valid"
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

func (r *SendPhoneVerificationDTO) Validate() error {
	r.Phone = security.SanitizeInput(r.Phone)
	return validation.ValidateStruct(r,
		validation.Field(&r.Phone, validation.Required, validation.By(validateAccountPhoneNumber)),
	)
}

func (r *VerifyPhoneDTO) Validate() error {
	r.Phone = security.SanitizeInput(r.Phone)
	r.Code = security.SanitizeInput(r.Code)
	return validation.ValidateStruct(r,
		validation.Field(&r.Phone, validation.Required, validation.By(validateAccountPhoneNumber)),
		validation.Field(&r.Code, validation.Required),
	)
}

// validateAccountPhoneNumber checks basic E.164-ish phone format. The Required
// rule handles emptiness, so an empty value passes here (no double error).
func validateAccountPhoneNumber(value any) error {
	s, _ := value.(string)
	if s == "" {
		return nil
	}
	if !valid.IsValidPhoneNumber(s) {
		return errors.New("must be a valid phone number")
	}
	return nil
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
