package migration

import (
	"gorm.io/gorm"
)

// CreateSecuritySettingsAuditTable creates the security_settings_audit table
// for tracking changes to pool-level security configuration.
func CreateSecuritySettingsAuditTable(db *gorm.DB) error {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS security_settings_audit (
    security_settings_audit_id   SERIAL PRIMARY KEY,
    security_settings_audit_uuid UUID NOT NULL UNIQUE,
    user_pool_id                 INTEGER NOT NULL,
    security_setting_id          INTEGER NOT NULL,
    change_type                  VARCHAR(50) NOT NULL,
    old_config                   JSONB,
    new_config                   JSONB,
    ip_address                   VARCHAR(50),
    user_agent                   TEXT,
    created_by                   INTEGER,
    created_at                   TIMESTAMPTZ DEFAULT now(),
    updated_at                   TIMESTAMPTZ DEFAULT now()
);

-- ADD CONSTRAINTS
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_security_settings_audit_user_pool_id'
    ) THEN
        ALTER TABLE security_settings_audit
            ADD CONSTRAINT fk_security_settings_audit_user_pool_id FOREIGN KEY (user_pool_id)
            REFERENCES user_pools(user_pool_id) ON DELETE CASCADE;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_security_settings_audit_security_setting_id'
    ) THEN
        ALTER TABLE security_settings_audit
            ADD CONSTRAINT fk_security_settings_audit_security_setting_id FOREIGN KEY (security_setting_id)
            REFERENCES security_settings(security_setting_id) ON DELETE CASCADE;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_security_settings_audit_created_by'
    ) THEN
        ALTER TABLE security_settings_audit
            ADD CONSTRAINT fk_security_settings_audit_created_by FOREIGN KEY (created_by)
            REFERENCES users(user_id) ON DELETE SET NULL;
    END IF;
END$$;

-- CREATE INDEXES
CREATE INDEX IF NOT EXISTS idx_security_settings_audit_uuid ON security_settings_audit (security_settings_audit_uuid);
CREATE INDEX IF NOT EXISTS idx_security_settings_audit_user_pool_id ON security_settings_audit (user_pool_id);
CREATE INDEX IF NOT EXISTS idx_security_settings_audit_security_setting_id ON security_settings_audit (security_setting_id);
CREATE INDEX IF NOT EXISTS idx_security_settings_audit_change_type ON security_settings_audit (change_type);
CREATE INDEX IF NOT EXISTS idx_security_settings_audit_created_by ON security_settings_audit (created_by);
CREATE INDEX IF NOT EXISTS idx_security_settings_audit_created_at ON security_settings_audit (created_at);
`

	return db.Exec(sql).Error
}
