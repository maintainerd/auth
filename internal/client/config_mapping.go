package client

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/lib/pq"
	"gorm.io/datatypes"
)

// validTokenEndpointAuthMethods is the set of client authentication methods the
// token endpoint actually IMPLEMENTS. Anything outside it is ignored, so a
// malformed config cannot silently weaken client authentication.
//
// The two mutual-TLS methods (tls_client_auth, self_signed_tls_client_auth) are
// deliberately absent even though the column's CHECK constraint still permits
// them. The token endpoint has no certificate-binding implementation and answers
// them with "not supported", so accepting them here let an operator configure a
// client into a state where it authenticates successfully in the console and
// then cannot obtain a token at all — a misconfiguration only discoverable at
// the moment the client first tries to work.
//
// This is not a gap in sender-constraining. A certificate-bound token is
// enforced at the point of USE on the control plane (RFC 8705 §3, see
// enforceGRPCCertBinding), which is where the property matters; the client still
// authenticates with private_key_jwt, which is stronger than a shared secret and
// needs no proxy to forward a certificate.
var validTokenEndpointAuthMethods = map[string]bool{
	TokenAuthMethodSecretBasic:     true,
	TokenAuthMethodSecretPost:      true,
	TokenAuthMethodNone:            true,
	TokenAuthMethodPrivateKeyJWT:   true,
	TokenAuthMethodClientSecretJWT: true,
}

// validRequiredACRs is the set of ACR (authentication context class) values a
// client may demand. "1" = password/single-factor, "2" = step-up/MFA. Anything
// outside this set is ignored so a malformed config cannot weaken enforcement.
var validRequiredACRs = map[string]bool{"1": true, "2": true}

// applyConfigToClientColumns mirrors the OAuth and security settings carried in
// the free-form `config` JSON into the first-class client columns the runtime
// actually reads — grant_types, response_types, token_endpoint_auth_method,
// allowed_scopes, require_consent, require_pkce, the access/refresh token TTLs,
// and the per-client security overrides (required_acr, session timeouts).
//
// The admin console persists these settings inside `config`, but the authorization,
// token-issuance, login, and session paths read them from the dedicated columns.
// Without this mapping, anything configured in the console would live only in the
// opaque config blob and never take effect at runtime.
//
// Security-override keys (required_acr, session_idle_timeout, session_absolute_timeout)
// are NULL-when-absent so the tenant security_settings default is inherited.
// Capability/credential policy (allowed MFA methods, password, lockout, threat,
// registration) is deliberately NOT mapped here — it stays tenant-level.
//
// Keys without a dedicated column (cors_enabled, refresh_token_rotation,
// multi_resource_refresh_token, and operator metadata) are intentionally left in
// `config` untouched.
func applyConfigToClientColumns(c *Client, config datatypes.JSON) {
	if c == nil || len(config) == 0 {
		return
	}

	var raw map[string]any
	if err := json.Unmarshal(config, &raw); err != nil {
		return
	}

	if v, ok := stringSliceFromConfig(raw["grant_types"]); ok {
		c.GrantTypes = pq.StringArray(v)
	}
	if v, ok := stringSliceFromConfig(raw["response_types"]); ok {
		c.ResponseTypes = pq.StringArray(v)
	}
	if v, ok := stringSliceFromConfig(raw["allowed_scopes"]); ok {
		c.AllowedScopes = pq.StringArray(v)
	}
	if v, ok := raw["token_endpoint_auth_method"].(string); ok {
		v = strings.TrimSpace(v)
		if validTokenEndpointAuthMethods[v] {
			c.TokenEndpointAuthMethod = v
		}
	}
	if v, ok := boolFromConfig(firstPresentConfigValue(raw, "require_consent", "consent_required")); ok {
		c.RequireConsent = &v
	}
	if v, ok := boolFromConfig(firstPresentConfigValue(raw, "require_pkce", "pkce_required")); ok {
		c.RequirePKCE = &v
	}
	// The nullable override columns below are CLEARED when their key is absent.
	// `config` is replaced wholesale on update, so a key the operator removed is
	// genuinely gone from the stored blob; leaving the column set would strand an
	// override the console no longer displays and make "revert to inherit"
	// impossible. The NOT NULL columns above keep leave-unchanged semantics because
	// they have DB defaults and clearing them would break a working client.
	if v, ok := intFromConfig(firstPresentConfigValue(raw, "access_token_lifetime", "access_token_ttl")); ok && v > 0 {
		c.AccessTokenTTL = &v
	} else {
		c.AccessTokenTTL = nil
	}
	if v, ok := intFromConfig(firstPresentConfigValue(raw, "refresh_token_lifetime", "refresh_token_ttl")); ok && v > 0 {
		c.RefreshTokenTTL = &v
	} else {
		c.RefreshTokenTTL = nil
	}

	// Per-client security overrides — NULL/absent means inherit the tenant default.
	// An out-of-range value clears instead of persisting: chk_clients_required_acr
	// would reject it anyway, and silently keeping the previous value would report
	// success for a setting the operator never asked for.
	if v, ok := raw["required_acr"].(string); ok && validRequiredACRs[strings.TrimSpace(v)] {
		acr := strings.TrimSpace(v)
		c.RequiredACR = &acr
	} else {
		c.RequiredACR = nil
	}
	if v, ok := intFromConfig(firstPresentConfigValue(raw, "session_idle_timeout", "session_idle_timeout_seconds")); ok && v > 0 {
		c.SessionIdleTimeout = &v
	} else {
		c.SessionIdleTimeout = nil
	}
	if v, ok := intFromConfig(firstPresentConfigValue(raw, "session_absolute_timeout", "session_absolute_timeout_seconds")); ok && v > 0 {
		c.SessionAbsoluteTimeout = &v
	} else {
		c.SessionAbsoluteTimeout = nil
	}

	applyAdvancedConfigToClientColumns(c, raw)
}

// applyAdvancedConfigToClientColumns mirrors the client's public keys, mTLS
// binding and claim-mapping settings from config into their columns.
//
// These columns are read at runtime — authenticatePrivateKeyJWT verifies the
// client assertion against jwks/jwks_uri, and token issuance reads
// scope_claim_mappings and claim_mappers — but nothing wrote them, so a
// private_key_jwt client could be created and then never authenticate. `config`
// is the console's existing write channel for everything the runtime reads from a
// column, so these keys join the same mapping rather than getting new DTO fields.
//
// Shape validation lives in validateAdvancedClientConfig so an operator gets a
// 422 rather than a silently ignored key; anything that reaches here and is still
// unusable is dropped instead of persisted.
func applyAdvancedConfigToClientColumns(c *Client, raw map[string]any) {
	// RFC 7591 §2: jwks and jwks_uri MUST NOT both be present. The validator
	// rejects that; this keeps the columns consistent if it is ever bypassed — an
	// inline JWKS wins because it needs no network fetch to be usable.
	if v, ok := jsonObjectFromConfig(raw["jwks"]); ok {
		c.JWKS = v
		c.JWKSUri = nil
	} else {
		c.JWKS = nil
		if uri, ok := nonEmptyStringFromConfig(raw["jwks_uri"]); ok {
			c.JWKSUri = &uri
		} else {
			c.JWKSUri = nil
		}
	}

	if v, ok := nonEmptyStringFromConfig(raw["mtls_bound_cert_thumbprint"]); ok {
		c.MTLSBoundCertThumbprint = &v
	} else {
		c.MTLSBoundCertThumbprint = nil
	}

	if v, ok := jsonObjectFromConfig(raw["scope_claim_mappings"]); ok {
		c.ScopeClaimMappings = v
	} else {
		c.ScopeClaimMappings = nil
	}
	if v, ok := jsonObjectFromConfig(raw["claim_mappers"]); ok {
		c.ClaimMappers = v
	} else {
		c.ClaimMappers = nil
	}
}

// jsonObjectFromConfig re-marshals a decoded JSON object back into a jsonb value.
// An empty object counts as absent so clearing the field in the console results in
// NULL rather than a `{}` that reads as "configured but empty".
func jsonObjectFromConfig(v any) (datatypes.JSON, bool) {
	obj, ok := v.(map[string]any)
	if !ok || len(obj) == 0 {
		return nil, false
	}
	encoded, err := json.Marshal(obj)
	if err != nil {
		return nil, false
	}
	return datatypes.JSON(encoded), true
}

func nonEmptyStringFromConfig(v any) (string, bool) {
	s, ok := v.(string)
	if !ok {
		return "", false
	}
	if s = strings.TrimSpace(s); s == "" {
		return "", false
	}
	return s, true
}

// firstPresentConfigValue returns the value of the first key present in raw,
// allowing several accepted aliases for the same setting.
func firstPresentConfigValue(raw map[string]any, keys ...string) any {
	for _, k := range keys {
		if v, ok := raw[k]; ok {
			return v
		}
	}
	return nil
}

// stringSliceFromConfig coerces a JSON array (or already-typed slice) into a
// trimmed, non-empty []string. The second return reports whether the key was a
// recognizable slice at all, so an absent key leaves the column unchanged.
func stringSliceFromConfig(v any) ([]string, bool) {
	switch t := v.(type) {
	case []any:
		out := make([]string, 0, len(t))
		for _, item := range t {
			if s, ok := item.(string); ok {
				if s = strings.TrimSpace(s); s != "" {
					out = append(out, s)
				}
			}
		}
		return out, true
	case []string:
		out := make([]string, 0, len(t))
		for _, s := range t {
			if s = strings.TrimSpace(s); s != "" {
				out = append(out, s)
			}
		}
		return out, true
	default:
		return nil, false
	}
}

func boolFromConfig(v any) (bool, bool) {
	switch t := v.(type) {
	case bool:
		return t, true
	case string:
		switch strings.ToLower(strings.TrimSpace(t)) {
		case "true":
			return true, true
		case "false":
			return false, true
		}
	}
	return false, false
}

func intFromConfig(v any) (int, bool) {
	switch t := v.(type) {
	case float64:
		return int(t), true
	case int:
		return t, true
	case int64:
		return int(t), true
	case string:
		if n, err := strconv.Atoi(strings.TrimSpace(t)); err == nil {
			return n, true
		}
	}
	return 0, false
}

// mirroredConfigKeys are the config keys applyConfigToClientColumns copies into a
// column, including the legacy aliases the mapper still accepts. effectiveClientConfig
// strips all of them before re-adding the canonical spelling from the column, so a
// blob can never present two spellings of one setting with different values.
var mirroredConfigKeys = []string{
	"grant_types", "response_types", "allowed_scopes", "token_endpoint_auth_method",
	"require_consent", "consent_required", "require_pkce", "pkce_required",
	"access_token_lifetime", "access_token_ttl", "refresh_token_lifetime", "refresh_token_ttl",
	"required_acr",
	"session_idle_timeout", "session_idle_timeout_seconds",
	"session_absolute_timeout", "session_absolute_timeout_seconds",
	"jwks", "jwks_uri", "mtls_bound_cert_thumbprint",
	"scope_claim_mappings", "claim_mappers",
}

// effectiveClientConfig returns the client's config blob with every mirrored
// setting replaced by the value in its column — i.e. what the runtime actually
// enforces, not what was last written to the blob.
//
// The columns are authoritative: the seeder, the gRPC path and any direct SQL can
// change a column without touching `config`. Serving the raw blob made the console
// display stale values and, worse, round-trip them back on save — silently
// reverting a column and (now that absent nullable keys clear) revoking settings
// the operator never saw. A column that is NULL/empty is reported as an ABSENT key,
// which is how "inherit the tenant default" is expressed.
//
// Keys with no column (cors_enabled, refresh_token_rotation,
// multi_resource_refresh_token, `custom` metadata) pass through untouched.
func effectiveClientConfig(c *Client) datatypes.JSON {
	if c == nil {
		return nil
	}

	raw := map[string]any{}
	if len(c.Config) > 0 {
		if err := json.Unmarshal(c.Config, &raw); err != nil {
			// A blob that cannot be decoded has no trustworthy passthrough keys;
			// report the columns alone rather than failing the read.
			raw = map[string]any{}
		}
	}

	for _, key := range mirroredConfigKeys {
		delete(raw, key)
	}

	if len(c.GrantTypes) > 0 {
		raw["grant_types"] = []string(c.GrantTypes)
	}
	if len(c.ResponseTypes) > 0 {
		raw["response_types"] = []string(c.ResponseTypes)
	}
	// An empty allowlist is meaningful ("every scope"), so it is emitted as [] rather
	// than omitted — omitting it would read as "not configured yet".
	// A nil pq.StringArray marshals to null, which consumers read as absent, so
	// normalize it to an empty array.
	scopes := []string(c.AllowedScopes)
	if scopes == nil {
		scopes = []string{}
	}
	raw["allowed_scopes"] = scopes
	if c.TokenEndpointAuthMethod != "" {
		raw["token_endpoint_auth_method"] = c.TokenEndpointAuthMethod
	}
	// NOT NULL columns with a DB default: nil means "the default applies", so report
	// the default rather than omitting the key and letting the client guess.
	raw["require_consent"] = c.RequireConsent == nil || *c.RequireConsent
	raw["require_pkce"] = c.RequirePKCE == nil || *c.RequirePKCE

	if c.AccessTokenTTL != nil {
		raw["access_token_lifetime"] = *c.AccessTokenTTL
	}
	if c.RefreshTokenTTL != nil {
		raw["refresh_token_lifetime"] = *c.RefreshTokenTTL
	}
	if c.RequiredACR != nil {
		raw["required_acr"] = *c.RequiredACR
	}
	if c.SessionIdleTimeout != nil {
		raw["session_idle_timeout"] = *c.SessionIdleTimeout
	}
	if c.SessionAbsoluteTimeout != nil {
		raw["session_absolute_timeout"] = *c.SessionAbsoluteTimeout
	}

	if obj, ok := decodeJSONObject(c.JWKS); ok {
		raw["jwks"] = obj
	}
	if c.JWKSUri != nil && *c.JWKSUri != "" {
		raw["jwks_uri"] = *c.JWKSUri
	}
	if c.MTLSBoundCertThumbprint != nil && *c.MTLSBoundCertThumbprint != "" {
		raw["mtls_bound_cert_thumbprint"] = *c.MTLSBoundCertThumbprint
	}
	if obj, ok := decodeJSONObject(c.ScopeClaimMappings); ok {
		raw["scope_claim_mappings"] = obj
	}
	if obj, ok := decodeJSONObject(c.ClaimMappers); ok {
		raw["claim_mappers"] = obj
	}

	encoded, err := json.Marshal(raw)
	if err != nil {
		return c.Config
	}
	return datatypes.JSON(encoded)
}

// decodeJSONObject unwraps a jsonb column into a map so it nests as an object in
// the response rather than as an escaped JSON string.
func decodeJSONObject(value datatypes.JSON) (map[string]any, bool) {
	if len(value) == 0 {
		return nil, false
	}
	var obj map[string]any
	if err := json.Unmarshal(value, &obj); err != nil || len(obj) == 0 {
		return nil, false
	}
	return obj, true
}
