package secpolicy

import "strings"

// RegistrationPolicy is the effective registration configuration for a tenant,
// resolved from registration_config with secure defaults applied.
type RegistrationPolicy struct {
	SelfRegistrationEnabled           bool
	RequireEmailVerification          bool
	RequirePhoneVerification          bool
	AllowedEmailDomains               []string
	BlockedEmailDomains               []string
	AutoConfirmEnabled                bool
	VerificationTokenTTLHours         int
	CaptchaOnSignup                   bool
	RegistrationRateLimitPerIPPerHour int
}

// LoadRegistrationPolicy returns the effective registration policy for a tenant,
// falling back to the seeded defaults when settings are missing or unreadable.
func LoadRegistrationPolicy(repo SecuritySettingRepository, tenantID int64) *RegistrationPolicy {
	cfg, _ := DefaultSecuritySettingConfig("registration")
	if repo != nil {
		if ss, err := repo.FindByTenantID(tenantID); err == nil && ss != nil {
			if merged, err := NormalizeSecuritySettingConfig("registration", mapFromJSON(ss.RegistrationConfig), nil); err == nil {
				cfg = merged
			}
		}
	}
	return mapToRegistrationPolicy(cfg)
}

func mapToRegistrationPolicy(cfg map[string]any) *RegistrationPolicy {
	p := &RegistrationPolicy{
		SelfRegistrationEnabled:           true,
		RequireEmailVerification:          true,
		VerificationTokenTTLHours:         24,
		CaptchaOnSignup:                   true,
		RegistrationRateLimitPerIPPerHour: 10,
	}
	if v, ok := cfg["self_registration_enabled"]; ok {
		p.SelfRegistrationEnabled = boolValue(v)
	}
	if v, ok := cfg["require_email_verification"]; ok {
		p.RequireEmailVerification = boolValue(v)
	}
	if v, ok := cfg["require_phone_verification"]; ok {
		p.RequirePhoneVerification = boolValue(v)
	}
	if v, ok := cfg["allowed_email_domains"]; ok {
		p.AllowedEmailDomains = stringSliceValue(v)
	}
	if v, ok := cfg["blocked_email_domains"]; ok {
		p.BlockedEmailDomains = stringSliceValue(v)
	}
	if v, ok := cfg["auto_confirm_enabled"]; ok {
		p.AutoConfirmEnabled = boolValue(v)
	}
	if v, ok := cfg["verification_token_ttl_hours"]; ok {
		p.VerificationTokenTTLHours = intValue(v)
	}
	if p.VerificationTokenTTLHours <= 0 {
		p.VerificationTokenTTLHours = 24
	}
	if v, ok := cfg["captcha_on_signup"]; ok {
		p.CaptchaOnSignup = boolValue(v)
	}
	if v, ok := cfg["registration_rate_limit_per_ip_per_hour"]; ok {
		p.RegistrationRateLimitPerIPPerHour = intValue(v)
	}
	if p.RegistrationRateLimitPerIPPerHour <= 0 {
		p.RegistrationRateLimitPerIPPerHour = 10
	}
	return p
}

// EmailVerified reports whether a freshly-registered account should be treated
// as email-verified given this policy. Auto-confirm short-circuits verification;
// otherwise an account is verified only when verification is not required.
func (p *RegistrationPolicy) EmailVerified() bool {
	if p == nil {
		return false
	}
	return p.AutoConfirmEnabled || !p.RequireEmailVerification
}

func (p *RegistrationPolicy) InitialUserStatus(email string) string {
	if p == nil || p.EmailVerified() || strings.TrimSpace(email) == "" {
		return "active"
	}
	return "pending"
}

// EmailDomainAllowed validates an email address against the allow/block lists.
// Blocklist takes precedence; when the allowlist is non-empty, the domain must
// match an entry. Entries may be exact ("acme.com") or wildcard ("*.acme.com").
// An empty email passes (domain rules only apply when an email is supplied).
func (p *RegistrationPolicy) EmailDomainAllowed(email string) bool {
	if p == nil {
		return true
	}
	at := strings.LastIndex(email, "@")
	if at < 0 {
		return true
	}
	domain := strings.ToLower(strings.TrimSpace(email[at+1:]))
	if domain == "" {
		return true
	}
	for _, blocked := range p.BlockedEmailDomains {
		if domainMatches(blocked, domain) {
			return false
		}
	}
	if len(p.AllowedEmailDomains) == 0 {
		return true
	}
	for _, allowed := range p.AllowedEmailDomains {
		if domainMatches(allowed, domain) {
			return true
		}
	}
	return false
}

func domainMatches(pattern, domain string) bool {
	pattern = strings.ToLower(strings.TrimSpace(pattern))
	if pattern == "" {
		return false
	}
	if strings.HasPrefix(pattern, "*.") {
		suffix := strings.TrimPrefix(pattern, "*.")
		return domain == suffix || strings.HasSuffix(domain, "."+suffix)
	}
	return domain == pattern
}
