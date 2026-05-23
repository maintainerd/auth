package migration

import (
	"gorm.io/gorm"
)

// CreateBrandingTable creates the branding table for tenant-level UI
// customisation consumed by auth-console (port 8080).
func CreateBrandingTable(db *gorm.DB) error {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS branding (
    branding_id          BIGSERIAL PRIMARY KEY,
    branding_uuid        UUID NOT NULL UNIQUE,
    tenant_id            BIGINT NOT NULL,
    company_name         VARCHAR(255),
    logo_url             TEXT,
    favicon_url          TEXT,
    primary_color        VARCHAR(20),
    secondary_color      VARCHAR(20),
    accent_color         VARCHAR(20),
    font_family          VARCHAR(100),
    custom_css           TEXT,
    support_url          TEXT,
    privacy_policy_url   TEXT,
    terms_of_service_url TEXT,
    metadata             JSONB DEFAULT '{}',
    created_by           BIGINT,
    updated_by           BIGINT,
    created_at           TIMESTAMPTZ DEFAULT now(),
    updated_at           TIMESTAMPTZ DEFAULT now(),
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
-- NOTE: FK constraints for created_by/updated_by are added in migration 026
-- (users table) since this table is created before users.

-- CREATE INDEXES
CREATE INDEX IF NOT EXISTS idx_branding_uuid ON branding (branding_uuid);
CREATE INDEX IF NOT EXISTS idx_branding_tenant_id ON branding (tenant_id);
CREATE INDEX IF NOT EXISTS idx_branding_created_at ON branding (created_at);
CREATE INDEX IF NOT EXISTS idx_branding_deleted_at ON branding (deleted_at) WHERE deleted_at IS NULL;
`

	return db.Exec(sql).Error
}
