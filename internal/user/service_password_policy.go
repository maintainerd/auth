package user

import (
	"github.com/maintainerd/auth/internal/platform/security"
	"github.com/maintainerd/auth/internal/secpolicy"
)

func loadPolicy(repo secpolicy.SecuritySettingRepository, tenantID int64) security.PasswordPolicy {
	if repo == nil {
		return security.DefaultPasswordPolicy()
	}
	ss, err := repo.FindDefaultByTenantID(tenantID)
	if err != nil || ss == nil {
		return security.DefaultPasswordPolicy()
	}
	return security.MergePasswordPolicy([]byte(ss.PasswordConfig))
}

func recordPasswordHistory(repo UserPasswordHistoryRepository, userID int64, historyCount int, newHash string) {
	if repo == nil || historyCount <= 0 {
		return
	}
	_ = repo.AddEntry(userID, newHash)
	_ = repo.PruneExcess(userID, historyCount)
}
