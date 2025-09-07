package migration

import (
	"log"

	"gorm.io/gorm"
)

func CreateOnboardingTable(db *gorm.DB) {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS onboardings (
    onboarding_id   	SERIAL PRIMARY KEY,
    onboarding_uuid		UUID NOT NULL UNIQUE,
    name							VARCHAR(100) NOT NULL,
    description				TEXT NOT NULL,
		identifier				VARCHAR(255) NOT NULL UNIQUE,
    is_active					BOOLEAN DEFAULT TRUE,
		auth_client_id		INTEGER NOT NULL,
    created_at				TIMESTAMPTZ DEFAULT now(),
    updated_at				TIMESTAMPTZ DEFAULT now()
);

-- ADD CONSTRAINTS
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_onboardings_auth_client_id'
    ) THEN
        ALTER TABLE onboardings
            ADD CONSTRAINT fk_onboardings_auth_client_id FOREIGN KEY (auth_client_id)
            REFERENCES auth_clients(auth_client_id) ON DELETE CASCADE;
    END IF;
END$$;

-- ADD INDEXES
CREATE INDEX IF NOT EXISTS idx_onboarding_uuid ON onboardings (onboarding_uuid);
CREATE INDEX IF NOT EXISTS idx_onboarding_name ON onboardings (name);
CREATE INDEX IF NOT EXISTS idx_onboarding_identifier ON onboardings (identifier);
CREATE INDEX IF NOT EXISTS idx_onboarding_is_active ON onboardings (is_active);
CREATE INDEX IF NOT EXISTS idx_onboarding_auth_client_id ON onboardings (auth_client_id);
CREATE INDEX IF NOT EXISTS idx_onboarding_created_at ON onboardings (created_at);
`
	if err := db.Exec(sql).Error; err != nil {
		log.Fatalf("❌ Failed to run migration 020_create_onboardings_table: %v", err)
	}

	log.Println("✅ Migration 020_create_onboardings_table executed")
}
