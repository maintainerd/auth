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
    oauth_refresh_token_id   BIGSERIAL    PRIMARY KEY,
    oauth_refresh_token_uuid UUID         NOT NULL DEFAULT gen_random_uuid() UNIQUE,
    token_hash               TEXT         NOT NULL UNIQUE,
    family_id                UUID         NOT NULL,
    client_id                BIGINT       NOT NULL,
    user_id                  BIGINT       NOT NULL,
    tenant_id                BIGINT       NOT NULL,
    scope                    TEXT[]       NOT NULL DEFAULT '{}',
    is_revoked               BOOLEAN      NOT NULL DEFAULT FALSE,
    revoked_at               TIMESTAMPTZ,
    expires_at               TIMESTAMPTZ  NOT NULL,
    last_used_at             TIMESTAMPTZ,
    -- Caller context recorded at issuance for the admin session console.
    ip_address               INET,
    user_agent               TEXT,
    -- Auth strength carried forward so introspection avoids a join.
    acr                      VARCHAR(10)  NOT NULL DEFAULT '1',
    amr                      TEXT[]       NOT NULL DEFAULT '{}',
    -- RFC 9449 §5: when the token set was issued to a DPoP-proofing client, the
    -- refresh token is bound to that client's key thumbprint (jkt) and may only
    -- be redeemed by a caller proving possession of the SAME key. NULL means the
    -- token was issued without DPoP and stays a bearer credential.
    dpop_jkt                 TEXT,
    -- The user_sessions row this token belongs to, so ending ONE session revokes
    -- exactly its refresh tokens and leaves the user's other browsers and mobile
    -- devices alone. NULL means the token predates session binding or was issued
    -- outside a browser session.
    user_session_uuid        UUID,
    created_at               TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_oauth_refresh_revoked CHECK (
        (is_revoked = FALSE AND revoked_at IS NULL) OR
        (is_revoked = TRUE  AND revoked_at IS NOT NULL)
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
CREATE INDEX IF NOT EXISTS idx_oauth_refresh_token_hash    ON oauth_refresh_tokens (token_hash);
CREATE INDEX IF NOT EXISTS idx_oauth_refresh_family        ON oauth_refresh_tokens (family_id);
CREATE INDEX IF NOT EXISTS idx_oauth_refresh_user_client   ON oauth_refresh_tokens (user_id, client_id);
CREATE INDEX IF NOT EXISTS idx_oauth_refresh_expires       ON oauth_refresh_tokens (expires_at);
CREATE INDEX IF NOT EXISTS idx_oauth_refresh_revoked       ON oauth_refresh_tokens (is_revoked) WHERE is_revoked = FALSE;
CREATE INDEX IF NOT EXISTS idx_oauth_refresh_tokens_scope  ON oauth_refresh_tokens USING GIN (scope);
CREATE INDEX IF NOT EXISTS idx_oauth_refresh_tokens_active ON oauth_refresh_tokens (user_id, client_id) WHERE is_revoked = false;
CREATE INDEX IF NOT EXISTS idx_oauth_refresh_amr           ON oauth_refresh_tokens USING GIN (amr);
CREATE INDEX IF NOT EXISTS idx_oauth_refresh_session       ON oauth_refresh_tokens (user_session_uuid) WHERE user_session_uuid IS NOT NULL;
`
	return db.Exec(sql).Error
}
