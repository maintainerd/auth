package migration

import "gorm.io/gorm"

func CreateClientIdentityProvidersTable(db *gorm.DB) error {
	sql := `
CREATE TABLE IF NOT EXISTS client_identity_providers (
    client_identity_provider_id   BIGSERIAL PRIMARY KEY,
    client_identity_provider_uuid UUID NOT NULL UNIQUE,
    tenant_id                     BIGINT NOT NULL,
    client_id                     BIGINT NOT NULL,
    identity_provider_id          BIGINT NOT NULL,
    is_default                    BOOLEAN NOT NULL DEFAULT FALSE,
    enabled                       BOOLEAN NOT NULL DEFAULT TRUE,
    display_order                 INTEGER NOT NULL DEFAULT 0,
    created_by                    BIGINT,
    updated_by                    BIGINT,
    created_at                    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at                    TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at                    TIMESTAMPTZ
);

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_client_identity_providers_tenant'
    ) THEN
        ALTER TABLE client_identity_providers
            ADD CONSTRAINT fk_client_identity_providers_tenant FOREIGN KEY (tenant_id)
            REFERENCES tenants(tenant_id) ON DELETE CASCADE;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_client_identity_providers_client'
    ) THEN
        ALTER TABLE client_identity_providers
            ADD CONSTRAINT fk_client_identity_providers_client FOREIGN KEY (client_id)
            REFERENCES clients(client_id) ON DELETE CASCADE;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_client_identity_providers_idp'
    ) THEN
        ALTER TABLE client_identity_providers
            ADD CONSTRAINT fk_client_identity_providers_idp FOREIGN KEY (identity_provider_id)
            REFERENCES identity_providers(identity_provider_id) ON DELETE CASCADE;
    END IF;
END$$;

CREATE UNIQUE INDEX IF NOT EXISTS uq_client_identity_providers_pair
    ON client_identity_providers (client_id, identity_provider_id)
    WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS uq_client_identity_providers_default
    ON client_identity_providers (client_id)
    WHERE is_default = TRUE AND deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_client_identity_providers_uuid
    ON client_identity_providers (client_identity_provider_uuid);
CREATE INDEX IF NOT EXISTS idx_client_identity_providers_client
    ON client_identity_providers (client_id)
    WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_client_identity_providers_tenant
    ON client_identity_providers (tenant_id);
CREATE INDEX IF NOT EXISTS idx_client_identity_providers_idp
    ON client_identity_providers (identity_provider_id);
CREATE INDEX IF NOT EXISTS idx_client_identity_providers_enabled
    ON client_identity_providers (enabled)
    WHERE enabled = TRUE AND deleted_at IS NULL;
`
	return db.Exec(sql).Error
}
