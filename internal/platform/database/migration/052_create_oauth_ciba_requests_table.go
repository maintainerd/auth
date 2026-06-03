package migration

import "gorm.io/gorm"

// CreateOAuthCIBARequestsTable creates the oauth_ciba_requests table which
// stores Client-Initiated Backchannel Authentication requests. The server
// notifies the identified user out-of-band; the client polls /oauth/token
// with the auth_req_id until the user approves, denies, or the request expires.
func CreateOAuthCIBARequestsTable(db *gorm.DB) error {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS oauth_ciba_requests (
    oauth_ciba_request_id   BIGSERIAL    PRIMARY KEY,
    oauth_ciba_request_uuid UUID         NOT NULL DEFAULT gen_random_uuid() UNIQUE,
    auth_req_id_hash        TEXT         NOT NULL UNIQUE,
    client_id               BIGINT       NOT NULL,
    tenant_id               BIGINT       NOT NULL,
    user_id                 BIGINT,
    scope                   TEXT         NOT NULL DEFAULT '',
    binding_message         TEXT,
    auth_acr                VARCHAR(32),
    auth_amr                JSONB        NOT NULL DEFAULT '[]'::jsonb,
    status                  VARCHAR(20)  NOT NULL DEFAULT 'pending',
    interval                SMALLINT     NOT NULL DEFAULT 5,
    last_poll_at            TIMESTAMPTZ,
    notification_sent_at    TIMESTAMPTZ,
    expires_at              TIMESTAMPTZ  NOT NULL,
    created_at              TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_oauth_ciba_requests_status CHECK (status IN ('pending', 'approved', 'denied', 'expired'))
);

-- ADD CONSTRAINTS (FOREIGN KEYS)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_oauth_ciba_requests_client_id'
    ) THEN
        ALTER TABLE oauth_ciba_requests
            ADD CONSTRAINT fk_oauth_ciba_requests_client_id FOREIGN KEY (client_id)
            REFERENCES clients(client_id) ON DELETE CASCADE;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_oauth_ciba_requests_tenant_id'
    ) THEN
        ALTER TABLE oauth_ciba_requests
            ADD CONSTRAINT fk_oauth_ciba_requests_tenant_id FOREIGN KEY (tenant_id)
            REFERENCES tenants(tenant_id) ON DELETE CASCADE;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_oauth_ciba_requests_user_id'
    ) THEN
        ALTER TABLE oauth_ciba_requests
            ADD CONSTRAINT fk_oauth_ciba_requests_user_id FOREIGN KEY (user_id)
            REFERENCES users(user_id) ON DELETE SET NULL;
    END IF;
END$$;

-- ADD INDEXES
CREATE INDEX IF NOT EXISTS idx_oauth_ciba_requests_uuid       ON oauth_ciba_requests (oauth_ciba_request_uuid);
CREATE INDEX IF NOT EXISTS idx_oauth_ciba_requests_client_id  ON oauth_ciba_requests (client_id);
CREATE INDEX IF NOT EXISTS idx_oauth_ciba_requests_user_id    ON oauth_ciba_requests (user_id);
CREATE INDEX IF NOT EXISTS idx_oauth_ciba_requests_expires_at ON oauth_ciba_requests (expires_at);
CREATE INDEX IF NOT EXISTS idx_oauth_ciba_requests_status     ON oauth_ciba_requests (status);
`
	return db.Exec(sql).Error
}
