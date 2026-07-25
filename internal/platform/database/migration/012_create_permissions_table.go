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
    -- NOT NULL DEFAULT '': see the note on apis.description.
    description         TEXT NOT NULL DEFAULT '',
    status              VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive')),
    is_system           BOOLEAN NOT NULL DEFAULT FALSE,
    created_by          BIGINT,
    updated_by          BIGINT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at          TIMESTAMPTZ,

    -- A permission name IS the authorization token that policies and roles match on,
    -- so a blank one is never meaningful.
    CONSTRAINT chk_permissions_name CHECK (btrim(name) <> '')
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
--
-- No index on permission_uuid (already UNIQUE), deleted_at (stores only NULLs), or
-- the low-cardinality status/is_system flags.

-- Permission names are unique PER TENANT, not per API.
--
-- The ownership chain is api → permission, so per-API uniqueness looks natural — but
-- authorization resolves permissions by NAME: the token carries names, and roles,
-- clients and policies all match on the name string. Two same-named permissions
-- under different APIs would therefore be indistinguishable at decision time, and
-- granting one would effectively grant the other. The seeded convention
-- (client:secret:read, workload-identity-federation:read) is already
-- resource-namespaced and globally unique by construction.
--
-- This also makes the DB agree with permissionService.Create, which already rejected
-- duplicates tenant-wide — previously the schema permitted something the app forbade.
CREATE UNIQUE INDEX IF NOT EXISTS uq_permissions_tenant_name ON permissions (tenant_id, name) WHERE deleted_at IS NULL;

-- Every listing is "tenant_id = ? ORDER BY created_at DESC".
CREATE INDEX IF NOT EXISTS idx_permissions_tenant_created_at ON permissions (tenant_id, created_at DESC) WHERE deleted_at IS NULL;

-- The API detail page lists an API's permissions, and the delete cascade updates by
-- api_id. Left non-partial so the cascade (which touches soft-deleted rows) is served.
CREATE INDEX IF NOT EXISTS idx_permissions_api_id ON permissions (api_id);

-- Permission names are resolved by name during token issuance.
CREATE INDEX IF NOT EXISTS idx_permissions_tenant_name ON permissions (tenant_id, name) WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_permissions_tenant_id ON permissions (tenant_id);
`
	return db.Exec(sql).Error
}
