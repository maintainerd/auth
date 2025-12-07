package migration

import (
	"log"

	"gorm.io/gorm"
)

func CreateApiPermissionTable(db *gorm.DB) {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS api_permissions (
    api_permission_id   	SERIAL PRIMARY KEY,
    api_permission_uuid		UUID NOT NULL UNIQUE,
    api_id              	INTEGER NOT NULL,
    permission_id       	INTEGER NOT NULL,
    created_at          	TIMESTAMPTZ DEFAULT now()
);

-- ADD CONSTRAINTS
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_api_permissions_api_id'
    ) THEN
        ALTER TABLE api_permissions
            ADD CONSTRAINT fk_api_permissions_api_id FOREIGN KEY (api_id)
            REFERENCES apis(api_id) ON DELETE CASCADE;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_api_permissions_permission_id'
    ) THEN
        ALTER TABLE api_permissions
            ADD CONSTRAINT fk_api_permissions_permission_id FOREIGN KEY (permission_id)
            REFERENCES permissions(permission_id) ON DELETE CASCADE;
    END IF;
END$$;

-- ADD INDEXES
CREATE INDEX IF NOT EXISTS idx_api_permissions_uuid ON api_permissions (api_permission_uuid);
CREATE INDEX IF NOT EXISTS idx_api_permissions_api_id ON api_permissions (api_id);
CREATE INDEX IF NOT EXISTS idx_api_permissions_permission_id ON api_permissions (permission_id);
`

	if err := db.Exec(sql).Error; err != nil {
		log.Fatalf("❌ Failed to run migration 009_create_api_permissions_table: %v", err)
	}

	log.Println("✅ Migration 009_create_api_permissions_table executed")
}
