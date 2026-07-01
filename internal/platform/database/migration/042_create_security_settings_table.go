package migration

import (
	"gorm.io/gorm"
)

// CreateSecuritySettingsTable creates the security_settings table scoped to a
// tenant. Each tenant gets exactly one row that holds JSONB configs for MFA,
// passwords, sessions, threat detection, lockout, registration, and tokens.
// 1:1 with tenant, cascade-deletes with parent — no soft delete needed.
func CreateSecuritySettingsTable(db *gorm.DB) error {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS security_settings (
    security_setting_id     BIGSERIAL PRIMARY KEY,
    security_setting_uuid   UUID NOT NULL UNIQUE,
    tenant_id               BIGINT NOT NULL,
    mfa_config              JSONB NOT NULL DEFAULT '{}'::jsonb,
    password_config         JSONB NOT NULL DEFAULT '{}'::jsonb,
    session_config          JSONB NOT NULL DEFAULT '{}'::jsonb,
    threat_config           JSONB NOT NULL DEFAULT '{}'::jsonb,
    lockout_config          JSONB NOT NULL DEFAULT '{}'::jsonb,
    registration_config     JSONB NOT NULL DEFAULT '{}'::jsonb,
    token_config            JSONB NOT NULL DEFAULT '{}'::jsonb,
    version                 INTEGER NOT NULL DEFAULT 1,
    created_by              BIGINT,
    updated_by              BIGINT,
    created_at              TIMESTAMPTZ DEFAULT now(),
    updated_at              TIMESTAMPTZ DEFAULT now()
);

-- ADD CONSTRAINTS
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_security_settings_tenant_id'
    ) THEN
        ALTER TABLE security_settings
            ADD CONSTRAINT fk_security_settings_tenant_id FOREIGN KEY (tenant_id)
            REFERENCES tenants(tenant_id) ON DELETE CASCADE;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_security_settings_created_by'
    ) THEN
        ALTER TABLE security_settings
            ADD CONSTRAINT fk_security_settings_created_by FOREIGN KEY (created_by)
            REFERENCES users(user_id) ON DELETE SET NULL;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_security_settings_updated_by'
    ) THEN
        ALTER TABLE security_settings
            ADD CONSTRAINT fk_security_settings_updated_by FOREIGN KEY (updated_by)
            REFERENCES users(user_id) ON DELETE SET NULL;
    END IF;
END$$;

-- CREATE INDEXES
CREATE INDEX IF NOT EXISTS idx_security_settings_uuid ON security_settings (security_setting_uuid);
CREATE UNIQUE INDEX IF NOT EXISTS uq_security_settings_tenant_id ON security_settings (tenant_id);
CREATE INDEX IF NOT EXISTS idx_security_settings_version ON security_settings (version);
CREATE INDEX IF NOT EXISTS idx_security_settings_created_at ON security_settings (created_at);
`

	return db.Exec(sql).Error
}
