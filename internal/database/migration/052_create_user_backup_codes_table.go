package migration

import "gorm.io/gorm"

// CreateUserBackupCodesTable stores hashed one-time backup codes for account recovery.
// code_hash is TEXT (not VARCHAR) to accommodate any hash algorithm output length.
func CreateUserBackupCodesTable(db *gorm.DB) error {
	return db.Exec(`
CREATE TABLE IF NOT EXISTS user_backup_codes (
    backup_code_id   BIGSERIAL PRIMARY KEY,
    backup_code_uuid UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),
    user_id          BIGINT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    code_hash        TEXT NOT NULL,
    used             BOOLEAN NOT NULL DEFAULT FALSE,
    used_at          TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_user_backup_codes_user_id ON user_backup_codes(user_id);
CREATE INDEX IF NOT EXISTS idx_user_backup_codes_code_hash ON user_backup_codes(code_hash);
`).Error
}
