package migration

import "gorm.io/gorm"

func CreateOAuthTokenExchangesTable(db *gorm.DB) error {
	return db.Exec(`
CREATE TABLE IF NOT EXISTS oauth_token_exchanges (
    oauth_token_exchange_id     BIGSERIAL    PRIMARY KEY,
    oauth_token_exchange_uuid   UUID         NOT NULL UNIQUE DEFAULT gen_random_uuid(),
    tenant_id                   BIGINT       NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
    actor_client_id             BIGINT       NOT NULL REFERENCES clients(client_id) ON DELETE CASCADE,
    subject_token_type          VARCHAR(100) NOT NULL,
    requested_token_type        VARCHAR(100) NOT NULL,
    subject_user_id             BIGINT       REFERENCES users(user_id) ON DELETE SET NULL,
    subject_client_id           BIGINT       REFERENCES clients(client_id) ON DELETE SET NULL,
    issued_jti                  VARCHAR(255),
    exchange_type               VARCHAR(20)  NOT NULL,
    scope                       TEXT[]       NOT NULL DEFAULT '{}',
    ip_address                  INET,
    created_at                  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT chk_oauth_token_exchanges_type CHECK (exchange_type IN ('impersonation', 'delegation'))
);
CREATE INDEX IF NOT EXISTS idx_oauth_token_exchanges_tenant_created
    ON oauth_token_exchanges (tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_oauth_token_exchanges_actor_client
    ON oauth_token_exchanges (actor_client_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_oauth_token_exchanges_subject_user
    ON oauth_token_exchanges (subject_user_id, created_at DESC) WHERE subject_user_id IS NOT NULL;

CREATE OR REPLACE FUNCTION prevent_oauth_token_exchanges_mutation() RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'oauth_token_exchanges rows are immutable';
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_oauth_token_exchanges_immutable ON oauth_token_exchanges;
CREATE TRIGGER trg_oauth_token_exchanges_immutable
    BEFORE UPDATE OR DELETE ON oauth_token_exchanges
    FOR EACH ROW EXECUTE FUNCTION prevent_oauth_token_exchanges_mutation();
`).Error
}
