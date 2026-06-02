package migration

import "gorm.io/gorm"

// AddSMSOTPFailedAttempts tracks wrong-code attempts and allows OTP invalidation
// after repeated failures.
func AddSMSOTPFailedAttempts(db *gorm.DB) error {
	return db.Exec(`
ALTER TABLE sms_otps
    ADD COLUMN IF NOT EXISTS failed_attempts INTEGER NOT NULL DEFAULT 0;
`).Error
}
