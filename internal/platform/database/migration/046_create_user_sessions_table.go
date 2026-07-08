package migration

import "gorm.io/gorm"

func CreateUserSessionsTable(db *gorm.DB) error {
	return db.Exec(`
CREATE TABLE IF NOT EXISTS user_sessions (
    user_session_id      BIGSERIAL    PRIMARY KEY,
    user_session_uuid    UUID         NOT NULL UNIQUE DEFAULT gen_random_uuid(),
    user_id              BIGINT       NOT NULL,
    tenant_id            BIGINT       NOT NULL,
    client_id            BIGINT,
    identity_provider_id BIGINT,
    auth_time            TIMESTAMPTZ  NOT NULL DEFAULT now(),
    ip_address           INET,
    user_agent           TEXT,
    amr                  TEXT[]       NOT NULL DEFAULT '{}',
    acr                  VARCHAR(10)  NOT NULL DEFAULT '1',
    idp_session_id       VARCHAR(255),
    idle_timeout_seconds INT          NOT NULL DEFAULT 1800,
    last_active_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    expires_at           TIMESTAMPTZ  NOT NULL,
    revoked_at           TIMESTAMPTZ,
    revoked_reason       VARCHAR(50),
    created_at           TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT fk_user_sessions_user FOREIGN KEY (user_id)
        REFERENCES users(user_id) ON DELETE CASCADE,
    CONSTRAINT fk_user_sessions_tenant FOREIGN KEY (tenant_id)
        REFERENCES tenants(tenant_id) ON DELETE CASCADE,
    CONSTRAINT fk_user_sessions_client FOREIGN KEY (client_id)
        REFERENCES clients(client_id) ON DELETE SET NULL,
    CONSTRAINT fk_user_sessions_identity_provider FOREIGN KEY (identity_provider_id)
        REFERENCES identity_providers(identity_provider_id) ON DELETE SET NULL,
    CONSTRAINT chk_user_sessions_revoked_reason CHECK (
        revoked_reason IS NULL OR revoked_reason IN (
            'logout', 'admin_revoke', 'password_change', 'mfa_change',
            'session_expired', 'concurrent_limit', 'suspicious_activity'
        )
    )
);
CREATE INDEX IF NOT EXISTS idx_user_sessions_user_active
    ON user_sessions (user_id, created_at ASC) WHERE revoked_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_user_sessions_tenant_user
    ON user_sessions (tenant_id, user_id) WHERE revoked_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_user_sessions_expires_at
    ON user_sessions (expires_at) WHERE revoked_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_user_sessions_last_active_at
    ON user_sessions (user_id, last_active_at DESC) WHERE revoked_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_user_sessions_client_id
    ON user_sessions (client_id) WHERE revoked_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_user_sessions_idp_session_id
    ON user_sessions (idp_session_id) WHERE idp_session_id IS NOT NULL;
`).Error
}
