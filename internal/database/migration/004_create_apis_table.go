package migration

import (
	"log"

	"gorm.io/gorm"
)

func CreateAPITable(db *gorm.DB) {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS apis (
    api_id              SERIAL PRIMARY KEY,
    api_uuid            UUID NOT NULL UNIQUE,
    api_name            VARCHAR(100) NOT NULL, -- 'auth', 'your-custom-api'
    display_name        TEXT NOT NULL,
    api_type            TEXT NOT NULL, -- 'default', 'custom'
    description         TEXT NOT NULL,
    identifier          TEXT NOT NULL, -- 'http://api.example.com'
    is_active           BOOLEAN DEFAULT FALSE,
    is_default          BOOLEAN DEFAULT FALSE,
    service_id          INTEGER NOT NULL,
    auth_container_id   INTEGER NOT NULL,
    created_at          TIMESTAMPTZ DEFAULT now(),
    updated_at          TIMESTAMPTZ DEFAULT now()
);

-- ADD CONSTRAINTS
ALTER TABLE apis
    ADD CONSTRAINT fk_apis_service_id FOREIGN KEY (service_id) REFERENCES services(service_id) ON DELETE CASCADE;
ALTER TABLE apis
    ADD CONSTRAINT fk_apis_auth_container_id FOREIGN KEY (auth_container_id) REFERENCES auth_containers(auth_container_id) ON DELETE CASCADE;

-- ADD INDEXES
CREATE INDEX idx_apis_api_name ON apis (api_name);
CREATE INDEX idx_apis_api_type ON apis (api_type);
CREATE INDEX idx_apis_identifier ON apis (identifier);
CREATE INDEX idx_apis_service_id ON apis (service_id);
CREATE INDEX idx_apis_auth_container_id ON apis (auth_container_id);
`
	if err := db.Exec(sql).Error; err != nil {
		log.Fatalf("❌ Failed to run migration 004_create_apis_table: %v", err)
	}

	log.Println("✅ Migration 004_create_apis_table executed")
}
