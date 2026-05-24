package migration

import (
	"gorm.io/gorm"
)

func CreateRoleTable(db *gorm.DB) error {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS roles (
    role_id     BIGSERIAL PRIMARY KEY,
    role_uuid   UUID UNIQUE NOT NULL,
    tenant_id   BIGINT NOT NULL,
    name        VARCHAR(255) NOT NULL,
    description TEXT NOT NULL,
    status      VARCHAR(16) NOT NULL DEFAULT 'inactive',
    is_default  BOOLEAN DEFAULT FALSE,
    is_system   BOOLEAN DEFAULT FALSE,
    created_by  BIGINT,
    updated_by  BIGINT,
    created_at  TIMESTAMPTZ DEFAULT now(),
    updated_at  TIMESTAMPTZ DEFAULT now(),
    deleted_at  TIMESTAMPTZ
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
CREATE UNIQUE INDEX IF NOT EXISTS uq_roles_tenant_name ON roles (tenant_id, name) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_roles_name ON roles (name);
CREATE INDEX IF NOT EXISTS idx_roles_description ON roles (description);
CREATE INDEX IF NOT EXISTS idx_roles_status ON roles (status);
CREATE INDEX IF NOT EXISTS idx_roles_is_default ON roles (is_default);
CREATE INDEX IF NOT EXISTS idx_roles_is_system ON roles (is_system);
CREATE INDEX IF NOT EXISTS idx_roles_tenant_id ON roles (tenant_id);
CREATE INDEX IF NOT EXISTS idx_roles_created_at ON roles (created_at);
CREATE INDEX IF NOT EXISTS idx_roles_deleted_at ON roles (deleted_at) WHERE deleted_at IS NULL;
`
	return db.Exec(sql).Error
}
