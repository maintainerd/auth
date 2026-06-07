package secpolicy

import "github.com/maintainerd/auth/internal/platform/security"

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
		return security.DefaultPasswordPolicy()
	}
	ss, err := repo.FindByTenantID(tenantID)
	if err != nil || ss == nil {
		return security.DefaultPasswordPolicy()
	}
	return security.MergePasswordPolicy([]byte(ss.PasswordConfig))
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
