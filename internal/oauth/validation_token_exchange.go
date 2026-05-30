package oauth

import (
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/maintainerd/auth/internal/platform/security"
)

func (r *OAuthTokenExchangeRequestDTO) Validate() error {
	r.SubjectToken = security.SanitizeInput(r.SubjectToken)
	r.SubjectTokenType = security.SanitizeInput(r.SubjectTokenType)
	r.ActorToken = security.SanitizeInput(r.ActorToken)
	r.ActorTokenType = security.SanitizeInput(r.ActorTokenType)
	r.RequestedTokenType = security.SanitizeInput(r.RequestedTokenType)
	r.Resource = security.SanitizeInput(r.Resource)
	r.Audience = security.SanitizeInput(r.Audience)
	r.Scope = security.SanitizeInput(r.Scope)
	r.ClientID = security.SanitizeInput(r.ClientID)
	r.ClientSecret = security.SanitizeInput(r.ClientSecret)

	const (
		tokenTypeAccessToken  = "urn:ietf:params:oauth:token-type:access_token"
		tokenTypeRefreshToken = "urn:ietf:params:oauth:token-type:refresh_token"
		tokenTypeIDToken      = "urn:ietf:params:oauth:token-type:id_token"
		tokenTypeJWT          = "urn:ietf:params:oauth:token-type:jwt"
	)

	return validation.ValidateStruct(r,
		validation.Field(&r.SubjectToken,
			validation.Required.Error("subject_token is required"),
		),
		validation.Field(&r.SubjectTokenType,
			validation.Required.Error("subject_token_type is required"),
			validation.In(tokenTypeAccessToken, tokenTypeRefreshToken, tokenTypeIDToken, tokenTypeJWT).
				Error("subject_token_type must be a valid token type URI"),
		),
		validation.Field(&r.RequestedTokenType,
			validation.In(tokenTypeAccessToken, tokenTypeRefreshToken, tokenTypeIDToken, tokenTypeJWT, "").
				Error("requested_token_type must be a valid token type URI"),
		),
		validation.Field(&r.ClientID,
			validation.Required.Error("client_id is required"),
		),
		validation.Field(&r.Scope,
			validation.Length(0, 1024).Error("scope must not exceed 1024 characters"),
		),
	)
}
