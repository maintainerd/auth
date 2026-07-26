package secpolicy

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSecuritySettingUpdateConfigRequestDto_Validate(t *testing.T) {
	t.Run("valid with data", func(t *testing.T) {
		d := SecuritySettingUpdateConfigRequestDTO{"min_length": 12}
		assert.NoError(t, d.Validate())
	})

	t.Run("empty config is invalid", func(t *testing.T) {
		d := SecuritySettingUpdateConfigRequestDTO{}
		require.Error(t, d.Validate())
	})

	t.Run("nil config is invalid", func(t *testing.T) {
		var d SecuritySettingUpdateConfigRequestDTO
		require.Error(t, d.Validate())
	})
}

func TestDecodeSecuritySettingUpdateConfig(t *testing.T) {
	t.Run("rejects unknown keys", func(t *testing.T) {
		_, err := DecodeSecuritySettingUpdateConfig("password", strings.NewReader(`{"min_lenght":12}`))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown field")
	})

	t.Run("rejects empty body object", func(t *testing.T) {
		_, err := DecodeSecuritySettingUpdateConfig("password", strings.NewReader(`{}`))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "empty")
	})

	t.Run("accepts typed partial body", func(t *testing.T) {
		cfg, err := DecodeSecuritySettingUpdateConfig("session", strings.NewReader(`{"idle_timeout_minutes":20}`))
		require.NoError(t, err)
		assert.EqualValues(t, float64(20), cfg["idle_timeout_minutes"])
	})
}

func TestNormalizeSecuritySettingConfig(t *testing.T) {
	t.Run("merges defaults, existing config, and patch", func(t *testing.T) {
		cfg, err := NormalizeSecuritySettingConfig(
			"password",
			map[string]any{"min_length": 14, "legacy_junk": true},
			map[string]any{"max_age_days": 90},
		)
		require.NoError(t, err)
		assert.EqualValues(t, float64(14), cfg["min_length"])
		assert.EqualValues(t, float64(90), cfg["max_age_days"])
		assert.EqualValues(t, float64(128), cfg["max_length"])
		assert.NotContains(t, cfg, "legacy_junk")
	})

	t.Run("rejects cross-field validation after defaults are merged", func(t *testing.T) {
		_, err := NormalizeSecuritySettingConfig("mfa", nil, map[string]any{"allowed_methods": []string{"totp"}})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "preferred_method")
	})

	// PS256 must remain accepted — it is the one non-default algorithm the RSA
	// key store can actually sign, so narrowing ES256 out must not catch it.
	t.Run("token accepts PS256 which the RSA key store can sign", func(t *testing.T) {
		cfg, err := NormalizeSecuritySettingConfig("token", nil, map[string]any{"signing_algorithm": "PS256"})
		require.NoError(t, err)
		assert.Equal(t, "PS256", cfg["signing_algorithm"])
	})

	// The two survivors — roles and tenant_id — are server-resolved authz/org
	// context and must stay accepted, and the shipped default (which ships both)
	// must keep validating so tenants are never forced off defaults.
	t.Run("token accepts the server-resolved roles and tenant_id claims", func(t *testing.T) {
		_, err := NormalizeSecuritySettingConfig("token", nil, map[string]any{
			"additional_access_token_claims": []string{"roles", "tenant_id"},
			"additional_id_token_claims":     []string{"roles", "tenant_id"},
		})
		require.NoError(t, err)
	})

	t.Run("shipped default token config still validates", func(t *testing.T) {
		def := MustDefaultSecuritySettingConfig("token")
		_, err := NormalizeSecuritySettingConfig("token", def, nil)
		require.NoError(t, err)
	})

	// The default threat config leaves the unimplemented toggles off, so it must
	// keep validating after they are rejected-when-true.
	t.Run("shipped default threat config still validates", func(t *testing.T) {
		def := MustDefaultSecuritySettingConfig("threat")
		_, err := NormalizeSecuritySettingConfig("threat", def, nil)
		require.NoError(t, err)
	})
}

func TestSecuritySettingValidationRules(t *testing.T) {
	cases := []struct {
		name       string
		configType string
		patch      map[string]any
		want       string
	}{
		{name: "password min greater than max", configType: "password", patch: map[string]any{"min_length": 129}, want: "min_length"},
		{name: "password history has safe upper bound", configType: "password", patch: map[string]any{"password_history_count": 25}, want: "password_history_count"},
		{name: "password age has safe upper bound", configType: "password", patch: map[string]any{"max_age_days": 3651}, want: "max_age_days"},
		{name: "temporary password validity has safe upper bound", configType: "password", patch: map[string]any{"temporary_password_validity_hours": 721}, want: "temporary_password_validity_hours"},
		{name: "mfa sms requires gate", configType: "mfa", patch: map[string]any{"allow_sms": false, "allowed_methods": []string{"sms"}}, want: "allow_sms"},
		// These three are enforced-MFA bypass windows. An unbounded value silently
		// neuters mode=enforced, so each has a safe ceiling like its already-bounded
		// siblings (step_up_ttl, totp_period).
		{name: "mfa grace period has safe upper bound", configType: "mfa", patch: map[string]any{"grace_period_days": 91}, want: "grace_period_days"},
		{name: "mfa admin grace period has safe upper bound", configType: "mfa", patch: map[string]any{"admin_grace_period_days": 91}, want: "admin_grace_period_days"},
		{name: "mfa trusted device period has safe upper bound", configType: "mfa", patch: map[string]any{"trusted_device_period_days": 366}, want: "trusted_device_period_days"},
		{name: "session SameSite None requires secure cookie", configType: "session", patch: map[string]any{"cookie_same_site": "None", "cookie_secure": false}, want: "cookie_secure"},
		{name: "token rejects unsupported algorithm", configType: "token", patch: map[string]any{"signing_algorithm": "HS256"}, want: "signing_algorithm"},
		// Auth-context claims must never be operator-configurable — setting acr/amr
		// without the real auth event forges step-up. These are on jwt.reservedClaims;
		// the token allowlist must not re-admit them.
		{name: "token rejects auth-context claim acr", configType: "token", patch: map[string]any{"additional_access_token_claims": []string{"acr"}}, want: "acr"},
		{name: "token rejects nonce as a static claim", configType: "token", patch: map[string]any{"additional_id_token_claims": []string{"nonce"}}, want: "nonce"},
		// PII does not belong in access tokens (RFC 9068 §6 least-disclosure); ID-token
		// identity claims come from scopes, not this list.
		{name: "token rejects PII claim email", configType: "token", patch: map[string]any{"additional_access_token_claims": []string{"email"}}, want: "email"},
		// ES256 must be rejected: validation used to accept it, but the RSA-only
		// key store cannot sign ES256, so saving it bricked all token issuance
		// for the tenant.
		{name: "token rejects ES256 the key store cannot sign", configType: "token", patch: map[string]any{"signing_algorithm": "ES256"}, want: "signing_algorithm"},
		{name: "lockout max duration must cap duration", configType: "lockout", patch: map[string]any{"lockout_duration_minutes": 30, "max_lockout_duration_minutes": 10}, want: "max_lockout_duration_minutes"},
		{name: "registration disallows auto-confirm with verification", configType: "registration", patch: map[string]any{"auto_confirm_enabled": true}, want: "auto_confirm_enabled"},
		{name: "threat step up cannot exceed block", configType: "threat", patch: map[string]any{"risk_step_up_threshold": 90}, want: "risk_step_up_threshold"},
		// Thresholds above the 0-100 score range are inert (can never fire); reject them.
		{name: "threat block threshold over 100 is inert", configType: "threat", patch: map[string]any{"risk_block_threshold": 150}, want: "risk_block_threshold"},
		// Unimplemented toggles must not be silently accepted (no provider = no enforcement).
		{name: "threat ip reputation not supported", configType: "threat", patch: map[string]any{"ip_reputation_check_enabled": true}, want: "not supported"},
		{name: "threat tor blocking not supported", configType: "threat", patch: map[string]any{"block_tor_exit_nodes": true}, want: "not supported"},
		{name: "threat distinct-account limit must be at least 1", configType: "threat", patch: map[string]any{"distinct_accounts_per_ip_per_hour": 0}, want: "distinct_accounts_per_ip_per_hour"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NormalizeSecuritySettingConfig(tc.configType, nil, tc.patch)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.want)
		})
	}
}

func TestResolveEffectiveSessionPolicy(t *testing.T) {
	t.Run("client can tighten but not loosen tenant timeouts", func(t *testing.T) {
		loose := 7200
		tight := 600
		policy, err := ResolveEffectiveSessionPolicy(
			map[string]any{"idle_timeout_minutes": 30, "absolute_timeout_hours": 24, "access_token_ttl_minutes": 15, "refresh_token_ttl_days": 30},
			map[string]any{"mode": "optional"},
			SecuritySettingClientOverrides{
				SessionIdleTimeout:     &loose,
				SessionAbsoluteTimeout: &tight,
				AccessTokenTTL:         &tight,
			},
		)
		require.NoError(t, err)
		assert.Equal(t, 1800, policy.IdleTimeoutSeconds)
		assert.Equal(t, 600, policy.AbsoluteTimeoutSeconds)
		assert.Equal(t, 600, policy.AccessTokenTTLSeconds)
		assert.Equal(t, "1", policy.RequiredACR)
	})

	t.Run("client and tenant can strengthen required acr", func(t *testing.T) {
		acr := "2"
		policy, err := ResolveEffectiveSessionPolicy(nil, map[string]any{"mode": "optional"}, SecuritySettingClientOverrides{RequiredACR: &acr})
		require.NoError(t, err)
		assert.Equal(t, "2", policy.RequiredACR)
	})
}

func TestResolveEffectiveTokenPolicy(t *testing.T) {
	t.Run("true pkce wins over client false", func(t *testing.T) {
		clientFalse := false
		policy, err := ResolveEffectiveTokenPolicy(map[string]any{"require_pkce": true}, SecuritySettingClientOverrides{RequirePKCE: &clientFalse})
		require.NoError(t, err)
		assert.True(t, policy.RequirePKCE)
	})

	t.Run("client true strengthens tenant false", func(t *testing.T) {
		clientTrue := true
		policy, err := ResolveEffectiveTokenPolicy(map[string]any{"require_pkce": false}, SecuritySettingClientOverrides{RequirePKCE: &clientTrue})
		require.NoError(t, err)
		assert.True(t, policy.RequirePKCE)
	})
}
