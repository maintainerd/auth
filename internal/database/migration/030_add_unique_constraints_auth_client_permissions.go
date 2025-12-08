package migration

import (
	"log"

	"gorm.io/gorm"
)

func AddUniqueConstraintsAuthClientPermissions(db *gorm.DB) {
	sql := `
-- ADD UNIQUE CONSTRAINTS TO PREVENT DUPLICATE RELATIONSHIPS
DO $$
BEGIN
    -- Add unique constraint to prevent duplicate auth_client_api + permission combinations
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'uq_auth_client_permissions_api_permission'
    ) THEN
        ALTER TABLE auth_client_permissions
            ADD CONSTRAINT uq_auth_client_permissions_api_permission UNIQUE (auth_client_api_id, permission_id);
    END IF;
END$$;
`

	if err := db.Exec(sql).Error; err != nil {
		log.Fatalf("❌ Failed to run migration 030_add_unique_constraints_auth_client_permissions: %v", err)
	}

	log.Println("✅ Migration 030_add_unique_constraints_auth_client_permissions executed")
}
