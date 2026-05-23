package migration

import "gorm.io/gorm"

// CreateOAuthPARRequestsTable creates the oauth_par_requests table which stores
// Pushed Authorization Requests (RFC 9126). Each row is a single authorization
// request submitted by a client before the user-facing /oauth/authorize redirect.
func CreateOAuthPARRequestsTable(db *gorm.DB) error {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS oauth_par_requests (
    oauth_par_request_id   BIGSERIAL    PRIMARY KEY,
    oauth_par_request_uuid UUID         NOT NULL DEFAULT gen_random_uuid() UNIQUE,
    request_uri_hash       TEXT         NOT NULL UNIQUE,
    client_id              BIGINT       NOT NULL,
    tenant_id              BIGINT       NOT NULL,
    response_type          VARCHAR(20)  NOT NULL DEFAULT 'code',
    redirect_uri           TEXT         NOT NULL,
    scope                  TEXT         NOT NULL DEFAULT '',
    state                  TEXT,
    nonce                  TEXT,
    code_challenge         TEXT         NOT NULL,
    code_challenge_method  VARCHAR(10)  NOT NULL DEFAULT 'S256',
    is_used                BOOLEAN      NOT NULL DEFAULT false,
    expires_at             TIMESTAMPTZ  NOT NULL,
    created_at             TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_oauth_par_code_challenge_method CHECK (code_challenge_method IN ('S256'))
);

-- ADD CONSTRAINTS (FOREIGN KEYS)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_oauth_par_requests_client_id'
    ) THEN
        ALTER TABLE oauth_par_requests
            ADD CONSTRAINT fk_oauth_par_requests_client_id FOREIGN KEY (client_id)
            REFERENCES clients(client_id) ON DELETE CASCADE;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_oauth_par_requests_tenant_id'
    ) THEN
        ALTER TABLE oauth_par_requests
            ADD CONSTRAINT fk_oauth_par_requests_tenant_id FOREIGN KEY (tenant_id)
            REFERENCES tenants(tenant_id) ON DELETE CASCADE;
    END IF;
END$$;

-- ADD INDEXES
CREATE INDEX IF NOT EXISTS idx_oauth_par_requests_uuid       ON oauth_par_requests (oauth_par_request_uuid);
CREATE INDEX IF NOT EXISTS idx_oauth_par_requests_client_id  ON oauth_par_requests (client_id);
CREATE INDEX IF NOT EXISTS idx_oauth_par_requests_expires_at ON oauth_par_requests (expires_at);
`
	return db.Exec(sql).Error
}
