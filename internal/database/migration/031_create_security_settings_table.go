package migration

import (
	"log"

	"gorm.io/gorm"
)

func CreateSecuritySettingsTable(db *gorm.DB) {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS security_settings (
    security_setting_id   SERIAL PRIMARY KEY,
    security_setting_uuid UUID NOT NULL UNIQUE,
    tenant_id             INTEGER NOT NULL,
    general_config        JSONB DEFAULT '{}'::jsonb,
    password_config       JSONB DEFAULT '{}'::jsonb,
    session_config        JSONB DEFAULT '{}'::jsonb,
    threat_config         JSONB DEFAULT '{}'::jsonb,
    ip_config             JSONB DEFAULT '{}'::jsonb,
    version               INTEGER DEFAULT 1,
    created_by            INTEGER,
    updated_by            INTEGER,
    created_at            TIMESTAMPTZ DEFAULT now(),
    updated_at            TIMESTAMPTZ DEFAULT now()
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
CREATE INDEX IF NOT EXISTS idx_security_settings_tenant_id ON security_settings (tenant_id);
CREATE INDEX IF NOT EXISTS idx_security_settings_version ON security_settings (version);
CREATE INDEX IF NOT EXISTS idx_security_settings_created_at ON security_settings (created_at);
`

	if err := db.Exec(sql).Error; err != nil {
		log.Fatalf("❌ Failed to run migration 031_create_security_settings_table: %v", err)
	}

	log.Println("✅ Migration 031_create_security_settings_table executed")
}
