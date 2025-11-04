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
    name           					VARCHAR(100) NOT NULL, -- 'default', 'cognito', 'auth0'
    display_name            TEXT NOT NULL,
    provider_type           VARCHAR(100) NOT NULL, -- 'primary', 'oauth2'
    identifier              TEXT,
    config                  JSONB,
    is_active               BOOLEAN DEFAULT FALSE,
    is_default              BOOLEAN DEFAULT FALSE,
    tenant_id               INTEGER NOT NULL,
    created_at              TIMESTAMPTZ DEFAULT now(),
    updated_at              TIMESTAMPTZ DEFAULT now()
);

-- ADD CONSTRAINTS (safe)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_identity_providers_tenant_id'
    ) THEN
        ALTER TABLE identity_providers
            ADD CONSTRAINT fk_identity_providers_tenant_id FOREIGN KEY (tenant_id)
            REFERENCES tenants(tenant_id) ON DELETE CASCADE;
    END IF;
END$$;

-- ADD INDEXES
CREATE INDEX IF NOT EXISTS idx_identity_providers_uuid ON identity_providers (identity_provider_uuid);
CREATE INDEX IF NOT EXISTS idx_identity_providers_name ON identity_providers (name);
CREATE INDEX IF NOT EXISTS idx_identity_providers_display_name ON identity_providers (display_name);
CREATE INDEX IF NOT EXISTS idx_identity_providers_provider_type ON identity_providers (provider_type);
CREATE INDEX IF NOT EXISTS idx_identity_providers_identifier ON identity_providers (identifier);
CREATE INDEX IF NOT EXISTS idx_identity_providers_is_active ON identity_providers (is_active);
CREATE INDEX IF NOT EXISTS idx_identity_providers_is_default ON identity_providers (is_default);
CREATE INDEX IF NOT EXISTS idx_identity_providers_tenant_id ON identity_providers (tenant_id);
CREATE INDEX IF NOT EXISTS idx_identity_providers_created_at ON identity_providers (created_at);
`
	if err := db.Exec(sql).Error; err != nil {
		log.Fatalf("❌ Failed to run migration 010_create_identity_providers_table: %v", err)
	}

	log.Println("✅ Migration 010_create_identity_providers_table executed")
}
