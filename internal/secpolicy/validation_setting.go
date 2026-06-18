package secpolicy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"
)

var validSecuritySettingConfigTypes = map[string]bool{
	"password":     true,
	"mfa":          true,
	"session":      true,
	"token":        true,
	"lockout":      true,
	"registration": true,
	"threat":       true,
}

func (r SecuritySettingUpdateConfigRequestDTO) Validate() error {
	if len(r) == 0 {
		return fmt.Errorf("config cannot be empty")
	}
	return nil
}

func DecodeSecuritySettingUpdateConfig(configType string, body io.Reader) (map[string]any, error) {
	if !validSecuritySettingConfigTypes[configType] {
		return nil, fmt.Errorf("invalid config type")
	}

	raw, err := io.ReadAll(body)
	if err != nil {
		return nil, err
	}
	if len(bytes.TrimSpace(raw)) == 0 {
		return nil, fmt.Errorf("config cannot be empty")
	}

	patch, err := decodeSecuritySettingPatch(configType, raw)
	if err != nil {
		return nil, err
	}
	if len(patch) == 0 {
		return nil, fmt.Errorf("config cannot be empty")
	}
	return patch, nil
}

func NormalizeSecuritySettingConfig(configType string, existing, patch map[string]any) (map[string]any, error) {
	defaults, ok := DefaultSecuritySettingConfig(configType)
	if !ok {
		return nil, fmt.Errorf("invalid config type")
	}

	merged := defaults
	allowed := map[string]bool{}
	for k := range defaults {
		allowed[k] = true
	}
	for k, v := range existing {
		if allowed[k] {
			merged[k] = v
		}
	}
	if patch != nil {
		encodedPatch, err := json.Marshal(patch)
		if err != nil {
			return nil, err
		}
		typedPatch, err := decodeSecuritySettingPatch(configType, encodedPatch)
		if err != nil {
			return nil, err
		}
		if len(typedPatch) == 0 {
			return nil, fmt.Errorf("config cannot be empty")
		}
		for k, v := range typedPatch {
			merged[k] = v
		}
	}

	encoded, err := json.Marshal(merged)
	if err != nil {
		return nil, err
	}
	full, err := decodeSecuritySettingPatch(configType, encoded)
	if err != nil {
		return nil, err
	}
	return full, nil
}

func decodeSecuritySettingPatch(configType string, raw []byte) (map[string]any, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()

	var dto any
	switch configType {
	case "password":
		var d PasswordConfigDTO
		if err := decoder.Decode(&d); err != nil {
			return nil, err
		}
		dto = d
	case "mfa":
		var d MFAConfigDTO
		if err := decoder.Decode(&d); err != nil {
			return nil, err
		}
		dto = d
	case "session":
		var d SessionConfigDTO
		if err := decoder.Decode(&d); err != nil {
			return nil, err
		}
		dto = d
	case "token":
		var d TokenConfigDTO
		if err := decoder.Decode(&d); err != nil {
			return nil, err
		}
		dto = d
	case "lockout":
		var d LockoutConfigDTO
		if err := decoder.Decode(&d); err != nil {
			return nil, err
		}
		dto = d
	case "registration":
		var d RegistrationConfigDTO
		if err := decoder.Decode(&d); err != nil {
			return nil, err
		}
		dto = d
	case "threat":
		var d ThreatConfigDTO
		if err := decoder.Decode(&d); err != nil {
			return nil, err
		}
		dto = d
	default:
		return nil, fmt.Errorf("invalid config type")
	}
	if decoder.Decode(&struct{}{}) != io.EOF {
		return nil, fmt.Errorf("request body must contain a single JSON object")
	}

	if err := validateSecuritySettingDTO(configType, dto); err != nil {
		return nil, err
	}
	return dtoToMap(dto)
}

func validateSecuritySettingDTO(configType string, dto any) error {
	switch d := dto.(type) {
	case PasswordConfigDTO:
		return validatePasswordConfig(d)
	case MFAConfigDTO:
		return validateMFAConfig(d)
	case SessionConfigDTO:
		return validateSessionConfig(d)
	case TokenConfigDTO:
		return validateTokenConfig(d)
	case LockoutConfigDTO:
		return validateLockoutConfig(d)
	case RegistrationConfigDTO:
		return validateRegistrationConfig(d)
	case ThreatConfigDTO:
		return validateThreatConfig(d)
	default:
		return fmt.Errorf("invalid config type: %s", configType)
	}
}

func validatePasswordConfig(d PasswordConfigDTO) error {
	if d.MinLength != nil && *d.MinLength < 1 {
		return fmt.Errorf("min_length must be at least 1")
	}
	if d.MaxLength != nil {
		if *d.MaxLength > 128 {
			return fmt.Errorf("max_length must be at most 128")
		}
		if *d.MaxLength < 64 {
			return fmt.Errorf("max_length must be at least 64")
		}
	}
	if d.MinLength != nil && d.MaxLength != nil && *d.MinLength > *d.MaxLength {
		return fmt.Errorf("min_length must be less than or equal to max_length")
	}
	if d.PasswordHistoryCount != nil && *d.PasswordHistoryCount < 0 {
		return fmt.Errorf("password_history_count must be non-negative")
	}
	if d.MaxAgeDays != nil && *d.MaxAgeDays < 0 {
		return fmt.Errorf("max_age_days must be non-negative")
	}
	if d.TemporaryPasswordValidityHours != nil && *d.TemporaryPasswordValidityHours < 1 {
		return fmt.Errorf("temporary_password_validity_hours must be at least 1")
	}
	if d.HashAlgorithm != nil && !oneOf(*d.HashAlgorithm, "argon2id", "bcrypt", "scrypt", "pbkdf2") {
		return fmt.Errorf("hash_algorithm must be one of argon2id, bcrypt, scrypt, pbkdf2")
	}
	if d.MinStrengthScore != nil && (*d.MinStrengthScore < 0 || *d.MinStrengthScore > 4) {
		return fmt.Errorf("min_strength_score must be between 0 and 4")
	}
	return nil
}

func validateMFAConfig(d MFAConfigDTO) error {
	if d.Mode != nil && !oneOf(*d.Mode, "disabled", "optional", "enforced") {
		return fmt.Errorf("mode must be one of disabled, optional, enforced")
	}
	allowedMethodSet := map[string]bool{
		"totp": true, "webauthn": true, "sms": true, "email_otp": true, "recovery_code": true,
	}
	methods := map[string]bool{}
	for _, method := range d.AllowedMethods {
		if !allowedMethodSet[method] {
			return fmt.Errorf("allowed_methods contains unsupported method %q", method)
		}
		methods[method] = true
	}
	if methods["sms"] && (d.AllowSMS == nil || !*d.AllowSMS) {
		return fmt.Errorf("sms is allowed only when allow_sms is true")
	}
	if methods["totp"] && d.TOTPIssuer != nil && strings.TrimSpace(*d.TOTPIssuer) == "" {
		return fmt.Errorf("totp_issuer is required when totp is allowed")
	}
	if d.PreferredMethod != nil && len(methods) > 0 && !methods[*d.PreferredMethod] {
		return fmt.Errorf("preferred_method must be included in allowed_methods")
	}
	if d.TOTPDigits != nil && *d.TOTPDigits != 6 && *d.TOTPDigits != 8 {
		return fmt.Errorf("totp_digits must be 6 or 8")
	}
	if d.TOTPPeriodSeconds != nil && (*d.TOTPPeriodSeconds < 30 || *d.TOTPPeriodSeconds > 90) {
		return fmt.Errorf("totp_period_seconds must be between 30 and 90")
	}
	if d.TrustedDevicePeriodDays != nil && *d.TrustedDevicePeriodDays < 0 {
		return fmt.Errorf("trusted_device_period_days must be non-negative")
	}
	if d.GracePeriodDays != nil && *d.GracePeriodDays < 0 {
		return fmt.Errorf("grace_period_days must be non-negative")
	}
	if d.RecoveryCodesCount != nil && *d.RecoveryCodesCount != 0 && (*d.RecoveryCodesCount < 8 || *d.RecoveryCodesCount > 16) {
		return fmt.Errorf("recovery_codes_count must be 0 or between 8 and 16")
	}
	if d.AdminGracePeriodDays != nil && *d.AdminGracePeriodDays < 0 {
		return fmt.Errorf("admin_grace_period_days must be non-negative")
	}
	if d.StepUpTTLMinutes != nil && *d.StepUpTTLMinutes < 1 {
		return fmt.Errorf("step_up_ttl_minutes must be at least 1")
	}
	if d.StepUpTTLMinutes != nil && *d.StepUpTTLMinutes > 60 {
		return fmt.Errorf("step_up_ttl_minutes must be at most 60")
	}
	return nil
}

func validateSessionConfig(d SessionConfigDTO) error {
	if d.AccessTokenTTLMinutes != nil && (*d.AccessTokenTTLMinutes < 1 || *d.AccessTokenTTLMinutes > 60) {
		return fmt.Errorf("access_token_ttl_minutes must be between 1 and 60")
	}
	if d.RefreshTokenTTLDays != nil && (*d.RefreshTokenTTLDays < 1 || *d.RefreshTokenTTLDays > 365) {
		return fmt.Errorf("refresh_token_ttl_days must be between 1 and 365")
	}
	if d.MaxConcurrentSessions != nil && *d.MaxConcurrentSessions < 0 {
		return fmt.Errorf("max_concurrent_sessions must be non-negative")
	}
	if d.IdleTimeoutMinutes != nil && *d.IdleTimeoutMinutes < 1 {
		return fmt.Errorf("idle_timeout_minutes must be at least 1")
	}
	if d.AbsoluteTimeoutHours != nil && *d.AbsoluteTimeoutHours < 1 {
		return fmt.Errorf("absolute_timeout_hours must be at least 1")
	}
	if d.RefreshTokenReuseIntervalSeconds != nil && *d.RefreshTokenReuseIntervalSeconds < 0 {
		return fmt.Errorf("refresh_token_reuse_interval_seconds must be non-negative")
	}
	if d.CookieSameSite != nil && !oneOf(*d.CookieSameSite, "Strict", "Lax", "None") {
		return fmt.Errorf("cookie_same_site must be one of Strict, Lax, None")
	}
	if d.CookieSameSite != nil && *d.CookieSameSite == "None" && d.CookieSecure != nil && !*d.CookieSecure {
		return fmt.Errorf("cookie_secure must be true when cookie_same_site is None")
	}
	return nil
}

func validateTokenConfig(d TokenConfigDTO) error {
	if d.ClockSkewLeewaySeconds != nil && (*d.ClockSkewLeewaySeconds < 0 || *d.ClockSkewLeewaySeconds > 300) {
		return fmt.Errorf("clock_skew_leeway_seconds must be between 0 and 300")
	}
	if d.SigningAlgorithm != nil && !oneOf(*d.SigningAlgorithm, "RS256", "ES256", "PS256") {
		return fmt.Errorf("signing_algorithm must be one of RS256, ES256, PS256")
	}
	for _, claim := range append(append([]string{}, d.AdditionalIDTokenClaims...), d.AdditionalAccessTokenClaims...) {
		if !knownTokenClaim(claim) {
			return fmt.Errorf("additional token claim %q is not supported", claim)
		}
	}
	encoded, _ := json.Marshal(d)
	if len(encoded) >= 4096 {
		return fmt.Errorf("token_config must stay below 4 KB")
	}
	return nil
}

func validateLockoutConfig(d LockoutConfigDTO) error {
	if d.MaxFailedAttempts != nil && (*d.MaxFailedAttempts < 1 || *d.MaxFailedAttempts > 100) {
		return fmt.Errorf("max_failed_attempts must be between 1 and 100")
	}
	if d.LockoutDurationMinutes != nil && *d.LockoutDurationMinutes < 1 {
		return fmt.Errorf("lockout_duration_minutes must be at least 1")
	}
	if d.ObservationWindowMinutes != nil && *d.ObservationWindowMinutes < 1 {
		return fmt.Errorf("observation_window_minutes must be at least 1")
	}
	if d.MaxLockoutDurationMinutes != nil && d.LockoutDurationMinutes != nil && *d.MaxLockoutDurationMinutes < *d.LockoutDurationMinutes {
		return fmt.Errorf("max_lockout_duration_minutes must be greater than or equal to lockout_duration_minutes")
	}
	if d.ProgressionResetHours != nil && *d.ProgressionResetHours < 1 {
		return fmt.Errorf("progression_reset_hours must be at least 1")
	}
	return nil
}

func validateRegistrationConfig(d RegistrationConfigDTO) error {
	allowed := normalizedDomainSet(d.AllowedEmailDomains)
	blocked := normalizedDomainSet(d.BlockedEmailDomains)
	for domain := range allowed {
		if blocked[domain] {
			return fmt.Errorf("allowed_email_domains and blocked_email_domains must not overlap")
		}
	}
	for _, domain := range append(append([]string{}, d.AllowedEmailDomains...), d.BlockedEmailDomains...) {
		if !validDomainPattern(domain) {
			return fmt.Errorf("email domain %q is invalid", domain)
		}
	}
	if d.AutoConfirmEnabled != nil && d.RequireEmailVerification != nil && *d.AutoConfirmEnabled && *d.RequireEmailVerification {
		return fmt.Errorf("auto_confirm_enabled and require_email_verification cannot both be true")
	}
	if d.VerificationTokenTTLHours != nil && *d.VerificationTokenTTLHours < 1 {
		return fmt.Errorf("verification_token_ttl_hours must be at least 1")
	}
	if d.RegistrationRateLimitPerIPPerHour != nil && *d.RegistrationRateLimitPerIPPerHour < 1 {
		return fmt.Errorf("registration_rate_limit_per_ip_per_hour must be at least 1")
	}
	return nil
}

func validateThreatConfig(d ThreatConfigDTO) error {
	if d.RiskStepUpThreshold != nil && *d.RiskStepUpThreshold < 0 {
		return fmt.Errorf("risk_step_up_threshold must be non-negative")
	}
	if d.RiskBlockThreshold != nil && *d.RiskBlockThreshold < 0 {
		return fmt.Errorf("risk_block_threshold must be non-negative")
	}
	if d.RiskStepUpThreshold != nil && d.RiskBlockThreshold != nil && *d.RiskStepUpThreshold > *d.RiskBlockThreshold {
		return fmt.Errorf("risk_step_up_threshold must be less than or equal to risk_block_threshold")
	}
	if d.VelocityFailuresPerIPPerHour != nil && *d.VelocityFailuresPerIPPerHour < 1 {
		return fmt.Errorf("velocity_failures_per_ip_per_hour must be at least 1")
	}
	return nil
}

func ResolveEffectiveSessionPolicy(tenantSession, tenantMFA map[string]any, client SecuritySettingClientOverrides) (EffectiveSessionPolicy, error) {
	session, err := NormalizeSecuritySettingConfig("session", tenantSession, nil)
	if err != nil {
		return EffectiveSessionPolicy{}, err
	}
	mfa, err := NormalizeSecuritySettingConfig("mfa", tenantMFA, nil)
	if err != nil {
		return EffectiveSessionPolicy{}, err
	}

	p := EffectiveSessionPolicy{
		AccessTokenTTLSeconds:            intValue(session["access_token_ttl_minutes"]) * 60,
		RefreshTokenTTLSeconds:           intValue(session["refresh_token_ttl_days"]) * 24 * 60 * 60,
		MaxConcurrentSessions:            intValue(session["max_concurrent_sessions"]),
		IdleTimeoutSeconds:               intValue(session["idle_timeout_minutes"]) * 60,
		AbsoluteTimeoutSeconds:           intValue(session["absolute_timeout_hours"]) * 60 * 60,
		RotateRefreshTokens:              boolValue(session["rotate_refresh_tokens"]),
		RefreshTokenReuseIntervalSeconds: intValue(session["refresh_token_reuse_interval_seconds"]),
		CookieSecure:                     boolValue(session["cookie_secure"]),
		CookieHTTPOnly:                   boolValue(session["cookie_http_only"]),
		CookieSameSite:                   stringValue(session["cookie_same_site"]),
		RevokeSessionsOnPasswordChange:   boolValue(session["revoke_sessions_on_password_change"]),
		RequiredACR:                      "1",
	}
	if stringValue(mfa["mode"]) == "enforced" {
		p.RequiredACR = "2"
	}
	if client.AccessTokenTTL != nil && *client.AccessTokenTTL > 0 && *client.AccessTokenTTL < p.AccessTokenTTLSeconds {
		p.AccessTokenTTLSeconds = *client.AccessTokenTTL
	}
	if client.RefreshTokenTTL != nil && *client.RefreshTokenTTL > 0 && *client.RefreshTokenTTL < p.RefreshTokenTTLSeconds {
		p.RefreshTokenTTLSeconds = *client.RefreshTokenTTL
	}
	if client.SessionIdleTimeout != nil && *client.SessionIdleTimeout > 0 && *client.SessionIdleTimeout < p.IdleTimeoutSeconds {
		p.IdleTimeoutSeconds = *client.SessionIdleTimeout
	}
	if client.SessionAbsoluteTimeout != nil && *client.SessionAbsoluteTimeout > 0 && *client.SessionAbsoluteTimeout < p.AbsoluteTimeoutSeconds {
		p.AbsoluteTimeoutSeconds = *client.SessionAbsoluteTimeout
	}
	if client.RequiredACR != nil && *client.RequiredACR == "2" {
		p.RequiredACR = "2"
	}
	return p, nil
}

func ResolveEffectiveTokenPolicy(tenantToken map[string]any, client SecuritySettingClientOverrides) (EffectiveTokenPolicy, error) {
	token, err := NormalizeSecuritySettingConfig("token", tenantToken, nil)
	if err != nil {
		return EffectiveTokenPolicy{}, err
	}
	p := EffectiveTokenPolicy{
		ClockSkewLeewaySeconds:      intValue(token["clock_skew_leeway_seconds"]),
		AdditionalIDTokenClaims:     stringSliceValue(token["additional_id_token_claims"]),
		AdditionalAccessTokenClaims: stringSliceValue(token["additional_access_token_claims"]),
		SigningAlgorithm:            stringValue(token["signing_algorithm"]),
		RequirePKCE:                 boolValue(token["require_pkce"]),
	}
	if client.RequirePKCE != nil && *client.RequirePKCE {
		p.RequirePKCE = true
	}
	return p, nil
}

func dtoToMap(dto any) (map[string]any, error) {
	b, err := json.Marshal(dto)
	if err != nil {
		return nil, err
	}
	out := map[string]any{}
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	for k, v := range out {
		if v == nil {
			delete(out, k)
		}
	}
	return out, nil
}

func oneOf(value string, allowed ...string) bool {
	for _, item := range allowed {
		if value == item {
			return true
		}
	}
	return false
}

func knownTokenClaim(claim string) bool {
	switch claim {
	case "roles", "tenant_id", "email", "email_verified", "phone", "phone_verified",
		"name", "given_name", "family_name", "picture", "locale", "auth_time",
		"nonce", "at_hash", "acr", "amr", "sid",
		"permissions", "fullname":
		return true
	default:
		return false
	}
}

func normalizedDomainSet(values []string) map[string]bool {
	out := map[string]bool{}
	for _, v := range values {
		out[strings.ToLower(strings.TrimSpace(v))] = true
	}
	return out
}

var domainLabelPattern = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$`)

func validDomainPattern(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return false
	}
	value = strings.TrimPrefix(value, "*.")
	parts := strings.Split(value, ".")
	if len(parts) < 2 {
		return false
	}
	for _, part := range parts {
		if !domainLabelPattern.MatchString(part) {
			return false
		}
	}
	return true
}

func intValue(v any) int {
	switch t := v.(type) {
	case int:
		return t
	case int64:
		return int(t)
	case float64:
		return int(t)
	case json.Number:
		n, _ := t.Int64()
		return int(n)
	default:
		return 0
	}
}

func boolValue(v any) bool {
	b, _ := v.(bool)
	return b
}

func stringValue(v any) string {
	s, _ := v.(string)
	return s
}

func stringSliceValue(v any) []string {
	switch values := v.(type) {
	case []string:
		return append([]string(nil), values...)
	case []any:
		out := make([]string, 0, len(values))
		for _, value := range values {
			if s, ok := value.(string); ok {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}
