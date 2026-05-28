package service

import (
	"errors"

	"github.com/maintainerd/auth/internal/platform/security"
	"github.com/maintainerd/auth/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

var errPasswordReused = errors.New("password was used recently and cannot be reused")

// loadPolicy returns the effective PasswordPolicy for a tenant.
// Falls back to DefaultPasswordPolicy when the repo is nil or no setting exists.
func loadPolicy(repo repository.SecuritySettingRepository, tenantID int64) security.PasswordPolicy {
	if repo == nil {
		return security.DefaultPasswordPolicy()
	}
	ss, err := repo.FindDefaultByTenantID(tenantID)
	if err != nil || ss == nil {
		return security.DefaultPasswordPolicy()
	}
	return security.MergePasswordPolicy([]byte(ss.PasswordConfig))
}

// checkPasswordHistory verifies the new plain-text password does not match any
// of the user's recent hashes. Returns an error if it does.
// A nil repo or a policy HistoryCount of 0 skips the check.
func checkPasswordHistory(repo repository.UserPasswordHistoryRepository, userID int64, historyCount int, newPassword string) error {
	if repo == nil || historyCount <= 0 {
		return nil
	}
	hashes, err := repo.FindRecentHashes(userID, historyCount)
	if err != nil {
		return err
	}
	for _, h := range hashes {
		if bcrypt.CompareHashAndPassword([]byte(h), []byte(newPassword)) == nil {
			return errPasswordReused
		}
	}
	return nil
}

// recordPasswordHistory adds the new hash to history and prunes excess entries.
// A nil repo or HistoryCount of 0 skips the operation (no-op).
func recordPasswordHistory(repo repository.UserPasswordHistoryRepository, userID int64, historyCount int, newHash string) {
	if repo == nil || historyCount <= 0 {
		return
	}
	_ = repo.AddEntry(userID, newHash)
	_ = repo.PruneExcess(userID, historyCount)
}
