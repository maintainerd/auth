package user

import "gorm.io/gorm"

// UserPasswordHistoryRepository manages previously used password hashes for
// a user so services can enforce PasswordPolicy.HistoryCount.
type UserPasswordHistoryRepository interface {
	WithTx(tx *gorm.DB) UserPasswordHistoryRepository
	// AddEntry inserts a new hash record for the user.
	AddEntry(userID int64, hash string) error
	// FindRecentHashes returns the most recent `count` hashes for the user,
	// ordered newest first.
	FindRecentHashes(userID int64, count int) ([]string, error)
	// PruneExcess deletes all but the most recent `keepCount` records for the user.
	PruneExcess(userID int64, keepCount int) error
}

type userPasswordHistoryRepository struct {
	db *gorm.DB
}

func NewUserPasswordHistoryRepository(db *gorm.DB) UserPasswordHistoryRepository {
	return &userPasswordHistoryRepository{db: db}
}

func (r *userPasswordHistoryRepository) WithTx(tx *gorm.DB) UserPasswordHistoryRepository {
	return &userPasswordHistoryRepository{db: tx}
}

func (r *userPasswordHistoryRepository) AddEntry(userID int64, hash string) error {
	entry := UserPasswordHistory{
		UserID:       userID,
		PasswordHash: hash,
	}
	return r.db.Create(&entry).Error
}

func (r *userPasswordHistoryRepository) FindRecentHashes(userID int64, count int) ([]string, error) {
	var entries []UserPasswordHistory
	err := r.db.
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(count).
		Find(&entries).Error
	if err != nil {
		return nil, err
	}
	hashes := make([]string, len(entries))
	for i, e := range entries {
		hashes[i] = e.PasswordHash
	}
	return hashes, nil
}

func (r *userPasswordHistoryRepository) PruneExcess(userID int64, keepCount int) error {
	// Delete all rows except the keepCount most recent ones.
	return r.db.Exec(`
DELETE FROM user_password_history
WHERE user_id = ?
  AND history_id NOT IN (
      SELECT history_id FROM user_password_history
      WHERE user_id = ?
      ORDER BY created_at DESC
      LIMIT ?
  )`, userID, userID, keepCount).Error
}
