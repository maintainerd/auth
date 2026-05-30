package oauth

import (
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/maintainerd/auth/internal/platform/security"
)

func (r *OAuthCIBARequestDTO) Validate() error {
	r.Scope = security.SanitizeInput(r.Scope)
	r.ClientNotificationToken = security.SanitizeInput(r.ClientNotificationToken)
	r.ACRValues = security.SanitizeInput(r.ACRValues)
	r.LoginHint = security.SanitizeInput(r.LoginHint)
	r.LoginHintToken = security.SanitizeInput(r.LoginHintToken)
	r.IDTokenHint = security.SanitizeInput(r.IDTokenHint)
	r.BindingMessage = security.SanitizeInput(r.BindingMessage)
	r.UserCode = security.SanitizeInput(r.UserCode)
	r.ClientID = security.SanitizeInput(r.ClientID)
	r.ClientSecret = security.SanitizeInput(r.ClientSecret)

	return validation.ValidateStruct(r,
		validation.Field(&r.ClientID,
			validation.Required.Error("client_id is required"),
		),
		validation.Field(&r.Scope,
			validation.Required.Error("scope is required"),
			validation.Length(1, 1024).Error("scope must not exceed 1024 characters"),
		),
		validation.Field(&r.LoginHint,
			validation.When(
				r.LoginHintToken == "" && r.IDTokenHint == "",
				validation.Required.Error("one of login_hint, login_hint_token, or id_token_hint is required"),
			),
		),
		validation.Field(&r.BindingMessage,
			validation.Length(0, 128).Error("binding_message must not exceed 128 characters"),
		),
	)
}

func (r *OAuthCIBATokenRequestDTO) Validate() error {
	r.AuthReqID = security.SanitizeInput(r.AuthReqID)
	r.ClientID = security.SanitizeInput(r.ClientID)
	r.ClientSecret = security.SanitizeInput(r.ClientSecret)

	return validation.ValidateStruct(r,
		validation.Field(&r.AuthReqID,
			validation.Required.Error("auth_req_id is required"),
		),
		validation.Field(&r.ClientID,
			validation.Required.Error("client_id is required"),
		),
	)
}
