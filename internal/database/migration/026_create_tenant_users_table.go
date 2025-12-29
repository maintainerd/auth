package migration

import (
	"log"

	"gorm.io/gorm"
)

func CreateTenantUsersTable(db *gorm.DB) {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS tenant_users (
    tenant_user_id   SERIAL PRIMARY KEY,
    tenant_user_uuid UUID NOT NULL UNIQUE,
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
        SELECT 1 FROM pg_constraint WHERE conname = 'chk_tenant_users_role'
    ) THEN
        ALTER TABLE tenant_users
            ADD CONSTRAINT chk_tenant_users_role
            CHECK (role IN ('owner', 'member'));
    END IF;
END$$;

-- ADD CONSTRAINTS (safe)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_tenant_users_tenant_id'
    ) THEN
        ALTER TABLE tenant_users
            ADD CONSTRAINT fk_tenant_users_tenant_id FOREIGN KEY (tenant_id)
            REFERENCES tenants(tenant_id) ON DELETE CASCADE;
    END IF;
END$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_tenant_users_user_id'
    ) THEN
        ALTER TABLE tenant_users
            ADD CONSTRAINT fk_tenant_users_user_id FOREIGN KEY (user_id)
            REFERENCES users(user_id) ON DELETE CASCADE;
    END IF;
END$$;

-- ADD UNIQUE CONSTRAINT
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'tenant_user_unique'
    ) THEN
        ALTER TABLE tenant_users
            ADD CONSTRAINT tenant_user_unique UNIQUE (tenant_id, user_id);
    END IF;
END$$;

-- ADD INDEXES
CREATE INDEX IF NOT EXISTS idx_tenant_users_uuid ON tenant_users (tenant_user_uuid);
CREATE INDEX IF NOT EXISTS idx_tenant_users_tenant_id ON tenant_users (tenant_id);
CREATE INDEX IF NOT EXISTS idx_tenant_users_user_id ON tenant_users (user_id);
CREATE INDEX IF NOT EXISTS idx_tenant_users_created_at ON tenant_users (created_at);
`
	if err := db.Exec(sql).Error; err != nil {
		log.Fatalf("❌ Failed to run migration 026_create_tenant_users_table: %v", err)
	}

	log.Println("✅ Migration 026_create_tenant_users_table executed")
}
