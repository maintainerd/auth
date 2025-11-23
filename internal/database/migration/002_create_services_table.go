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
    name    		VARCHAR(100) NOT NULL,
    display_name    TEXT NOT NULL,
    description     TEXT NOT NULL,
    version         VARCHAR(20) NOT NULL,
    status          VARCHAR(20) DEFAULT 'inactive',
	is_public		BOOLEAN DEFAULT FALSE,
	is_default      BOOLEAN DEFAULT FALSE,
	is_system       BOOLEAN DEFAULT FALSE,
    created_at      TIMESTAMPTZ DEFAULT now(),
    updated_at      TIMESTAMPTZ DEFAULT now()
);

-- ADD INDEXES
CREATE INDEX IF NOT EXISTS idx_services_uuid ON services (service_uuid);
CREATE INDEX IF NOT EXISTS idx_services_name ON services (name);
CREATE INDEX IF NOT EXISTS idx_services_display_name ON services (display_name);
CREATE INDEX IF NOT EXISTS idx_services_status ON services (status);
CREATE INDEX IF NOT EXISTS idx_services_is_public ON services (is_public);
CREATE INDEX IF NOT EXISTS idx_services_is_default ON services (is_default);
CREATE INDEX IF NOT EXISTS idx_services_is_system ON services (is_system);
CREATE INDEX IF NOT EXISTS idx_services_created_at ON services (created_at);

-- ADD CONSTRAINTS
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'chk_services_status'
    ) THEN
        ALTER TABLE services ADD CONSTRAINT chk_services_status
            CHECK (status IN ('active', 'maintenance', 'deprecated', 'inactive'));
    END IF;
END$$;
`
	if err := db.Exec(sql).Error; err != nil {
		log.Fatalf("❌ Failed to run migration 002_create_services_table: %v", err)
	}

	log.Println("✅ Migration 002_create_services_table executed")
}
