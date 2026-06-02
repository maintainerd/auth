package authn

import (
	"errors"

	"github.com/maintainerd/auth/internal/platform/security"
)

var errPasswordReused = errors.New("password was used recently and cannot be reused")

// checkPasswordHistory verifies the new plain-text password does not match any
// of the user's recent hashes. Returns an error if it does.
// A nil repo or a policy HistoryCount of 0 skips the check.
func checkPasswordHistory(repo UserPasswordHistoryRepository, userID int64, historyCount int, newPassword string) error {
	if repo == nil || historyCount <= 0 {
		return nil
	}
	hashes, err := repo.FindRecentHashes(userID, historyCount)
	if err != nil {
		return err
	}
	for _, h := range hashes {
		if security.ComparePassword([]byte(h), []byte(newPassword)) {
			return errPasswordReused
		}
	}
	return nil
}
