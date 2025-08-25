package migration

import (
	"log"

	"gorm.io/gorm"
)

func CreateEmailTemplatesTable(db *gorm.DB) {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS email_templates (
    template_id     SERIAL PRIMARY KEY,
    template_uuid   UUID NOT NULL UNIQUE,
    name            VARCHAR(100) NOT NULL UNIQUE,
    subject         VARCHAR(255) NOT NULL,
    body_html       TEXT NOT NULL,
    body_plain      TEXT,
    is_active       BOOLEAN DEFAULT TRUE,
    created_at      TIMESTAMPTZ DEFAULT now(),
    updated_at      TIMESTAMPTZ DEFAULT now()
);

-- CREATE INDEXES
CREATE INDEX IF NOT EXISTS idx_email_templates_name ON email_templates(name);
CREATE INDEX IF NOT EXISTS idx_email_templates_is_active ON email_templates(is_active);
`

	if err := db.Exec(sql).Error; err != nil {
		log.Fatalf("❌ Failed to run migration 022_create_email_templates_table: %v", err)
	}

	log.Println("✅ Migration 022_create_email_templates_table executed")
}
