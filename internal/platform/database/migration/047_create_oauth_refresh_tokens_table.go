package migration

import (
	"gorm.io/gorm"
)

// CreateOAuthRefreshTokensTable creates the oauth_refresh_tokens table which
// stores refresh tokens with family tracking for rotation and reuse detection.
func CreateOAuthRefreshTokensTable(db *gorm.DB) error {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS oauth_refresh_tokens (
    oauth_refresh_token_id   BIGSERIAL     PRIMARY KEY,
    oauth_refresh_token_uuid UUID          NOT NULL DEFAULT gen_random_uuid() UNIQUE,
    token_hash               TEXT          NOT NULL UNIQUE,
    family_id                UUID          NOT NULL,
    client_id                INTEGER       NOT NULL,
    user_id                  INTEGER       NOT NULL,
    tenant_id                INTEGER       NOT NULL,
    scope                    TEXT          NOT NULL DEFAULT '',
    is_revoked               BOOLEAN       NOT NULL DEFAULT FALSE,
    revoked_at               TIMESTAMPTZ,
    expires_at               TIMESTAMPTZ   NOT NULL,
    last_used_at             TIMESTAMPTZ,
    created_at               TIMESTAMPTZ   NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_oauth_refresh_revoked CHECK (
        (is_revoked = FALSE AND revoked_at IS NULL) OR
        (is_revoked = TRUE AND revoked_at IS NOT NULL)
    )
);

-- ADD CONSTRAINTS (FOREIGN KEYS)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_oauth_refresh_tokens_client_id'
    ) THEN
        ALTER TABLE oauth_refresh_tokens
            ADD CONSTRAINT fk_oauth_refresh_tokens_client_id FOREIGN KEY (client_id)
            REFERENCES clients(client_id) ON DELETE CASCADE;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_oauth_refresh_tokens_user_id'
    ) THEN
        ALTER TABLE oauth_refresh_tokens
            ADD CONSTRAINT fk_oauth_refresh_tokens_user_id FOREIGN KEY (user_id)
            REFERENCES users(user_id) ON DELETE CASCADE;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_oauth_refresh_tokens_tenant_id'
    ) THEN
        ALTER TABLE oauth_refresh_tokens
            ADD CONSTRAINT fk_oauth_refresh_tokens_tenant_id FOREIGN KEY (tenant_id)
            REFERENCES tenants(tenant_id) ON DELETE CASCADE;
    END IF;
END$$;

-- ADD INDEXES
CREATE INDEX IF NOT EXISTS idx_oauth_refresh_token_hash ON oauth_refresh_tokens (token_hash);
CREATE INDEX IF NOT EXISTS idx_oauth_refresh_family ON oauth_refresh_tokens (family_id);
CREATE INDEX IF NOT EXISTS idx_oauth_refresh_user_client ON oauth_refresh_tokens (user_id, client_id);
CREATE INDEX IF NOT EXISTS idx_oauth_refresh_expires ON oauth_refresh_tokens (expires_at);
CREATE INDEX IF NOT EXISTS idx_oauth_refresh_revoked ON oauth_refresh_tokens (is_revoked) WHERE is_revoked = FALSE;
`
	return db.Exec(sql).Error
}
