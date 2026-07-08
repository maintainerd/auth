package migration

import "gorm.io/gorm"

func CreateUserLockoutsTable(db *gorm.DB) error {
	return db.Exec(`
CREATE TABLE IF NOT EXISTS user_lockouts (
    user_lockout_id   BIGSERIAL    PRIMARY KEY,
    user_lockout_uuid UUID         NOT NULL UNIQUE DEFAULT gen_random_uuid(),
    tenant_id         BIGINT       NOT NULL,
    user_id           BIGINT,
    identifier        VARCHAR(255) NOT NULL,
    failed_count      INTEGER      NOT NULL DEFAULT 0,
    last_failed_at    TIMESTAMPTZ,
    locked_until      TIMESTAMPTZ,
    ip_address        INET,
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT fk_user_lockouts_tenant FOREIGN KEY (tenant_id)
        REFERENCES tenants(tenant_id) ON DELETE CASCADE,
    CONSTRAINT fk_user_lockouts_user FOREIGN KEY (user_id)
        REFERENCES users(user_id) ON DELETE CASCADE
);
CREATE UNIQUE INDEX IF NOT EXISTS uq_user_lockouts_tenant_identifier
    ON user_lockouts (tenant_id, identifier);
CREATE INDEX IF NOT EXISTS idx_user_lockouts_user_id
    ON user_lockouts (user_id) WHERE user_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_user_lockouts_locked_until
    ON user_lockouts (locked_until) WHERE locked_until IS NOT NULL;
`).Error
}
