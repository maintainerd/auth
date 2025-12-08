package migration

import (
	"log"

	"gorm.io/gorm"
)

func CreateAuthClientApiTable(db *gorm.DB) {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS auth_client_apis (
    auth_client_api_id   		SERIAL PRIMARY KEY,
    auth_client_api_uuid		UUID NOT NULL UNIQUE,
    auth_client_id              INTEGER NOT NULL,
    api_id                      INTEGER NOT NULL,
    created_at                  TIMESTAMPTZ DEFAULT now()
);

-- ADD CONSTRAINTS
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_auth_client_apis_auth_client_id'
    ) THEN
        ALTER TABLE auth_client_apis
            ADD CONSTRAINT fk_auth_client_apis_auth_client_id FOREIGN KEY (auth_client_id)
            REFERENCES auth_clients(auth_client_id) ON DELETE CASCADE;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_auth_client_apis_api_id'
    ) THEN
        ALTER TABLE auth_client_apis
            ADD CONSTRAINT fk_auth_client_apis_api_id FOREIGN KEY (api_id)
            REFERENCES apis(api_id) ON DELETE CASCADE;
    END IF;
END$$;

-- ADD INDEXES
CREATE INDEX IF NOT EXISTS idx_auth_client_apis_uuid ON auth_client_apis (auth_client_api_uuid);
CREATE INDEX IF NOT EXISTS idx_auth_client_apis_auth_client_id ON auth_client_apis (auth_client_id);
CREATE INDEX IF NOT EXISTS idx_auth_client_apis_api_id ON auth_client_apis (api_id);
`

	if err := db.Exec(sql).Error; err != nil {
		log.Fatalf("❌ Failed to run migration 013_create_auth_client_apis_table: %v", err)
	}

	log.Println("✅ Migration 013_create_auth_client_apis_table executed")
}
