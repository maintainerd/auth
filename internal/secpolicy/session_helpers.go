package secpolicy

// ShouldRevokeSessionsOnPasswordChange reads the tenant session config and
// returns whether sessions should be revoked on password change. Returns true
// (the secure default) when the repo is nil, the settings are missing, or the
// field is absent/unreadable.
func ShouldRevokeSessionsOnPasswordChange(repo SecuritySettingRepository, tenantID int64) bool {
	if repo == nil {
		return true
	}
	ss, err := repo.FindByTenantID(tenantID)
	if err != nil || ss == nil {
		return true
	}
	cfg, err := NormalizeSecuritySettingConfig("session", mapFromJSON(ss.SessionConfig), nil)
	if err != nil {
		return true
	}
	if v, ok := cfg["revoke_sessions_on_password_change"]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return true
}
