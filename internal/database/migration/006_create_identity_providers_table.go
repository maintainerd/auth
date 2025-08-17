package migration

import (
	"log"

	"gorm.io/gorm"
)

func CreateIdentityProviderTable(db *gorm.DB) {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS identity_providers (
    identity_provider_id    SERIAL PRIMARY KEY,
    identity_provider_uuid  UUID NOT NULL UNIQUE,
    provider_name           VARCHAR(100) NOT NULL, -- 'default', 'cognito', 'auth0'
    display_name            TEXT NOT NULL,
    provider_type           VARCHAR(100) NOT NULL, -- 'primary', 'oauth2'
    identifier              TEXT,
    config                  JSONB,
    is_active               BOOLEAN DEFAULT FALSE,
    is_default              BOOLEAN DEFAULT FALSE,
    auth_container_id       INTEGER NOT NULL,
    created_at              TIMESTAMPTZ DEFAULT now(),
    updated_at              TIMESTAMPTZ DEFAULT now()
);

-- ADD CONSTRAINTS
ALTER TABLE identity_providers
    ADD CONSTRAINT fk_identity_providers_auth_container_id FOREIGN KEY (auth_container_id) REFERENCES auth_containers(auth_container_id) ON DELETE CASCADE;

-- ADD INDEXES
CREATE INDEX idx_identity_providers_auth_container_id ON identity_providers (auth_container_id);
CREATE INDEX idx_identity_providers_provider_name ON identity_providers (provider_name);
`
	if err := db.Exec(sql).Error; err != nil {
		log.Fatalf("❌ Failed to run migration 006_create_identity_providers_table: %v", err)
	}

	log.Println("✅ Migration 006_create_identity_providers_table executed")
}
