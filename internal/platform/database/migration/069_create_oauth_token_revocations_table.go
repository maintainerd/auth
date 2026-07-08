package migration

import "gorm.io/gorm"

func CreateOAuthTokenRevocationsTable(db *gorm.DB) error {
	return db.Exec(`
CREATE TABLE IF NOT EXISTS oauth_token_revocations (
    oauth_token_revocation_id   BIGSERIAL    PRIMARY KEY,
    oauth_token_revocation_uuid UUID         NOT NULL UNIQUE DEFAULT gen_random_uuid(),
    tenant_id                   BIGINT       NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
    jti                         VARCHAR(255) NOT NULL,
    token_type                  VARCHAR(20)  NOT NULL DEFAULT 'access_token',
    revoked_by_user_id          BIGINT       REFERENCES users(user_id) ON DELETE SET NULL,
    revoked_by_client_id        BIGINT       REFERENCES clients(client_id) ON DELETE SET NULL,
    reason                      VARCHAR(100) NOT NULL DEFAULT 'logout',
    expires_at                  TIMESTAMPTZ  NOT NULL,
    created_at                  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT uq_oauth_token_revocations_jti UNIQUE (jti),
    CONSTRAINT chk_oauth_token_revocations_type CHECK (token_type IN ('access_token', 'id_token')),
    CONSTRAINT chk_oauth_token_revocations_reason CHECK (reason IN ('logout', 'password_change', 'admin_revoke', 'security_event'))
);
CREATE INDEX IF NOT EXISTS idx_oauth_token_revocations_tenant_jti
    ON oauth_token_revocations (tenant_id, jti);
CREATE INDEX IF NOT EXISTS idx_oauth_token_revocations_expires_at
    ON oauth_token_revocations (expires_at);
`).Error
}
