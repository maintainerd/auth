package migration

import "gorm.io/gorm"

func CreateUserTOTPSecretsTable(db *gorm.DB) error {
	return db.Exec(`
CREATE TABLE IF NOT EXISTS user_totp_secrets (
    totp_secret_id   BIGSERIAL PRIMARY KEY,
    totp_secret_uuid UUID        NOT NULL UNIQUE DEFAULT gen_random_uuid(),
    user_id          BIGINT      NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    -- Base32-encoded TOTP secret. At rest this should be encrypted via KMS
    -- in production; for now stored as plaintext pending KMS integration.
    secret           TEXT        NOT NULL,
    is_enabled       BOOLEAN     NOT NULL DEFAULT FALSE,
    -- Set to now() when the user completes enrollment (verifies first code).
    enrolled_at      TIMESTAMPTZ,
    last_used_at     TIMESTAMPTZ,
    last_used_step   BIGINT,
    digits           INTEGER    NOT NULL DEFAULT 6,
    period           INTEGER    NOT NULL DEFAULT 30,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX IF NOT EXISTS uq_user_totp_secrets_user_id ON user_totp_secrets(user_id);
CREATE INDEX IF NOT EXISTS idx_user_totp_secrets_user_id ON user_totp_secrets(user_id);
`).Error
}
