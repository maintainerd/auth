package secpolicy

import "log/slog"

// MFAPolicy is the effective MFA configuration for a tenant, resolved from
// security_settings.mfa_config with seeded defaults applied. It serves gaps
// #3 (TOTP params), #4 (recovery codes), #5 (allow_sms enrollment gate),
// #6 (sensitive-action step-up), #7 (trusted device), #8 (grace periods).
type MFAPolicy struct {
	Mode                          string
	AllowedMethods                []string
	TOTPIssuer                    string
	TrustedDevicePeriodDays       int
	GracePeriodDays               int
	PreferredMethod               string
	AllowSMS                      bool
	AllowEmailOTP                 bool
	TOTPDigits                    int
	TOTPPeriodSeconds             int
	RecoveryCodesCount            int
	RequireMFAForSensitiveActions bool
	AdminGracePeriodDays          int
	StepUpTTLMinutes              int
}

// LoadMFAPolicy returns the effective MFA policy for a tenant, falling
// back to the seeded defaults when settings are missing or unreadable.
// Returns nil when repo is nil (no security-settings available — all
// methods are allowed; callers must treat nil as permissive).
//
// The default mode is "optional", so any fallback here is a DOWNGRADE: a tenant
// configured mode "enforced" degrades to optional and users with no enrolled
// factor walk straight in. That is the correct availability trade (a settings
// outage must not lock every user out of their account) but it was previously
// silent — the lookup error was discarded by `err == nil && ss != nil` and
// nothing was logged, so the only symptom of a database blip was MFA quietly
// not being required. Every degradation now says so, matching LoadPasswordPolicy
// and LoadLockoutPolicy.
func LoadMFAPolicy(repo SecuritySettingRepository, tenantID int64) *MFAPolicy {
	if repo == nil {
		// Not an anomaly: several call sites are constructed without a settings
		// repository and are documented to treat nil as permissive.
		return nil
	}
	cfg, _ := DefaultSecuritySettingConfig("mfa")
	ss, err := repo.FindByTenantID(tenantID)
	switch {
	case err != nil:
		mfaPolicyDegraded(tenantID, "security settings lookup failed", err)
	case ss == nil:
		// A tenant with no row yet is expected during provisioning.
	default:
		raw := mapFromJSON(ss.MFAConfig)
		if merged, nerr := NormalizeSecuritySettingConfig("mfa", raw, nil); nerr == nil {
			cfg = merged
		} else {
			// Enforcement must honor already-stored tenant policy even when an
			// older row does not satisfy today's write-time validation.
			for k := range cfg {
				if v, ok := raw[k]; ok {
					cfg[k] = v
				}
			}
			mfaPolicyDegraded(tenantID, "stored MFA config does not satisfy current validation; enforcing stored values over defaults where present", nerr)
		}
	}
	return &MFAPolicy{
		Mode:                          stringValue(cfg["mode"]),
		AllowedMethods:                normalizeMFAMethods(stringSliceValue(cfg["allowed_methods"])),
		TOTPIssuer:                    stringValue(cfg["totp_issuer"]),
		TrustedDevicePeriodDays:       intValue(cfg["trusted_device_period_days"]),
		GracePeriodDays:               intValue(cfg["grace_period_days"]),
		PreferredMethod:               stringValue(cfg["preferred_method"]),
		AllowSMS:                      boolValue(cfg["allow_sms"]),
		AllowEmailOTP:                 boolValue(cfg["allow_email_otp"]),
		TOTPDigits:                    intValue(cfg["totp_digits"]),
		TOTPPeriodSeconds:             intValue(cfg["totp_period_seconds"]),
		RecoveryCodesCount:            intValue(cfg["recovery_codes_count"]),
		RequireMFAForSensitiveActions: boolValue(cfg["require_mfa_for_sensitive_actions"]),
		AdminGracePeriodDays:          intValue(cfg["admin_grace_period_days"]),
		StepUpTTLMinutes:              stepUpTTLWithDefault(intValue(cfg["step_up_ttl_minutes"])),
	}
}

// mfaPolicyDegraded records that the tenant's configured MFA policy could not be
// read or trusted, so enforcement may be weaker than the admin UI shows.
func mfaPolicyDegraded(tenantID int64, reason string, err error) {
	slog.Warn("MFA policy degraded; the tenant's configured MFA mode may NOT be enforced (defaults are mode=optional)",
		"tenant_id", tenantID,
		"reason", reason,
		"error", err,
	)
}

func stepUpTTLWithDefault(v int) int {
	if v <= 0 {
		return 5
	}
	return v
}

func (p *MFAPolicy) StepUpTTLSeconds() int64 {
	if p == nil {
		return 300
	}
	if p.StepUpTTLMinutes <= 0 {
		return 300
	}
	return int64(p.StepUpTTLMinutes) * 60
}

func normalizeMFAMethods(methods []string) []string {
	out := make([]string, 0, len(methods))
	for _, m := range methods {
		if m == "recovery_code" {
			m = "backup_code"
		}
		out = append(out, m)
	}
	return out
}
