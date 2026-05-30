package oauth

import (
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/maintainerd/auth/internal/platform/security"
)

// Validate sanitises inputs and checks grant-type-specific required fields.
func (r *OAuthTokenRequestDTO) Validate() error {
	r.GrantType = security.SanitizeInput(r.GrantType)
	r.Code = security.SanitizeInput(r.Code)
	r.RedirectURI = security.SanitizeInput(r.RedirectURI)
	r.CodeVerifier = security.SanitizeInput(r.CodeVerifier)
	r.RefreshToken = security.SanitizeInput(r.RefreshToken)
	r.Scope = security.SanitizeInput(r.Scope)
	r.ClientID = security.SanitizeInput(r.ClientID)
	r.ClientSecret = security.SanitizeInput(r.ClientSecret)

	return validation.ValidateStruct(r,
		validation.Field(&r.GrantType,
			validation.Required.Error("grant_type is required"),
			validation.In("authorization_code", "refresh_token", "client_credentials").
				Error("grant_type must be one of: authorization_code, refresh_token, client_credentials"),
		),
	)
}

// Validate sanitises inputs and checks the required token field.
func (r *OAuthRevokeRequestDTO) Validate() error {
	r.Token = security.SanitizeInput(r.Token)
	r.TokenTypeHint = security.SanitizeInput(r.TokenTypeHint)
	r.ClientID = security.SanitizeInput(r.ClientID)
	r.ClientSecret = security.SanitizeInput(r.ClientSecret)

	return validation.ValidateStruct(r,
		validation.Field(&r.Token,
			validation.Required.Error("token is required"),
		),
		validation.Field(&r.TokenTypeHint,
			validation.In("access_token", "refresh_token", "").
				Error("token_type_hint must be 'access_token' or 'refresh_token'"),
		),
	)
}

// Validate sanitises inputs and checks the required token field.
func (r *OAuthIntrospectRequestDTO) Validate() error {
	r.Token = security.SanitizeInput(r.Token)
	r.TokenTypeHint = security.SanitizeInput(r.TokenTypeHint)

	return validation.ValidateStruct(r,
		validation.Field(&r.Token,
			validation.Required.Error("token is required"),
		),
		validation.Field(&r.TokenTypeHint,
			validation.In("access_token", "refresh_token", "").
				Error("token_type_hint must be 'access_token' or 'refresh_token'"),
		),
	)
}
