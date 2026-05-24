package migration

import "gorm.io/gorm"

// CreateUserPasswordHistoryTable stores previous password hashes per user so
// services can enforce PasswordPolicy.HistoryCount (no re-use of last N passwords).
func CreateUserPasswordHistoryTable(db *gorm.DB) error {
	return db.Exec(`
CREATE TABLE IF NOT EXISTS user_password_history (
    history_id    BIGSERIAL PRIMARY KEY,
    user_id       BIGINT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    password_hash TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_uph_user_id_created ON user_password_history(user_id, created_at DESC);
`).Error
}
