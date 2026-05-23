package migration

import (
	"gorm.io/gorm"
)

func CreateSignupFlowTable(db *gorm.DB) error {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS signup_flows (
    signup_flow_id    BIGSERIAL PRIMARY KEY,
    signup_flow_uuid  UUID NOT NULL UNIQUE,
    tenant_id         BIGINT NOT NULL,
    name              VARCHAR(100) NOT NULL,
    description       TEXT NOT NULL,
    identifier        VARCHAR(255) NOT NULL,
    config            JSONB DEFAULT '{}'::jsonb,
    status            VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'inactive')),
    client_id         BIGINT NOT NULL,
    created_by        BIGINT,
    updated_by        BIGINT,
    created_at        TIMESTAMPTZ DEFAULT now(),
    updated_at        TIMESTAMPTZ DEFAULT now(),
    deleted_at        TIMESTAMPTZ
);

-- ADD CONSTRAINTS
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_signup_flows_tenant_id'
    ) THEN
        ALTER TABLE signup_flows
            ADD CONSTRAINT fk_signup_flows_tenant_id FOREIGN KEY (tenant_id)
            REFERENCES tenants(tenant_id) ON DELETE CASCADE;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_signup_flows_client_id'
    ) THEN
        ALTER TABLE signup_flows
            ADD CONSTRAINT fk_signup_flows_client_id FOREIGN KEY (client_id)
            REFERENCES clients(client_id) ON DELETE CASCADE;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_signup_flows_created_by'
    ) THEN
        ALTER TABLE signup_flows
            ADD CONSTRAINT fk_signup_flows_created_by FOREIGN KEY (created_by)
            REFERENCES users(user_id) ON DELETE SET NULL;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_signup_flows_updated_by'
    ) THEN
        ALTER TABLE signup_flows
            ADD CONSTRAINT fk_signup_flows_updated_by FOREIGN KEY (updated_by)
            REFERENCES users(user_id) ON DELETE SET NULL;
    END IF;
END$$;

-- ADD INDEXES
CREATE INDEX IF NOT EXISTS idx_signup_flow_uuid ON signup_flows (signup_flow_uuid);
CREATE INDEX IF NOT EXISTS idx_signup_flow_tenant_id ON signup_flows (tenant_id);
CREATE INDEX IF NOT EXISTS idx_signup_flow_name ON signup_flows (name);
-- identifier should be unique per tenant, not globally (was UNIQUE before)
CREATE UNIQUE INDEX IF NOT EXISTS uq_signup_flows_tenant_identifier ON signup_flows (tenant_id, identifier) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_signup_flow_status ON signup_flows (status);
CREATE INDEX IF NOT EXISTS idx_signup_flow_client_id ON signup_flows (client_id);
CREATE INDEX IF NOT EXISTS idx_signup_flow_created_at ON signup_flows (created_at);
CREATE INDEX IF NOT EXISTS idx_signup_flow_deleted_at ON signup_flows (deleted_at) WHERE deleted_at IS NULL;
`
	return db.Exec(sql).Error
}
