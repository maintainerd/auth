package migration

import (
	"log"

	"gorm.io/gorm"
)

func CreateAPITable(db *gorm.DB) {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS apis (
    api_id					SERIAL PRIMARY KEY,
    api_uuid				UUID NOT NULL UNIQUE,
    name            VARCHAR(100) NOT NULL,
    display_name		TEXT NOT NULL,
		description			TEXT NOT NULL,
    api_type				TEXT NOT NULL,
    identifier			TEXT NOT NULL,
		service_id			INTEGER NOT NULL,
    status					TEXT DEFAULT 'inactive' CHECK (status IN ('active', 'inactive')),
    is_default			BOOLEAN DEFAULT FALSE,
    is_system				BOOLEAN DEFAULT FALSE,
    created_at			TIMESTAMPTZ DEFAULT now(),
    updated_at			TIMESTAMPTZ DEFAULT now()
);

-- ADD CONSTRAINTS
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_apis_service_id'
    ) THEN
        ALTER TABLE apis
            ADD CONSTRAINT fk_apis_service_id FOREIGN KEY (service_id)
            REFERENCES services(service_id) ON DELETE CASCADE;
    END IF;
END$$;

-- ADD INDEXES
CREATE INDEX IF NOT EXISTS idx_apis_uuid ON apis (api_uuid);
CREATE INDEX IF NOT EXISTS idx_apis_name ON apis (name);
CREATE INDEX IF NOT EXISTS idx_apis_display_name ON apis (display_name);
CREATE INDEX IF NOT EXISTS idx_apis_api_type ON apis (api_type);
CREATE INDEX IF NOT EXISTS idx_apis_identifier ON apis (identifier);
CREATE INDEX IF NOT EXISTS idx_apis_service_id ON apis (service_id);
CREATE INDEX IF NOT EXISTS idx_apis_status ON apis (status);
CREATE INDEX IF NOT EXISTS idx_apis_is_default ON apis (is_default);
CREATE INDEX IF NOT EXISTS idx_apis_is_system ON apis (is_system);
CREATE INDEX IF NOT EXISTS idx_apis_created_at ON apis (created_at);
`
	if err := db.Exec(sql).Error; err != nil {
		log.Fatalf("❌ Failed to run migration 007_create_apis_table: %v", err)
	}

	log.Println("✅ Migration 007_create_apis_table executed")
}
