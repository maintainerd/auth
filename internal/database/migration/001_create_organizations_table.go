package migration

import (
	"log"

	"gorm.io/gorm"
)

func CreateOrganizationTable(db *gorm.DB) {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS organizations (
    organization_id			SERIAL PRIMARY KEY,
    organization_uuid		UUID NOT NULL UNIQUE,
    name								VARCHAR(255) NOT NULL,
    description					TEXT,
    email								VARCHAR(255),
    phone								VARCHAR(50),
    is_active						BOOLEAN DEFAULT TRUE,
    created_at					TIMESTAMPTZ DEFAULT now(),
    updated_at					TIMESTAMPTZ DEFAULT now()
);

-- ADD INDEXES
CREATE INDEX IF NOT EXISTS idx_organizations_uuid ON organizations (organization_uuid);
CREATE INDEX IF NOT EXISTS idx_organizations_name ON organizations (name);
CREATE INDEX IF NOT EXISTS idx_organizations_email ON organizations (email);
CREATE INDEX IF NOT EXISTS idx_organizations_phone ON organizations (phone);
CREATE INDEX IF NOT EXISTS idx_organizations_is_active ON organizations (is_active);
CREATE INDEX IF NOT EXISTS idx_organizations_created_at ON organizations (created_at);
`
	if err := db.Exec(sql).Error; err != nil {
		log.Fatalf("❌ Failed to run migration 001_create_organizations_table: %v", err)
	}

	log.Println("✅ Migration 001_create_organizations_table executed")
}
