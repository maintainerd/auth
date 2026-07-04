package migration

import "gorm.io/gorm"

func CreateOAuthAuthorizeRequestsTable(db *gorm.DB) error {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS oauth_authorize_requests (
    oauth_authorize_request_id   BIGSERIAL    PRIMARY KEY,
    oauth_authorize_request_uuid UUID         NOT NULL DEFAULT gen_random_uuid() UNIQUE,
    client_id                    BIGINT       NOT NULL,
    tenant_id                    BIGINT,
    redirect_uri                 VARCHAR(2048) NOT NULL,
    scope                        TEXT[],
    state                        TEXT,
    nonce                        TEXT,
    response_type                VARCHAR(20)  NOT NULL,
    code_challenge               TEXT,
    code_challenge_method        VARCHAR(10),
    screen_hint                  VARCHAR(20),
    registration_flow            TEXT,
    status                       VARCHAR(20)  NOT NULL DEFAULT 'pending',
    expires_at                   TIMESTAMPTZ  NOT NULL,
    consumed_at                  TIMESTAMPTZ,
    created_at                   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at                   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at                   TIMESTAMPTZ
);

-- ADD CONSTRAINTS
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_oauth_authorize_requests_client_id'
    ) THEN
        ALTER TABLE oauth_authorize_requests
            ADD CONSTRAINT fk_oauth_authorize_requests_client_id FOREIGN KEY (client_id)
            REFERENCES clients(client_id) ON DELETE CASCADE;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_oauth_authorize_requests_tenant_id'
    ) THEN
        ALTER TABLE oauth_authorize_requests
            ADD CONSTRAINT fk_oauth_authorize_requests_tenant_id FOREIGN KEY (tenant_id)
            REFERENCES tenants(tenant_id) ON DELETE CASCADE;
    END IF;
END$$;

-- ADD INDEXES
CREATE INDEX IF NOT EXISTS idx_oauth_authorize_requests_uuid       ON oauth_authorize_requests (oauth_authorize_request_uuid);
CREATE INDEX IF NOT EXISTS idx_oauth_authorize_requests_client_id    ON oauth_authorize_requests (client_id);
CREATE INDEX IF NOT EXISTS idx_oauth_authorize_requests_expires_at   ON oauth_authorize_requests (expires_at);
CREATE INDEX IF NOT EXISTS idx_oauth_authorize_requests_deleted_at   ON oauth_authorize_requests (deleted_at) WHERE deleted_at IS NULL;
`
	return db.Exec(sql).Error
}
