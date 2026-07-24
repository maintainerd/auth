package migration

import (
	"gorm.io/gorm"
)

func CreateRegistrationFlowTable(db *gorm.DB) error {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS registration_flows (
    registration_flow_id   BIGSERIAL PRIMARY KEY,
    registration_flow_uuid UUID NOT NULL UNIQUE,
    tenant_id              BIGINT NOT NULL,
    client_id              BIGINT NOT NULL,
    -- name doubles as the public registration-link selector
    -- (?registration_flow=<name>), so it is slug-shaped and tenant-unique.
    name                   VARCHAR(100) NOT NULL,
    -- NOT NULL because the GORM model holds a non-pointer string; a NULL here
    -- would fail to scan.
    description            TEXT NOT NULL DEFAULT '',
    -- Always a JSON array of field names; the service unmarshals it into
    -- []string, so a non-array would be a runtime error rather than a 400.
    required_fields        JSONB NOT NULL DEFAULT '[]' CHECK (jsonb_typeof(required_fields) = 'array'),
    verification_required  BOOLEAN NOT NULL DEFAULT FALSE,
    is_system              BOOLEAN NOT NULL DEFAULT FALSE,
    status                 VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive')),
    created_by             BIGINT,
    updated_by             BIGINT,
    created_at             TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at             TIMESTAMPTZ
);

-- ADD CONSTRAINTS
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_registration_flows_tenant_id'
    ) THEN
        ALTER TABLE registration_flows
            ADD CONSTRAINT fk_registration_flows_tenant_id FOREIGN KEY (tenant_id)
            REFERENCES tenants(tenant_id) ON DELETE CASCADE;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_registration_flows_client_id'
    ) THEN
        ALTER TABLE registration_flows
            ADD CONSTRAINT fk_registration_flows_client_id FOREIGN KEY (client_id)
            REFERENCES clients(client_id) ON DELETE RESTRICT;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_registration_flows_created_by'
    ) THEN
        ALTER TABLE registration_flows
            ADD CONSTRAINT fk_registration_flows_created_by FOREIGN KEY (created_by)
            REFERENCES users(user_id) ON DELETE SET NULL;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_registration_flows_updated_by'
    ) THEN
        ALTER TABLE registration_flows
            ADD CONSTRAINT fk_registration_flows_updated_by FOREIGN KEY (updated_by)
            REFERENCES users(user_id) ON DELETE SET NULL;
    END IF;
END$$;

-- ADD INDEXES
-- registration_flow_uuid is declared UNIQUE above, which already creates an
-- index; a second one on the same column would only cost writes.
CREATE INDEX IF NOT EXISTS idx_registration_flows_tenant_id ON registration_flows (tenant_id);
-- Every name lookup is tenant-scoped, so uq_registration_flows_tenant_name below
-- serves them. A name-only index would be dead weight: the free-text search uses
-- LOWER(name) LIKE, which a plain btree cannot answer either.
CREATE UNIQUE INDEX IF NOT EXISTS uq_registration_flows_tenant_name ON registration_flows (tenant_id, name) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_registration_flows_status ON registration_flows (status);
CREATE INDEX IF NOT EXISTS idx_registration_flows_client_id ON registration_flows (client_id);
CREATE INDEX IF NOT EXISTS idx_registration_flows_created_at ON registration_flows (created_at);
CREATE INDEX IF NOT EXISTS idx_registration_flows_deleted_at ON registration_flows (deleted_at) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_registration_flows_is_system ON registration_flows (is_system) WHERE is_system = TRUE;
`
	return db.Exec(sql).Error
}
