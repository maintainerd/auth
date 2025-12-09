package migration

import (
	"log"

	"gorm.io/gorm"
)

func CreateAPIKeyPermissionsTable(db *gorm.DB) {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS api_key_permissions (
    api_key_permission_id   SERIAL PRIMARY KEY,
    api_key_permission_uuid UUID NOT NULL UNIQUE,
    api_key_api_id          INTEGER NOT NULL,
    permission_id           INTEGER NOT NULL,
    created_at              TIMESTAMPTZ DEFAULT now()
);

-- ADD CONSTRAINTS
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_api_key_permissions_api_key_api_id'
    ) THEN
        ALTER TABLE api_key_permissions
            ADD CONSTRAINT fk_api_key_permissions_api_key_api_id FOREIGN KEY (api_key_api_id)
            REFERENCES api_key_apis(api_key_api_id) ON DELETE CASCADE;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_api_key_permissions_permission_id'
    ) THEN
        ALTER TABLE api_key_permissions
            ADD CONSTRAINT fk_api_key_permissions_permission_id FOREIGN KEY (permission_id)
            REFERENCES permissions(permission_id) ON DELETE CASCADE;
    END IF;

    -- Add unique constraint to prevent duplicate api_key_api + permission combinations
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'uq_api_key_permissions_api_key_api_permission'
    ) THEN
        ALTER TABLE api_key_permissions
            ADD CONSTRAINT uq_api_key_permissions_api_key_api_permission UNIQUE (api_key_api_id, permission_id);
    END IF;
END$$;

-- ADD INDEXES
CREATE INDEX IF NOT EXISTS idx_api_key_permissions_uuid ON api_key_permissions (api_key_permission_uuid);
CREATE INDEX IF NOT EXISTS idx_api_key_permissions_api_key_api_id ON api_key_permissions (api_key_api_id);
CREATE INDEX IF NOT EXISTS idx_api_key_permissions_permission_id ON api_key_permissions (permission_id);
CREATE INDEX IF NOT EXISTS idx_api_key_permissions_created_at ON api_key_permissions (created_at);
`

	if err := db.Exec(sql).Error; err != nil {
		log.Fatalf("❌ Failed to run migration 017_create_api_key_permissions_table: %v", err)
	}

	log.Println("✅ Migration 017_create_api_key_permissions_table executed")
}
