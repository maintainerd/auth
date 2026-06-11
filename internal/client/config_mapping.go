package client

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/lib/pq"
	"gorm.io/datatypes"
)

// validTokenEndpointAuthMethods is the set of RFC-defined client authentication
// methods the token endpoint understands. Anything outside this set is ignored
// so a malformed config cannot silently weaken client authentication.
var validTokenEndpointAuthMethods = map[string]bool{
	TokenAuthMethodSecretBasic:             true,
	TokenAuthMethodSecretPost:              true,
	TokenAuthMethodNone:                    true,
	TokenAuthMethodPrivateKeyJWT:           true,
	TokenAuthMethodClientSecretJWT:         true,
	TokenAuthMethodTLSClientAuth:           true,
	TokenAuthMethodSelfSignedTLSClientAuth: true,
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
		c.RequireConsent = v
	}
	if v, ok := boolFromConfig(firstPresentConfigValue(raw, "require_pkce", "pkce_required")); ok {
		c.RequirePKCE = &v
	}
	if v, ok := intFromConfig(firstPresentConfigValue(raw, "access_token_lifetime", "access_token_ttl")); ok {
		c.AccessTokenTTL = &v
	}
	if v, ok := intFromConfig(firstPresentConfigValue(raw, "refresh_token_lifetime", "refresh_token_ttl")); ok {
		c.RefreshTokenTTL = &v
	}

	// Per-client security overrides — NULL/absent means inherit the tenant default.
	if v, ok := raw["required_acr"].(string); ok {
		if v = strings.TrimSpace(v); validRequiredACRs[v] {
			c.RequiredACR = &v
		}
	}
	if v, ok := intFromConfig(firstPresentConfigValue(raw, "session_idle_timeout", "session_idle_timeout_seconds")); ok && v > 0 {
		c.SessionIdleTimeout = &v
	}
	if v, ok := intFromConfig(firstPresentConfigValue(raw, "session_absolute_timeout", "session_absolute_timeout_seconds")); ok && v > 0 {
		c.SessionAbsoluteTimeout = &v
	}
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
