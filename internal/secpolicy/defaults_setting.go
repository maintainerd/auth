package secpolicy

import (
	"encoding/json"

	"gorm.io/datatypes"
)

var defaultSecuritySettingConfigs = map[string]map[string]any{
	"password": {
		"min_length":                        12,
		"max_length":                        128,
		"require_uppercase":                 false,
		"require_lowercase":                 false,
		"require_number":                    false,
		"require_symbol":                    false,
		"reject_common_passwords":           true,
		"check_hibp":                        true,
		"password_history_count":            5,
		"max_age_days":                      0,
		"temporary_password_validity_hours": 72,
		"hash_algorithm":                    "argon2id",
		"min_strength_score":                2,
	},
	"mfa": {
		"mode":                              "optional",
		"allowed_methods":                   []string{"totp", "webauthn", "recovery_code"},
		"totp_issuer":                       "Maintainerd-Auth",
		"trusted_device_period_days":        14,
		"grace_period_days":                 30,
		"preferred_method":                  "webauthn",
		"allow_sms":                         false,
		"allow_email_otp":                   false,
		"totp_digits":                       6,
		"totp_period_seconds":               30,
		"recovery_codes_count":              10,
		"require_mfa_for_sensitive_actions": true,
		"admin_grace_period_days":           0,
		"step_up_ttl_minutes":               5,
	},
	"session": {
		"access_token_ttl_minutes":             15,
		"refresh_token_ttl_days":               30,
		"max_concurrent_sessions":              5,
		"idle_timeout_minutes":                 30,
		"absolute_timeout_hours":               24,
		"rotate_refresh_tokens":                true,
		"refresh_token_reuse_interval_seconds": 10,
		"cookie_secure":                        true,
		"cookie_http_only":                     true,
		"cookie_same_site":                     "Lax",
		"revoke_sessions_on_password_change":   true,
	},
	"token": {
		"clock_skew_leeway_seconds":      30,
		"additional_id_token_claims":     []string{"roles", "tenant_id"},
		"additional_access_token_claims": []string{"roles", "tenant_id"},
		"signing_algorithm":              "RS256",
		"require_pkce":                   true,
	},
	"lockout": {
		"enabled":                      true,
		"max_failed_attempts":          5,
		"lockout_duration_minutes":     30,
		"progressive_lockout":          true,
		"auto_unlock":                  true,
		"reset_count_on_success":       true,
		"observation_window_minutes":   15,
		"max_lockout_duration_minutes": 60,
		"progression_reset_hours":      24,
		"notify_user_on_lockout":       true,
	},
	"registration": {
		"self_registration_enabled":    true,
		"require_email_verification":   true,
		"require_phone_verification":   false,
		"allowed_email_domains":        []string{},
		"blocked_email_domains":        []string{},
		"auto_confirm_enabled":         false,
		"verification_token_ttl_hours": 24,
		// Captcha is deferred to a later release and no first-party form emits a
		// captcha_token, so seeding this true made every tenant reject 100% of
		// self-service registration the moment CAPTCHA_SECRET was configured — with
		// no way to turn it off. It must default off until the feature ships.
		"captcha_on_signup":                       false,
		"registration_rate_limit_per_ip_per_hour": 10,
	},
	"threat": {
		"brute_force_detection_enabled":             true,
		"impossible_travel_detection_enabled":       true,
		"new_device_notification_enabled":           true,
		"velocity_check_enabled":                    true,
		"risk_based_step_up_enabled":                false,
		"compromised_credential_monitoring_enabled": true,
		"ip_reputation_check_enabled":               false,
		"block_tor_exit_nodes":                      false,
		"risk_step_up_threshold":                    21,
		"risk_block_threshold":                      81,
		"velocity_failures_per_ip_per_hour":         50,
		"distinct_accounts_per_ip_per_hour":         10,
	},
}

func DefaultSecuritySettingConfig(configType string) (map[string]any, bool) {
	cfg, ok := defaultSecuritySettingConfigs[configType]
	if !ok {
		return nil, false
	}
	return copyConfigMap(cfg), true
}

func MustDefaultSecuritySettingConfig(configType string) map[string]any {
	cfg, ok := DefaultSecuritySettingConfig(configType)
	if !ok {
		panic("unknown security setting config type: " + configType)
	}
	return cfg
}

func DefaultSecuritySettingJSON(configType string) datatypes.JSON {
	cfg := MustDefaultSecuritySettingConfig(configType)
	b, err := json.Marshal(cfg)
	if err != nil {
		panic(err)
	}
	return datatypes.JSON(b)
}

func NewDefaultSecuritySetting(tenantID int64) SecuritySetting {
	return SecuritySetting{
		TenantID:           tenantID,
		MFAConfig:          DefaultSecuritySettingJSON("mfa"),
		PasswordConfig:     DefaultSecuritySettingJSON("password"),
		SessionConfig:      DefaultSecuritySettingJSON("session"),
		ThreatConfig:       DefaultSecuritySettingJSON("threat"),
		LockoutConfig:      DefaultSecuritySettingJSON("lockout"),
		RegistrationConfig: DefaultSecuritySettingJSON("registration"),
		TokenConfig:        DefaultSecuritySettingJSON("token"),
		Version:            1,
	}
}

func ApplySecuritySettingDefaults(setting *SecuritySetting) bool {
	if setting == nil {
		return false
	}
	changed := false
	changed = applyDefaultJSON(&setting.MFAConfig, "mfa") || changed
	changed = applyDefaultJSON(&setting.PasswordConfig, "password") || changed
	changed = applyDefaultJSON(&setting.SessionConfig, "session") || changed
	changed = applyDefaultJSON(&setting.ThreatConfig, "threat") || changed
	changed = applyDefaultJSON(&setting.LockoutConfig, "lockout") || changed
	changed = applyDefaultJSON(&setting.RegistrationConfig, "registration") || changed
	changed = applyDefaultJSON(&setting.TokenConfig, "token") || changed
	return changed
}

func applyDefaultJSON(raw *datatypes.JSON, configType string) bool {
	current := map[string]any{}
	if raw != nil && len(*raw) > 0 {
		_ = json.Unmarshal(*raw, &current)
	}
	merged, err := NormalizeSecuritySettingConfig(configType, current, nil)
	if err != nil {
		return false
	}
	b, err := json.Marshal(merged)
	if err != nil {
		return false
	}
	if string(*raw) == string(b) {
		return false
	}
	*raw = datatypes.JSON(b)
	return true
}

func copyConfigMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		switch values := v.(type) {
		case []string:
			cp := append([]string(nil), values...)
			out[k] = cp
		case []any:
			cp := append([]any(nil), values...)
			out[k] = cp
		default:
			out[k] = v
		}
	}
	return out
}
