package secpolicy

import (
	"encoding/json"

	"github.com/maintainerd/auth/internal/platform/security"
)

// PasswordHistoryRecorder is the minimal password-history writer needed by
// domains that hash a new user password.
type PasswordHistoryRecorder interface {
	AddEntry(userID int64, passwordHash string) error
	PruneExcess(userID int64, keep int) error
}

// LoadPasswordPolicy returns the effective password policy for a tenant,
// falling back to defaults when settings are missing or unreadable.
func LoadPasswordPolicy(repo SecuritySettingRepository, tenantID int64) security.PasswordPolicy {
	if repo == nil {
		return defaultPasswordPolicy()
	}
	ss, err := repo.FindByTenantID(tenantID)
	if err != nil || ss == nil {
		return defaultPasswordPolicy()
	}
	cfg, err := NormalizeSecuritySettingConfig("password", canonicalPasswordConfigAliases(mapFromJSON(ss.PasswordConfig)), nil)
	if err != nil {
		return defaultPasswordPolicy()
	}
	b, err := json.Marshal(cfg)
	if err != nil {
		return defaultPasswordPolicy()
	}
	return security.MergePasswordPolicy(b)
}

// RecordPasswordHistory stores the latest password hash and prunes older
// entries according to the configured history count.
func RecordPasswordHistory(repo PasswordHistoryRecorder, userID int64, historyCount int, newHash string) {
	if repo == nil || historyCount <= 0 {
		return
	}
	_ = repo.AddEntry(userID, newHash)
	_ = repo.PruneExcess(userID, historyCount)
}

func defaultPasswordPolicy() security.PasswordPolicy {
	cfg := MustDefaultSecuritySettingConfig("password")
	b, err := json.Marshal(cfg)
	if err != nil {
		return security.DefaultPasswordPolicy()
	}
	return security.MergePasswordPolicy(b)
}

func mapFromJSON(raw []byte) map[string]any {
	out := map[string]any{}
	if len(raw) == 0 {
		return out
	}
	_ = json.Unmarshal(raw, &out)
	return out
}

func canonicalPasswordConfigAliases(cfg map[string]any) map[string]any {
	if cfg == nil {
		return nil
	}
	if v, ok := cfg["require_upper"]; ok {
		cfg["require_uppercase"] = v
	}
	if v, ok := cfg["require_lower"]; ok {
		cfg["require_lowercase"] = v
	}
	if v, ok := cfg["require_digit"]; ok {
		cfg["require_number"] = v
	}
	if v, ok := cfg["require_special"]; ok {
		cfg["require_symbol"] = v
	}
	if v, ok := cfg["blocklist_enabled"]; ok {
		cfg["reject_common_passwords"] = v
	}
	if v, ok := cfg["history_count"]; ok {
		cfg["password_history_count"] = v
	}
	if v, ok := cfg["expiry_days"]; ok {
		cfg["max_age_days"] = v
	}
	return cfg
}
