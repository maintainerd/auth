package migration

import (
	"log"

	"gorm.io/gorm"
)

func CreateAuthContainerTable(db *gorm.DB) {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS auth_containers (
    auth_container_id   	SERIAL PRIMARY KEY,
    auth_container_uuid		UUID NOT NULL UNIQUE,
    name                	VARCHAR(255) NOT NULL,
    description         	TEXT NOT NULL,
    identifier          	TEXT NOT NULL,
		organization_id     	INTEGER NOT NULL,
    is_active           	BOOLEAN DEFAULT FALSE,
		is_public							BOOLEAN DEFAULT FALSE,
    is_default          	BOOLEAN DEFAULT FALSE,
    created_at          	TIMESTAMPTZ DEFAULT now(),
    updated_at          	TIMESTAMPTZ DEFAULT now()
);

-- ADD CONSTRAINTS
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_auth_containers_organization_id'
    ) THEN
        ALTER TABLE auth_containers
            ADD CONSTRAINT fk_auth_containers_organization_id
            FOREIGN KEY (organization_id) REFERENCES organizations(organization_id) ON DELETE CASCADE;
    END IF;
END$$;

-- ADD INDEXES
CREATE INDEX IF NOT EXISTS idx_auth_containers_uuid ON auth_containers (auth_container_uuid);
CREATE INDEX IF NOT EXISTS idx_auth_containers_name ON auth_containers (name);
CREATE INDEX IF NOT EXISTS idx_auth_containers_identifier ON auth_containers (identifier);
CREATE INDEX IF NOT EXISTS idx_auth_containers_organization_id ON auth_containers (organization_id);
CREATE INDEX IF NOT EXISTS idx_auth_containers_is_active ON auth_containers (is_active);
CREATE INDEX IF NOT EXISTS idx_auth_containers_is_public ON auth_containers (is_public);
CREATE INDEX IF NOT EXISTS idx_auth_containers_is_default ON auth_containers (is_default);
CREATE INDEX IF NOT EXISTS idx_auth_containers_created_at ON auth_containers (created_at);
`
	if err := db.Exec(sql).Error; err != nil {
		log.Fatalf("❌ Failed to run migration 009_create_auth_containers_table: %v", err)
	}

	log.Println("✅ Migration 009_create_auth_containers_table executed")
}
