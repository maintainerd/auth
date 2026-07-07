package migration

import "gorm.io/gorm"

func CreateSCIMConfigurationsTable(db *gorm.DB) error {
	return db.Exec(`
CREATE TABLE IF NOT EXISTS scim_configurations (
    scim_configuration_id       BIGSERIAL    PRIMARY KEY,
    scim_configuration_uuid     UUID         NOT NULL UNIQUE DEFAULT gen_random_uuid(),
    tenant_id                   BIGINT       NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
    identity_provider_id        BIGINT       REFERENCES identity_providers(identity_provider_id) ON DELETE SET NULL,
    display_name                VARCHAR(255) NOT NULL,
    base_url                    VARCHAR(2048),
    bearer_token_hash           VARCHAR(255),
    sync_users                  BOOLEAN      NOT NULL DEFAULT TRUE,
    sync_groups                 BOOLEAN      NOT NULL DEFAULT FALSE,
    sync_direction              VARCHAR(20)  NOT NULL DEFAULT 'inbound',
    attribute_mapping           JSONB        NOT NULL DEFAULT '{}',
    is_active                   BOOLEAN      NOT NULL DEFAULT TRUE,
    last_sync_at                TIMESTAMPTZ,
    last_sync_status            VARCHAR(20),
    last_sync_error             TEXT,
    created_by                  BIGINT       REFERENCES users(user_id) ON DELETE SET NULL,
    updated_by                  BIGINT       REFERENCES users(user_id) ON DELETE SET NULL,
    created_at                  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at                  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at                  TIMESTAMPTZ,
    CONSTRAINT chk_scim_configurations_sync_direction CHECK (sync_direction IN ('inbound', 'outbound', 'bidirectional')),
    CONSTRAINT chk_scim_configurations_last_sync_status CHECK (
        last_sync_status IS NULL OR last_sync_status IN ('success', 'partial', 'failed')
    )
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_scim_configurations_tenant
    ON scim_configurations (tenant_id) WHERE deleted_at IS NULL AND is_active = TRUE;
CREATE INDEX IF NOT EXISTS idx_scim_configurations_tenant
    ON scim_configurations (tenant_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_scim_configurations_identity_provider
    ON scim_configurations (identity_provider_id) WHERE identity_provider_id IS NOT NULL AND deleted_at IS NULL;
`).Error
}
