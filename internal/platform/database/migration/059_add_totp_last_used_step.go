package migration

import "gorm.io/gorm"

// AddTOTPLastUsedStep stores the last accepted TOTP counter so codes cannot
// be replayed within the validity window.
func AddTOTPLastUsedStep(db *gorm.DB) error {
	return db.Exec(`
ALTER TABLE user_totp_secrets
    ADD COLUMN IF NOT EXISTS last_used_step BIGINT;
`).Error
}
