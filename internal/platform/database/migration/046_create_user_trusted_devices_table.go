package migration

import "gorm.io/gorm"

func CreateUserTrustedDevicesTable(db *gorm.DB) error {
	return db.Exec(`
CREATE TABLE IF NOT EXISTS user_trusted_devices (
    user_trusted_device_id   BIGSERIAL    PRIMARY KEY,
    user_trusted_device_uuid UUID         NOT NULL UNIQUE DEFAULT gen_random_uuid(),
    user_id                  BIGINT       NOT NULL,
    tenant_id                BIGINT       NOT NULL,
    device_fingerprint       VARCHAR(255) NOT NULL,
    device_token_hash        VARCHAR(255) NOT NULL DEFAULT '',
    device_name              VARCHAR(255),
    location                 VARCHAR(255),
    ip_address               INET,
    user_agent               TEXT,
    trusted_until            TIMESTAMPTZ  NOT NULL,
    last_seen_at             TIMESTAMPTZ,
    created_at               TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at               TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at               TIMESTAMPTZ,
    CONSTRAINT fk_user_trusted_devices_user FOREIGN KEY (user_id)
        REFERENCES users(user_id) ON DELETE CASCADE,
    CONSTRAINT fk_user_trusted_devices_tenant FOREIGN KEY (tenant_id)
        REFERENCES tenants(tenant_id) ON DELETE CASCADE
);
CREATE UNIQUE INDEX IF NOT EXISTS uq_user_trusted_devices_user_fingerprint
    ON user_trusted_devices (user_id, device_fingerprint) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_user_trusted_devices_token
    ON user_trusted_devices (user_id, device_token_hash) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_user_trusted_devices_user_id
    ON user_trusted_devices (user_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_user_trusted_devices_trusted_until
    ON user_trusted_devices (trusted_until) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_user_trusted_devices_active
    ON user_trusted_devices (tenant_id, user_id) WHERE deleted_at IS NULL;
`).Error
}
