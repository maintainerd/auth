package migration

import (
	"gorm.io/gorm"
)

func CreatePoliciesTable(db *gorm.DB) error {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS policies (
    policy_id       BIGSERIAL PRIMARY KEY,
    policy_uuid     UUID NOT NULL UNIQUE,
    tenant_id       BIGINT NOT NULL,
    name            VARCHAR(150) NOT NULL,
    -- NOT NULL DEFAULT '': the Go model is a non-pointer string and can only write
    -- '', but a seeder or direct SQL could write NULL, and an ILIKE filter never
    -- matches NULL — giving "no description" two on-disk representations.
    description     TEXT NOT NULL DEFAULT '',
    document        JSONB NOT NULL,
    version         VARCHAR(20) NOT NULL,
    status          VARCHAR(20) NOT NULL DEFAULT 'inactive' CHECK (status IN ('active', 'inactive')),
    is_system       BOOLEAN NOT NULL DEFAULT FALSE,
    created_by      BIGINT,
    updated_by      BIGINT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at      TIMESTAMPTZ,

    -- A policy document IS the authorization rule. These invariants must hold even
    -- when a seeder, a migration or direct SQL bypasses the DTO layer.
    CONSTRAINT chk_policies_name CHECK (btrim(name) <> ''),
    CONSTRAINT chk_policies_version CHECK (btrim(version) <> ''),
    -- The evaluator expects an object with a statement array. A scalar or array
    -- document would parse into an empty PolicyDocument and grant nothing, silently.
    CONSTRAINT chk_policies_document CHECK (jsonb_typeof(document) = 'object'),
    CONSTRAINT chk_policies_document_stmt CHECK (
        jsonb_typeof(document -> 'statement') = 'array'
    )
);

-- ADD CONSTRAINTS (safe)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_policies_tenant_id'
    ) THEN
        ALTER TABLE policies
            ADD CONSTRAINT fk_policies_tenant_id FOREIGN KEY (tenant_id)
            REFERENCES tenants(tenant_id) ON DELETE CASCADE;
    END IF;
END$$;

-- ADD INDEXES
--
-- No index on policy_uuid: the column is already UNIQUE. No index on the
-- low-cardinality version/status flags.

CREATE UNIQUE INDEX IF NOT EXISTS uq_policies_tenant_name ON policies (tenant_id, name) WHERE deleted_at IS NULL;

-- Every listing is "tenant_id = ? ORDER BY created_at DESC".
CREATE INDEX IF NOT EXISTS idx_policies_tenant_created_at ON policies (tenant_id, created_at DESC) WHERE deleted_at IS NULL;

-- Supports "which policies grant action X" over the document body.
CREATE INDEX IF NOT EXISTS idx_policies_document ON policies USING GIN (document);

-- tenant_id alone stays for the ON DELETE CASCADE, which touches soft-deleted rows.
CREATE INDEX IF NOT EXISTS idx_policies_tenant_id ON policies (tenant_id);
`

	return db.Exec(sql).Error
}
