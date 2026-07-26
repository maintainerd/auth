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

const (
	maxPasswordHistoryCount           = 24
	maxPasswordAgeDays                = 3650
	maxTemporaryPasswordValidityHours = 720

	// Upper bounds on the MFA bypass windows. A year of trusted-device memory and
	// a quarter of enrollment grace are past any legitimate policy; the point is
	// only to stop mode=enforced being neutered by an absurd value. Kept
	// deliberately generous so no real configuration is rejected. See the
	// comments in validateMFAConfig.
	maxTrustedDevicePeriodDays = 365
	maxMFAGracePeriodDays      = 90

	// minPasswordMinLength is the floor a tenant may configure for min_length.
	//
	// DO NOT raise this to the NIST/ASVS recommended minimum. It is tempting,
	// and it is a breaking change: validatePasswordConfig runs on READ as well
	// as write — LoadPasswordPolicy → NormalizeSecuritySettingConfig →
	// decodeSecuritySettingPatch → here. Raising the floor makes every already
	// stored config below it fail to normalize, so those tenants silently fall
	// back to the SHIPPED DEFAULTS for all thirteen fields, not just this one.
	//
	// The recommended minimum belongs in the shipped default (12) and in the
	// console's guidance, both of which already say so. A tenant choosing to go
	// lower is making a policy decision this validator should record, not veto.
	minPasswordMinLength = 1
)

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
	if d.MinLength != nil && *d.MinLength < minPasswordMinLength {
		return fmt.Errorf("min_length must be at least %d", minPasswordMinLength)
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
	if d.PasswordHistoryCount != nil && *d.PasswordHistoryCount > maxPasswordHistoryCount {
		return fmt.Errorf("password_history_count must be at most %d", maxPasswordHistoryCount)
	}
	if d.MaxAgeDays != nil && *d.MaxAgeDays < 0 {
		return fmt.Errorf("max_age_days must be non-negative")
	}
	if d.MaxAgeDays != nil && *d.MaxAgeDays > maxPasswordAgeDays {
		return fmt.Errorf("max_age_days must be at most %d", maxPasswordAgeDays)
	}
	if d.TemporaryPasswordValidityHours != nil && *d.TemporaryPasswordValidityHours < 1 {
		return fmt.Errorf("temporary_password_validity_hours must be at least 1")
	}
	if d.TemporaryPasswordValidityHours != nil && *d.TemporaryPasswordValidityHours > maxTemporaryPasswordValidityHours {
		return fmt.Errorf("temporary_password_validity_hours must be at most %d", maxTemporaryPasswordValidityHours)
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
	if methods["email_otp"] && (d.AllowEmailOTP == nil || !*d.AllowEmailOTP) {
		return fmt.Errorf("email_otp is allowed only when allow_email_otp is true")
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
	// These three are all windows during which enforced MFA is bypassed. Left
	// unbounded above, a tenant admin could set grace_period_days: 100000 and
	// silently neuter mode=enforced — the sibling fields here (step_up_ttl 1-60,
	// totp_period 30-90) are already bounded on both ends, so the missing ceilings
	// were the odd ones out. The ceilings are generous (a year of trusted-device
	// memory, a quarter of enrollment grace) so no legitimate policy is rejected;
	// they only stop enforcement being turned off by way of an absurd window.
	if d.TrustedDevicePeriodDays != nil && (*d.TrustedDevicePeriodDays < 0 || *d.TrustedDevicePeriodDays > maxTrustedDevicePeriodDays) {
		return fmt.Errorf("trusted_device_period_days must be between 0 and %d", maxTrustedDevicePeriodDays)
	}
	if d.GracePeriodDays != nil && (*d.GracePeriodDays < 0 || *d.GracePeriodDays > maxMFAGracePeriodDays) {
		return fmt.Errorf("grace_period_days must be between 0 and %d", maxMFAGracePeriodDays)
	}
	if d.RecoveryCodesCount != nil && *d.RecoveryCodesCount != 0 && (*d.RecoveryCodesCount < 8 || *d.RecoveryCodesCount > 16) {
		return fmt.Errorf("recovery_codes_count must be 0 or between 8 and 16")
	}
	if d.AdminGracePeriodDays != nil && (*d.AdminGracePeriodDays < 0 || *d.AdminGracePeriodDays > maxMFAGracePeriodDays) {
		return fmt.Errorf("admin_grace_period_days must be between 0 and %d", maxMFAGracePeriodDays)
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
	// ES256 is deliberately NOT accepted. The signing key store is RSA-only
	// (platform/jwt: signingKey returns *rsa.PrivateKey), and token generation
	// returns an error for ES256 ("requires an ECDSA key store"). Accepting it
	// here would let a tenant admin save a value that then makes EVERY token
	// issuance fail — a self-inflicted, settings-blessed auth outage for the
	// whole tenant. Only advertise what the server can actually sign; restore
	// ES256 here if and when an ECDSA key store is added.
	if d.SigningAlgorithm != nil && !oneOf(*d.SigningAlgorithm, "RS256", "PS256") {
		return fmt.Errorf("signing_algorithm must be one of RS256, PS256")
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
	// Risk score is 0-100 (see security.AssessLoginThreat), so a threshold above
	// 100 can never be reached and silently makes the control inert. Reject it
	// rather than let an admin configure a block/step-up that never fires.
	if d.RiskStepUpThreshold != nil && (*d.RiskStepUpThreshold < 0 || *d.RiskStepUpThreshold > 100) {
		return fmt.Errorf("risk_step_up_threshold must be between 0 and 100")
	}
	if d.RiskBlockThreshold != nil && (*d.RiskBlockThreshold < 0 || *d.RiskBlockThreshold > 100) {
		return fmt.Errorf("risk_block_threshold must be between 0 and 100")
	}
	if d.RiskStepUpThreshold != nil && d.RiskBlockThreshold != nil && *d.RiskStepUpThreshold > *d.RiskBlockThreshold {
		return fmt.Errorf("risk_step_up_threshold must be less than or equal to risk_block_threshold")
	}
	if d.VelocityFailuresPerIPPerHour != nil && *d.VelocityFailuresPerIPPerHour < 1 {
		return fmt.Errorf("velocity_failures_per_ip_per_hour must be at least 1")
	}
	if d.DistinctAccountsPerIPPerHour != nil && *d.DistinctAccountsPerIPPerHour < 1 {
		return fmt.Errorf("distinct_accounts_per_ip_per_hour must be at least 1")
	}
	// ip_reputation_check_enabled and block_tor_exit_nodes require an external
	// data source (an IP-reputation feed / a Tor exit-node list) that this
	// standalone server does not ship. Accepting `true` would be a SILENT
	// fail-open: the admin believes the control is on while nothing enforces it.
	// Reject enabling them until a provider is wired — the same "don't advertise
	// what you can't enforce" rule applied to unsupported signing algorithms.
	if d.IPReputationCheckEnabled != nil && *d.IPReputationCheckEnabled {
		return fmt.Errorf("ip_reputation_check_enabled is not supported: no IP reputation provider is configured")
	}
	if d.BlockTorExitNodes != nil && *d.BlockTorExitNodes {
		return fmt.Errorf("block_tor_exit_nodes is not supported: no Tor exit-node source is configured")
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

// knownTokenClaim is the allowlist of claim names a tenant may add to its access
// and ID tokens via token_config.additional_*_claims.
//
// It is deliberately narrow — only server-resolved organization/authorization
// context. The previous allowlist admitted three dangerous categories, all now
// removed:
//
//   - Auth-context / computed claims (acr, amr, auth_time, nonce, at_hash, sid).
//     These are on the jwt.reservedClaims denylist precisely because "forging
//     them defeats step-up and nonce binding" — an operator setting acr to an
//     MFA value without MFA, or a static nonce, forges the authentication
//     context. They must only ever be issuer-stamped from the real auth event,
//     never operator-configurable. Their presence here directly contradicted
//     that denylist.
//   - PII (email, phone, name, given_name, family_name, picture, locale,
//     fullname). RFC 9068 §6 least-disclosure: access tokens travel to resource
//     servers and must not carry personal data. For ID tokens these are already
//     delivered by the OIDC profile/email/phone SCOPES (OIDC Core §5.4), so a
//     second config path is redundant and divergence-prone.
//   - permissions. Authorization here is DB-driven (PermissionMiddleware), so a
//     permissions claim in a token is stale-prone and bloats it; excluded to
//     keep tokens minimal and avoid anyone enforcing on a stale claim.
//
// What remains — roles and tenant_id — are authorization/tenancy context (RFC
// 9068 §2.2.3.1), always resolved from real per-request values, never operator
// literals. This set is a strict subset of jwt.reservedClaims, so the allowlist
// and the denylist no longer disagree.
func knownTokenClaim(claim string) bool {
	switch claim {
	case "roles", "tenant_id":
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
