package secpolicy

import (
	"time"
)

// IPRestrictionRuleResponseDTO is the JSON representation of an IP restriction
// rule.
type IPRestrictionRuleResponseDTO struct {
	IPRestrictionRuleID string    `json:"ip_restriction_rule_id"`
	Description         string    `json:"description"`
	Type                string    `json:"type"`
	IPAddress           string    `json:"ip_address"`
	Status              string    `json:"status"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

// IPRestrictionRuleCreateRequestDTO is the request body for creating an IP
// restriction rule.
type IPRestrictionRuleCreateRequestDTO struct {
	Description string  `json:"description"`
	Type        string  `json:"type"`
	IPAddress   string  `json:"ip_address"`
	Status      *string `json:"status,omitempty"`
}

// IPRestrictionRuleUpdateRequestDTO is the request body for updating an IP
// restriction rule.
type IPRestrictionRuleUpdateRequestDTO struct {
	Description string  `json:"description"`
	Type        string  `json:"type"`
	IPAddress   string  `json:"ip_address"`
	Status      *string `json:"status,omitempty"`
}

// IPRestrictionRuleUpdateStatusRequestDTO is the request body for updating an
// IP restriction rule's status.
type IPRestrictionRuleUpdateStatusRequestDTO struct {
	Status string `json:"status"`
}

// IPRestrictionRuleFilterDTO holds query parameters for listing and filtering
// IP restriction rules.
type IPRestrictionRuleFilterDTO struct {
	Type        *string  `json:"type"`
	Status      []string `json:"status"`
	IPAddress   *string  `json:"ip_address"`
	Description *string  `json:"description"`

	// Pagination and sorting
	PaginationRequestDTO
}

// Security setting config response - returns config directly
type SecuritySettingConfigResponseDTO map[string]any

// Update config request - accepts config directly
type SecuritySettingUpdateConfigRequestDTO map[string]any

type PasswordConfigDTO struct {
	MinLength                      *int    `json:"min_length,omitempty"`
	MaxLength                      *int    `json:"max_length,omitempty"`
	RequireUppercase               *bool   `json:"require_uppercase,omitempty"`
	RequireLowercase               *bool   `json:"require_lowercase,omitempty"`
	RequireNumber                  *bool   `json:"require_number,omitempty"`
	RequireSymbol                  *bool   `json:"require_symbol,omitempty"`
	RejectCommonPasswords          *bool   `json:"reject_common_passwords,omitempty"`
	CheckHIBP                      *bool   `json:"check_hibp,omitempty"`
	PasswordHistoryCount           *int    `json:"password_history_count,omitempty"`
	MaxAgeDays                     *int    `json:"max_age_days,omitempty"`
	TemporaryPasswordValidityHours *int    `json:"temporary_password_validity_hours,omitempty"`
	HashAlgorithm                  *string `json:"hash_algorithm,omitempty"`
	MinStrengthScore               *int    `json:"min_strength_score,omitempty"`
}

type MFAConfigDTO struct {
	Mode                          *string  `json:"mode,omitempty"`
	AllowedMethods                []string `json:"allowed_methods,omitempty"`
	TOTPIssuer                    *string  `json:"totp_issuer,omitempty"`
	TrustedDevicePeriodDays       *int     `json:"trusted_device_period_days,omitempty"`
	GracePeriodDays               *int     `json:"grace_period_days,omitempty"`
	PreferredMethod               *string  `json:"preferred_method,omitempty"`
	AllowSMS                      *bool    `json:"allow_sms,omitempty"`
	TOTPDigits                    *int     `json:"totp_digits,omitempty"`
	TOTPPeriodSeconds             *int     `json:"totp_period_seconds,omitempty"`
	RecoveryCodesCount            *int     `json:"recovery_codes_count,omitempty"`
	RequireMFAForSensitiveActions *bool    `json:"require_mfa_for_sensitive_actions,omitempty"`
	AdminGracePeriodDays          *int     `json:"admin_grace_period_days,omitempty"`
}

type SessionConfigDTO struct {
	AccessTokenTTLMinutes            *int    `json:"access_token_ttl_minutes,omitempty"`
	RefreshTokenTTLDays              *int    `json:"refresh_token_ttl_days,omitempty"`
	MaxConcurrentSessions            *int    `json:"max_concurrent_sessions,omitempty"`
	IdleTimeoutMinutes               *int    `json:"idle_timeout_minutes,omitempty"`
	AbsoluteTimeoutHours             *int    `json:"absolute_timeout_hours,omitempty"`
	RotateRefreshTokens              *bool   `json:"rotate_refresh_tokens,omitempty"`
	RefreshTokenReuseIntervalSeconds *int    `json:"refresh_token_reuse_interval_seconds,omitempty"`
	CookieSecure                     *bool   `json:"cookie_secure,omitempty"`
	CookieHTTPOnly                   *bool   `json:"cookie_http_only,omitempty"`
	CookieSameSite                   *string `json:"cookie_same_site,omitempty"`
	RevokeSessionsOnPasswordChange   *bool   `json:"revoke_sessions_on_password_change,omitempty"`
}

type TokenConfigDTO struct {
	ClockSkewLeewaySeconds      *int     `json:"clock_skew_leeway_seconds,omitempty"`
	AdditionalIDTokenClaims     []string `json:"additional_id_token_claims,omitempty"`
	AdditionalAccessTokenClaims []string `json:"additional_access_token_claims,omitempty"`
	SigningAlgorithm            *string  `json:"signing_algorithm,omitempty"`
	RequirePKCE                 *bool    `json:"require_pkce,omitempty"`
}

type LockoutConfigDTO struct {
	Enabled                   *bool `json:"enabled,omitempty"`
	MaxFailedAttempts         *int  `json:"max_failed_attempts,omitempty"`
	LockoutDurationMinutes    *int  `json:"lockout_duration_minutes,omitempty"`
	ProgressiveLockout        *bool `json:"progressive_lockout,omitempty"`
	AutoUnlock                *bool `json:"auto_unlock,omitempty"`
	ResetCountOnSuccess       *bool `json:"reset_count_on_success,omitempty"`
	ObservationWindowMinutes  *int  `json:"observation_window_minutes,omitempty"`
	MaxLockoutDurationMinutes *int  `json:"max_lockout_duration_minutes,omitempty"`
	ProgressionResetHours     *int  `json:"progression_reset_hours,omitempty"`
	NotifyUserOnLockout       *bool `json:"notify_user_on_lockout,omitempty"`
}

type RegistrationConfigDTO struct {
	SelfRegistrationEnabled           *bool    `json:"self_registration_enabled,omitempty"`
	RequireEmailVerification          *bool    `json:"require_email_verification,omitempty"`
	RequirePhoneVerification          *bool    `json:"require_phone_verification,omitempty"`
	AllowedEmailDomains               []string `json:"allowed_email_domains,omitempty"`
	BlockedEmailDomains               []string `json:"blocked_email_domains,omitempty"`
	AutoConfirmEnabled                *bool    `json:"auto_confirm_enabled,omitempty"`
	VerificationTokenTTLHours         *int     `json:"verification_token_ttl_hours,omitempty"`
	DefaultRole                       *string  `json:"default_role,omitempty"`
	CaptchaOnSignup                   *bool    `json:"captcha_on_signup,omitempty"`
	RegistrationRateLimitPerIPPerHour *int     `json:"registration_rate_limit_per_ip_per_hour,omitempty"`
}

type ThreatConfigDTO struct {
	BruteForceDetectionEnabled             *bool `json:"brute_force_detection_enabled,omitempty"`
	ImpossibleTravelDetectionEnabled       *bool `json:"impossible_travel_detection_enabled,omitempty"`
	NewDeviceNotificationEnabled           *bool `json:"new_device_notification_enabled,omitempty"`
	VelocityCheckEnabled                   *bool `json:"velocity_check_enabled,omitempty"`
	RiskBasedStepUpEnabled                 *bool `json:"risk_based_step_up_enabled,omitempty"`
	CompromisedCredentialMonitoringEnabled *bool `json:"compromised_credential_monitoring_enabled,omitempty"`
	IPReputationCheckEnabled               *bool `json:"ip_reputation_check_enabled,omitempty"`
	BlockTorExitNodes                      *bool `json:"block_tor_exit_nodes,omitempty"`
	RiskStepUpThreshold                    *int  `json:"risk_step_up_threshold,omitempty"`
	RiskBlockThreshold                     *int  `json:"risk_block_threshold,omitempty"`
	VelocityFailuresPerIPPerHour           *int  `json:"velocity_failures_per_ip_per_hour,omitempty"`
}

type SecuritySettingClientOverrides struct {
	AccessTokenTTL         *int
	RefreshTokenTTL        *int
	SessionIdleTimeout     *int
	SessionAbsoluteTimeout *int
	RequiredACR            *string
	RequirePKCE            *bool
}

type EffectiveSessionPolicy struct {
	AccessTokenTTLSeconds            int
	RefreshTokenTTLSeconds           int
	MaxConcurrentSessions            int
	IdleTimeoutSeconds               int
	AbsoluteTimeoutSeconds           int
	RotateRefreshTokens              bool
	RefreshTokenReuseIntervalSeconds int
	CookieSecure                     bool
	CookieHTTPOnly                   bool
	CookieSameSite                   string
	RevokeSessionsOnPasswordChange   bool
	RequiredACR                      string
}

type EffectiveTokenPolicy struct {
	ClockSkewLeewaySeconds      int
	AdditionalIDTokenClaims     []string
	AdditionalAccessTokenClaims []string
	SigningAlgorithm            string
	RequirePKCE                 bool
}
