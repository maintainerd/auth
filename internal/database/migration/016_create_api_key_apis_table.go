package migration

import (
	"log"

	"gorm.io/gorm"
)

func CreateAPIKeyApiTable(db *gorm.DB) {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS api_key_apis (
    api_key_api_id   		SERIAL PRIMARY KEY,
    api_key_api_uuid		UUID NOT NULL UNIQUE,
    api_key_id              INTEGER NOT NULL,
    api_id                  INTEGER NOT NULL,
    created_at              TIMESTAMPTZ DEFAULT now()
);

-- ADD CONSTRAINTS
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_api_key_apis_api_key_id'
    ) THEN
        ALTER TABLE api_key_apis
            ADD CONSTRAINT fk_api_key_apis_api_key_id FOREIGN KEY (api_key_id)
            REFERENCES api_keys(api_key_id) ON DELETE CASCADE;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_api_key_apis_api_id'
    ) THEN
        ALTER TABLE api_key_apis
            ADD CONSTRAINT fk_api_key_apis_api_id FOREIGN KEY (api_id)
            REFERENCES apis(api_id) ON DELETE CASCADE;
    END IF;

    -- Add unique constraint to prevent duplicate api_key + api combinations
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'uq_api_key_apis_key_api'
    ) THEN
        ALTER TABLE api_key_apis
            ADD CONSTRAINT uq_api_key_apis_key_api UNIQUE (api_key_id, api_id);
    END IF;
END$$;

-- ADD INDEXES
CREATE INDEX IF NOT EXISTS idx_api_key_apis_uuid ON api_key_apis (api_key_api_uuid);
CREATE INDEX IF NOT EXISTS idx_api_key_apis_api_key_id ON api_key_apis (api_key_id);
CREATE INDEX IF NOT EXISTS idx_api_key_apis_api_id ON api_key_apis (api_id);
`

	if err := db.Exec(sql).Error; err != nil {
		log.Fatalf("❌ Failed to run migration 016_create_api_key_apis_table: %v", err)
	}

	log.Println("✅ Migration 016_create_api_key_apis_table executed")
}
