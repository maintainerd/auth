package migration

import (
	"gorm.io/gorm"
)

// CreateTenantEventTypesTable creates the per-tenant master switch table.
func CreateTenantEventTypesTable(db *gorm.DB) error {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS tenant_event_types (
    tenant_event_type_id        BIGSERIAL PRIMARY KEY,
    tenant_event_type_uuid      UUID NOT NULL UNIQUE,
    tenant_id                   BIGINT NOT NULL,
    event_type_id               BIGINT NOT NULL,
    enabled                     BOOLEAN NOT NULL DEFAULT true,
    created_at                  TIMESTAMPTZ DEFAULT now(),
    updated_at                  TIMESTAMPTZ DEFAULT now(),
    UNIQUE (tenant_id, event_type_id)
);

-- ADD CONSTRAINTS
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_tenant_event_types_tenant_id'
    ) THEN
        ALTER TABLE tenant_event_types
            ADD CONSTRAINT fk_tenant_event_types_tenant_id FOREIGN KEY (tenant_id)
            REFERENCES tenants(tenant_id) ON DELETE CASCADE;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_tenant_event_types_event_type_id'
    ) THEN
        ALTER TABLE tenant_event_types
            ADD CONSTRAINT fk_tenant_event_types_event_type_id FOREIGN KEY (event_type_id)
            REFERENCES event_types(event_type_id) ON DELETE CASCADE;
    END IF;
END$$;

-- CREATE INDEXES
CREATE INDEX IF NOT EXISTS idx_tenant_event_types_uuid ON tenant_event_types (tenant_event_type_uuid);
CREATE INDEX IF NOT EXISTS idx_tenant_event_types_tenant_id ON tenant_event_types (tenant_id);
CREATE INDEX IF NOT EXISTS idx_tenant_event_types_tenant_enabled ON tenant_event_types (tenant_id, enabled);
`

	return db.Exec(sql).Error
}
