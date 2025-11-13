package migration

import (
	"log"

	"gorm.io/gorm"
)

func CreateTenantTable(db *gorm.DB) {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS tenants (
    tenant_id				SERIAL PRIMARY KEY,
    tenant_uuid			UUID NOT NULL UNIQUE,
    name						VARCHAR(255) NOT NULL,
    description			TEXT,
    identifier			VARCHAR(255) NOT NULL UNIQUE,
    is_active				BOOLEAN DEFAULT FALSE,
    is_public				BOOLEAN DEFAULT FALSE,
    is_default			BOOLEAN DEFAULT FALSE,
    is_system				BOOLEAN DEFAULT FALSE,
    metadata				JSONB DEFAULT '{}',
    created_at			TIMESTAMPTZ DEFAULT now(),
    updated_at			TIMESTAMPTZ DEFAULT now()
);

-- ADD INDEXES
CREATE INDEX IF NOT EXISTS idx_tenants_uuid ON tenants (tenant_uuid);
CREATE INDEX IF NOT EXISTS idx_tenants_name ON tenants (name);
CREATE INDEX IF NOT EXISTS idx_tenants_identifier ON tenants (identifier);
CREATE INDEX IF NOT EXISTS idx_tenants_is_active ON tenants (is_active);
CREATE INDEX IF NOT EXISTS idx_tenants_is_public ON tenants (is_public);
CREATE INDEX IF NOT EXISTS idx_tenants_is_default ON tenants (is_default);
CREATE INDEX IF NOT EXISTS idx_tenants_is_system ON tenants (is_system);
CREATE INDEX IF NOT EXISTS idx_tenants_metadata ON tenants USING GIN (metadata);
CREATE INDEX IF NOT EXISTS idx_tenants_created_at ON tenants (created_at);
`
	if err := db.Exec(sql).Error; err != nil {
		log.Fatalf("❌ Failed to run migration 001_create_tenants_table: %v", err)
	}

	log.Println("✅ Migration 001_create_tenants_table executed")
}
