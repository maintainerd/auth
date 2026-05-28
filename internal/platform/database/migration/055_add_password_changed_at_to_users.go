package migration

import "gorm.io/gorm"

// AddPasswordChangedAtToUsers adds the password_changed_at column to the users
// table. Used to enforce PasswordPolicy.ExpiryDays (forced rotation).
func AddPasswordChangedAtToUsers(db *gorm.DB) error {
	return db.Exec(`
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS password_changed_at TIMESTAMPTZ;
CREATE INDEX IF NOT EXISTS idx_users_password_changed_at ON users(password_changed_at)
    WHERE password_changed_at IS NOT NULL;
`).Error
}
