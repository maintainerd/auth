package migration

import (
	"gorm.io/gorm"
)

func CreateEmailTemplatesTable(db *gorm.DB) error {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS email_templates (
    email_template_id   BIGSERIAL PRIMARY KEY,
    email_template_uuid UUID NOT NULL UNIQUE,
    tenant_id           BIGINT NOT NULL,
    name                VARCHAR(100) NOT NULL,
    subject             VARCHAR(255) NOT NULL,
    body_html           TEXT NOT NULL,
    body_plain          TEXT,
    parameters_doc      TEXT,
    status              VARCHAR(20) DEFAULT 'active',
    is_default          BOOLEAN DEFAULT FALSE,
    is_system           BOOLEAN DEFAULT FALSE,
    created_by          BIGINT,
    updated_by          BIGINT,
    created_at          TIMESTAMPTZ DEFAULT now(),
    updated_at          TIMESTAMPTZ DEFAULT now(),
    deleted_at          TIMESTAMPTZ,
    CONSTRAINT chk_email_templates_status CHECK (status IN ('active', 'inactive'))
);

-- ADD CONSTRAINTS
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_email_templates_tenant_id'
    ) THEN
        ALTER TABLE email_templates
            ADD CONSTRAINT fk_email_templates_tenant_id FOREIGN KEY (tenant_id)
            REFERENCES tenants(tenant_id) ON DELETE CASCADE;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_email_templates_created_by'
    ) THEN
        ALTER TABLE email_templates
            ADD CONSTRAINT fk_email_templates_created_by FOREIGN KEY (created_by)
            REFERENCES users(user_id) ON DELETE SET NULL;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_email_templates_updated_by'
    ) THEN
        ALTER TABLE email_templates
            ADD CONSTRAINT fk_email_templates_updated_by FOREIGN KEY (updated_by)
            REFERENCES users(user_id) ON DELETE SET NULL;
    END IF;
END$$;

-- CREATE INDEXES
CREATE INDEX IF NOT EXISTS idx_email_templates_uuid ON email_templates (email_template_uuid);
CREATE INDEX IF NOT EXISTS idx_email_templates_tenant_id ON email_templates (tenant_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_email_templates_tenant_name ON email_templates (tenant_id, name) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_email_templates_status ON email_templates (status);
CREATE INDEX IF NOT EXISTS idx_email_templates_is_default ON email_templates (is_default);
CREATE INDEX IF NOT EXISTS idx_email_templates_is_system ON email_templates (is_system);
CREATE INDEX IF NOT EXISTS idx_email_templates_created_at ON email_templates (created_at);
CREATE INDEX IF NOT EXISTS idx_email_templates_deleted_at ON email_templates (deleted_at) WHERE deleted_at IS NULL;
`

	return db.Exec(sql).Error
}
