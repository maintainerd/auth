package migration

import (
	"log"

	"gorm.io/gorm"
)

func CreateTenantMembersTable(db *gorm.DB) {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS tenant_members (
    tenant_member_id   SERIAL PRIMARY KEY,
    tenant_member_uuid UUID NOT NULL UNIQUE,
    tenant_id        BIGINT NOT NULL,
    user_id          BIGINT NOT NULL,
    role             VARCHAR(32) NOT NULL DEFAULT 'member',
    created_at       TIMESTAMPTZ DEFAULT now(),
    updated_at       TIMESTAMPTZ DEFAULT now()
);

-- ADD CHECK CONSTRAINT FOR ROLE
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'chk_tenant_members_role'
    ) THEN
        ALTER TABLE tenant_members
            ADD CONSTRAINT chk_tenant_members_role
            CHECK (role IN ('owner', 'member'));
    END IF;
END$$;

-- ADD CONSTRAINTS (safe)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_tenant_members_tenant_id'
    ) THEN
        ALTER TABLE tenant_members
            ADD CONSTRAINT fk_tenant_members_tenant_id FOREIGN KEY (tenant_id)
            REFERENCES tenants(tenant_id) ON DELETE CASCADE;
    END IF;
END$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_tenant_members_user_id'
    ) THEN
        ALTER TABLE tenant_members
            ADD CONSTRAINT fk_tenant_members_user_id FOREIGN KEY (user_id)
            REFERENCES users(user_id) ON DELETE CASCADE;
    END IF;
END$$;

-- ADD UNIQUE CONSTRAINT
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'tenant_member_unique'
    ) THEN
        ALTER TABLE tenant_members
            ADD CONSTRAINT tenant_member_unique UNIQUE (tenant_id, user_id);
    END IF;
END$$;

-- ADD INDEXES
CREATE INDEX IF NOT EXISTS idx_tenant_members_uuid ON tenant_members (tenant_member_uuid);
CREATE INDEX IF NOT EXISTS idx_tenant_members_tenant_id ON tenant_members (tenant_id);
CREATE INDEX IF NOT EXISTS idx_tenant_members_user_id ON tenant_members (user_id);
CREATE INDEX IF NOT EXISTS idx_tenant_members_created_at ON tenant_members (created_at);
`
	if err := db.Exec(sql).Error; err != nil {
		log.Fatalf("❌ Failed to run migration 026_create_tenant_members_table: %v", err)
	}

	log.Println("✅ Migration 026_create_tenant_members_table executed")
}
