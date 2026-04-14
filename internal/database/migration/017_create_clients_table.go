package migration

import (
	"gorm.io/gorm"
)

func CreateClientTable(db *gorm.DB) error {
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

    -- OAuth 2.0 fields
    token_endpoint_auth_method VARCHAR(30) NOT NULL DEFAULT 'client_secret_basic',
    grant_types                TEXT[]       NOT NULL DEFAULT '{authorization_code}',
    response_types             TEXT[]       NOT NULL DEFAULT '{code}',
    access_token_ttl           INTEGER,
    refresh_token_ttl          INTEGER,
    require_consent            BOOLEAN      NOT NULL DEFAULT TRUE,

    created_at              TIMESTAMPTZ DEFAULT now(),
    updated_at              TIMESTAMPTZ DEFAULT now(),

    -- Constraint for allowed auth methods
    CONSTRAINT chk_clients_token_auth_method CHECK (
        token_endpoint_auth_method IN ('client_secret_basic', 'client_secret_post', 'none')
    )
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

-- OAuth indexes
CREATE INDEX IF NOT EXISTS idx_clients_grant_types ON clients USING GIN (grant_types);
`
	return db.Exec(sql).Error
}
