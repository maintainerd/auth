package migration

import (
	"gorm.io/gorm"
)

// CreateIPRestrictionRulesTable creates the ip_restriction_rules table.
// Changes from prior versions:
//   - ip_address: VARCHAR(50) → INET (native Postgres type, supports IPv4/IPv6 + CIDR)
//   - type enum deduped: dropped 'whitelist' (== 'allow') and 'blacklist' (== 'deny')
//   - added deleted_at for soft delete
func CreateIPRestrictionRulesTable(db *gorm.DB) error {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS ip_restriction_rules (
    ip_restriction_rule_id   BIGSERIAL PRIMARY KEY,
    ip_restriction_rule_uuid UUID NOT NULL UNIQUE,
    tenant_id                BIGINT NOT NULL,
    description              TEXT,
    type                     VARCHAR(20) NOT NULL,
    ip_address               INET NOT NULL,
    status                   VARCHAR(20) NOT NULL DEFAULT 'active',
    created_by               BIGINT,
    updated_by               BIGINT,
    created_at               TIMESTAMPTZ DEFAULT now(),
    updated_at               TIMESTAMPTZ DEFAULT now(),
    deleted_at               TIMESTAMPTZ
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
            ADD CONSTRAINT chk_ip_restriction_rules_type CHECK (type IN ('allow', 'deny'));
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
CREATE INDEX IF NOT EXISTS idx_ip_restriction_rules_deleted_at ON ip_restriction_rules (deleted_at) WHERE deleted_at IS NULL;
`

	return db.Exec(sql).Error
}
