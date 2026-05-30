package oauth

import (
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/maintainerd/auth/internal/platform/security"
)

func (r *OAuthBackchannelLogoutRequestDTO) Validate() error {
	r.LogoutToken = security.SanitizeInput(r.LogoutToken)

	return validation.ValidateStruct(r,
		validation.Field(&r.LogoutToken,
			validation.Required.Error("logout_token is required"),
		),
	)
}

func (r *OAuthEndSessionRequestDTO) Validate() error {
	r.IDTokenHint = security.SanitizeInput(r.IDTokenHint)
	r.PostLogoutRedirectURI = security.SanitizeInput(r.PostLogoutRedirectURI)
	r.State = security.SanitizeInput(r.State)
	r.ClientID = security.SanitizeInput(r.ClientID)

	return validation.ValidateStruct(r,
		validation.Field(&r.PostLogoutRedirectURI,
			validation.Length(0, 2048).Error("post_logout_redirect_uri must not exceed 2048 characters"),
		),
		validation.Field(&r.State,
			validation.Length(0, 512).Error("state must not exceed 512 characters"),
		),
	)
}
