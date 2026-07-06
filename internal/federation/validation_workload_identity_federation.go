package federation

import (
	"net/url"
	"strings"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/maintainerd/maintainerd-auth/internal/platform/security"
)

// httpsURLRule requires a syntactically valid absolute https:// URL. OIDC
// issuers must be reachable over TLS (OpenID Connect Discovery §4).
var httpsURLRule = validation.By(func(value any) error {
	raw, _ := value.(string)
	if raw == "" {
		return nil
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.Host == "" {
		return validation.NewError("validation_issuer_url", "issuer_url must be a valid https URL")
	}
	return nil
})

// scopeListRule validates each requested scope string is non-empty and bounded.
var scopeListRule = validation.By(func(value any) error {
	scopes, ok := value.([]string)
	if !ok {
		return nil
	}
	for _, s := range scopes {
		trimmed := strings.TrimSpace(s)
		if trimmed == "" {
			return validation.NewError("validation_scope", "allowed_scopes must not contain empty values")
		}
		if len(trimmed) > 128 {
			return validation.NewError("validation_scope", "each scope must not exceed 128 characters")
		}
	}
	return nil
})

// Validate validates the create request.
func (r *WorkloadIdentityFederationCreateRequestDTO) Validate() error {
	r.ClientUUID = security.SanitizeInput(r.ClientUUID)
	r.Name = security.SanitizeInput(r.Name)
	r.Description = security.SanitizeInput(r.Description)
	r.IssuerURL = strings.TrimSpace(r.IssuerURL)
	r.Audience = security.SanitizeInput(r.Audience)
	r.SubjectClaim = security.SanitizeInput(r.SubjectClaim)
	r.SubjectPattern = security.SanitizeInput(r.SubjectPattern)

	return validation.ValidateStruct(r,
		validation.Field(&r.ClientUUID,
			validation.Required.Error("client_uuid is required"),
			is.UUID.Error("client_uuid must be a valid UUID"),
		),
		validation.Field(&r.Name,
			validation.Required.Error("name is required"),
			validation.Length(1, 100).Error("name must be between 1 and 100 characters"),
		),
		validation.Field(&r.Description,
			validation.Length(0, 2000).Error("description must not exceed 2000 characters"),
		),
		validation.Field(&r.IssuerURL,
			validation.Required.Error("issuer_url is required"),
			validation.Length(0, 2048).Error("issuer_url must not exceed 2048 characters"),
			httpsURLRule,
		),
		validation.Field(&r.Audience,
			validation.Required.Error("audience is required"),
			validation.Length(1, 512).Error("audience must be between 1 and 512 characters"),
		),
		validation.Field(&r.SubjectClaim,
			validation.Length(0, 100).Error("subject_claim must not exceed 100 characters"),
		),
		validation.Field(&r.SubjectPattern,
			validation.Required.Error("subject_pattern is required"),
			validation.Length(1, 512).Error("subject_pattern must be between 1 and 512 characters"),
		),
		validation.Field(&r.AllowedScopes, scopeListRule),
	)
}

// Validate validates the update request.
func (r *WorkloadIdentityFederationUpdateRequestDTO) Validate() error {
	r.Name = security.SanitizeInput(r.Name)
	r.Description = security.SanitizeInput(r.Description)
	r.IssuerURL = strings.TrimSpace(r.IssuerURL)
	r.Audience = security.SanitizeInput(r.Audience)
	r.SubjectClaim = security.SanitizeInput(r.SubjectClaim)
	r.SubjectPattern = security.SanitizeInput(r.SubjectPattern)

	return validation.ValidateStruct(r,
		validation.Field(&r.Name,
			validation.Required.Error("name is required"),
			validation.Length(1, 100).Error("name must be between 1 and 100 characters"),
		),
		validation.Field(&r.Description,
			validation.Length(0, 2000).Error("description must not exceed 2000 characters"),
		),
		validation.Field(&r.IssuerURL,
			validation.Required.Error("issuer_url is required"),
			validation.Length(0, 2048).Error("issuer_url must not exceed 2048 characters"),
			httpsURLRule,
		),
		validation.Field(&r.Audience,
			validation.Required.Error("audience is required"),
			validation.Length(1, 512).Error("audience must be between 1 and 512 characters"),
		),
		validation.Field(&r.SubjectClaim,
			validation.Length(0, 100).Error("subject_claim must not exceed 100 characters"),
		),
		validation.Field(&r.SubjectPattern,
			validation.Required.Error("subject_pattern is required"),
			validation.Length(1, 512).Error("subject_pattern must be between 1 and 512 characters"),
		),
		validation.Field(&r.AllowedScopes, scopeListRule),
	)
}

// Validate validates the list filter.
func (f WorkloadIdentityFederationFilterDTO) Validate() error {
	return validation.ValidateStruct(&f,
		validation.Field(&f.PaginationRequestDTO),
	)
}
