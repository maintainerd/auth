package migration

import (
	"log"

	"gorm.io/gorm"
)

func CreateRegistrationRouteTable(db *gorm.DB) {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS registration_routes (
    registration_route_id   SERIAL PRIMARY KEY,
    registration_route_uuid UUID NOT NULL UNIQUE,
    name                    VARCHAR(100) NOT NULL,
    identifier              VARCHAR(255) NOT NULL UNIQUE,
    description             TEXT NOT NULL,
    auth_container_id       INTEGER NOT NULL,
    is_active               BOOLEAN DEFAULT TRUE,
    created_at              TIMESTAMPTZ DEFAULT now(),
    updated_at              TIMESTAMPTZ DEFAULT now()
);

-- ADD CONSTRAINTS (safe)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_registration_routes_auth_container_id'
    ) THEN
        ALTER TABLE registration_routes
            ADD CONSTRAINT fk_registration_routes_auth_container_id FOREIGN KEY (auth_container_id)
            REFERENCES auth_containers(auth_container_id) ON DELETE CASCADE;
    END IF;
END$$;

-- ADD INDEXES
CREATE INDEX IF NOT EXISTS idx_registration_routes_registration_route_uuid ON registration_routes (registration_route_uuid);
CREATE INDEX IF NOT EXISTS idx_registration_routes_identifier ON registration_routes (identifier);
CREATE INDEX IF NOT EXISTS idx_registration_routes_is_active ON registration_routes (is_active);
`
	if err := db.Exec(sql).Error; err != nil {
		log.Fatalf("❌ Failed to run migration 015_create_registration_routes_table: %v", err)
	}

	log.Println("✅ Migration 015_create_registration_routes_table executed")
}
