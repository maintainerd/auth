package secpolicy

import (
	"log/slog"
	"time"

	"github.com/maintainerd/maintainerd-auth/internal/platform/security"
)

// LoadLockoutPolicy returns the effective lockout (rate-limit) policy for a
// tenant, falling back to the shipped defaults when settings are missing or
// unreadable.
//
// It must never return nil on a failure. The login path treats a nil policy as
// "lockout is off" and early-returns before ever counting a failed attempt, so
// the previous behaviour — nil on any repo error or normalization failure —
// meant a single stale config row or one transient database blip silently
// disabled account lockout for the whole tenant. Online password guessing then
// became unlimited while the admin UI still displayed max_failed_attempts: 5.
// Falling back to the shipped defaults keeps the control ON through a fault,
// which is the same call every sibling loader makes (LoadPasswordPolicy,
// LoadMFAPolicy), and each degradation is logged so it is not silent.
func LoadLockoutPolicy(repo SecuritySettingRepository, tenantID int64) *security.RateLimitConfig {
	if repo == nil {
		// Not an anomaly: several call sites are constructed without a settings
		// repository and are documented to run on defaults.
		return defaultLockoutConfig()
	}
	ss, err := repo.FindByTenantID(tenantID)
	if err != nil {
		return lockoutPolicyFallback(tenantID, "security settings lookup failed", err)
	}
	if ss == nil {
		// A tenant with no row yet is expected during provisioning.
		return defaultLockoutConfig()
	}
	cfg, err := NormalizeSecuritySettingConfig("lockout", mapFromJSON(ss.LockoutConfig), nil)
	if err != nil {
		return lockoutPolicyFallback(tenantID, "stored lockout config is not valid", err)
	}
	return mapToLockoutConfig(cfg)
}

// lockoutPolicyFallback logs the degradation and returns the shipped defaults.
func lockoutPolicyFallback(tenantID int64, reason string, err error) *security.RateLimitConfig {
	slog.Warn("lockout policy falling back to defaults; the tenant's configured lockout thresholds are NOT being enforced",
		"tenant_id", tenantID,
		"reason", reason,
		"error", err,
	)
	return defaultLockoutConfig()
}

// DefaultLockoutConfig is the shipped tenant baseline (enabled, 5 attempts,
// 30-minute progressive lockout). Exported so bootstrap paths that run before a
// tenant's settings exist are held to the same threshold.
func DefaultLockoutConfig() *security.RateLimitConfig {
	return defaultLockoutConfig()
}

func defaultLockoutConfig() *security.RateLimitConfig {
	return mapToLockoutConfig(MustDefaultSecuritySettingConfig("lockout"))
}

func mapToLockoutConfig(cfg map[string]any) *security.RateLimitConfig {
	rc := &security.RateLimitConfig{
		Enabled:             true,
		ResetCountOnSuccess: true,
		AutoUnlock:          true,
	}
	if v, ok := cfg["enabled"]; ok {
		rc.Enabled = boolValue(v)
	}
	if v, ok := cfg["max_failed_attempts"]; ok {
		rc.MaxFailedAttempts = intValue(v)
	}
	if v, ok := cfg["lockout_duration_minutes"]; ok {
		rc.LockoutDuration = time.Duration(intValue(v)) * time.Minute
	}
	if v, ok := cfg["observation_window_minutes"]; ok {
		rc.ObservationWindow = time.Duration(intValue(v)) * time.Minute
	}
	if v, ok := cfg["auto_unlock"]; ok {
		rc.AutoUnlock = boolValue(v)
	}
	if v, ok := cfg["reset_count_on_success"]; ok {
		rc.ResetCountOnSuccess = boolValue(v)
	}
	if v, ok := cfg["notify_user_on_lockout"]; ok {
		rc.NotifyUserOnLockout = boolValue(v)
	}
	if v, ok := cfg["progressive_lockout"]; ok {
		rc.ProgressiveLockout = boolValue(v)
	}
	if v, ok := cfg["max_lockout_duration_minutes"]; ok {
		rc.MaxLockoutDuration = time.Duration(intValue(v)) * time.Minute
	}
	if v, ok := cfg["progression_reset_hours"]; ok {
		rc.ProgressionReset = time.Duration(intValue(v)) * time.Hour
	}
	return rc
}
