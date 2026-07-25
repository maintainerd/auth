package migration

import (
	"gorm.io/gorm"
)

func CreateAPITable(db *gorm.DB) error {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS apis (
    api_id          BIGSERIAL PRIMARY KEY,
    api_uuid        UUID NOT NULL UNIQUE,
    tenant_id       BIGINT NOT NULL,
    service_id      BIGINT NOT NULL,
    name            VARCHAR(100) NOT NULL,
    display_name    VARCHAR(255) NOT NULL,
    -- NOT NULL DEFAULT '' so "no description" has one representation: the Go model
    -- is a non-pointer string and can only write '', but a seeder or direct SQL could
    -- write NULL, and an ILIKE filter never matches NULL.
    description     TEXT NOT NULL DEFAULT '',
    identifier      VARCHAR(512) NOT NULL,
    status          VARCHAR(20) NOT NULL DEFAULT 'inactive' CHECK (status IN ('active', 'inactive')),
    is_system       BOOLEAN NOT NULL DEFAULT FALSE,
    created_by      BIGINT,
    updated_by      BIGINT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at      TIMESTAMPTZ,

    -- NOT NULL does not exclude ''. Without these the unique indexes admit one
    -- whitespace-only API per tenant, and seeders bypass the DTO length rules.
    CONSTRAINT chk_apis_name CHECK (btrim(name) <> ''),
    CONSTRAINT chk_apis_display_name CHECK (btrim(display_name) <> ''),
    CONSTRAINT chk_apis_identifier CHECK (btrim(identifier) <> '')
);

-- ADD CONSTRAINTS
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_apis_service_id'
    ) THEN
        ALTER TABLE apis
            ADD CONSTRAINT fk_apis_service_id FOREIGN KEY (service_id)
            REFERENCES services(service_id) ON DELETE CASCADE;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_apis_tenant_id'
    ) THEN
        ALTER TABLE apis
            ADD CONSTRAINT fk_apis_tenant_id FOREIGN KEY (tenant_id)
            REFERENCES tenants(tenant_id) ON DELETE CASCADE;
    END IF;
END$$;

-- ADD INDEXES
--
-- No index on api_uuid: the column is already UNIQUE, which creates its own index.
-- No index on deleted_at: a partial index WHERE deleted_at IS NULL stores only NULLs
-- and matches ~every live row, so the planner always prefers a scan.
-- No index on display_name: the only predicate is LOWER(display_name) LIKE '%…%',
-- which a btree cannot serve.

-- Identifier is the API's externally-addressable name and must be unique per tenant.
CREATE UNIQUE INDEX IF NOT EXISTS uq_apis_tenant_identifier ON apis (tenant_id, identifier) WHERE deleted_at IS NULL;

-- FindByNameAndTenantID looks up by (name, tenant_id) and the service layer treats a
-- hit as a conflict. Without this index that check is a race, and two APIs in one
-- tenant could share a name.
CREATE UNIQUE INDEX IF NOT EXISTS uq_apis_tenant_name ON apis (tenant_id, name) WHERE deleted_at IS NULL;

-- Every listing is "tenant_id = ? ORDER BY created_at DESC".
CREATE INDEX IF NOT EXISTS idx_apis_tenant_created_at ON apis (tenant_id, created_at DESC) WHERE deleted_at IS NULL;

-- The service detail page lists a service's APIs, and the delete cascade plucks by
-- service.
CREATE INDEX IF NOT EXISTS idx_apis_tenant_service ON apis (tenant_id, service_id) WHERE deleted_at IS NULL;

-- tenant_id alone stays for the ON DELETE CASCADE, which touches soft-deleted rows
-- that no partial index can serve.
CREATE INDEX IF NOT EXISTS idx_apis_tenant_id ON apis (tenant_id);
`
	return db.Exec(sql).Error
}
