package migration

import (
	"gorm.io/gorm"
)

func CreatePermissionTable(db *gorm.DB) error {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS permissions (
    permission_id       BIGSERIAL PRIMARY KEY,
    permission_uuid     UUID NOT NULL UNIQUE,
    tenant_id           BIGINT NOT NULL,
    api_id              BIGINT NOT NULL,
    name                VARCHAR(255) NOT NULL,
    description         TEXT NOT NULL,
    status              VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'inactive')),
    is_default          BOOLEAN DEFAULT FALSE,
    is_system           BOOLEAN DEFAULT FALSE,
    created_by          BIGINT,
    updated_by          BIGINT,
    created_at          TIMESTAMPTZ DEFAULT now(),
    updated_at          TIMESTAMPTZ DEFAULT now(),
    deleted_at          TIMESTAMPTZ
);

-- ADD CONSTRAINTS
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_permissions_tenant_id'
    ) THEN
        ALTER TABLE permissions
            ADD CONSTRAINT fk_permissions_tenant_id FOREIGN KEY (tenant_id)
            REFERENCES tenants(tenant_id) ON DELETE CASCADE;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_permissions_api_id'
    ) THEN
        ALTER TABLE permissions
            ADD CONSTRAINT fk_permissions_api_id FOREIGN KEY (api_id)
            REFERENCES apis(api_id) ON DELETE CASCADE;
    END IF;
END$$;

-- ADD INDEXES
CREATE INDEX IF NOT EXISTS idx_permissions_uuid ON permissions (permission_uuid);
CREATE INDEX IF NOT EXISTS idx_permissions_tenant_id ON permissions (tenant_id);
CREATE INDEX IF NOT EXISTS idx_permissions_api_id ON permissions (api_id);
-- Permission names are unique per (tenant, api) — not globally — and exclude soft-deleted rows.
CREATE UNIQUE INDEX IF NOT EXISTS uq_permissions_tenant_api_name ON permissions (tenant_id, api_id, name) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_permissions_name ON permissions (name);
CREATE INDEX IF NOT EXISTS idx_permissions_status ON permissions (status);
CREATE INDEX IF NOT EXISTS idx_permissions_is_default ON permissions (is_default);
CREATE INDEX IF NOT EXISTS idx_permissions_is_system ON permissions (is_system);
CREATE INDEX IF NOT EXISTS idx_permissions_created_at ON permissions (created_at);
CREATE INDEX IF NOT EXISTS idx_permissions_deleted_at ON permissions (deleted_at) WHERE deleted_at IS NULL;
`
	return db.Exec(sql).Error
}
