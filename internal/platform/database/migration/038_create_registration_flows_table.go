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
    name                   VARCHAR(100) NOT NULL,
    description            TEXT,
    identifier             VARCHAR(255) NOT NULL,
    required_fields        JSONB NOT NULL DEFAULT '[]',
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
CREATE INDEX IF NOT EXISTS idx_registration_flows_uuid ON registration_flows (registration_flow_uuid);
CREATE INDEX IF NOT EXISTS idx_registration_flows_tenant_id ON registration_flows (tenant_id);
CREATE INDEX IF NOT EXISTS idx_registration_flows_name ON registration_flows (name);
CREATE UNIQUE INDEX IF NOT EXISTS uq_registration_flows_tenant_identifier ON registration_flows (tenant_id, identifier) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_registration_flows_status ON registration_flows (status);
CREATE INDEX IF NOT EXISTS idx_registration_flows_client_id ON registration_flows (client_id);
CREATE INDEX IF NOT EXISTS idx_registration_flows_created_at ON registration_flows (created_at);
CREATE INDEX IF NOT EXISTS idx_registration_flows_deleted_at ON registration_flows (deleted_at) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_registration_flows_is_system ON registration_flows (is_system) WHERE is_system = TRUE;
`
	return db.Exec(sql).Error
}
