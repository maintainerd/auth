package migration

import (
	"log"

	"gorm.io/gorm"
)

func CreateServiceTable(db *gorm.DB) {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS services (
    service_id      SERIAL PRIMARY KEY,
    service_uuid    UUID NOT NULL UNIQUE,
    service_name    VARCHAR(100) NOT NULL, -- 'auth', 'your-custom-service'
    display_name    TEXT NOT NULL,
    description     TEXT NOT NULL,
    service_type    TEXT NOT NULL, -- 'default', 'custom'
    version         VARCHAR(20) NOT NULL,
    config          JSONB,
    is_active       BOOLEAN DEFAULT FALSE,
    is_default      BOOLEAN DEFAULT FALSE,
    created_at      TIMESTAMPTZ DEFAULT now(),
    updated_at      TIMESTAMPTZ DEFAULT now()
);

-- ADD INDEXES
CREATE INDEX idx_services_service_name ON services (service_name);
CREATE INDEX idx_services_display_name ON services (display_name);
CREATE INDEX idx_services_service_type ON services (service_type);
CREATE INDEX idx_services_service_uuid ON services (service_uuid);
`

	if err := db.Exec(sql).Error; err != nil {
		log.Fatalf("❌ Failed to run migration 001_create_services_table: %v", err)
	}

	log.Println("✅ Migration 001_create_services_table executed")
}
