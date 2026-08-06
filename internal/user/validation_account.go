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
		validation.Field(&r.NewUsername, validation.Required, validation.Length(3, 50), validation.By(validateUsernameCharset)),
		validation.Field(&r.CurrentPassword, validation.Required),
	)
}

// validateUsernameCharset rejects a username that could be mistaken for an email
// address at login time.
//
// authn.findLoginUser resolves the USERNAME column first and only falls back to
// email, and the two columns carry separate uniqueness indexes. With only a
// length rule here, Mallory could rename herself to "alice@corp.com": the
// uniqueness pre-check queries usernames alone so nothing collides, but Alice's
// own email login then resolves to Mallory's row and Alice is locked out for
// good. The allowlist — letters, digits, dot, underscore, hyphen — excludes '@'
// by construction rather than banning one character an encoding trick could
// smuggle past.
//
// authn's registration DTO keeps an identical copy; internal/user and
// internal/authn are siblings, so neither imports the other. Both must agree —
// a rule enforced only at registration is bypassed by a rename, and one enforced
// only at rename is bypassed by registering the name directly.
func validateUsernameCharset(value any) error {
	s, _ := value.(string)
	if s == "" {
		return nil // Required handles emptiness; no double error.
	}
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		case r == '.' || r == '_' || r == '-':
		default:
			return errors.New("username may only contain letters, digits, and . _ -")
		}
	}
	return nil
}

// Validate checks only presence. SanitizeInput is deliberately NOT applied to
// either password — it strips and rewrites characters, which would silently
// mutate the secret being set. Length and composition are the tenant policy's
// job (security.ValidatePasswordPolicyForUser); declaring them here too would
// give two sources of truth that drift.
func (r *ChangePasswordDTO) Validate() error {
	return validation.ValidateStruct(r,
		validation.Field(&r.CurrentPassword, validation.Required),
		validation.Field(&r.NewPassword, validation.Required),
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

// Validate requires a password alongside the backup code. A backup code is a
// recovery SECOND factor, not a standalone primary credential: without the
// password this endpoint minted a full access + refresh token set from an email
// address and one 8–10 character code, which bypasses the tenant's enforced-MFA
// policy outright. SanitizeInput is deliberately NOT applied to the password —
// it rewrites characters, which would mutate the secret being compared.
func (r *VerifyBackupCodeDTO) Validate() error {
	r.Email = security.SanitizeInput(r.Email)
	return validation.ValidateStruct(r,
		validation.Field(&r.Email, validation.Required, is.Email),
		validation.Field(&r.Password, validation.Required),
		validation.Field(&r.Code, validation.Required),
		validation.Field(&r.ClientID, validation.Required),
		validation.Field(&r.ProviderID, validation.Required),
	)
}
