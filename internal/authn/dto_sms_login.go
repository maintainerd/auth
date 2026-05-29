package authn

import (
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/maintainerd/auth/internal/platform/security"
)

// SMSLoginSendDTO is the request to send a one-time SMS code.
type SMSLoginSendDTO struct {
	Phone      string `json:"phone"`
	ClientID   string `json:"client_id"`
	ProviderID string `json:"provider_id"`
}

func (r *SMSLoginSendDTO) Validate() error {
	r.Phone = security.SanitizeInput(r.Phone)
	return validation.ValidateStruct(r,
		validation.Field(&r.Phone, validation.Required, validation.Length(1, 20)),
		validation.Field(&r.ClientID, validation.Required),
		validation.Field(&r.ProviderID, validation.Required),
	)
}

// SMSLoginVerifyDTO is the request to verify an SMS OTP and obtain tokens.
type SMSLoginVerifyDTO struct {
	Phone      string `json:"phone"`
	OTP        string `json:"otp"`
	ClientID   string `json:"client_id"`
	ProviderID string `json:"provider_id"`
}

func (r *SMSLoginVerifyDTO) Validate() error {
	r.Phone = security.SanitizeInput(r.Phone)
	return validation.ValidateStruct(r,
		validation.Field(&r.Phone, validation.Required, validation.Length(1, 20)),
		validation.Field(&r.OTP, validation.Required, validation.Length(6, 6)),
		validation.Field(&r.ClientID, validation.Required),
		validation.Field(&r.ProviderID, validation.Required),
	)
}
