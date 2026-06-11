package migration

import (
	"gorm.io/gorm"
)

func CreateAuthFlowTable(db *gorm.DB) error {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS auth_flows (
    auth_flow_id           BIGSERIAL PRIMARY KEY,
    auth_flow_uuid         UUID NOT NULL UNIQUE,
    tenant_id              BIGINT NOT NULL,
    name                   VARCHAR(100) NOT NULL,
    description            TEXT NOT NULL,
    identifier             VARCHAR(255) NOT NULL,
    status                 VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'inactive')),
    client_id              BIGINT,
    branding_id            BIGINT,
    created_by             BIGINT,
    updated_by             BIGINT,
    created_at             TIMESTAMPTZ DEFAULT now(),
    updated_at             TIMESTAMPTZ DEFAULT now(),
    deleted_at             TIMESTAMPTZ
);

-- ADD CONSTRAINTS
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_auth_flows_tenant_id'
    ) THEN
        ALTER TABLE auth_flows
            ADD CONSTRAINT fk_auth_flows_tenant_id FOREIGN KEY (tenant_id)
            REFERENCES tenants(tenant_id) ON DELETE CASCADE;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_auth_flows_client_id'
    ) THEN
        ALTER TABLE auth_flows
            ADD CONSTRAINT fk_auth_flows_client_id FOREIGN KEY (client_id)
            REFERENCES clients(client_id) ON DELETE SET NULL;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_auth_flows_branding_id'
    ) THEN
        ALTER TABLE auth_flows
            ADD CONSTRAINT fk_auth_flows_branding_id FOREIGN KEY (branding_id)
            REFERENCES branding(branding_id) ON DELETE SET NULL;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_auth_flows_created_by'
    ) THEN
        ALTER TABLE auth_flows
            ADD CONSTRAINT fk_auth_flows_created_by FOREIGN KEY (created_by)
            REFERENCES users(user_id) ON DELETE SET NULL;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_auth_flows_updated_by'
    ) THEN
        ALTER TABLE auth_flows
            ADD CONSTRAINT fk_auth_flows_updated_by FOREIGN KEY (updated_by)
            REFERENCES users(user_id) ON DELETE SET NULL;
    END IF;
END$$;

-- ADD INDEXES
CREATE INDEX IF NOT EXISTS idx_auth_flow_uuid ON auth_flows (auth_flow_uuid);
CREATE INDEX IF NOT EXISTS idx_auth_flow_tenant_id ON auth_flows (tenant_id);
CREATE INDEX IF NOT EXISTS idx_auth_flow_name ON auth_flows (name);
CREATE UNIQUE INDEX IF NOT EXISTS uq_auth_flows_tenant_identifier ON auth_flows (tenant_id, identifier) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_auth_flow_status ON auth_flows (status);
CREATE INDEX IF NOT EXISTS idx_auth_flow_client_id ON auth_flows (client_id);
CREATE INDEX IF NOT EXISTS idx_auth_flow_branding_id ON auth_flows (branding_id);
CREATE INDEX IF NOT EXISTS idx_auth_flow_created_at ON auth_flows (created_at);
CREATE INDEX IF NOT EXISTS idx_auth_flow_deleted_at ON auth_flows (deleted_at) WHERE deleted_at IS NULL;
`
	return db.Exec(sql).Error
}
