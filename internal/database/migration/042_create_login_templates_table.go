package migration

import (
	"gorm.io/gorm"
)

func CreateLoginTemplatesTable(db *gorm.DB) error {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS login_templates (
    login_template_id   BIGSERIAL PRIMARY KEY,
    login_template_uuid UUID NOT NULL UNIQUE,
    tenant_id           BIGINT NOT NULL,
    name                VARCHAR(100) NOT NULL,
    description         TEXT,
    template            VARCHAR(20) NOT NULL,
    status              VARCHAR(20) NOT NULL DEFAULT 'active',
    metadata            JSONB DEFAULT '{}',
    is_default          BOOLEAN DEFAULT false,
    is_system           BOOLEAN DEFAULT false,
    created_by          BIGINT,
    updated_by          BIGINT,
    created_at          TIMESTAMPTZ DEFAULT now(),
    updated_at          TIMESTAMPTZ DEFAULT now(),
    deleted_at          TIMESTAMPTZ
);

-- ADD CONSTRAINTS
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_login_templates_tenant_id'
    ) THEN
        ALTER TABLE login_templates
            ADD CONSTRAINT fk_login_templates_tenant_id FOREIGN KEY (tenant_id)
            REFERENCES tenants(tenant_id) ON DELETE CASCADE;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_login_templates_created_by'
    ) THEN
        ALTER TABLE login_templates
            ADD CONSTRAINT fk_login_templates_created_by FOREIGN KEY (created_by)
            REFERENCES users(user_id) ON DELETE SET NULL;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_login_templates_updated_by'
    ) THEN
        ALTER TABLE login_templates
            ADD CONSTRAINT fk_login_templates_updated_by FOREIGN KEY (updated_by)
            REFERENCES users(user_id) ON DELETE SET NULL;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'chk_login_templates_status'
    ) THEN
        ALTER TABLE login_templates
            ADD CONSTRAINT chk_login_templates_status CHECK (status IN ('active', 'inactive'));
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'chk_login_templates_template'
    ) THEN
        ALTER TABLE login_templates
            ADD CONSTRAINT chk_login_templates_template CHECK (template IN ('modern', 'classic', 'minimal', 'corporate', 'creative', 'custom'));
    END IF;
END$$;

-- CREATE INDEXES
CREATE INDEX IF NOT EXISTS idx_login_templates_uuid ON login_templates (login_template_uuid);
CREATE INDEX IF NOT EXISTS idx_login_templates_tenant_id ON login_templates (tenant_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_login_templates_tenant_name ON login_templates (tenant_id, name) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_login_templates_status ON login_templates (status);
CREATE INDEX IF NOT EXISTS idx_login_templates_is_default ON login_templates (is_default);
CREATE INDEX IF NOT EXISTS idx_login_templates_is_system ON login_templates (is_system);
CREATE INDEX IF NOT EXISTS idx_login_templates_created_at ON login_templates (created_at);
CREATE INDEX IF NOT EXISTS idx_login_templates_deleted_at ON login_templates (deleted_at) WHERE deleted_at IS NULL;
`

	return db.Exec(sql).Error
}
