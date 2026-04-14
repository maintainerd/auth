package migration

import (
	"gorm.io/gorm"
)

// CreateOAuthAuthorizationCodesTable creates the oauth_authorization_codes
// table which stores pending authorization codes. Codes are short-lived
// (10 minutes), single-use, and bound to a PKCE challenge.
func CreateOAuthAuthorizationCodesTable(db *gorm.DB) error {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS oauth_authorization_codes (
    oauth_authorization_code_id   BIGSERIAL     PRIMARY KEY,
    oauth_authorization_code_uuid UUID          NOT NULL DEFAULT gen_random_uuid() UNIQUE,
    code_hash                     TEXT          NOT NULL UNIQUE,
    client_id                     INTEGER       NOT NULL,
    user_id                       INTEGER       NOT NULL,
    tenant_id                     INTEGER       NOT NULL,
    redirect_uri                  TEXT          NOT NULL,
    scope                         TEXT          NOT NULL DEFAULT '',
    state                         TEXT,
    nonce                         TEXT,
    code_challenge                TEXT          NOT NULL,
    code_challenge_method         VARCHAR(10)   NOT NULL DEFAULT 'S256',
    is_used                       BOOLEAN       NOT NULL DEFAULT FALSE,
    used_at                       TIMESTAMPTZ,
    expires_at                    TIMESTAMPTZ   NOT NULL,
    created_at                    TIMESTAMPTZ   NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_oauth_auth_code_method CHECK (code_challenge_method IN ('S256'))
);

-- ADD CONSTRAINTS (FOREIGN KEYS)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_oauth_auth_codes_client_id'
    ) THEN
        ALTER TABLE oauth_authorization_codes
            ADD CONSTRAINT fk_oauth_auth_codes_client_id FOREIGN KEY (client_id)
            REFERENCES clients(client_id) ON DELETE CASCADE;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_oauth_auth_codes_user_id'
    ) THEN
        ALTER TABLE oauth_authorization_codes
            ADD CONSTRAINT fk_oauth_auth_codes_user_id FOREIGN KEY (user_id)
            REFERENCES users(user_id) ON DELETE CASCADE;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_oauth_auth_codes_tenant_id'
    ) THEN
        ALTER TABLE oauth_authorization_codes
            ADD CONSTRAINT fk_oauth_auth_codes_tenant_id FOREIGN KEY (tenant_id)
            REFERENCES tenants(tenant_id) ON DELETE CASCADE;
    END IF;
END$$;

-- ADD INDEXES
CREATE INDEX IF NOT EXISTS idx_oauth_auth_codes_code_hash ON oauth_authorization_codes (code_hash);
CREATE INDEX IF NOT EXISTS idx_oauth_auth_codes_expires ON oauth_authorization_codes (expires_at);
CREATE INDEX IF NOT EXISTS idx_oauth_auth_codes_client_user ON oauth_authorization_codes (client_id, user_id);
`
	return db.Exec(sql).Error
}
