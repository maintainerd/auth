package migration

import (
	"gorm.io/gorm"
)

func CreateSMSTemplatesTable(db *gorm.DB) error {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS sms_templates (
    sms_template_id   BIGSERIAL PRIMARY KEY,
    sms_template_uuid UUID NOT NULL UNIQUE,
    tenant_id         BIGINT NOT NULL,
    name              VARCHAR(100) NOT NULL,
    description       TEXT,
    message           TEXT NOT NULL,
    parameters_doc    TEXT,
    status            VARCHAR(20) NOT NULL DEFAULT 'active',
    is_default        BOOLEAN NOT NULL DEFAULT false,
    is_system         BOOLEAN NOT NULL DEFAULT false,
    created_by        BIGINT,
    updated_by        BIGINT,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at        TIMESTAMPTZ
);

-- ADD CONSTRAINTS
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_sms_templates_tenant_id'
    ) THEN
        ALTER TABLE sms_templates
            ADD CONSTRAINT fk_sms_templates_tenant_id
            FOREIGN KEY (tenant_id) REFERENCES tenants(tenant_id) ON DELETE CASCADE;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_sms_templates_created_by'
    ) THEN
        ALTER TABLE sms_templates
            ADD CONSTRAINT fk_sms_templates_created_by FOREIGN KEY (created_by)
            REFERENCES users(user_id) ON DELETE SET NULL;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_sms_templates_updated_by'
    ) THEN
        ALTER TABLE sms_templates
            ADD CONSTRAINT fk_sms_templates_updated_by FOREIGN KEY (updated_by)
            REFERENCES users(user_id) ON DELETE SET NULL;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'chk_sms_templates_status'
    ) THEN
        ALTER TABLE sms_templates
            ADD CONSTRAINT chk_sms_templates_status CHECK (status IN ('active', 'inactive'));
    END IF;
END$$;

-- CREATE INDEXES
CREATE INDEX IF NOT EXISTS idx_sms_templates_uuid ON sms_templates (sms_template_uuid);
CREATE INDEX IF NOT EXISTS idx_sms_templates_tenant_id ON sms_templates (tenant_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_sms_templates_tenant_name ON sms_templates (tenant_id, name) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_sms_templates_status ON sms_templates (status);
CREATE INDEX IF NOT EXISTS idx_sms_templates_is_default ON sms_templates (is_default);
CREATE INDEX IF NOT EXISTS idx_sms_templates_is_system ON sms_templates (is_system);
CREATE INDEX IF NOT EXISTS idx_sms_templates_created_at ON sms_templates (created_at);
CREATE INDEX IF NOT EXISTS idx_sms_templates_deleted_at ON sms_templates (deleted_at) WHERE deleted_at IS NULL;
`

	return db.Exec(sql).Error
}
