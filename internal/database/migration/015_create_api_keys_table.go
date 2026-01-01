package migration

import (
	"log"

	"gorm.io/gorm"
)

func CreateAPIKeysTable(db *gorm.DB) {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS api_keys (
    api_key_id              SERIAL PRIMARY KEY,
    api_key_uuid            UUID NOT NULL UNIQUE,
    tenant_id               INTEGER NOT NULL,
    name                    VARCHAR(100) NOT NULL,
    description             TEXT,
    key_hash                TEXT NOT NULL UNIQUE, -- Hashed version of the API key
    key_prefix              VARCHAR(20) NOT NULL, -- First few characters for identification
    config                  JSONB, -- Configuration and additional fields
    expires_at              TIMESTAMPTZ,
    rate_limit              INTEGER, -- Requests per minute/hour
    status                  TEXT DEFAULT 'active' CHECK (status IN ('active', 'inactive')),
    created_at              TIMESTAMPTZ DEFAULT now(),
    updated_at              TIMESTAMPTZ DEFAULT now()
);

-- ADD CONSTRAINTS (safe)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_api_keys_tenant_id'
    ) THEN
        ALTER TABLE api_keys
            ADD CONSTRAINT fk_api_keys_tenant_id FOREIGN KEY (tenant_id)
            REFERENCES tenants(tenant_id) ON DELETE CASCADE;
    END IF;
END$$;

-- ADD INDEXES
CREATE INDEX IF NOT EXISTS idx_api_keys_uuid ON api_keys (api_key_uuid);
CREATE INDEX IF NOT EXISTS idx_api_keys_tenant_id ON api_keys (tenant_id);
CREATE INDEX IF NOT EXISTS idx_api_keys_name ON api_keys (name);
CREATE INDEX IF NOT EXISTS idx_api_keys_key_hash ON api_keys (key_hash);
CREATE INDEX IF NOT EXISTS idx_api_keys_key_prefix ON api_keys (key_prefix);
CREATE INDEX IF NOT EXISTS idx_api_keys_status ON api_keys (status);
CREATE INDEX IF NOT EXISTS idx_api_keys_expires_at ON api_keys (expires_at);
CREATE INDEX IF NOT EXISTS idx_api_keys_created_at ON api_keys (created_at);
`

	if err := db.Exec(sql).Error; err != nil {
		log.Fatalf("❌ Failed to run migration 015_create_api_keys_table: %v", err)
	}

	log.Println("✅ Migration 015_create_api_keys_table executed")
}
