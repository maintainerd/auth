package migration

import (
	"gorm.io/gorm"
)

func CreateClientTable(db *gorm.DB) error {
	sql := `
-- CREATE TABLE
--
-- A client row represents a DOWNSTREAM relying-party application (SPA, traditional
-- web, mobile/native, or M2M) that authenticates against this authorization server.
-- It is NOT a place to store an external provider's own OAuth credentials — upstream
-- federation credentials (Cognito/Auth0/Google client_id/secret) live in the dedicated
-- identity_providers.provider_client_id / provider_client_secret_encrypted columns
-- (the secret encrypted at rest). Login connections are enabled through the
-- client_identity_providers join table; the OAuth columns below always describe
-- how THIS app talks to our token endpoint, regardless of provider.
--
-- Columns are grouped: identity & ownership, descriptive, secret storage, config &
-- lifecycle, OAuth core, token lifetime, security overrides, advanced client auth,
-- claims, and audit. Security-override columns are nullable: NULL = inherit the
-- tenant default from security_settings (resolved coalesce(client, tenant, system)).
CREATE TABLE IF NOT EXISTS clients (
    -- Identity & ownership
    client_id                    BIGSERIAL PRIMARY KEY,
    client_uuid                  UUID NOT NULL UNIQUE,
    tenant_id                    BIGINT NOT NULL,
    service_id                   BIGINT,

    -- Descriptive
    name                         VARCHAR(100) NOT NULL,
    display_name                 TEXT NOT NULL,
    client_type                  VARCHAR(20) NOT NULL,
    domain                       TEXT,
    identifier                   TEXT,

    -- Secret storage: bcrypt hash for password-style auth plus encrypted copy
    -- for client_secret_jwt HMAC verification. Plaintext is returned once only.
    -- All nullable: public clients (SPA, mobile) carry no secret.
    secret_hash                  TEXT,
    secret_encrypted             TEXT,
    previous_secret_hash         TEXT,
    previous_secret_encrypted    TEXT,
    previous_secret_expires_at   TIMESTAMPTZ,

    -- Free-form config blob + lifecycle
    config                       JSONB,
    status                       VARCHAR(20) DEFAULT 'inactive',
    is_default                   BOOLEAN DEFAULT FALSE,
    is_system                    BOOLEAN DEFAULT FALSE,
    branding_id                  BIGINT,
    allow_registration           BOOLEAN NOT NULL DEFAULT TRUE,

    -- OAuth 2.0 core
    token_endpoint_auth_method   VARCHAR(30) NOT NULL DEFAULT 'client_secret_basic',
    grant_types                  TEXT[]      NOT NULL DEFAULT '{authorization_code}',
    response_types               TEXT[]      NOT NULL DEFAULT '{code}',
    require_consent              BOOLEAN     NOT NULL DEFAULT TRUE,
    -- PKCE (RFC 7636). Mandatory for public clients (SPA/mobile) per OAuth 2.0
    -- Security BCP; defaults TRUE so omitting it is the secure choice.
    require_pkce                 BOOLEAN     NOT NULL DEFAULT TRUE,
    -- Per-client scope allowlist (empty = all scopes permitted).
    allowed_scopes               TEXT[]      NOT NULL DEFAULT '{}',

    -- Token lifetime overrides (seconds). NULL = inherit tenant token_config.
    access_token_ttl             INTEGER,
    refresh_token_ttl            INTEGER,

    -- Security overrides — runtime enforcement only, tighten-only, NULL = inherit
    -- the tenant security_settings default. Capability/credential policy
    -- (allowed MFA methods, password, lockout, threat, registration) is NOT
    -- overridable per client and stays at the tenant level.
    required_acr                 VARCHAR(10),   -- MFA/step-up: '1' = pwd, '2' = step-up
    session_idle_timeout         INTEGER,       -- sliding idle window (seconds)
    session_absolute_timeout     INTEGER,       -- hard session cap (seconds)

    -- JWT client auth (RFC 7523): embedded JWKS or URI for private_key_jwt / client_secret_jwt.
    jwks                         JSONB,
    jwks_uri                     TEXT,

    -- mTLS client auth (RFC 8705): expected certificate SHA-256 thumbprint.
    mtls_bound_cert_thumbprint   TEXT,

    -- Scope-to-claim mapping: maps scope names to OIDC claim names to include in tokens.
    -- Format: {"email": ["email", "email_verified"], "profile": ["given_name", "family_name"]}
    -- When null, the standard OIDC default mapping applies.
    scope_claim_mappings         JSONB,

    -- Custom claim mappers: static or metadata-derived claims injected into tokens per client/tenant.
    -- Format: {"claim_name": "static_value", "org_id": "{{user.metadata.org_id}}"}
    claim_mappers                JSONB,

    -- Audit
    created_by                   BIGINT,
    updated_by                   BIGINT,
    created_at                   TIMESTAMPTZ DEFAULT now(),
    updated_at                   TIMESTAMPTZ DEFAULT now(),
    deleted_at                   TIMESTAMPTZ,

    CONSTRAINT chk_clients_token_auth_method CHECK (
        token_endpoint_auth_method IN (
            'client_secret_basic', 'client_secret_post', 'none',
            'private_key_jwt', 'client_secret_jwt',
            'tls_client_auth', 'self_signed_tls_client_auth'
        )
    ),
    CONSTRAINT chk_clients_client_type CHECK (
        client_type IN ('traditional', 'spa', 'mobile', 'm2m')
    ),
    CONSTRAINT chk_clients_required_acr CHECK (
        required_acr IS NULL OR required_acr IN ('1', '2')
    ),
    CONSTRAINT chk_clients_session_idle_timeout CHECK (
        session_idle_timeout IS NULL OR session_idle_timeout > 0
    ),
    CONSTRAINT chk_clients_session_absolute_timeout CHECK (
        session_absolute_timeout IS NULL OR session_absolute_timeout > 0
    ),
    CONSTRAINT chk_clients_session_timeout_order CHECK (
        session_idle_timeout IS NULL
        OR session_absolute_timeout IS NULL
        OR session_absolute_timeout >= session_idle_timeout
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
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_clients_branding_id'
    ) THEN
        ALTER TABLE clients
            ADD CONSTRAINT fk_clients_branding_id FOREIGN KEY (branding_id)
            REFERENCES branding(branding_id) ON DELETE SET NULL;
    END IF;

END$$;

-- ADD INDEXES
-- Composite indexes for common query patterns
CREATE INDEX IF NOT EXISTS idx_clients_tenant_id_status ON clients (tenant_id, status);
CREATE INDEX IF NOT EXISTS idx_clients_tenant_id_is_default ON clients (tenant_id, is_default) WHERE is_default = TRUE;
CREATE INDEX IF NOT EXISTS idx_clients_service_id ON clients (service_id) WHERE service_id IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS uq_clients_tenant_name ON clients (tenant_id, name) WHERE deleted_at IS NULL;

-- Single column indexes
CREATE INDEX IF NOT EXISTS idx_clients_identifier ON clients (identifier) WHERE identifier IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS uq_clients_identifier ON clients (identifier) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_clients_is_system ON clients (is_system) WHERE is_system = TRUE;
CREATE INDEX IF NOT EXISTS idx_clients_branding_id ON clients (branding_id) WHERE branding_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_clients_created_at ON clients (created_at);
CREATE INDEX IF NOT EXISTS idx_clients_deleted_at ON clients (deleted_at) WHERE deleted_at IS NULL;

-- OAuth indexes
CREATE INDEX IF NOT EXISTS idx_clients_grant_types ON clients USING GIN (grant_types);
`
	return db.Exec(sql).Error
}
