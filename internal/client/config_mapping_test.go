package client

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
)

func TestApplyConfigToClientColumns_MirrorsOAuthSettings(t *testing.T) {
	c := &Client{}
	applyConfigToClientColumns(c, datatypes.JSON(`{
		"grant_types": ["authorization_code", "refresh_token"],
		"response_types": ["code"],
		"allowed_scopes": ["openid", " profile "],
		"token_endpoint_auth_method": "client_secret_basic",
		"require_consent": true,
		"pkce_required": true,
		"access_token_lifetime": 3600,
		"refresh_token_lifetime": 604800,
		"required_acr": "2",
		"session_idle_timeout": 900,
		"session_absolute_timeout": 28800
	}`))

	assert.Equal(t, []string{"authorization_code", "refresh_token"}, []string(c.GrantTypes))
	assert.Equal(t, []string{"code"}, []string(c.ResponseTypes))
	// Whitespace is trimmed so a scope never fails an exact comparison at runtime.
	assert.Equal(t, []string{"openid", "profile"}, []string(c.AllowedScopes))
	assert.Equal(t, TokenAuthMethodSecretBasic, c.TokenEndpointAuthMethod)
	require.NotNil(t, c.RequireConsent)
	assert.True(t, *c.RequireConsent)
	require.NotNil(t, c.RequirePKCE)
	assert.True(t, *c.RequirePKCE)
	require.NotNil(t, c.AccessTokenTTL)
	assert.Equal(t, 3600, *c.AccessTokenTTL)
	require.NotNil(t, c.RefreshTokenTTL)
	assert.Equal(t, 604800, *c.RefreshTokenTTL)
	require.NotNil(t, c.RequiredACR)
	assert.Equal(t, "2", *c.RequiredACR)
	require.NotNil(t, c.SessionIdleTimeout)
	assert.Equal(t, 900, *c.SessionIdleTimeout)
	require.NotNil(t, c.SessionAbsoluteTimeout)
	assert.Equal(t, 28800, *c.SessionAbsoluteTimeout)
}

// An empty array must clear the column rather than be ignored, otherwise an
// operator can never remove the last scope from a client.
func TestApplyConfigToClientColumns_EmptyArrayClearsScopes(t *testing.T) {
	c := &Client{AllowedScopes: []string{"api:read"}}
	applyConfigToClientColumns(c, datatypes.JSON(`{"allowed_scopes": []}`))
	assert.Empty(t, c.AllowedScopes)
}

// config is replaced wholesale on update, so an override the operator removed
// must return to NULL — which is what the runtime reads as "inherit the tenant
// security setting". Leaving the previous value would make the override
// permanent and invisible.
func TestApplyConfigToClientColumns_AbsentOverridesRevertToInherit(t *testing.T) {
	ttl := 3600
	refresh := 604800
	acr := "2"
	idle := 900
	absolute := 28800
	c := &Client{
		AccessTokenTTL:         &ttl,
		RefreshTokenTTL:        &refresh,
		RequiredACR:            &acr,
		SessionIdleTimeout:     &idle,
		SessionAbsoluteTimeout: &absolute,
	}

	applyConfigToClientColumns(c, datatypes.JSON(`{"grant_types": ["authorization_code"]}`))

	assert.Nil(t, c.AccessTokenTTL)
	assert.Nil(t, c.RefreshTokenTTL)
	assert.Nil(t, c.RequiredACR)
	assert.Nil(t, c.SessionIdleTimeout)
	assert.Nil(t, c.SessionAbsoluteTimeout)
}

// A nil or empty config means "unchanged" — the column values must survive, and
// Client.Config must not be replaced with an unusable value.
func TestApplyConfigToClientColumns_EmptyConfigLeavesColumnsAlone(t *testing.T) {
	acr := "2"
	for _, config := range []datatypes.JSON{nil, datatypes.JSON(``)} {
		c := &Client{RequiredACR: &acr, GrantTypes: []string{"authorization_code"}}
		applyConfigToClientColumns(c, config)
		require.NotNil(t, c.RequiredACR)
		assert.Equal(t, "2", *c.RequiredACR)
		assert.Equal(t, []string{"authorization_code"}, []string(c.GrantTypes))
	}
}

// Values outside the allowlist must not reach the DB: chk_clients_required_acr
// and the token endpoint's method dispatch would both reject them.
func TestApplyConfigToClientColumns_IgnoresInvalidValues(t *testing.T) {
	c := &Client{TokenEndpointAuthMethod: TokenAuthMethodSecretBasic}
	applyConfigToClientColumns(c, datatypes.JSON(`{
		"token_endpoint_auth_method": "made_up_method",
		"required_acr": "9",
		"session_idle_timeout": -1
	}`))

	// An unrecognized method leaves the working one in place rather than blanking
	// client authentication.
	assert.Equal(t, TokenAuthMethodSecretBasic, c.TokenEndpointAuthMethod)
	assert.Nil(t, c.RequiredACR)
	assert.Nil(t, c.SessionIdleTimeout)
}

// Malformed JSON must be inert: the caller's validation layer owns rejection, and
// half-applying a blob would leave the columns disagreeing with the stored config.
func TestApplyConfigToClientColumns_MalformedJSONIsInert(t *testing.T) {
	acr := "2"
	c := &Client{RequiredACR: &acr}
	applyConfigToClientColumns(c, datatypes.JSON(`{"grant_types":`))
	require.NotNil(t, c.RequiredACR)
	assert.Equal(t, "2", *c.RequiredACR)
}

// The console persists the older alias names; both spellings must map so a client
// saved by an earlier build keeps working.
func TestApplyConfigToClientColumns_AcceptsKeyAliases(t *testing.T) {
	c := &Client{}
	applyConfigToClientColumns(c, datatypes.JSON(`{
		"consent_required": false,
		"require_pkce": false,
		"access_token_ttl": 1800,
		"refresh_token_ttl": 7200,
		"session_idle_timeout_seconds": 600,
		"session_absolute_timeout_seconds": 3600
	}`))

	require.NotNil(t, c.RequireConsent)
	assert.False(t, *c.RequireConsent)
	require.NotNil(t, c.RequirePKCE)
	assert.False(t, *c.RequirePKCE)
	require.NotNil(t, c.AccessTokenTTL)
	assert.Equal(t, 1800, *c.AccessTokenTTL)
	require.NotNil(t, c.RefreshTokenTTL)
	assert.Equal(t, 7200, *c.RefreshTokenTTL)
	require.NotNil(t, c.SessionIdleTimeout)
	assert.Equal(t, 600, *c.SessionIdleTimeout)
	require.NotNil(t, c.SessionAbsoluteTimeout)
	assert.Equal(t, 3600, *c.SessionAbsoluteTimeout)
}

func decodeConfig(t *testing.T, config datatypes.JSON) map[string]any {
	t.Helper()
	var raw map[string]any
	require.NoError(t, json.Unmarshal(config, &raw))
	return raw
}

// The columns are authoritative — the seeder, gRPC and direct SQL all change them
// without touching the blob. Reporting the blob made the console show stale values
// and round-trip them back on save.
func TestEffectiveClientConfig_ColumnsWinOverStaleBlob(t *testing.T) {
	ttl := 1800
	acr := "2"
	c := &Client{
		Config: datatypes.JSON(`{
			"grant_types": ["implicit"],
			"token_endpoint_auth_method": "none",
			"access_token_ttl": 99999,
			"required_acr": "1",
			"cors_enabled": true,
			"custom": {"team": "platform"}
		}`),
		GrantTypes:              []string{"authorization_code", "refresh_token"},
		TokenEndpointAuthMethod: TokenAuthMethodSecretBasic,
		AccessTokenTTL:          &ttl,
		RequiredACR:             &acr,
	}

	raw := decodeConfig(t, effectiveClientConfig(c))

	assert.Equal(t, []any{"authorization_code", "refresh_token"}, raw["grant_types"])
	assert.Equal(t, TokenAuthMethodSecretBasic, raw["token_endpoint_auth_method"])
	assert.Equal(t, float64(1800), raw["access_token_lifetime"])
	assert.Equal(t, "2", raw["required_acr"])

	// The stale alias must be gone, or a save would round-trip it and the mapper
	// could pick the wrong spelling.
	assert.NotContains(t, raw, "access_token_ttl")

	// Keys with no column pass through untouched.
	assert.Equal(t, true, raw["cors_enabled"])
	assert.Equal(t, map[string]any{"team": "platform"}, raw["custom"])
}

// A NULL override column is how "inherit the tenant default" is stored, so the key
// must be absent rather than reported as a zero.
func TestEffectiveClientConfig_OmitsInheritedOverrides(t *testing.T) {
	c := &Client{Config: datatypes.JSON(`{"required_acr":"1","session_idle_timeout":900}`)}

	raw := decodeConfig(t, effectiveClientConfig(c))

	assert.NotContains(t, raw, "required_acr")
	assert.NotContains(t, raw, "session_idle_timeout")
	assert.NotContains(t, raw, "session_absolute_timeout")
	assert.NotContains(t, raw, "access_token_lifetime")
}

// require_consent / require_pkce are NOT NULL with a TRUE default, so a nil pointer
// means the default applies — reporting nothing would let the console hydrate false
// and disable PKCE on the next save.
func TestEffectiveClientConfig_ReportsBooleanDefaults(t *testing.T) {
	raw := decodeConfig(t, effectiveClientConfig(&Client{}))
	assert.Equal(t, true, raw["require_consent"])
	assert.Equal(t, true, raw["require_pkce"])

	no := false
	raw = decodeConfig(t, effectiveClientConfig(&Client{RequireConsent: &no, RequirePKCE: &no}))
	assert.Equal(t, false, raw["require_consent"])
	assert.Equal(t, false, raw["require_pkce"])
}

// An empty scope allowlist means "every scope" — a real setting, so it must be
// reported as [] rather than omitted like an inherited override.
func TestEffectiveClientConfig_ReportsEmptyScopesExplicitly(t *testing.T) {
	raw := decodeConfig(t, effectiveClientConfig(&Client{}))
	require.Contains(t, raw, "allowed_scopes")
	assert.Equal(t, []any{}, raw["allowed_scopes"])
}

// The advanced settings live only in columns; without this the console cannot show
// a private_key_jwt client's keys, and saving the form would clear them.
func TestEffectiveClientConfig_SurfacesAdvancedColumns(t *testing.T) {
	uri := "https://app.example.com/jwks.json"
	thumb := "A1b2C3d4E5f6G7h8I9j0K1l2M3n4O5p6Q7r8S9t0U1v"
	c := &Client{
		JWKSUri:                 &uri,
		MTLSBoundCertThumbprint: &thumb,
		ScopeClaimMappings:      datatypes.JSON(`{"profile":["name"]}`),
		ClaimMappers:            datatypes.JSON(`{"department":"metadata.dept"}`),
	}

	raw := decodeConfig(t, effectiveClientConfig(c))

	assert.Equal(t, uri, raw["jwks_uri"])
	assert.Equal(t, thumb, raw["mtls_bound_cert_thumbprint"])
	// Nested as objects, not escaped JSON strings.
	assert.Equal(t, map[string]any{"profile": []any{"name"}}, raw["scope_claim_mappings"])
	assert.Equal(t, map[string]any{"department": "metadata.dept"}, raw["claim_mappers"])
	assert.NotContains(t, raw, "jwks")
}

func TestEffectiveClientConfig_SurfacesInlineJWKS(t *testing.T) {
	c := &Client{JWKS: datatypes.JSON(`{"keys":[{"kty":"RSA","kid":"a"}]}`)}
	raw := decodeConfig(t, effectiveClientConfig(c))
	jwks, ok := raw["jwks"].(map[string]any)
	require.True(t, ok, "jwks must nest as an object")
	assert.Len(t, jwks["keys"], 1)
}

// Round-tripping the reported config back through the mapper must be a no-op,
// otherwise opening a client in the console and saving it unchanged mutates it.
func TestEffectiveClientConfig_RoundTripsThroughMapper(t *testing.T) {
	ttl, refresh, idle := 1800, 7200, 900
	acr := "2"
	uri := "https://app.example.com/jwks.json"
	original := &Client{
		Config:                  datatypes.JSON(`{"cors_enabled":true}`),
		GrantTypes:              []string{"authorization_code"},
		ResponseTypes:           []string{"code"},
		AllowedScopes:           []string{"openid"},
		TokenEndpointAuthMethod: TokenAuthMethodPrivateKeyJWT,
		AccessTokenTTL:          &ttl,
		RefreshTokenTTL:         &refresh,
		RequiredACR:             &acr,
		SessionIdleTimeout:      &idle,
		JWKSUri:                 &uri,
	}

	reported := effectiveClientConfig(original)
	// The reported blob must also be accepted by the write-path validator.
	require.NoError(t, validateClientConfig(reported))

	roundTripped := &Client{}
	applyConfigToClientColumns(roundTripped, reported)

	assert.Equal(t, []string(original.GrantTypes), []string(roundTripped.GrantTypes))
	assert.Equal(t, []string(original.ResponseTypes), []string(roundTripped.ResponseTypes))
	assert.Equal(t, []string(original.AllowedScopes), []string(roundTripped.AllowedScopes))
	assert.Equal(t, original.TokenEndpointAuthMethod, roundTripped.TokenEndpointAuthMethod)
	assert.Equal(t, original.AccessTokenTTL, roundTripped.AccessTokenTTL)
	assert.Equal(t, original.RefreshTokenTTL, roundTripped.RefreshTokenTTL)
	assert.Equal(t, original.RequiredACR, roundTripped.RequiredACR)
	assert.Equal(t, original.SessionIdleTimeout, roundTripped.SessionIdleTimeout)
	assert.Nil(t, roundTripped.SessionAbsoluteTimeout)
	require.NotNil(t, roundTripped.JWKSUri)
	assert.Equal(t, uri, *roundTripped.JWKSUri)
}

// A blob that cannot be decoded has no trustworthy passthrough keys, but the
// columns are still readable — the read must not fail or leak the broken blob.
func TestEffectiveClientConfig_MalformedBlobFallsBackToColumns(t *testing.T) {
	c := &Client{Config: datatypes.JSON(`{"grant_types":`), GrantTypes: []string{"authorization_code"}}
	raw := decodeConfig(t, effectiveClientConfig(c))
	assert.Equal(t, []any{"authorization_code"}, raw["grant_types"])
}

// An update replaces the whole client, config included, so without a version token
// two operators editing the same client silently overwrite each other and the loser
// sees a 200.
func TestAssertClientUnchangedSince(t *testing.T) {
	loaded := time.Date(2026, 7, 25, 10, 30, 0, 0, time.UTC)
	client := &Client{UpdatedAt: loaded}

	t.Run("a nil expectation opts out", func(t *testing.T) {
		assert.NoError(t, assertClientUnchangedSince(client, nil))
	})

	t.Run("the loaded version passes", func(t *testing.T) {
		expected := loaded
		assert.NoError(t, assertClientUnchangedSince(client, &expected))
	})

	// The console echoes back an RFC3339 string; a different zone is the same instant.
	t.Run("an equal instant in another zone passes", func(t *testing.T) {
		expected := loaded.In(time.FixedZone("UTC+8", 8*3600))
		assert.NoError(t, assertClientUnchangedSince(client, &expected))
	})

	// Postgres stores microseconds, so a JSON round trip can carry digits the column
	// never had. Comparing raw would make every save a false conflict.
	t.Run("sub-microsecond drift is not a conflict", func(t *testing.T) {
		expected := loaded.Add(400 * time.Nanosecond)
		assert.NoError(t, assertClientUnchangedSince(client, &expected))
	})

	t.Run("a stale version is a conflict", func(t *testing.T) {
		stale := loaded.Add(-time.Second)
		err := assertClientUnchangedSince(client, &stale)
		require.Error(t, err)
		var conflict *apperror.ConflictError
		require.ErrorAs(t, err, &conflict)
		assert.Contains(t, err.Error(), "modified by someone else")
	})

	// A token from the future is equally wrong: it did not come from this row.
	t.Run("a newer version is also a conflict", func(t *testing.T) {
		ahead := loaded.Add(time.Second)
		require.Error(t, assertClientUnchangedSince(client, &ahead))
	})
}

// JSON null and an omitted key both decode to a nil pointer, so "clear this field"
// had no representation at all: a logout URI could be set but never removed.
func TestResolveOptionalString(t *testing.T) {
	current := "https://app.example.com/logout"

	t.Run("nil leaves the current value", func(t *testing.T) {
		got := resolveOptionalString(&current, nil)
		require.NotNil(t, got)
		assert.Equal(t, current, *got)
	})

	t.Run("empty clears", func(t *testing.T) {
		empty := ""
		assert.Nil(t, resolveOptionalString(&current, &empty))

		blank := "   "
		assert.Nil(t, resolveOptionalString(&current, &blank))
	})

	t.Run("a value replaces, trimmed", func(t *testing.T) {
		next := "  https://app.example.com/new  "
		got := resolveOptionalString(&current, &next)
		require.NotNil(t, got)
		assert.Equal(t, "https://app.example.com/new", *got)
	})

	t.Run("setting from empty works", func(t *testing.T) {
		next := "https://app.example.com/logout"
		got := resolveOptionalString(nil, &next)
		require.NotNil(t, got)
		assert.Equal(t, next, *got)
	})
}
