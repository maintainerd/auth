package migration

import (
	"log"

	"gorm.io/gorm"
)

func CreateOrganizationTable(db *gorm.DB) {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS organizations (
    organization_id       SERIAL PRIMARY KEY,
    organization_uuid     UUID NOT NULL UNIQUE,
    name                  VARCHAR(255) NOT NULL,
    description           TEXT,
    email                 VARCHAR(255),
    phone_number          VARCHAR(50),
    website_url           TEXT,
    logo_url              TEXT,
    external_reference_id VARCHAR(255), -- Optional for external integrations
    is_default            BOOLEAN DEFAULT FALSE,
    is_active             BOOLEAN DEFAULT TRUE,
    created_at            TIMESTAMPTZ DEFAULT now(),
    updated_at            TIMESTAMPTZ DEFAULT now()
);

-- ADD INDEXES
CREATE INDEX IF NOT EXISTS idx_organizations_organization_uuid ON organizations (organization_uuid);
CREATE INDEX IF NOT EXISTS idx_organizations_name ON organizations (name);
CREATE INDEX IF NOT EXISTS idx_organizations_email ON organizations (email);
CREATE INDEX IF NOT EXISTS idx_organizations_phone_number ON organizations (phone_number);
CREATE INDEX IF NOT EXISTS idx_organizations_is_active ON organizations (is_active);
CREATE INDEX IF NOT EXISTS idx_organizations_is_default ON organizations (is_default);
CREATE INDEX IF NOT EXISTS idx_organizations_external_reference_id ON organizations (external_reference_id);
`
	if err := db.Exec(sql).Error; err != nil {
		log.Fatalf("❌ Failed to run migration 003_create_organizations_table: %v", err)
	}

	log.Println("✅ Migration 003_create_organizations_table executed")
}
