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
    tenant_id               INTEGER NOT NULL,
	identity_provider_id    INTEGER NOT NULL,
    name             		VARCHAR(100) NOT NULL, -- 'default', 'google', 'facebook', 'github'
    display_name            TEXT NOT NULL,
    client_type             VARCHAR(100) NOT NULL, -- 'traditional', 'spa', 'native', 'm2m'
    domain                  TEXT,
    client_id               TEXT,
    client_secret           TEXT,
    config                  JSONB,
    status                  VARCHAR(20) DEFAULT 'inactive',
    is_default              BOOLEAN DEFAULT FALSE,
    is_system               BOOLEAN DEFAULT FALSE,
    created_at              TIMESTAMPTZ DEFAULT now(),
    updated_at              TIMESTAMPTZ DEFAULT now()
);

-- ADD CONSTRAINTS (safe)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_auth_clients_tenant_id'
    ) THEN
        ALTER TABLE auth_clients
            ADD CONSTRAINT fk_auth_clients_tenant_id FOREIGN KEY (tenant_id)
            REFERENCES tenants(tenant_id) ON DELETE CASCADE;
    END IF;
END$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_auth_clients_identity_provider_id'
    ) THEN
        ALTER TABLE auth_clients
            ADD CONSTRAINT fk_auth_clients_identity_provider_id FOREIGN KEY (identity_provider_id)
            REFERENCES identity_providers(identity_provider_id) ON DELETE CASCADE;
    END IF;
END$$;

-- ADD INDEXES
CREATE INDEX IF NOT EXISTS idx_auth_clients_uuid ON auth_clients (auth_client_uuid);
CREATE INDEX IF NOT EXISTS idx_auth_clients_tenant_id ON auth_clients (tenant_id);
CREATE INDEX IF NOT EXISTS idx_auth_clients_identity_provider_id ON auth_clients (identity_provider_id);
CREATE INDEX IF NOT EXISTS idx_auth_clients_name ON auth_clients (name);
CREATE INDEX IF NOT EXISTS idx_auth_clients_display_name ON auth_clients (display_name);
CREATE INDEX IF NOT EXISTS idx_auth_clients_client_type ON auth_clients (client_type);
CREATE INDEX IF NOT EXISTS idx_auth_clients_status ON auth_clients (status);
CREATE INDEX IF NOT EXISTS idx_auth_clients_is_system ON auth_clients (is_system);
CREATE INDEX IF NOT EXISTS idx_auth_clients_is_default ON auth_clients (is_default);
CREATE INDEX IF NOT EXISTS idx_auth_clients_created_at ON auth_clients (created_at);
`
	if err := db.Exec(sql).Error; err != nil {
		log.Fatalf("❌ Failed to run migration 011_create_auth_clients_table: %v", err)
	}

	log.Println("✅ Migration 011_create_auth_clients_table executed")
}
