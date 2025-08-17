package migration

import (
	"log"

	"gorm.io/gorm"
)

func CreatePermissionTable(db *gorm.DB) {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS permissions (
    permission_id       SERIAL PRIMARY KEY,
    permission_uuid     UUID UNIQUE NOT NULL,
    name                VARCHAR(255) UNIQUE NOT NULL,
    description         TEXT NOT NULL,
    is_active           BOOLEAN DEFAULT FALSE,
    is_default          BOOLEAN DEFAULT FALSE,
    api_id              INTEGER NOT NULL,
    auth_container_id   INTEGER NOT NULL,
    created_at          TIMESTAMPTZ DEFAULT now(),
    updated_at          TIMESTAMPTZ DEFAULT now()
);

-- ADD CONSTRAINTS
ALTER TABLE permissions
    ADD CONSTRAINT fk_permissions_api_id FOREIGN KEY (api_id) REFERENCES apis(api_id) ON DELETE CASCADE;
ALTER TABLE permissions
    ADD CONSTRAINT fk_permissions_auth_container_id FOREIGN KEY (auth_container_id) REFERENCES auth_containers(auth_container_id) ON DELETE CASCADE;

-- ADD INDEXES
CREATE INDEX idx_permissions_permission_uuid ON permissions(permission_uuid);
CREATE INDEX idx_permissions_name ON permissions(name);
CREATE INDEX idx_permissions_api_id ON permissions(api_id);
CREATE INDEX idx_permissions_auth_container_id ON permissions(auth_container_id);
`
	if err := db.Exec(sql).Error; err != nil {
		log.Fatalf("❌ Failed to run migration 005_create_permissions_table: %v", err)
	}

	log.Println("✅ Migration 005_create_permissions_table executed")
}
