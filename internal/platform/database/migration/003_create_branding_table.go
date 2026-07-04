package migration

import (
	"gorm.io/gorm"
)

// CreateBrandingTable creates the branding table — per-tenant visual identity
// (logo + colors) consumed by both auth-console and auth-identity. A tenant has
// many branding rows: one immutable system record (is_system) and any number of
// custom ones, with exactly one active (is_active) that drives the global look.
// All theme tokens (primary/secondary colors, panel/sidebar/app backgrounds,
// fonts, …) live in the `metadata` JSONB so the palette can grow without schema
// changes. Only stable, first-class fields get columns (logo, favicon, legal
// URLs and the selected hosted-login layout). Custom CSS remains unsupported.
func CreateBrandingTable(db *gorm.DB) error {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS branding (
    branding_id          BIGSERIAL PRIMARY KEY,
    branding_uuid        UUID NOT NULL UNIQUE,
    tenant_id            BIGINT NOT NULL,
    name                 VARCHAR(100),
    is_system            BOOLEAN NOT NULL DEFAULT false,
    is_active            BOOLEAN NOT NULL DEFAULT false,
    layout               VARCHAR(32) NOT NULL DEFAULT 'centered',
    company_name         VARCHAR(255),
    logo_url             TEXT,
    logo_data            BYTEA,
    logo_content_type    VARCHAR(255),
    favicon_url          TEXT,
    support_url          TEXT,
    privacy_policy_url   TEXT,
    terms_of_service_url TEXT,
    metadata             JSONB NOT NULL DEFAULT '{}',
    created_by           BIGINT,
    updated_by           BIGINT,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at           TIMESTAMPTZ
);

-- ADD CONSTRAINTS
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_branding_tenant_id'
    ) THEN
        ALTER TABLE branding
            ADD CONSTRAINT fk_branding_tenant_id FOREIGN KEY (tenant_id)
            REFERENCES tenants(tenant_id) ON DELETE CASCADE;
    END IF;
END$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'chk_branding_layout'
    ) THEN
        ALTER TABLE branding
            ADD CONSTRAINT chk_branding_layout
            CHECK (layout IN ('centered', 'full_page', 'split'));
    END IF;
END$$;
-- NOTE: FK constraints for created_by/updated_by are added in migration 026
-- (users table) since this table is created before users.

-- CREATE INDEXES
CREATE INDEX IF NOT EXISTS idx_branding_uuid ON branding (branding_uuid);
CREATE INDEX IF NOT EXISTS idx_branding_tenant_id ON branding (tenant_id);
CREATE INDEX IF NOT EXISTS idx_branding_created_at ON branding (created_at);
CREATE INDEX IF NOT EXISTS idx_branding_deleted_at ON branding (deleted_at) WHERE deleted_at IS NULL;

-- At most one active branding per tenant (ignoring soft-deleted rows).
CREATE UNIQUE INDEX IF NOT EXISTS uq_branding_active_per_tenant
    ON branding (tenant_id) WHERE is_active AND deleted_at IS NULL;
`

	return db.Exec(sql).Error
}
