package migration

import "gorm.io/gorm"

func CreateUserMFATOTPSecretsTable(db *gorm.DB) error {
	return db.Exec(`
CREATE TABLE IF NOT EXISTS user_mfa_totp_secrets (
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
    digits           INTEGER     NOT NULL DEFAULT 6,
    period           INTEGER     NOT NULL DEFAULT 30,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX IF NOT EXISTS uq_user_mfa_totp_secrets_user_id ON user_mfa_totp_secrets(user_id);
CREATE INDEX IF NOT EXISTS idx_user_mfa_totp_secrets_user_id ON user_mfa_totp_secrets(user_id);

-- Keep users.is_totp_enabled / first_mfa_enrolled_at in sync. Fires on
-- INSERT/UPDATE/DELETE because a pending secret is inserted (is_enabled=false)
-- at enrollment-begin and enable/disable is an is_enabled UPDATE. See
-- sync_totp_flag() in 024_create_users_table.go.
DROP TRIGGER IF EXISTS trg_sync_totp_flag ON user_mfa_totp_secrets;
CREATE TRIGGER trg_sync_totp_flag
    AFTER INSERT OR UPDATE OR DELETE ON user_mfa_totp_secrets
    FOR EACH ROW EXECUTE FUNCTION sync_totp_flag();
`).Error
}
