package secpolicy

import (
	"time"

	"github.com/maintainerd/maintainerd-auth/internal/platform/security"
)

// LoadLockoutPolicy returns the effective lockout (rate-limit) policy for a
// tenant, falling back to defaults when settings are missing or unreadable.
func LoadLockoutPolicy(repo SecuritySettingRepository, tenantID int64) *security.RateLimitConfig {
	if repo == nil {
		return nil
	}
	ss, err := repo.FindByTenantID(tenantID)
	if err != nil || ss == nil {
		return nil
	}
	cfg, err := NormalizeSecuritySettingConfig("lockout", mapFromJSON(ss.LockoutConfig), nil)
	if err != nil {
		return nil
	}
	return mapToLockoutConfig(cfg)
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
