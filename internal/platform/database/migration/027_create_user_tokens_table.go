package migration

import (
	"gorm.io/gorm"
)

// CreateUserTokenTable stores short-lived URL tokens and session records for users.
// Token types (enforced by chk_user_tokens_token_type):
//   - user:session            — live login session (rerouted to user_sessions after §3.15)
//   - user:email:verification — link emailed to user to verify email address
//   - user:password:reset     — link emailed to user to reset password
//   - user:magic_link         — link for passwordless login
//   - user:mfa:trusted_device — cookie for MFA step-down (rerouted to user_trusted_devices after §3.12)
func CreateUserTokenTable(db *gorm.DB) error {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS user_tokens (
    user_token_id         BIGSERIAL PRIMARY KEY,
    user_token_uuid       UUID NOT NULL UNIQUE,
    user_id               BIGINT NOT NULL,
    token_type            VARCHAR(50) NOT NULL, -- 'user:session', 'user:email:verification', 'user:password:reset', 'user:magic_link', 'user:mfa:trusted_device'
    token                 TEXT NOT NULL, -- hashed token string
    user_agent            TEXT,
    ip_address            VARCHAR(50),
    expires_at            TIMESTAMPTZ,
    is_revoked            BOOLEAN NOT NULL DEFAULT FALSE,

    -- Session-specific fields. NULL for non-session token types.
    last_used_at          TIMESTAMPTZ,
    idle_timeout_seconds  INTEGER,
    absolute_expires_at   TIMESTAMPTZ,

    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ADD CONSTRAINTS
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_user_tokens_user'
    ) THEN
        ALTER TABLE user_tokens
            ADD CONSTRAINT fk_user_tokens_user FOREIGN KEY (user_id)
            REFERENCES users(user_id) ON DELETE CASCADE;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'chk_user_tokens_token_type'
    ) THEN
        ALTER TABLE user_tokens ADD CONSTRAINT chk_user_tokens_token_type
            CHECK (token_type IN (
                'user:session',
                'user:email:verification',
                'user:password:reset',
                'user:magic_link',
                'user:mfa:trusted_device'
            ));
    END IF;
END$$;

-- ADD INDEXES
CREATE INDEX IF NOT EXISTS idx_user_tokens_uuid ON user_tokens (user_token_uuid);
CREATE INDEX IF NOT EXISTS idx_user_tokens_user_id ON user_tokens (user_id);
CREATE INDEX IF NOT EXISTS idx_user_tokens_token_type ON user_tokens (token_type);
CREATE UNIQUE INDEX IF NOT EXISTS idx_user_tokens_token_unique ON user_tokens (token);
CREATE INDEX IF NOT EXISTS idx_user_tokens_created_at ON user_tokens (created_at);
CREATE INDEX IF NOT EXISTS idx_user_tokens_expires_at ON user_tokens (expires_at) WHERE expires_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_user_tokens_session_active ON user_tokens (user_id, token_type, is_revoked, absolute_expires_at);
CREATE INDEX IF NOT EXISTS idx_user_tokens_active ON user_tokens (user_id, token_type) WHERE is_revoked = false;
`
	return db.Exec(sql).Error
}
