package migration

import (
	"gorm.io/gorm"
)

func CreateIdentityProviderTable(db *gorm.DB) error {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS identity_providers (
    identity_provider_id    BIGSERIAL PRIMARY KEY,
    identity_provider_uuid  UUID NOT NULL UNIQUE,
    tenant_id               BIGINT NOT NULL,
    name                    VARCHAR(100) NOT NULL,
    display_name            TEXT NOT NULL,
    provider                VARCHAR(100) NOT NULL,
    provider_type           VARCHAR(100) NOT NULL,
    identifier              TEXT,
    issuer                          TEXT,
    provider_client_id              TEXT,
    provider_client_secret_encrypted TEXT,
    allow_jit_provisioning          BOOLEAN NOT NULL DEFAULT FALSE,
    config                  JSONB,
    status                  VARCHAR(20) DEFAULT 'inactive',
    is_default              BOOLEAN DEFAULT FALSE,
    is_system               BOOLEAN DEFAULT FALSE,
    created_by              BIGINT,
    updated_by              BIGINT,
    created_at              TIMESTAMPTZ DEFAULT now(),
    updated_at              TIMESTAMPTZ DEFAULT now(),
    deleted_at              TIMESTAMPTZ
);

-- ADD CONSTRAINTS (safe)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_identity_providers_tenant_id'
    ) THEN
        ALTER TABLE identity_providers
            ADD CONSTRAINT fk_identity_providers_tenant_id FOREIGN KEY (tenant_id)
            REFERENCES tenants(tenant_id) ON DELETE CASCADE;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'chk_identity_providers_provider_type'
    ) THEN
        ALTER TABLE identity_providers
            ADD CONSTRAINT chk_identity_providers_provider_type
            CHECK (provider_type IN ('system', 'social', 'enterprise'));
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'chk_identity_providers_status'
    ) THEN
        ALTER TABLE identity_providers
            ADD CONSTRAINT chk_identity_providers_status
            CHECK (status IN ('active', 'inactive'));
    END IF;
END$$;

-- ADD INDEXES
CREATE INDEX IF NOT EXISTS idx_identity_providers_uuid ON identity_providers (identity_provider_uuid);
CREATE UNIQUE INDEX IF NOT EXISTS uq_identity_providers_tenant_name ON identity_providers (tenant_id, name) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_identity_providers_display_name ON identity_providers (display_name);
CREATE INDEX IF NOT EXISTS idx_identity_providers_provider ON identity_providers (provider);
CREATE INDEX IF NOT EXISTS idx_identity_providers_provider_type ON identity_providers (provider_type);
CREATE INDEX IF NOT EXISTS idx_identity_providers_identifier ON identity_providers (identifier);
CREATE UNIQUE INDEX IF NOT EXISTS uq_identity_providers_identifier ON identity_providers (identifier) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_identity_providers_issuer ON identity_providers (issuer) WHERE issuer IS NOT NULL AND deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS uq_identity_providers_issuer ON identity_providers (issuer) WHERE issuer IS NOT NULL AND deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_identity_providers_status ON identity_providers (status);
CREATE INDEX IF NOT EXISTS idx_identity_providers_is_default ON identity_providers (is_default);
CREATE INDEX IF NOT EXISTS idx_identity_providers_is_system ON identity_providers (is_system);
CREATE INDEX IF NOT EXISTS idx_identity_providers_tenant_id ON identity_providers (tenant_id);
CREATE INDEX IF NOT EXISTS idx_identity_providers_created_at ON identity_providers (created_at);
CREATE INDEX IF NOT EXISTS idx_identity_providers_deleted_at ON identity_providers (deleted_at) WHERE deleted_at IS NULL;
`
	return db.Exec(sql).Error
}
