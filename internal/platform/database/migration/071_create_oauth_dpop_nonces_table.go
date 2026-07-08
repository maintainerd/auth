package migration

import "gorm.io/gorm"

func CreateOAuthDPoPNoncesTable(db *gorm.DB) error {
	return db.Exec(`
CREATE TABLE IF NOT EXISTS oauth_dpop_nonces (
    oauth_dpop_nonce_id     BIGSERIAL    PRIMARY KEY,
    oauth_dpop_nonce_uuid   UUID         NOT NULL UNIQUE DEFAULT gen_random_uuid(),
    tenant_id               BIGINT       NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
    client_id               BIGINT       NOT NULL REFERENCES clients(client_id) ON DELETE CASCADE,
    nonce                   VARCHAR(512) NOT NULL UNIQUE,
    used_at                 TIMESTAMPTZ,
    expires_at              TIMESTAMPTZ  NOT NULL,
    created_at              TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_oauth_dpop_nonces_nonce
    ON oauth_dpop_nonces (nonce) WHERE used_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_oauth_dpop_nonces_expires_at
    ON oauth_dpop_nonces (expires_at) WHERE used_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_oauth_dpop_nonces_client
    ON oauth_dpop_nonces (client_id, expires_at);
`).Error
}
