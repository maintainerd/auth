package migration

import (
	"gorm.io/gorm"
)

// CreateTenantSettingsTable creates the tenant_settings table for tenant-level
// operational configuration (rate limits, audit, maintenance).
// This is a 1:1 config row per tenant — no soft delete or audit columns since
// the row's lifecycle is bound to the parent tenant via ON DELETE CASCADE.
func CreateTenantSettingsTable(db *gorm.DB) error {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS tenant_settings (
    tenant_setting_id   BIGSERIAL PRIMARY KEY,
    tenant_setting_uuid UUID NOT NULL UNIQUE,
    tenant_id           BIGINT NOT NULL,
    rate_limit_config   JSONB NOT NULL DEFAULT '{"enabled":false,"requests_per_window":100,"window_duration_seconds":60,"per_ip":true,"per_api_key":true,"exempt_ips":[],"endpoint_overrides":{}}',
    audit_config        JSONB NOT NULL DEFAULT '{"enabled":true,"retention_days":90,"pii_masking":true,"log_level":"info","event_types":[]}',
    maintenance_config  JSONB NOT NULL DEFAULT '{"enabled":false,"message":"The system is currently undergoing maintenance. Please try again later.","scheduled_start":null,"scheduled_end":null}',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
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
CREATE UNIQUE INDEX IF NOT EXISTS idx_tenant_settings_tenant_id ON tenant_settings (tenant_id);
CREATE INDEX IF NOT EXISTS idx_tenant_settings_created_at ON tenant_settings (created_at);
`

	return db.Exec(sql).Error
}
