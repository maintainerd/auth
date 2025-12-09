package migration

import (
	"log"

	"gorm.io/gorm"
)

func CreateAuthClientPermissionTable(db *gorm.DB) {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS auth_client_permissions (
    auth_client_permission_id   	SERIAL PRIMARY KEY,
    auth_client_permission_uuid		UUID NOT NULL UNIQUE,
    auth_client_api_id              INTEGER NOT NULL,
    permission_id               	INTEGER NOT NULL,
    created_at                  	TIMESTAMPTZ DEFAULT now()
);

-- ADD CONSTRAINTS
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_auth_client_permissions_auth_client_api_id'
    ) THEN
        ALTER TABLE auth_client_permissions
            ADD CONSTRAINT fk_auth_client_permissions_auth_client_api_id FOREIGN KEY (auth_client_api_id)
            REFERENCES auth_client_apis(auth_client_api_id) ON DELETE CASCADE;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_auth_client_permissions_permission_id'
    ) THEN
        ALTER TABLE auth_client_permissions
            ADD CONSTRAINT fk_auth_client_permissions_permission_id FOREIGN KEY (permission_id)
            REFERENCES permissions(permission_id) ON DELETE CASCADE;
    END IF;

    -- Add unique constraint to prevent duplicate auth_client_api + permission combinations
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'uq_auth_client_permissions_api_permission'
    ) THEN
        ALTER TABLE auth_client_permissions
            ADD CONSTRAINT uq_auth_client_permissions_api_permission UNIQUE (auth_client_api_id, permission_id);
    END IF;
END$$;

-- ADD INDEXES
CREATE INDEX IF NOT EXISTS idx_auth_client_permissions_uuid ON auth_client_permissions (auth_client_permission_uuid);
CREATE INDEX IF NOT EXISTS idx_auth_client_permissions_auth_client_api_id ON auth_client_permissions (auth_client_api_id);
CREATE INDEX IF NOT EXISTS idx_auth_client_permissions_permission_id ON auth_client_permissions (permission_id);
`

	if err := db.Exec(sql).Error; err != nil {
		log.Fatalf("❌ Failed to run migration 014_create_auth_client_permissions_table: %v", err)
	}

	log.Println("✅ Migration 014_create_auth_client_permissions_table executed")
}
