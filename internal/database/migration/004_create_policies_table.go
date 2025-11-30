package migration

import (
	"log"

	"gorm.io/gorm"
)

func CreatePoliciesTable(db *gorm.DB) {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS policies (
    policy_id       SERIAL PRIMARY KEY,
    policy_uuid     UUID NOT NULL UNIQUE,
    name            VARCHAR(150) NOT NULL,
    description     TEXT,
    document        JSONB NOT NULL,
    version         VARCHAR(20) NOT NULL,
    status          VARCHAR(20) DEFAULT 'inactive' CHECK (status IN ('active', 'inactive')),
    is_default      BOOLEAN DEFAULT FALSE,
    is_system       BOOLEAN DEFAULT FALSE,
    created_at      TIMESTAMPTZ DEFAULT now(),
    updated_at      TIMESTAMPTZ DEFAULT now()
);

-- ADD INDEXES
CREATE INDEX IF NOT EXISTS idx_policies_uuid ON policies (policy_uuid);
CREATE INDEX IF NOT EXISTS idx_policies_name ON policies (name);
CREATE INDEX IF NOT EXISTS idx_policies_document ON policies (document);
CREATE INDEX IF NOT EXISTS idx_policies_version ON policies (version);
CREATE INDEX IF NOT EXISTS idx_policies_status ON policies (status);
CREATE INDEX IF NOT EXISTS idx_policies_is_default ON policies (is_default);
CREATE INDEX IF NOT EXISTS idx_policies_is_system ON policies (is_system);
CREATE INDEX IF NOT EXISTS idx_policies_created_at ON policies (created_at);
`

	if err := db.Exec(sql).Error; err != nil {
		log.Fatalf("❌ Failed to run migration 004_create_policies_table: %v", err)
	}

	log.Println("✅ Migration 004_create_policies_table executed")
}
