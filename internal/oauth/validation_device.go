package oauth

import (
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/maintainerd/auth/internal/platform/security"
)

func (r *OAuthDeviceAuthorizationRequestDTO) Validate() error {
	r.ClientID = security.SanitizeInput(r.ClientID)
	r.ClientSecret = security.SanitizeInput(r.ClientSecret)
	r.Scope = security.SanitizeInput(r.Scope)

	return validation.ValidateStruct(r,
		validation.Field(&r.ClientID,
			validation.Required.Error("client_id is required"),
			validation.Length(1, 255).Error("client_id must not exceed 255 characters"),
		),
		validation.Field(&r.Scope,
			validation.Length(0, 1024).Error("scope must not exceed 1024 characters"),
		),
	)
}

func (r *OAuthDeviceVerifyRequestDTO) Validate() error {
	r.UserCode = security.SanitizeInput(r.UserCode)

	return validation.ValidateStruct(r,
		validation.Field(&r.UserCode,
			validation.Required.Error("user_code is required"),
			validation.Length(8, 9).Error("user_code must be 8 characters (XXXX-XXXX format)"),
		),
	)
}

func (r *OAuthDeviceTokenRequestDTO) Validate() error {
	r.DeviceCode = security.SanitizeInput(r.DeviceCode)
	r.ClientID = security.SanitizeInput(r.ClientID)
	r.ClientSecret = security.SanitizeInput(r.ClientSecret)

	return validation.ValidateStruct(r,
		validation.Field(&r.DeviceCode,
			validation.Required.Error("device_code is required"),
		),
		validation.Field(&r.ClientID,
			validation.Required.Error("client_id is required"),
		),
	)
}
