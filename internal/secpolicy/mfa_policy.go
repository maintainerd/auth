package secpolicy

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
func LoadMFAPolicy(repo SecuritySettingRepository, tenantID int64) *MFAPolicy {
	if repo == nil {
		return nil
	}
	cfg, _ := DefaultSecuritySettingConfig("mfa")
	if ss, err := repo.FindByTenantID(tenantID); err == nil && ss != nil {
		raw := mapFromJSON(ss.MFAConfig)
		if merged, err := NormalizeSecuritySettingConfig("mfa", raw, nil); err == nil {
			cfg = merged
		} else {
			// Enforcement must honor already-stored tenant policy even when an
			// older row does not satisfy today's write-time validation.
			for k := range cfg {
				if v, ok := raw[k]; ok {
					cfg[k] = v
				}
			}
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
