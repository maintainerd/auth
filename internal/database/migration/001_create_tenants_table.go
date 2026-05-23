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
    is_public      BOOLEAN DEFAULT FALSE,
    is_system      BOOLEAN DEFAULT FALSE,
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
CREATE INDEX IF NOT EXISTS idx_tenants_is_public ON tenants (is_public);
CREATE INDEX IF NOT EXISTS idx_tenants_is_system ON tenants (is_system);
CREATE INDEX IF NOT EXISTS idx_tenants_metadata ON tenants USING GIN (metadata);
CREATE INDEX IF NOT EXISTS idx_tenants_created_at ON tenants (created_at);
CREATE INDEX IF NOT EXISTS idx_tenants_deleted_at ON tenants (deleted_at) WHERE deleted_at IS NULL;
`
	return db.Exec(sql).Error
}
