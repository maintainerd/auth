package oauth

import (
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/maintainerd/auth/internal/platform/security"
)

func (r *OAuthClientRegistrationRequestDTO) Validate() error {
	r.ClientName = security.SanitizeInput(r.ClientName)
	r.Scope = security.SanitizeInput(r.Scope)
	r.TokenEndpointAuthMethod = security.SanitizeInput(r.TokenEndpointAuthMethod)
	r.LogoURI = security.SanitizeInput(r.LogoURI)
	r.PolicyURI = security.SanitizeInput(r.PolicyURI)
	r.TOSURI = security.SanitizeInput(r.TOSURI)

	for i, u := range r.RedirectURIs {
		r.RedirectURIs[i] = security.SanitizeInput(u)
	}
	for i, g := range r.GrantTypes {
		r.GrantTypes[i] = security.SanitizeInput(g)
	}
	for i, rt := range r.ResponseTypes {
		r.ResponseTypes[i] = security.SanitizeInput(rt)
	}

	return validation.ValidateStruct(r,
		validation.Field(&r.ClientName,
			validation.Required.Error("client_name is required"),
			validation.Length(1, 255).Error("client_name must not exceed 255 characters"),
		),
		validation.Field(&r.RedirectURIs,
			validation.Required.Error("redirect_uris is required"),
			validation.Length(1, 10).Error("between 1 and 10 redirect_uris are allowed"),
		),
		validation.Field(&r.TokenEndpointAuthMethod,
			validation.In("client_secret_basic", "client_secret_post", "none", "").
				Error("token_endpoint_auth_method must be client_secret_basic, client_secret_post, or none"),
		),
		validation.Field(&r.Scope,
			validation.Length(0, 1024).Error("scope must not exceed 1024 characters"),
		),
		validation.Field(&r.IdentityProviderID,
			validation.Required.Error("identity_provider_id is required"),
			validation.Min(int64(1)).Error("identity_provider_id must be a positive integer"),
		),
	)
}
