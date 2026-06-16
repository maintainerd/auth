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
    name           VARCHAR(255) NOT NULL,
    display_name   VARCHAR(255),
    description    TEXT,
    identifier     VARCHAR(255) NOT NULL,
    status         VARCHAR(20) DEFAULT 'active',
    is_system      BOOLEAN DEFAULT FALSE,
    -- is_completed marks a tenant as fully provisioned. Regular tenants default
    -- to TRUE (usable immediately on creation). The system tenant is created
    -- with FALSE during bootstrap and flipped TRUE once it has an admin + owner.
    is_completed   BOOLEAN NOT NULL DEFAULT TRUE,
    metadata       JSONB DEFAULT '{}',
    created_by     BIGINT,
    updated_by     BIGINT,
    created_at     TIMESTAMPTZ DEFAULT now(),
    updated_at     TIMESTAMPTZ DEFAULT now(),
    deleted_at     TIMESTAMPTZ
);

-- ADD INDEXES
CREATE INDEX IF NOT EXISTS idx_tenants_uuid ON tenants (tenant_uuid);
CREATE INDEX IF NOT EXISTS idx_tenants_name ON tenants (name);
CREATE UNIQUE INDEX IF NOT EXISTS uq_tenants_identifier ON tenants (identifier) WHERE deleted_at IS NULL;
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
