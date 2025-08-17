package migration

import (
	"log"

	"gorm.io/gorm"
)

func CreateAuthClientTable(db *gorm.DB) {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS auth_clients (
    auth_client_id          SERIAL PRIMARY KEY,
    auth_client_uuid        UUID NOT NULL UNIQUE,
    client_name             VARCHAR(100) NOT NULL, -- 'default', 'google', 'facebook', 'github'
    display_name            TEXT NOT NULL,
    client_type             VARCHAR(100) NOT NULL, -- 'traditional', 'spa', 'native', 'm2m'
    domain                  TEXT, -- optional
    client_id               TEXT, -- optional
    client_secret           TEXT, -- optional
    redirect_uri            TEXT, -- optional
    config                  JSONB,
    is_active               BOOLEAN DEFAULT FALSE,
    is_default              BOOLEAN DEFAULT FALSE,
    identity_provider_id    INTEGER NOT NULL,
    auth_container_id       INTEGER NOT NULL,
    created_at              TIMESTAMPTZ DEFAULT now(),
    updated_at              TIMESTAMPTZ DEFAULT now()
);

-- ADD CONSTRAINTS
ALTER TABLE auth_clients
    ADD CONSTRAINT fk_auth_clients_identity_provider_id FOREIGN KEY (identity_provider_id) REFERENCES identity_providers(identity_provider_id) ON DELETE CASCADE;
ALTER TABLE auth_clients
    ADD CONSTRAINT fk_auth_clients_auth_container_id FOREIGN KEY (auth_container_id) REFERENCES auth_containers(auth_container_id) ON DELETE CASCADE;

-- ADD INDEXES
CREATE INDEX idx_auth_clients_identity_provider_id ON auth_clients (identity_provider_id);
CREATE INDEX idx_auth_clients_auth_container_id ON auth_clients (auth_container_id);
`
	if err := db.Exec(sql).Error; err != nil {
		log.Fatalf("❌ Failed to run migration 007_create_auth_clients_table: %v", err)
	}

	log.Println("✅ Migration 007_create_auth_clients_table executed")
}
