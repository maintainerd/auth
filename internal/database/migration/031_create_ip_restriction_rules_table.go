package migration

import (
	"log"

	"gorm.io/gorm"
)

func CreateIpRestrictionRulesTable(db *gorm.DB) {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS ip_restriction_rules (
    ip_restriction_rule_id   SERIAL PRIMARY KEY,
    ip_restriction_rule_uuid UUID NOT NULL UNIQUE,
    tenant_id                INTEGER NOT NULL,
    description              TEXT,
    type                     VARCHAR(20) NOT NULL,
    ip_address               VARCHAR(50) NOT NULL,
    status                   VARCHAR(20) NOT NULL DEFAULT 'active',
    created_by               INTEGER,
    updated_by               INTEGER,
    created_at               TIMESTAMPTZ DEFAULT now(),
    updated_at               TIMESTAMPTZ DEFAULT now()
);

-- ADD CONSTRAINTS
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_ip_restriction_rules_tenant_id'
    ) THEN
        ALTER TABLE ip_restriction_rules
            ADD CONSTRAINT fk_ip_restriction_rules_tenant_id FOREIGN KEY (tenant_id)
            REFERENCES tenants(tenant_id) ON DELETE CASCADE;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'chk_ip_restriction_rules_type'
    ) THEN
        ALTER TABLE ip_restriction_rules
            ADD CONSTRAINT chk_ip_restriction_rules_type CHECK (type IN ('allow', 'deny', 'whitelist', 'blacklist'));
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'chk_ip_restriction_rules_status'
    ) THEN
        ALTER TABLE ip_restriction_rules
            ADD CONSTRAINT chk_ip_restriction_rules_status CHECK (status IN ('active', 'inactive'));
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_ip_restriction_rules_created_by'
    ) THEN
        ALTER TABLE ip_restriction_rules
            ADD CONSTRAINT fk_ip_restriction_rules_created_by FOREIGN KEY (created_by)
            REFERENCES users(user_id) ON DELETE SET NULL;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_ip_restriction_rules_updated_by'
    ) THEN
        ALTER TABLE ip_restriction_rules
            ADD CONSTRAINT fk_ip_restriction_rules_updated_by FOREIGN KEY (updated_by)
            REFERENCES users(user_id) ON DELETE SET NULL;
    END IF;
END$$;

-- CREATE INDEXES
CREATE INDEX IF NOT EXISTS idx_ip_restriction_rules_uuid ON ip_restriction_rules (ip_restriction_rule_uuid);
CREATE INDEX IF NOT EXISTS idx_ip_restriction_rules_tenant_id ON ip_restriction_rules (tenant_id);
CREATE INDEX IF NOT EXISTS idx_ip_restriction_rules_type ON ip_restriction_rules (type);
CREATE INDEX IF NOT EXISTS idx_ip_restriction_rules_status ON ip_restriction_rules (status);
CREATE INDEX IF NOT EXISTS idx_ip_restriction_rules_ip_address ON ip_restriction_rules (ip_address);
CREATE INDEX IF NOT EXISTS idx_ip_restriction_rules_created_at ON ip_restriction_rules (created_at);
`

	if err := db.Exec(sql).Error; err != nil {
		log.Fatalf("❌ Failed to run migration 031_create_ip_restriction_rules_table: %v", err)
	}

	log.Println("✅ Migration 031_create_ip_restriction_rules_table executed")
}
