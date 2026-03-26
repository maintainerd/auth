package migration

import (
	"log"

	"gorm.io/gorm"
)

func CreateClientTable(db *gorm.DB) {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS clients (
    client_id          SERIAL PRIMARY KEY,
    client_uuid        UUID NOT NULL UNIQUE,
    tenant_id               INTEGER NOT NULL,
	identity_provider_id    INTEGER NOT NULL,
    name             		VARCHAR(100) NOT NULL, -- 'default', 'google', 'facebook', 'github'
    display_name            TEXT NOT NULL,
    client_type             VARCHAR(100) NOT NULL, -- 'traditional', 'spa', 'native', 'm2m'
    domain                  TEXT,
    identifier              TEXT,
    secret                  TEXT,
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
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_clients_tenant_id'
    ) THEN
        ALTER TABLE clients
            ADD CONSTRAINT fk_clients_tenant_id FOREIGN KEY (tenant_id)
            REFERENCES tenants(tenant_id) ON DELETE CASCADE;
    END IF;
END$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_clients_identity_provider_id'
    ) THEN
        ALTER TABLE clients
            ADD CONSTRAINT fk_clients_identity_provider_id FOREIGN KEY (identity_provider_id)
            REFERENCES identity_providers(identity_provider_id) ON DELETE CASCADE;
    END IF;
END$$;

-- ADD INDEXES
-- Composite indexes for common query patterns
CREATE INDEX IF NOT EXISTS idx_clients_tenant_id_status ON clients (tenant_id, status);
CREATE INDEX IF NOT EXISTS idx_clients_tenant_id_is_default ON clients (tenant_id, is_default) WHERE is_default = TRUE;
CREATE INDEX IF NOT EXISTS idx_clients_tenant_id_identity_provider_id ON clients (tenant_id, identity_provider_id);
CREATE INDEX IF NOT EXISTS idx_clients_tenant_id_name ON clients (tenant_id, name);

-- Single column indexes
CREATE INDEX IF NOT EXISTS idx_clients_identifier ON clients (identifier) WHERE identifier IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_clients_identity_provider_id ON clients (identity_provider_id);
CREATE INDEX IF NOT EXISTS idx_clients_is_system ON clients (is_system) WHERE is_system = TRUE;
CREATE INDEX IF NOT EXISTS idx_clients_created_at ON clients (created_at);
`
	if err := db.Exec(sql).Error; err != nil {
		log.Fatalf("❌ Failed to run migration 011_create_clients_table: %v", err)
	}

	log.Println("✅ Migration 011_create_clients_table executed")
}
