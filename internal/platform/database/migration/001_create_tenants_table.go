package migration

import (
	"gorm.io/gorm"
)

func CreateTenantTable(db *gorm.DB) error {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS tenants (
    tenant_id      BIGSERIAL PRIMARY KEY,
    tenant_uuid    UUID NOT NULL UNIQUE,
    name           VARCHAR(63) NOT NULL,
    display_name   VARCHAR(255),
    description    TEXT,
    status         VARCHAR(20) NOT NULL DEFAULT 'active',
    is_system      BOOLEAN NOT NULL DEFAULT FALSE,
    metadata       JSONB NOT NULL DEFAULT '{}',
    created_by     BIGINT,
    updated_by     BIGINT,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at     TIMESTAMPTZ
);

-- ADD INDEXES
CREATE INDEX IF NOT EXISTS idx_tenants_uuid ON tenants (tenant_uuid);
-- name is the unique, DNS-safe subdomain slug ({tenant}.auth.maintainerd.local).
CREATE UNIQUE INDEX IF NOT EXISTS uq_tenants_name ON tenants (name) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_tenants_status ON tenants (status);
CREATE INDEX IF NOT EXISTS idx_tenants_is_system ON tenants (is_system);
-- Singleton guarantee: at most one live system tenant can ever exist (the root).
CREATE UNIQUE INDEX IF NOT EXISTS uq_tenants_single_system ON tenants (is_system) WHERE is_system = TRUE AND deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_tenants_metadata ON tenants USING GIN (metadata);
CREATE INDEX IF NOT EXISTS idx_tenants_created_at ON tenants (created_at);
CREATE INDEX IF NOT EXISTS idx_tenants_deleted_at ON tenants (deleted_at) WHERE deleted_at IS NULL;
`
	return db.Exec(sql).Error
}
