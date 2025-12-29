package migration

import (
	"log"

	"gorm.io/gorm"
)

func CreateSmsTemplatesTable(db *gorm.DB) {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS sms_templates (
    sms_template_id   SERIAL PRIMARY KEY,
    sms_template_uuid UUID NOT NULL UNIQUE,
    name              VARCHAR(100) NOT NULL UNIQUE,
    description       TEXT,
    message           TEXT NOT NULL,
    sender_id         VARCHAR(20),
    status            VARCHAR(20) NOT NULL DEFAULT 'active',
    metadata          JSONB DEFAULT '{}',
    is_default        BOOLEAN DEFAULT false,
    is_system         BOOLEAN DEFAULT false,
    created_at        TIMESTAMPTZ DEFAULT now(),
    updated_at        TIMESTAMPTZ DEFAULT now()
);

-- ADD CONSTRAINTS
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'chk_sms_templates_status'
    ) THEN
        ALTER TABLE sms_templates
            ADD CONSTRAINT chk_sms_templates_status CHECK (status IN ('active', 'inactive'));
    END IF;
END$$;

-- CREATE INDEXES
CREATE INDEX IF NOT EXISTS idx_sms_templates_uuid ON sms_templates (sms_template_uuid);
CREATE INDEX IF NOT EXISTS idx_sms_templates_name ON sms_templates (name);
CREATE INDEX IF NOT EXISTS idx_sms_templates_status ON sms_templates (status);
CREATE INDEX IF NOT EXISTS idx_sms_templates_sender_id ON sms_templates (sender_id);
CREATE INDEX IF NOT EXISTS idx_sms_templates_is_default ON sms_templates (is_default);
CREATE INDEX IF NOT EXISTS idx_sms_templates_is_system ON sms_templates (is_system);
CREATE INDEX IF NOT EXISTS idx_sms_templates_created_at ON sms_templates (created_at);
`

	if err := db.Exec(sql).Error; err != nil {
		log.Fatalf("❌ Failed to run migration 036_create_sms_templates_table: %v", err)
	}

	log.Println("✅ Migration 036_create_sms_templates_table executed")
}
