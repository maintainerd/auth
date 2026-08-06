package federation

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/maintainerd/maintainerd-auth/internal/platform/config"
	"github.com/maintainerd/maintainerd-auth/internal/platform/jwt"
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

// selfIssuerRule refuses a federation that trusts THIS server as its issuer.
//
// The platform serves a valid OIDC discovery document, so probeIssuer would happily
// accept its own hostname. A federation trusting it turns the exchange endpoint into
// a token-refresh loop: any platform-issued token can be re-exchanged for a fresh
// one with a new TTL, indefinitely, with no re-authentication and nothing to revoke.
var selfIssuerRule = validation.By(func(value any) error {
	raw, _ := value.(string)
	issuer := strings.TrimSpace(raw)
	if issuer == "" {
		return nil
	}
	self := strings.TrimSpace(config.AppPublicHostname)
	if self == "" {
		return nil
	}
	issuerHost, err := url.Parse(issuer)
	if err != nil {
		return nil // shape is reported by httpsURLRule
	}
	selfHost, err := url.Parse(self)
	if err != nil {
		return nil
	}
	if selfHost.Host != "" && strings.EqualFold(issuerHost.Host, selfHost.Host) {
		return validation.NewError("validation_issuer_url",
			"issuer_url must not be this authorization server: re-exchanging its own tokens "+
				"would let a token be refreshed indefinitely without re-authentication")
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

// maxAttributeMappingEntries bounds the claim mapping. Every exchange decodes it
// and inflates the issued JWT, and no legitimate federation needs many.
const maxAttributeMappingEntries = 16

// internalClaimNamePattern constrains a mapped DESTINATION claim name so it cannot
// collide with a claim a consumer resolves case-insensitively or by prefix.
var internalClaimNamePattern = regexp.MustCompile(`^[a-z][a-z0-9_]{0,63}$`)

// subjectPatternRule rejects a subject_pattern too broad to be a trust boundary.
//
// subject_pattern is the ONLY thing that distinguishes one workload from another on
// a shared issuer. Issuers like token.actions.githubusercontent.com are global —
// any GitHub user can obtain a token from them, and the `aud` value is chosen by
// the requesting workflow, so audience is a routing key rather than a boundary.
// A pattern of "*" (or one that starts with "*", leaving the org segment
// unanchored) therefore lets ANY workload on that issuer exchange its token for
// this tenant's access token, unauthenticated. A partially-wildcarded org segment
// like "repo:a*" is the same hole with extra steps.
var subjectPatternRule = validation.By(func(value any) error {
	raw, _ := value.(string)
	pattern := strings.TrimSpace(raw)
	if pattern == "" {
		return nil // Required is reported separately.
	}
	if !SubjectPatternTooBroad(pattern) {
		return nil
	}
	// Below: distinguish WHY it is too broad so the operator gets an actionable
	// message. SubjectPatternTooBroad is the single source of truth for the verdict;
	// the exchange path enforces the same rule at match time.
	if strings.HasPrefix(pattern, "*") || strings.HasPrefix(pattern, "?") {
		return validation.NewError("validation_subject_pattern",
			"subject_pattern must not start with a wildcard: it would match every workload from this issuer. "+
				"Anchor it on the organisation or namespace, e.g. \"repo:my-org/my-repo:*\"")
	}
	return validation.NewError("validation_subject_pattern",
		"subject_pattern is too broad to identify a workload: the wildcard must come after a whole "+
			"organisation or namespace segment, not part-way through one. \"repo:my-org/*\" is anchored; "+
			"\"repo:a*\" matches every organisation starting with \"a\"")
})

// attributeMappingRule validates the external-claim -> internal-claim map.
//
// The VALUES are destination claim names in the issued token, and ExtraClaims is
// merged LAST over the standard claim set — so a value of "sub", "client_id" or
// "svc" is a token-forgery primitive on an endpoint that takes no client
// credentials. The exchange path also drops reserved destinations at issuance;
// this rejects them at configuration time so an operator is told rather than left
// with a mapping that silently does nothing.
var attributeMappingRule = validation.By(func(value any) error {
	mapping, ok := value.(map[string]string)
	if !ok || len(mapping) == 0 {
		return nil
	}
	if len(mapping) > maxAttributeMappingEntries {
		return validation.NewError("validation_attribute_mapping",
			fmt.Sprintf("attribute_mapping must not contain more than %d entries", maxAttributeMappingEntries))
	}
	for externalClaim, internalClaim := range mapping {
		ext := strings.TrimSpace(externalClaim)
		internal := strings.TrimSpace(internalClaim)
		if ext == "" {
			return validation.NewError("validation_attribute_mapping",
				"attribute_mapping keys must not be empty")
		}
		if len(ext) > 128 {
			return validation.NewError("validation_attribute_mapping",
				"each attribute_mapping key must not exceed 128 characters")
		}
		if internal == "" {
			return validation.NewError("validation_attribute_mapping",
				"attribute_mapping values must not be empty: the value is the claim name to write")
		}
		if jwt.IsReservedClaim(internal) {
			return validation.NewError("validation_attribute_mapping",
				fmt.Sprintf("attribute_mapping must not write the reserved claim %q; it is set by the token issuer "+
					"and overriding it would forge the token's identity", internal))
		}
		if !internalClaimNamePattern.MatchString(internal) {
			return validation.NewError("validation_attribute_mapping",
				fmt.Sprintf("%q is not a valid claim name: use lowercase letters, digits and underscores, "+
					"starting with a letter", internal))
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
			validation.Length(0, 512).Error("issuer_url must not exceed 512 characters"),
			httpsURLRule,
			selfIssuerRule,
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
			subjectPatternRule,
		),
		validation.Field(&r.AllowedScopes, scopeListRule),
		validation.Field(&r.AttributeMapping, attributeMappingRule),
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
			validation.Length(0, 512).Error("issuer_url must not exceed 512 characters"),
			httpsURLRule,
			selfIssuerRule,
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
			subjectPatternRule,
		),
		validation.Field(&r.AllowedScopes, scopeListRule),
		validation.Field(&r.AttributeMapping, attributeMappingRule),
	)
}

// Validate validates the list filter.
func (f WorkloadIdentityFederationFilterDTO) Validate() error {
	return validation.ValidateStruct(&f,
		validation.Field(&f.PaginationRequestDTO),
	)
}
