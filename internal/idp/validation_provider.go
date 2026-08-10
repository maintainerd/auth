package idp

import (
	"encoding/json"
	"net/url"
	"regexp"
	"strings"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
)

// idpNamePattern constrains the provider name to a DNS/slug-safe form: lowercase
// letters, digits and hyphens, and it cannot start or end with a hyphen. This
// mirrors the frontend yup rule so backend and frontend agree on what a valid
// identity provider name is.
var idpNamePattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)

// idpScopeTokenPattern constrains each OAuth2/OIDC scope token to the RFC 6749
// scope-token character set plus the separators real-world providers use. It
// mirrors the frontend rule so backend and frontend reject the same rubbish.
var idpScopeTokenPattern = regexp.MustCompile(`^[A-Za-z0-9._:/-]+$`)

// oidcConfigAllowedKeys is the exhaustive allow-list of config JSON keys for
// OIDC and OAUTH2_ONLY (social/enterprise) providers. Any other key is rejected
// so we never persist arbitrary/rubbish structure.
var oidcConfigAllowedKeys = map[string]struct{}{
	"scopes":                 {},
	"attribute_mapping":      {},
	"authorization_endpoint": {},
	"token_endpoint":         {},
	"userinfo_endpoint":      {},
}

// samlConfigAllowedKeys is the exhaustive allow-list of config JSON keys for
// SAML providers.
var samlConfigAllowedKeys = map[string]struct{}{
	"entity_id":         {},
	"sso_url":           {},
	"slo_url":           {},
	"certificate":       {},
	"name_id_format":    {},
	"attribute_mapping": {},
}

// knownAttributeTargets is the set of IdentityMetadata fields the federation
// broker honors when it maps upstream claims/attributes (see extractMetadata in
// service_federation.go). attribute_mapping values outside this set are rejected.
var knownAttributeTargets = map[string]struct{}{
	"email":          {},
	"email_verified": {},
	"name":           {},
	"given_name":     {},
	"family_name":    {},
	"picture":        {},
	"locale":         {},
}

// requireHTTPSURL enforces that an external endpoint URL is transport-secure.
// This mirrors the frontend rule exactly: a URL is acceptable only if its scheme
// is https, EXCEPT http is allowed when the host is localhost or 127.0.0.1 (local
// development). An empty value is treated as valid here — presence/required-ness
// is enforced separately by the surrounding validators.
//
// Applied to every externally-dialed endpoint (issuer, authorization/token/
// userinfo endpoints, SAML sso_url/slo_url) so a mis-configured or malicious
// provider config can never downgrade federation traffic to cleartext http.
func requireHTTPSURL(value string) error {
	v := strings.TrimSpace(value)
	if v == "" {
		return nil
	}
	u, err := url.Parse(v)
	if err != nil || u.Host == "" {
		return validation.NewError("validation_url", "must be a valid URL")
	}
	switch strings.ToLower(u.Scheme) {
	case "https":
		return nil
	case "http":
		if host := u.Hostname(); host == "localhost" || host == "127.0.0.1" {
			return nil
		}
		return validation.NewError("validation_url_https", "must use https")
	default:
		return validation.NewError("validation_url_https", "must use https")
	}
}

// requireHTTPSURLRule adapts requireHTTPSURL to an ozzo validation.By rule so it
// can guard string fields (e.g. Issuer) directly.
func requireHTTPSURLRule(value interface{}) error {
	s, _ := value.(string)
	return requireHTTPSURL(s)
}

// providerAllowedHosts is the exact host allow-list for providers whose official
// OIDC issuer / OAuth2 endpoints live on fixed, well-known domains (verified
// against each provider's current documentation). It stops an operator from
// registering a known provider that points at an attacker-controlled host — e.g.
// a `github` provider whose token_endpoint is evil.com, which would leak the
// client secret on the first federated login, or a `google` provider whose
// issuer is a fake host.
//
// For OIDC providers the ISSUER is bound to this set; for OAuth2-only providers
// (github/facebook/twitter) the explicit endpoints are bound. Providers that are
// deliberately ABSENT are host-unrestricted because they legitimately use custom
// or self-managed domains — auth0 (custom domains), gitlab (self-managed) and
// external maintainerd (enterprise, any org) — and SAML (any IdP). Those remain
// constrained to https by requireHTTPSURL. Cognito uses the regex below because
// its host is regional.
var providerAllowedHosts = map[string][]string{
	shared.IDPProviderGoogle:    {"accounts.google.com"},
	shared.IDPProviderLinkedIn:  {"www.linkedin.com"},
	shared.IDPProviderMicrosoft: {"login.microsoftonline.com", "login.microsoftonline.us", "login.partner.microsoftonline.cn"},
	shared.IDPProviderGitHub:    {"github.com", "api.github.com"},
	shared.IDPProviderFacebook:  {"facebook.com", "www.facebook.com", "graph.facebook.com"},
	shared.IDPProviderTwitter:   {"x.com", "twitter.com", "api.x.com", "api.twitter.com"},
}

// cognitoIssuerHostPattern matches AWS Cognito's regional issuer hosts, including
// the newer issuer-cognito-idp resilience prefix and the China amazonaws.com.cn
// suffix. The user-pool id lives in the issuer path and is not host-checked here.
var cognitoIssuerHostPattern = regexp.MustCompile(`^(issuer-)?cognito-idp\.[a-z0-9-]+\.amazonaws\.com(\.cn)?$`)

// requireAllowedHost enforces that a URL's host belongs to the provider's set of
// official domains. It uses EXACT host equality (never suffix-contains, which is
// bypassable by evilgithub.com / github.com.evil.com). url.Hostname() strips the
// port and any embedded user:pass@ credentials, so https://api.github.com@evil.com
// resolves to host evil.com and is correctly rejected. An empty value and any
// provider not in the allow-list (or cognito regex) are no-ops.
func requireAllowedHost(provider, value string) error {
	v := strings.TrimSpace(value)
	if v == "" {
		return nil
	}
	u, err := url.Parse(v)
	if err != nil || u.Host == "" {
		return validation.NewError("validation_url", "must be a valid URL")
	}
	host := strings.ToLower(u.Hostname())

	if provider == shared.IDPProviderCognito {
		if !cognitoIssuerHostPattern.MatchString(host) {
			return validation.NewError("validation_provider_host", "host \""+host+"\" is not a valid AWS Cognito domain")
		}
		return nil
	}

	allowed, restricted := providerAllowedHosts[provider]
	if !restricted {
		return nil
	}
	for _, h := range allowed {
		if host == h {
			return nil
		}
	}
	return validation.NewError("validation_provider_host", "host \""+host+"\" is not permitted for the "+provider+" provider")
}

// isExternalProviderType reports whether a provider type needs upstream OIDC
// credentials (issuer + provider_client_id). System providers do not.
func isExternalProviderType(providerType string) bool {
	return providerType == shared.IDPTypeSocial || providerType == shared.IDPTypeEnterprise
}

// isSAMLProviderType reports whether the provider type is SAML.
func isSAMLProviderType(providerType string) bool {
	return providerType == shared.IDPTypeSAML
}

// isOAuth2OnlyProvider reports whether the provider is an OAuth2-only provider
// that has NO OIDC discovery document. These providers cannot derive endpoints
// from an issuer, so they must ship explicit authorization/token/userinfo
// endpoints in their config instead of an issuer. The set is defined once in the
// provider registry (provider_profiles.go).
func isOAuth2OnlyProvider(provider string) bool {
	return profileFor(provider).oauth2Only
}

// attribute_mapping maps between our known IdentityMetadata targets and the
// upstream provider's claim/attribute names — but the DIRECTION differs by
// provider family, matching how the broker consumes it at runtime:
//   - OIDC (extractMetadata): mapping[target] = upstreamClaim, so the KEYS are
//     our known targets and the values are arbitrary upstream claim names.
//   - SAML (extractSAMLClaims): mapping[samlAttr] = target, so the VALUES are
//     our known targets and the keys are arbitrary SAML attribute names.
// Each validator checks the side that must be a known target; the other side is
// a free-form upstream name. Empty mapping is a no-op.

// validateOIDCAttributeMapping validates an OIDC attribute_mapping (keys are targets).
func validateOIDCAttributeMapping(mapping map[string]string) error {
	for target := range mapping {
		if _, ok := knownAttributeTargets[target]; !ok {
			return validation.NewError("validation_attribute_mapping",
				"config.attribute_mapping references an unknown target attribute: "+target)
		}
	}
	return nil
}

// validateSAMLAttributeMapping validates a SAML attribute_mapping (values are targets).
func validateSAMLAttributeMapping(mapping map[string]string) error {
	for _, target := range mapping {
		if _, ok := knownAttributeTargets[target]; !ok {
			return validation.NewError("validation_attribute_mapping",
				"config.attribute_mapping references an unknown target attribute: "+target)
		}
	}
	return nil
}

// validateExternalProviderConfig validates the config JSONB for OIDC and
// OAUTH2_ONLY (social/enterprise) providers. It enforces the key allow-list,
// scope-token format, attribute-mapping targets and endpoint URL validity. When
// requireEndpoints is true (an active OAuth2-only provider), all three endpoints
// must be present because those providers have no OIDC discovery to fall back on.
func validateExternalProviderConfig(provider string, requireEndpoints bool) validation.RuleFunc {
	oauth2Only := isOAuth2OnlyProvider(provider)
	return func(value interface{}) error {
		raw, err := json.Marshal(value)
		if err != nil {
			return validation.NewError("validation_oidc_config", "config must be valid JSON")
		}
		if len(raw) == 0 || string(raw) == "null" {
			if requireEndpoints {
				return validation.NewError("validation_oidc_config",
					"config.authorization_endpoint, config.token_endpoint and config.userinfo_endpoint are required for active github/facebook/twitter providers")
			}
			return nil
		}

		// Allow-list: reject any key we do not explicitly permit. Unmarshaling into
		// a map catches unknown keys that a typed struct would silently drop.
		var keys map[string]json.RawMessage
		if err := json.Unmarshal(raw, &keys); err != nil {
			return validation.NewError("validation_oidc_config", "config must be a JSON object")
		}
		for k := range keys {
			if _, ok := oidcConfigAllowedKeys[k]; !ok {
				return validation.NewError("validation_oidc_config", "unknown configuration key: "+k)
			}
		}

		// OIDC providers (google/microsoft/cognito/auth0/gitlab/linkedin/maintainerd-
		// enterprise) MUST derive their authorization/token/userinfo endpoints from
		// OIDC discovery — the runtime prefers any config override (see
		// resolveTokenEndpoint), so accepting an operator-supplied token_endpoint would
		// let it be pointed at an attacker host and exfiltrate the upstream client
		// secret. Reject endpoint overrides outright; only OAuth2-only providers
		// (github/facebook/twitter), which have no discovery, may ship explicit
		// endpoints (host-bound below). OIDC config reduces to scopes + attribute_mapping.
		if !oauth2Only {
			for _, epKey := range []string{"authorization_endpoint", "token_endpoint", "userinfo_endpoint"} {
				if _, present := keys[epKey]; present {
					return validation.NewError("validation_oidc_config",
						"endpoint overrides are not permitted for the "+provider+" provider; endpoints are obtained automatically via OIDC discovery")
				}
			}
		}

		// scopes: JSON array of strings, each matching the scope-token pattern.
		if rawScopes, ok := keys["scopes"]; ok {
			var scopes []string
			if err := json.Unmarshal(rawScopes, &scopes); err != nil {
				return validation.NewError("validation_oidc_config", "config.scopes must be an array of strings")
			}
			for _, s := range scopes {
				if !idpScopeTokenPattern.MatchString(s) {
					return validation.NewError("validation_oidc_config", "config.scopes contains an invalid scope token: "+s)
				}
			}
		}

		// attribute_mapping: object of string→string whose values are known targets.
		if rawMapping, ok := keys["attribute_mapping"]; ok {
			var mapping map[string]string
			if err := json.Unmarshal(rawMapping, &mapping); err != nil {
				return validation.NewError("validation_oidc_config", "config.attribute_mapping must be an object of string values")
			}
			if err := validateOIDCAttributeMapping(mapping); err != nil {
				return err
			}
		}

		var cfg OIDCProviderConfig
		if err := json.Unmarshal(raw, &cfg); err != nil {
			return validation.NewError("validation_oidc_config", "config must be valid JSON")
		}
		endpoints := []struct {
			field string
			value string
		}{
			{"authorization_endpoint", cfg.AuthorizationEndpoint},
			{"token_endpoint", cfg.TokenEndpoint},
			{"userinfo_endpoint", cfg.UserinfoEndpoint},
		}
		for _, ep := range endpoints {
			if strings.TrimSpace(ep.value) == "" {
				continue
			}
			if err := requireHTTPSURL(ep.value); err != nil {
				return validation.NewError("validation_oidc_config", "config."+ep.field+" must be a valid https URL (http is allowed only for localhost)")
			}
			// Only OAuth2-only providers reach here with endpoints set (OIDC overrides
			// were rejected above). They POST the client secret to their token endpoint
			// and have fixed official hosts, so bind every endpoint to the provider's
			// domain — a bogus host would exfiltrate the secret.
			if oauth2Only {
				if err := requireAllowedHost(provider, ep.value); err != nil {
					return validation.NewError("validation_provider_host", "config."+ep.field+": "+err.Error())
				}
			}
		}
		if requireEndpoints {
			for _, ep := range endpoints {
				if strings.TrimSpace(ep.value) == "" {
					return validation.NewError("validation_oidc_config",
						"config."+ep.field+" is required for active github/facebook/twitter providers")
				}
			}
		}
		return nil
	}
}

// validateSAMLConfig validates the config JSONB for SAML providers. It enforces
// the key allow-list, URL validity for sso_url/slo_url, attribute-mapping targets
// and that any certificate parses as PEM X.509. When active is true the required
// fields (entity_id, sso_url, certificate) must be present.
func validateSAMLConfig(active bool) validation.RuleFunc {
	return func(value interface{}) error {
		raw, err := json.Marshal(value)
		if err != nil {
			return validation.NewError("validation_saml_config", "config must be valid JSON")
		}
		if len(raw) == 0 || string(raw) == "null" {
			if active {
				return validation.NewError("validation_saml_config", "config is required for active SAML providers")
			}
			return nil
		}

		var keys map[string]json.RawMessage
		if err := json.Unmarshal(raw, &keys); err != nil {
			return validation.NewError("validation_saml_config", "config must be a JSON object")
		}
		for k := range keys {
			if _, ok := samlConfigAllowedKeys[k]; !ok {
				return validation.NewError("validation_saml_config", "unknown configuration key: "+k)
			}
		}

		var cfg SAMLProviderConfig
		if err := json.Unmarshal(raw, &cfg); err != nil {
			return validation.NewError("validation_saml_config", "config must be valid JSON")
		}
		if strings.TrimSpace(cfg.SSOURL) != "" {
			if err := requireHTTPSURL(cfg.SSOURL); err != nil {
				return validation.NewError("validation_saml_config", "config.sso_url must be a valid https URL (http is allowed only for localhost)")
			}
		}
		if strings.TrimSpace(cfg.SLOURL) != "" {
			if err := requireHTTPSURL(cfg.SLOURL); err != nil {
				return validation.NewError("validation_saml_config", "config.slo_url must be a valid https URL (http is allowed only for localhost)")
			}
		}
		if err := validateSAMLAttributeMapping(cfg.AttributeMapping); err != nil {
			return err
		}
		if strings.TrimSpace(cfg.Certificate) != "" {
			if _, err := ParsePEMCertExpiry(cfg.Certificate); err != nil {
				return validation.NewError("validation_saml_config", "config.certificate must be a valid PEM X.509 certificate")
			}
		}
		if active {
			if cfg.EntityID == "" {
				return validation.NewError("validation_saml_config", "config.entity_id is required for active SAML providers")
			}
			if cfg.SSOURL == "" {
				return validation.NewError("validation_saml_config", "config.sso_url is required for active SAML providers")
			}
			if cfg.Certificate == "" {
				return validation.NewError("validation_saml_config", "config.certificate is required for active SAML providers")
			}
		}
		return nil
	}
}

// validateProviderTypeConsistency returns a rule that enforces the correct
// Provider ↔ ProviderType pairing. The maintainerd provider may be either the
// built-in 'system' provider OR an 'enterprise' provider — the latter is how an
// external, self-hosted Maintainerd instance is registered for generic OIDC
// federation; 'social'/'saml' are never valid for maintainerd. 'system' remains
// exclusive to the built-in maintainerd provider (no other provider may claim
// it), and the saml provider must use the saml provider type.
func validateProviderTypeConsistency(provider string) validation.RuleFunc {
	return func(value interface{}) error {
		providerType, _ := value.(string)
		switch {
		case provider == shared.IDPProviderMaintainerd &&
			providerType != shared.IDPTypeSystem && providerType != shared.IDPTypeEnterprise:
			return validation.NewError("validation_provider_type", "Provider type must be 'system' (built-in) or 'enterprise' (external federation) for the maintainerd provider")
		case providerType == shared.IDPTypeSystem && provider != shared.IDPProviderMaintainerd:
			return validation.NewError("validation_provider_type", "Provider type 'system' is only valid for the maintainerd provider")
		case provider == shared.IDPProviderSAML && providerType != shared.IDPTypeSAML:
			return validation.NewError("validation_provider_type", "Provider type must be 'saml' for the saml provider")
		}
		return nil
	}
}

// tokenFederationRejectedForOAuth2Only errors when token federation is enabled
// on an OAuth2-only provider. Those providers (github/facebook/twitter) issue no
// OIDC ID token and expose no issuer, so federating foreign OIDC tokens "from
// this issuer" is meaningless — rejecting it here gives a clear message instead
// of the confusing "Issuer is required".
func tokenFederationRejectedForOAuth2Only(oauth2Only, enabled bool) validation.RuleFunc {
	return func(interface{}) error {
		if oauth2Only && enabled {
			return validation.NewError("validation_token_federation",
				"Token federation is not available for GitHub, Facebook or X — these providers issue no OIDC ID token")
		}
		return nil
	}
}

// Validation for create request
func (r IdentityProviderCreateRequestDTO) Validate() error {
	requireExternalCreds := isExternalProviderType(r.ProviderType) && r.Status == shared.StatusActive
	requireTokenFederation := r.AllowTokenFederation && r.Status == shared.StatusActive
	oauth2Only := isOAuth2OnlyProvider(r.Provider)
	// OAuth2-only providers (github/facebook/twitter) have no discovery, so they
	// require explicit endpoints instead of an issuer. OIDC providers still
	// require an issuer. Token federation always requires an issuer.
	requireOAuth2Endpoints := requireExternalCreds && oauth2Only
	requireIssuer := (requireExternalCreds && !oauth2Only) || requireTokenFederation
	return validation.ValidateStruct(&r,
		validation.Field(&r.Name,
			validation.Required.Error("Name is required"),
			validation.Length(3, 50).Error("Name must be between 3 and 50 characters"),
			validation.Match(idpNamePattern).Error("Name must contain only lowercase letters, numbers, and hyphens, and cannot start or end with a hyphen"),
		),
		validation.Field(&r.DisplayName,
			validation.Required.Error("Display name is required"),
			validation.Length(2, 100).Error("Display name must be between 2 and 100 characters"),
		),
		validation.Field(&r.Provider,
			validation.Required.Error("Provider is required"),
			validation.In(shared.IDPProviderMaintainerd, shared.IDPProviderCognito, shared.IDPProviderAuth0, shared.IDPProviderGoogle, shared.IDPProviderFacebook, shared.IDPProviderGitHub, shared.IDPProviderGitLab, shared.IDPProviderMicrosoft, shared.IDPProviderLinkedIn, shared.IDPProviderTwitter, shared.IDPProviderSAML).Error("Provider must be one of: maintainerd, cognito, auth0, google, facebook, github, gitlab, microsoft, linkedin, twitter, saml"),
		),
		validation.Field(&r.ProviderType,
			validation.Required.Error("Provider type is required"),
			validation.In(shared.IDPTypeSystem, shared.IDPTypeSocial, shared.IDPTypeEnterprise, shared.IDPTypeSAML).Error("Provider type must be one of: system, social, enterprise, saml"),
			validation.By(validateProviderTypeConsistency(r.Provider)),
		),
		validation.Field(&r.Config,
			validation.When(isSAMLProviderType(r.ProviderType), validation.By(validateSAMLConfig(r.Status == shared.StatusActive))),
			validation.When(isExternalProviderType(r.ProviderType), validation.By(validateExternalProviderConfig(r.Provider, requireOAuth2Endpoints))),
		),
		validation.Field(&r.Issuer,
			validation.When(requireIssuer, validation.Required.Error("Issuer is required for active OIDC social/enterprise or token-federation providers")),
			validation.When(strings.TrimSpace(r.Issuer) != "", validation.By(requireHTTPSURLRule)),
			// Bind the issuer to the provider's official domain(s) for fixed-domain
			// providers (google/microsoft/cognito/linkedin) so a known provider
			// can't be pointed at a fake issuer. Variable-domain providers (auth0
			// custom domains, gitlab self-managed, external maintainerd) are no-ops.
			validation.When(strings.TrimSpace(r.Issuer) != "", validation.By(func(interface{}) error { return requireAllowedHost(r.Provider, r.Issuer) })),
		),
		validation.Field(&r.ProviderClientID,
			validation.When(requireExternalCreds, validation.Required.Error("Provider client ID is required for active social/enterprise providers")),
		),
		validation.Field(&r.ProviderClientSecret,
			// On create, an active external (social/enterprise) provider must ship
			// a client secret. On update the secret is write-only (blank keeps the
			// stored value), so it is only required in the create path.
			validation.When(requireExternalCreds, validation.Required.Error("Provider client secret is required for active social/enterprise providers")),
		),
		validation.Field(&r.AllowTokenFederation,
			validation.By(tokenFederationRejectedForOAuth2Only(oauth2Only, r.AllowTokenFederation)),
		),
		validation.Field(&r.AllowedAudiences,
			validation.When(requireTokenFederation,
				validation.By(func(value interface{}) error {
					auds, ok := value.([]string)
					if !ok || len(auds) == 0 {
						return validation.NewError("validation_required", "At least one allowed audience is required when token federation is enabled")
					}
					return nil
				}),
			),
		),
		validation.Field(&r.EmailDomains,
			validation.When(len(r.EmailDomains) > 0,
				validation.Each(is.Domain.Error("Each email domain must be a valid domain")),
			),
		),
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			validation.In(shared.StatusActive, shared.StatusInactive).Error("Status must be either 'active' or 'inactive'"),
		),
	)
}

// Validation for update request
func (r IdentityProviderUpdateRequestDTO) Validate() error {
	requireExternalCreds := isExternalProviderType(r.ProviderType) && r.Status == shared.StatusActive
	requireTokenFederation := r.AllowTokenFederation && r.Status == shared.StatusActive
	oauth2Only := isOAuth2OnlyProvider(r.Provider)
	requireOAuth2Endpoints := requireExternalCreds && oauth2Only
	requireIssuer := (requireExternalCreds && !oauth2Only) || requireTokenFederation
	return validation.ValidateStruct(&r,
		validation.Field(&r.Name,
			validation.Required.Error("Name is required"),
			validation.Length(3, 50).Error("Name must be between 3 and 50 characters"),
			validation.Match(idpNamePattern).Error("Name must contain only lowercase letters, numbers, and hyphens, and cannot start or end with a hyphen"),
		),
		validation.Field(&r.DisplayName,
			validation.Required.Error("Display name is required"),
			validation.Length(2, 100).Error("Display name must be between 2 and 100 characters"),
		),
		validation.Field(&r.Provider,
			validation.Required.Error("Provider is required"),
			validation.In(shared.IDPProviderMaintainerd, shared.IDPProviderCognito, shared.IDPProviderAuth0, shared.IDPProviderGoogle, shared.IDPProviderFacebook, shared.IDPProviderGitHub, shared.IDPProviderGitLab, shared.IDPProviderMicrosoft, shared.IDPProviderLinkedIn, shared.IDPProviderTwitter, shared.IDPProviderSAML).Error("Provider must be one of: maintainerd, cognito, auth0, google, facebook, github, gitlab, microsoft, linkedin, twitter, saml"),
		),
		validation.Field(&r.ProviderType,
			validation.Required.Error("Provider type is required"),
			validation.In(shared.IDPTypeSystem, shared.IDPTypeSocial, shared.IDPTypeEnterprise, shared.IDPTypeSAML).Error("Provider type must be one of: system, social, enterprise, saml"),
			validation.By(validateProviderTypeConsistency(r.Provider)),
		),
		validation.Field(&r.Config,
			validation.When(isSAMLProviderType(r.ProviderType), validation.By(validateSAMLConfig(r.Status == shared.StatusActive))),
			validation.When(isExternalProviderType(r.ProviderType), validation.By(validateExternalProviderConfig(r.Provider, requireOAuth2Endpoints))),
		),
		validation.Field(&r.Issuer,
			validation.When(requireIssuer, validation.Required.Error("Issuer is required for active OIDC social/enterprise or token-federation providers")),
			validation.When(strings.TrimSpace(r.Issuer) != "", validation.By(requireHTTPSURLRule)),
			// Bind the issuer to the provider's official domain(s) for fixed-domain
			// providers (google/microsoft/cognito/linkedin) so a known provider
			// can't be pointed at a fake issuer. Variable-domain providers (auth0
			// custom domains, gitlab self-managed, external maintainerd) are no-ops.
			validation.When(strings.TrimSpace(r.Issuer) != "", validation.By(func(interface{}) error { return requireAllowedHost(r.Provider, r.Issuer) })),
		),
		validation.Field(&r.ProviderClientID,
			validation.When(requireExternalCreds, validation.Required.Error("Provider client ID is required for active social/enterprise providers")),
		),
		validation.Field(&r.AllowTokenFederation,
			validation.By(tokenFederationRejectedForOAuth2Only(oauth2Only, r.AllowTokenFederation)),
		),
		validation.Field(&r.AllowedAudiences,
			validation.When(requireTokenFederation,
				validation.By(func(value interface{}) error {
					auds, ok := value.([]string)
					if !ok || len(auds) == 0 {
						return validation.NewError("validation_required", "At least one allowed audience is required when token federation is enabled")
					}
					return nil
				}),
			),
		),
		validation.Field(&r.EmailDomains,
			validation.When(len(r.EmailDomains) > 0,
				validation.Each(is.Domain.Error("Each email domain must be a valid domain")),
			),
		),
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			validation.In(shared.StatusActive, shared.StatusInactive).Error("Status must be either 'active' or 'inactive'"),
		),
	)
}

func (r IdentityProviderStatusUpdateDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			validation.In(shared.StatusActive, shared.StatusInactive).Error("Status must be either 'active' or 'inactive'"),
		),
	)
}

// Validate validates the identity provider filter DTO.
func (f IdentityProviderFilterDTO) Validate() error {
	return validation.ValidateStruct(&f,
		validation.Field(&f.Provider,
			validation.When(len(f.Provider) > 0,
				validation.Each(validation.In(shared.IDPProviderMaintainerd, shared.IDPProviderCognito, shared.IDPProviderAuth0, shared.IDPProviderGoogle, shared.IDPProviderFacebook, shared.IDPProviderGitHub, shared.IDPProviderGitLab, shared.IDPProviderMicrosoft, shared.IDPProviderLinkedIn, shared.IDPProviderTwitter, shared.IDPProviderSAML).Error("Invalid identity provider")),
			),
		),
		validation.Field(&f.ProviderType,
			validation.When(f.ProviderType != nil,
				validation.In(shared.IDPTypeSystem, shared.IDPTypeSocial, shared.IDPTypeEnterprise, shared.IDPTypeSAML).Error("Provider type must be one of: system, social, enterprise, saml"),
			),
		),
		validation.Field(&f.Status,
			validation.When(len(f.Status) > 0,
				validation.Each(validation.In(shared.StatusActive, shared.StatusInactive).Error("Status must be 'active' or 'inactive'")),
			),
		),
		validation.Field(&f.PaginationRequestDTO),
	)
}
