package migration

import (
	"gorm.io/gorm"
)

// CreateTenantSettingsTable creates the tenant_settings table for tenant-level
// operational configuration (rate limits, audit, maintenance, feature flags).
func CreateTenantSettingsTable(db *gorm.DB) error {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS tenant_settings (
    tenant_setting_id   SERIAL PRIMARY KEY,
    tenant_setting_uuid UUID NOT NULL UNIQUE,
    tenant_id           INTEGER NOT NULL,
    rate_limit_config   JSONB DEFAULT '{}',
    audit_config        JSONB DEFAULT '{}',
    maintenance_config  JSONB DEFAULT '{}',
    feature_flags       JSONB DEFAULT '{}',
    created_at          TIMESTAMPTZ DEFAULT now(),
    updated_at          TIMESTAMPTZ DEFAULT now()
);

-- ADD CONSTRAINTS
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_tenant_settings_tenant_id'
    ) THEN
        ALTER TABLE tenant_settings
            ADD CONSTRAINT fk_tenant_settings_tenant_id FOREIGN KEY (tenant_id)
            REFERENCES tenants(tenant_id) ON DELETE CASCADE;
    END IF;
END$$;

-- CREATE INDEXES
CREATE INDEX IF NOT EXISTS idx_tenant_settings_uuid ON tenant_settings (tenant_setting_uuid);
CREATE INDEX IF NOT EXISTS idx_tenant_settings_tenant_id ON tenant_settings (tenant_id);
CREATE INDEX IF NOT EXISTS idx_tenant_settings_created_at ON tenant_settings (created_at);
`

	return db.Exec(sql).Error
}
