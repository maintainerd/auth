package migration

import (
	"log"

	"gorm.io/gorm"
)

func CreateOnboardingRouteTable(db *gorm.DB) {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS onboarding_routes (
    onboarding_route_id   	SERIAL PRIMARY KEY,
    onboarding_route_uuid 	UUID NOT NULL UNIQUE,
    name                    VARCHAR(100) NOT NULL,
    identifier              VARCHAR(255) NOT NULL UNIQUE,
    description             TEXT NOT NULL,
    auth_client_id       		INTEGER NOT NULL,
    is_active               BOOLEAN DEFAULT TRUE,
    created_at              TIMESTAMPTZ DEFAULT now(),
    updated_at              TIMESTAMPTZ DEFAULT now()
);

-- ADD CONSTRAINTS (safe)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_onboarding_routes_auth_client_id'
    ) THEN
        ALTER TABLE onboarding_routes
            ADD CONSTRAINT fk_onboarding_routes_auth_client_id FOREIGN KEY (auth_client_id)
            REFERENCES auth_clients(auth_client_id) ON DELETE CASCADE;
    END IF;
END$$;

-- ADD INDEXES
CREATE INDEX IF NOT EXISTS idx_onboarding_routes_onboarding_route_uuid ON onboarding_routes (onboarding_route_uuid);
CREATE INDEX IF NOT EXISTS idx_onboarding_routes_identifier ON onboarding_routes (identifier);
CREATE INDEX IF NOT EXISTS idx_onboarding_routes_auth_client_id ON onboarding_routes (auth_client_id);
CREATE INDEX IF NOT EXISTS idx_onboarding_routes_is_active ON onboarding_routes (is_active);
`
	if err := db.Exec(sql).Error; err != nil {
		log.Fatalf("❌ Failed to run migration 016_create_onboarding_routes_table: %v", err)
	}

	log.Println("✅ Migration 016_create_onboarding_routes_table executed")
}
