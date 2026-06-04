package migration

import (
	"gorm.io/gorm"
)

func CreateClientTable(db *gorm.DB) error {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS clients (
    client_id               BIGSERIAL PRIMARY KEY,
    client_uuid             UUID NOT NULL UNIQUE,
    tenant_id               BIGINT NOT NULL,
    service_id              BIGINT,
    identity_provider_id    BIGINT NOT NULL,
    name                    VARCHAR(100) NOT NULL,
    display_name            TEXT NOT NULL,
    client_type             VARCHAR(100) NOT NULL,
    domain                  TEXT,
    identifier              TEXT,

    -- Secret storage: bcrypt hash for password-style auth plus encrypted copy
    -- for client_secret_jwt HMAC verification. Plaintext is returned once only.
    secret_hash                  TEXT,
    secret_encrypted             TEXT,
    previous_secret_hash         TEXT,
    previous_secret_encrypted    TEXT,
    previous_secret_expires_at   TIMESTAMPTZ,

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

    -- Per-client scope allowlist (empty = all scopes permitted).
    allowed_scopes             TEXT[]       NOT NULL DEFAULT '{}',

    -- JWT client auth (RFC 7523): embedded JWKS or URI for private_key_jwt / client_secret_jwt.
    jwks                       JSONB,
    jwks_uri                   TEXT,

    -- mTLS client auth (RFC 8705): expected certificate SHA-256 thumbprint.
    mtls_bound_cert_thumbprint TEXT,

    -- Scope-to-claim mapping: maps scope names to OIDC claim names to include in tokens.
    -- Format: {"email": ["email", "email_verified"], "profile": ["given_name", "family_name"]}
    -- When null, the standard OIDC default mapping applies.
    scope_claim_mappings       JSONB,

    -- Custom claim mappers: static or metadata-derived claims injected into tokens per client/tenant.
    -- Format: {"claim_name": "static_value", "org_id": "{{user.metadata.org_id}}"}
    claim_mappers              JSONB,

    created_by              BIGINT,
    updated_by              BIGINT,
    created_at              TIMESTAMPTZ DEFAULT now(),
    updated_at              TIMESTAMPTZ DEFAULT now(),
    deleted_at              TIMESTAMPTZ,

    CONSTRAINT chk_clients_token_auth_method CHECK (
        token_endpoint_auth_method IN (
            'client_secret_basic', 'client_secret_post', 'none',
            'private_key_jwt', 'client_secret_jwt',
            'tls_client_auth', 'self_signed_tls_client_auth'
        )
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

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_clients_service_id'
    ) THEN
        ALTER TABLE clients
            ADD CONSTRAINT fk_clients_service_id FOREIGN KEY (service_id)
            REFERENCES services(service_id) ON DELETE SET NULL;
    END IF;

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
CREATE INDEX IF NOT EXISTS idx_clients_service_id ON clients (service_id) WHERE service_id IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS uq_clients_tenant_name ON clients (tenant_id, name) WHERE deleted_at IS NULL;

-- Single column indexes
CREATE INDEX IF NOT EXISTS idx_clients_identifier ON clients (identifier) WHERE identifier IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_clients_identity_provider_id ON clients (identity_provider_id);
CREATE INDEX IF NOT EXISTS idx_clients_is_system ON clients (is_system) WHERE is_system = TRUE;
CREATE INDEX IF NOT EXISTS idx_clients_created_at ON clients (created_at);
CREATE INDEX IF NOT EXISTS idx_clients_deleted_at ON clients (deleted_at) WHERE deleted_at IS NULL;

-- OAuth indexes
CREATE INDEX IF NOT EXISTS idx_clients_grant_types ON clients USING GIN (grant_types);
`
	return db.Exec(sql).Error
}
