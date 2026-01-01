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
    tenant_id           INTEGER NOT NULL,
    name                VARCHAR(255) NOT NULL,
    description         TEXT NOT NULL,
    status              VARCHAR(16) NOT NULL DEFAULT 'inactive',
    is_default          BOOLEAN DEFAULT FALSE,
    is_system           BOOLEAN DEFAULT FALSE,
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

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'uq_roles_tenant_id_name'
    ) THEN
        ALTER TABLE roles
            ADD CONSTRAINT uq_roles_tenant_id_name UNIQUE (tenant_id, name);
    END IF;
END$$;

-- ADD INDEXES
CREATE INDEX IF NOT EXISTS idx_roles_role_uuid ON roles (role_uuid);
CREATE INDEX IF NOT EXISTS idx_roles_name ON roles (name);
CREATE INDEX IF NOT EXISTS idx_roles_description ON roles (description);
CREATE INDEX IF NOT EXISTS idx_roles_status ON roles (status);
CREATE INDEX IF NOT EXISTS idx_roles_is_default ON roles (is_default);
CREATE INDEX IF NOT EXISTS idx_roles_is_system ON roles (is_system);
CREATE INDEX IF NOT EXISTS idx_roles_tenant_id ON roles (tenant_id);
CREATE INDEX IF NOT EXISTS idx_roles_created_at ON roles (created_at);
`
	if err := db.Exec(sql).Error; err != nil {
		log.Fatalf("❌ Failed to run migration 018_create_roles_table: %v", err)
	}

	log.Println("✅ Migration 018_create_roles_table executed")
}
