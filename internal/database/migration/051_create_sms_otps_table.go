package migration

import "gorm.io/gorm"

// CreateSMSOtpsTable stores short-lived SMS OTPs for user verification.
// otp_hash is TEXT (not VARCHAR) to accommodate any hash algorithm output length.
func CreateSMSOtpsTable(db *gorm.DB) error {
	return db.Exec(`
CREATE TABLE IF NOT EXISTS sms_otps (
    sms_otp_id      BIGSERIAL PRIMARY KEY,
    sms_otp_uuid    UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),
    user_id         BIGINT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    phone           VARCHAR(20) NOT NULL,
    otp_hash        TEXT NOT NULL,
    expires_at      TIMESTAMPTZ NOT NULL,
    used            BOOLEAN NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_sms_otps_user_id ON sms_otps(user_id);
CREATE INDEX IF NOT EXISTS idx_sms_otps_phone ON sms_otps(phone);
CREATE INDEX IF NOT EXISTS idx_sms_otps_otp_hash ON sms_otps(otp_hash);
CREATE INDEX IF NOT EXISTS idx_sms_otps_expires_at ON sms_otps(expires_at);
`).Error
}
