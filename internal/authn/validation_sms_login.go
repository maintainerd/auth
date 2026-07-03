package authn

import (
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/maintainerd/maintainerd-auth/internal/platform/security"
)

func (r *SMSLoginSendDTO) Validate() error {
	r.Phone = security.SanitizeInput(r.Phone)
	return validation.ValidateStruct(r,
		validation.Field(&r.Phone, validation.Required, validation.Length(1, 20)),
	)
}

func (r *SMSLoginVerifyDTO) Validate() error {
	r.Phone = security.SanitizeInput(r.Phone)
	return validation.ValidateStruct(r,
		validation.Field(&r.Phone, validation.Required, validation.Length(1, 20)),
		validation.Field(&r.OTP, validation.Required, validation.Length(6, 6)),
	)
}
