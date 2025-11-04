package migration

import (
	"log"

	"gorm.io/gorm"
)

func CreateRoleTable(db *gorm.DB) {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS roles (
    role_id             SERIAL PRIMARY KEY,
    role_uuid           UUID UNIQUE NOT NULL,
    name                VARCHAR(255) UNIQUE NOT NULL,
    description         TEXT NOT NULL,
    is_active           BOOLEAN DEFAULT FALSE,
    is_default          BOOLEAN DEFAULT FALSE,
    tenant_id           INTEGER NOT NULL,
    created_at          TIMESTAMPTZ DEFAULT now(),
    updated_at          TIMESTAMPTZ DEFAULT now()
);

-- ADD CONSTRAINTS (safe)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_roles_tenant_id'
    ) THEN
        ALTER TABLE roles
            ADD CONSTRAINT fk_roles_tenant_id FOREIGN KEY (tenant_id)
            REFERENCES tenants(tenant_id) ON DELETE CASCADE;
    END IF;
END$$;

-- ADD INDEXES
CREATE INDEX IF NOT EXISTS idx_roles_role_uuid ON roles (role_uuid);
CREATE INDEX IF NOT EXISTS idx_roles_name ON roles (name);
CREATE INDEX IF NOT EXISTS idx_roles_description ON roles (description);
CREATE INDEX IF NOT EXISTS idx_roles_is_active ON roles (is_active);
CREATE INDEX IF NOT EXISTS idx_roles_is_default ON roles (is_default);
CREATE INDEX IF NOT EXISTS idx_roles_tenant_id ON roles (tenant_id);
CREATE INDEX IF NOT EXISTS idx_roles_created_at ON roles (created_at);
`
	if err := db.Exec(sql).Error; err != nil {
		log.Fatalf("❌ Failed to run migration 014_create_roles_table: %v", err)
	}

	log.Println("✅ Migration 014_create_roles_table executed")
}
