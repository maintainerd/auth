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
    permission_uuid     UUID NOT NULL UNIQUE,
    name                VARCHAR(255) NOT NULL UNIQUE,
    description         TEXT NOT NULL,
    api_id              INTEGER NOT NULL,
    status              VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'inactive')),
    is_default          BOOLEAN DEFAULT FALSE,
    is_system           BOOLEAN DEFAULT FALSE,
    created_at          TIMESTAMPTZ DEFAULT now(),
    updated_at          TIMESTAMPTZ DEFAULT now()
);

-- ADD CONSTRAINTS
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_permissions_api_id'
    ) THEN
        ALTER TABLE permissions
            ADD CONSTRAINT fk_permissions_api_id FOREIGN KEY (api_id)
            REFERENCES apis(api_id) ON DELETE CASCADE;
    END IF;
END$$;

-- ADD INDEXES
CREATE INDEX IF NOT EXISTS idx_permissions_uuid ON permissions (permission_uuid);
CREATE INDEX IF NOT EXISTS idx_permissions_name ON permissions (name);
CREATE INDEX IF NOT EXISTS idx_permissions_api_id ON permissions (api_id);
CREATE INDEX IF NOT EXISTS idx_permissions_status ON permissions (status);
CREATE INDEX IF NOT EXISTS idx_permissions_is_default ON permissions (is_default);
CREATE INDEX IF NOT EXISTS idx_permissions_is_system ON permissions (is_system);
CREATE INDEX IF NOT EXISTS idx_permissions_created_at ON permissions (created_at);
`
	if err := db.Exec(sql).Error; err != nil {
		log.Fatalf("❌ Failed to run migration 008_create_permissions_table: %v", err)
	}

	log.Println("✅ Migration 008_create_permissions_table executed")
}
