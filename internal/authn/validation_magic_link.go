package authn

import (
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/maintainerd/maintainerd-auth/internal/platform/security"
)

func (r *SendMagicLinkRequestDTO) Validate() error {
	r.Email = security.SanitizeInput(r.Email)

	return validation.ValidateStruct(r,
		validation.Field(&r.Email,
			validation.Required.Error("Email is required"),
			is.Email.Error("Email must be a valid email address"),
			validation.Length(1, 255).Error("Email must not exceed 255 characters"),
		),
	)
}

func (r *VerifyMagicLinkRequestDTO) Validate() error {
	r.Token = security.SanitizeInput(r.Token)

	return validation.ValidateStruct(r,
		validation.Field(&r.Token,
			validation.Required.Error("Token is required"),
			validation.Length(16, 256).Error("Token has an invalid length"),
		),
	)
}
