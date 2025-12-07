package migration

import (
	"log"

	"gorm.io/gorm"
)

func AddUniqueConstraintsAuthClientApis(db *gorm.DB) {
	sql := `
-- ADD UNIQUE CONSTRAINTS TO PREVENT DUPLICATE RELATIONSHIPS
DO $$
BEGIN
    -- Add unique constraint to prevent duplicate auth_client + api combinations
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'uq_auth_client_apis_client_api'
    ) THEN
        ALTER TABLE auth_client_apis
            ADD CONSTRAINT uq_auth_client_apis_client_api UNIQUE (auth_client_id, api_id);
    END IF;
END$$;
`

	if err := db.Exec(sql).Error; err != nil {
		log.Fatalf("❌ Failed to run migration 029_add_unique_constraints_auth_client_apis: %v", err)
	}

	log.Println("✅ Migration 029_add_unique_constraints_auth_client_apis executed")
}
