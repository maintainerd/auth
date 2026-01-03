package migration

import (
	"log"

	"gorm.io/gorm"
)

func CreateEmailTemplatesTable(db *gorm.DB) {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS email_templates (
    email_template_id     SERIAL PRIMARY KEY,
    email_template_uuid		UUID NOT NULL UNIQUE,
    tenant_id             BIGINT NOT NULL,
    name            			VARCHAR(100) NOT NULL UNIQUE,
    subject         			VARCHAR(255) NOT NULL,
    body_html       			TEXT NOT NULL,
    body_plain      			TEXT,
    status          			VARCHAR(20) DEFAULT 'active',
    is_default      			BOOLEAN DEFAULT FALSE,
    is_system       			BOOLEAN DEFAULT FALSE,
    created_at      			TIMESTAMPTZ DEFAULT now(),
    updated_at      			TIMESTAMPTZ DEFAULT now(),
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
END$$;

-- CREATE INDEXES
CREATE INDEX IF NOT EXISTS idx_email_templates_uuid ON email_templates (email_template_uuid);
CREATE INDEX IF NOT EXISTS idx_email_templates_tenant_id ON email_templates (tenant_id);
CREATE INDEX IF NOT EXISTS idx_email_templates_name ON email_templates (name);
CREATE INDEX IF NOT EXISTS idx_email_templates_status ON email_templates (status);
CREATE INDEX IF NOT EXISTS idx_email_templates_is_default ON email_templates (is_default);
CREATE INDEX IF NOT EXISTS idx_email_templates_is_system ON email_templates (is_system);
CREATE INDEX IF NOT EXISTS idx_email_templates_created_at ON email_templates (created_at);
`

	if err := db.Exec(sql).Error; err != nil {
		log.Fatalf("❌ Failed to run migration 036_create_email_templates_table: %v", err)
	}

	log.Println("✅ Migration 036_create_email_templates_table executed")
}
