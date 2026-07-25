package migration

import (
	"gorm.io/gorm"
)

func CreateServiceTable(db *gorm.DB) error {
	sql := `
-- CREATE TABLE
--
-- A service is a first-party or tenant-registered application that participates in
-- this authorization server's policy model. Services are always tenant-scoped via
-- tenant_id NOT NULL — there is no cross-tenant sharing, which is why migration 007
-- (tenant_services) intentionally creates nothing.
CREATE TABLE IF NOT EXISTS services (
    service_id      BIGSERIAL PRIMARY KEY,
    service_uuid    UUID NOT NULL UNIQUE,
    tenant_id       BIGINT NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
    name            VARCHAR(100) NOT NULL,
    display_name    VARCHAR(255) NOT NULL,
    -- NOT NULL DEFAULT '' so "no description" has exactly one representation. The Go
    -- model is a non-pointer string and can only ever write '', but a seeder or
    -- direct SQL could write NULL — and an ILIKE filter never matches NULL, so the
    -- two would behave differently in search.
    description     TEXT NOT NULL DEFAULT '',
    version         VARCHAR(20) NOT NULL,
    status          VARCHAR(20) NOT NULL DEFAULT 'inactive',
    is_system       BOOLEAN NOT NULL DEFAULT FALSE,
    created_by      BIGINT,
    updated_by      BIGINT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at      TIMESTAMPTZ,

    CONSTRAINT chk_services_status CHECK (
        status IN ('active', 'maintenance', 'deprecated', 'inactive')
    ),
    -- NOT NULL does not exclude ''. Without these the (tenant_id, name) unique index
    -- admits one whitespace-only service per tenant, and any seeder or direct SQL
    -- write bypasses the DTO length rules entirely.
    CONSTRAINT chk_services_name CHECK (btrim(name) <> ''),
    CONSTRAINT chk_services_display_name CHECK (btrim(display_name) <> ''),
    CONSTRAINT chk_services_version CHECK (btrim(version) <> '')
);

-- ADD INDEXES
--
-- No index on service_uuid: the column is already UNIQUE, which creates
-- services_service_uuid_key. A second one would only cost writes.
--
-- Not partial on deleted_at: tenant_id carries ON DELETE CASCADE, and the cascade
-- touches soft-deleted rows too, which a WHERE deleted_at IS NULL index cannot serve.
CREATE INDEX IF NOT EXISTS idx_services_tenant_id ON services (tenant_id);

-- Services are tenant-scoped: name is unique per tenant, not globally.
CREATE UNIQUE INDEX IF NOT EXISTS uq_services_tenant_name ON services (tenant_id, name) WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_services_tenant_status ON services (tenant_id, status) WHERE deleted_at IS NULL;

-- Every listing is "tenant_id = ? ORDER BY created_at DESC", so the index is
-- composite rather than created_at alone.
CREATE INDEX IF NOT EXISTS idx_services_tenant_created_at ON services (tenant_id, created_at DESC) WHERE deleted_at IS NULL;

-- Deliberately absent:
--   display_name — the only predicate is LOWER(display_name) LIKE '%…%', which a
--     btree cannot serve. Reinstate as a pg_trgm GIN index alongside the first query
--     that needs real substring search.
--   deleted_at — a partial index WHERE deleted_at IS NULL stores only NULLs and
--     matches ~every live row, so the planner always prefers a scan.
`
	return db.Exec(sql).Error
}
