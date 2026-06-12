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

// applyConfigToClientColumns mirrors the OAuth settings carried in the free-form
// `config` JSON into the first-class client columns the OAuth runtime actually
// reads — grant_types, response_types, token_endpoint_auth_method, allowed_scopes,
// require_consent, and the access/refresh token TTLs.
//
// The admin console persists these settings inside `config`, but the authorization
// and token-issuance paths (internal/oauth) read them from the dedicated columns.
// Without this mapping, anything configured in the console would live only in the
// opaque config blob and never take effect at runtime.
//
// Keys without a dedicated column (pkce_required, cors_enabled, refresh_token_rotation,
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
	if v, ok := intFromConfig(firstPresentConfigValue(raw, "access_token_lifetime", "access_token_ttl")); ok {
		c.AccessTokenTTL = &v
	}
	if v, ok := intFromConfig(firstPresentConfigValue(raw, "refresh_token_lifetime", "refresh_token_ttl")); ok {
		c.RefreshTokenTTL = &v
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
