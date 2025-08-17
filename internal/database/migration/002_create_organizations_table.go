package migration

import (
	"log"

	"gorm.io/gorm"
)

func CreateOrganizationTable(db *gorm.DB) {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS auth_containers (
    auth_container_id   SERIAL PRIMARY KEY,
    auth_container_uuid UUID NOT NULL UNIQUE,
    name                VARCHAR(255) NOT NULL,
    description         TEXT NOT NULL,
    identifier          TEXT NOT NULL,
    is_active           BOOLEAN DEFAULT FALSE,
    is_default          BOOLEAN DEFAULT FALSE,
    organization_id     INTEGER NOT NULL,
    created_at          TIMESTAMPTZ DEFAULT now(),
    updated_at          TIMESTAMPTZ
);

-- ADD INDEXES
CREATE INDEX idx_auth_containers_auth_container_uuid ON auth_containers(auth_container_uuid);
CREATE INDEX idx_auth_containers_name ON auth_containers(name);
CREATE INDEX idx_auth_containers_is_active ON auth_containers(is_active);
CREATE INDEX idx_auth_containers_is_default ON auth_containers(is_default);
`
	if err := db.Exec(sql).Error; err != nil {
		log.Fatalf("❌ Failed to run migration 002_create_organizations_table: %v", err)
	}

	log.Println("✅ Migration 002_create_organizations_table executed")
}
