package migration

import (
	"log"

	"gorm.io/gorm"
)

func CreateLoginTemplatesTable(db *gorm.DB) {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS login_templates (
    login_template_id   SERIAL PRIMARY KEY,
    login_template_uuid UUID NOT NULL UNIQUE,
    name                VARCHAR(100) NOT NULL UNIQUE,
    description         TEXT,
    template            VARCHAR(20) NOT NULL,
    status              VARCHAR(20) NOT NULL DEFAULT 'active',
    metadata            JSONB DEFAULT '{}',
    is_default          BOOLEAN DEFAULT false,
    is_system           BOOLEAN DEFAULT false,
    created_at          TIMESTAMPTZ DEFAULT now(),
    updated_at          TIMESTAMPTZ DEFAULT now()
);

-- ADD CONSTRAINTS
DO $$
BEGIN
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
CREATE INDEX IF NOT EXISTS idx_login_templates_name ON login_templates (name);
CREATE INDEX IF NOT EXISTS idx_login_templates_status ON login_templates (status);
CREATE INDEX IF NOT EXISTS idx_login_templates_is_default ON login_templates (is_default);
CREATE INDEX IF NOT EXISTS idx_login_templates_is_system ON login_templates (is_system);
CREATE INDEX IF NOT EXISTS idx_login_templates_created_at ON login_templates (created_at);
`

	if err := db.Exec(sql).Error; err != nil {
		log.Fatalf("❌ Failed to run migration 033_create_login_templates_table: %v", err)
	}

	log.Println("✅ Migration 033_create_login_templates_table executed")
}
