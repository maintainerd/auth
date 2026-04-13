package migration

import (
	"gorm.io/gorm"
)

// CreateEmailConfigTable creates the email_config table for tenant-level
// SMTP/SES/SendGrid delivery configuration.
func CreateEmailConfigTable(db *gorm.DB) error {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS email_config (
    email_config_id     SERIAL PRIMARY KEY,
    email_config_uuid   UUID NOT NULL UNIQUE,
    tenant_id           INTEGER NOT NULL,
    provider            VARCHAR(50) NOT NULL,
    host                VARCHAR(255),
    port                INTEGER,
    username            VARCHAR(255),
    password_encrypted  TEXT,
    from_address        VARCHAR(255) NOT NULL,
    from_name           VARCHAR(255),
    reply_to            VARCHAR(255),
    encryption          VARCHAR(20),
    test_mode           BOOLEAN NOT NULL DEFAULT false,
    status              VARCHAR(20) NOT NULL DEFAULT 'active',
    metadata            JSONB DEFAULT '{}',
    created_at          TIMESTAMPTZ DEFAULT now(),
    updated_at          TIMESTAMPTZ DEFAULT now()
);

-- ADD CONSTRAINTS
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_email_config_tenant_id'
    ) THEN
        ALTER TABLE email_config
            ADD CONSTRAINT fk_email_config_tenant_id FOREIGN KEY (tenant_id)
            REFERENCES tenants(tenant_id) ON DELETE CASCADE;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'chk_email_config_status'
    ) THEN
        ALTER TABLE email_config
            ADD CONSTRAINT chk_email_config_status CHECK (status IN ('active', 'inactive'));
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'chk_email_config_encryption'
    ) THEN
        ALTER TABLE email_config
            ADD CONSTRAINT chk_email_config_encryption CHECK (encryption IN ('tls', 'ssl', 'none'));
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'chk_email_config_provider'
    ) THEN
        ALTER TABLE email_config
            ADD CONSTRAINT chk_email_config_provider CHECK (provider IN ('smtp', 'ses', 'sendgrid', 'mailgun', 'postmark', 'resend'));
    END IF;
END$$;

-- CREATE INDEXES
CREATE INDEX IF NOT EXISTS idx_email_config_uuid ON email_config (email_config_uuid);
CREATE INDEX IF NOT EXISTS idx_email_config_tenant_id ON email_config (tenant_id);
CREATE INDEX IF NOT EXISTS idx_email_config_status ON email_config (status);
CREATE INDEX IF NOT EXISTS idx_email_config_created_at ON email_config (created_at);
`

	return db.Exec(sql).Error
}
