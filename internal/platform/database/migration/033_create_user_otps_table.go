package migration

import "gorm.io/gorm"

// CreateUserOTPsTable stores short-lived verification codes across channels.
// The channel column discriminates sms, email, voice, etc. so a single table
// serves every OTP-based verification flow.
func CreateUserOTPsTable(db *gorm.DB) error {
	return db.Exec(`
CREATE TABLE IF NOT EXISTS user_otps (
    user_otp_id     BIGSERIAL PRIMARY KEY,
    user_otp_uuid   UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),
    user_id         BIGINT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    channel         VARCHAR(20) NOT NULL,
    recipient       VARCHAR(255) NOT NULL,
    otp_hash        TEXT NOT NULL,
    metadata        JSONB NOT NULL DEFAULT '{}',
    expires_at      TIMESTAMPTZ NOT NULL,
    used            BOOLEAN NOT NULL DEFAULT FALSE,
    failed_attempts INT NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_user_otps_user_id ON user_otps(user_id);
CREATE INDEX IF NOT EXISTS idx_user_otps_channel_recipient ON user_otps(channel, recipient);
CREATE INDEX IF NOT EXISTS idx_user_otps_otp_hash ON user_otps(otp_hash);
CREATE INDEX IF NOT EXISTS idx_user_otps_expires_at ON user_otps(expires_at);
`).Error
}
