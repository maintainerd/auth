package migration

import "gorm.io/gorm"

func CreateUserMFAEmailsTable(db *gorm.DB) error {
	sql := `
CREATE TABLE IF NOT EXISTS user_mfa_emails (
    mfa_email_id   BIGSERIAL PRIMARY KEY,
    mfa_email_uuid UUID NOT NULL UNIQUE,
    user_id        BIGINT NOT NULL,
    email          VARCHAR(255) NOT NULL,
    is_verified    BOOLEAN NOT NULL DEFAULT FALSE,
    verified_at    TIMESTAMPTZ,
    last_used_at   TIMESTAMPTZ,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT fk_user_mfa_emails_user_id FOREIGN KEY (user_id)
        REFERENCES users(user_id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_user_mfa_emails_user_id ON user_mfa_emails (user_id);
CREATE INDEX IF NOT EXISTS idx_user_mfa_emails_uuid ON user_mfa_emails (mfa_email_uuid);
`
	return db.Exec(sql).Error
}
