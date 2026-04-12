package migration

import (
	"gorm.io/gorm"
)

// CreateSMSConfigTable creates the sms_config table for tenant-level
// SMS delivery configuration (Twilio, SNS, Vonage, etc.).
func CreateSMSConfigTable(db *gorm.DB) error {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS sms_config (
    sms_config_id           SERIAL PRIMARY KEY,
    sms_config_uuid         UUID NOT NULL UNIQUE,
    tenant_id               INTEGER NOT NULL,
    provider                VARCHAR(50) NOT NULL,
    account_sid             VARCHAR(255),
    auth_token_encrypted    TEXT,
    from_number             VARCHAR(50),
    sender_id               VARCHAR(50),
    test_mode               BOOLEAN NOT NULL DEFAULT false,
    status                  VARCHAR(20) NOT NULL DEFAULT 'active',
    metadata                JSONB DEFAULT '{}',
    created_at              TIMESTAMPTZ DEFAULT now(),
    updated_at              TIMESTAMPTZ DEFAULT now()
);

-- ADD CONSTRAINTS
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_sms_config_tenant_id'
    ) THEN
        ALTER TABLE sms_config
            ADD CONSTRAINT fk_sms_config_tenant_id FOREIGN KEY (tenant_id)
            REFERENCES tenants(tenant_id) ON DELETE CASCADE;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'chk_sms_config_status'
    ) THEN
        ALTER TABLE sms_config
            ADD CONSTRAINT chk_sms_config_status CHECK (status IN ('active', 'inactive'));
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'chk_sms_config_provider'
    ) THEN
        ALTER TABLE sms_config
            ADD CONSTRAINT chk_sms_config_provider CHECK (provider IN ('twilio', 'sns', 'vonage', 'messagebird'));
    END IF;
END$$;

-- CREATE INDEXES
CREATE INDEX IF NOT EXISTS idx_sms_config_uuid ON sms_config (sms_config_uuid);
CREATE INDEX IF NOT EXISTS idx_sms_config_tenant_id ON sms_config (tenant_id);
CREATE INDEX IF NOT EXISTS idx_sms_config_status ON sms_config (status);
CREATE INDEX IF NOT EXISTS idx_sms_config_created_at ON sms_config (created_at);
`

	return db.Exec(sql).Error
}
