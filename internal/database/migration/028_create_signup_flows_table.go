package migration

import (
	"log"

	"gorm.io/gorm"
)

func CreateSignupFlowTable(db *gorm.DB) {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS signup_flows (
    signup_flow_id   	SERIAL PRIMARY KEY,
    signup_flow_uuid  UUID NOT NULL UNIQUE,
    tenant_id					BIGINT NOT NULL,
    name							VARCHAR(100) NOT NULL,
    description				TEXT NOT NULL,
		identifier				VARCHAR(255) NOT NULL UNIQUE,
		config						JSONB DEFAULT '{}'::jsonb,
    status						VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'inactive')),
		auth_client_id		INTEGER NOT NULL,
    created_at				TIMESTAMPTZ DEFAULT now(),
    updated_at				TIMESTAMPTZ DEFAULT now()
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
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_signup_flows_auth_client_id'
    ) THEN
        ALTER TABLE signup_flows
            ADD CONSTRAINT fk_signup_flows_auth_client_id FOREIGN KEY (auth_client_id)
            REFERENCES auth_clients(auth_client_id) ON DELETE CASCADE;
    END IF;
END$$;

-- ADD INDEXES
CREATE INDEX IF NOT EXISTS idx_signup_flow_uuid ON signup_flows (signup_flow_uuid);
CREATE INDEX IF NOT EXISTS idx_signup_flow_tenant_id ON signup_flows (tenant_id);
CREATE INDEX IF NOT EXISTS idx_signup_flow_name ON signup_flows (name);
CREATE INDEX IF NOT EXISTS idx_signup_flow_identifier ON signup_flows (identifier);
CREATE INDEX IF NOT EXISTS idx_signup_flow_status ON signup_flows (status);
CREATE INDEX IF NOT EXISTS idx_signup_flow_auth_client_id ON signup_flows (auth_client_id);
CREATE INDEX IF NOT EXISTS idx_signup_flow_created_at ON signup_flows (created_at);
`
	if err := db.Exec(sql).Error; err != nil {
		log.Fatalf("❌ Failed to run migration 028_create_signup_flows_table: %v", err)
	}

	log.Println("✅ Migration 028_create_signup_flows_table executed")
}
