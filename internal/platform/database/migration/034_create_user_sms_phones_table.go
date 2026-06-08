package migration

import "gorm.io/gorm"

// CreateUserSMSPhonesTable stores verified SMS MFA phone numbers.
// One row per user — UNIQUE constraint on user_id ensures a single
// active MFA phone at any time. The users.phone column remains
// informational (profile display, notifications) and is unrelated.
func CreateUserSMSPhonesTable(db *gorm.DB) error {
	return db.Exec(`
CREATE TABLE IF NOT EXISTS user_sms_phones (
    mfa_phone_id   BIGSERIAL PRIMARY KEY,
    mfa_phone_uuid UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),
    user_id        BIGINT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    phone          VARCHAR(20) NOT NULL,
    is_verified    BOOLEAN NOT NULL DEFAULT FALSE,
    verified_at    TIMESTAMPTZ,
    last_used_at   TIMESTAMPTZ,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX IF NOT EXISTS uq_user_sms_phones_user_id ON user_sms_phones(user_id);
CREATE INDEX IF NOT EXISTS idx_user_sms_phones_phone ON user_sms_phones(phone);
`).Error
}
