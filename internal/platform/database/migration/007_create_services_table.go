package migration

import (
	"gorm.io/gorm"
)

func CreateServiceTable(db *gorm.DB) error {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS services (
    service_id      BIGSERIAL PRIMARY KEY,
    service_uuid    UUID NOT NULL UNIQUE,
    tenant_id       BIGINT NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
    name            VARCHAR(100) NOT NULL,
    display_name    TEXT NOT NULL,
    description     TEXT NOT NULL,
    version         VARCHAR(20) NOT NULL,
    status          VARCHAR(20) DEFAULT 'inactive',
    is_system       BOOLEAN DEFAULT FALSE,
    created_by      BIGINT,
    updated_by      BIGINT,
    created_at      TIMESTAMPTZ DEFAULT now(),
    updated_at      TIMESTAMPTZ DEFAULT now(),
    deleted_at      TIMESTAMPTZ
);

-- ADD INDEXES
CREATE INDEX IF NOT EXISTS idx_services_uuid ON services (service_uuid);
CREATE INDEX IF NOT EXISTS idx_services_tenant_id ON services (tenant_id);
-- Services are tenant-scoped: name is unique per tenant, not globally.
CREATE UNIQUE INDEX IF NOT EXISTS uq_services_tenant_name ON services (tenant_id, name) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_services_display_name ON services (display_name);
CREATE INDEX IF NOT EXISTS idx_services_status ON services (status);
CREATE INDEX IF NOT EXISTS idx_services_is_system ON services (is_system);
CREATE INDEX IF NOT EXISTS idx_services_created_at ON services (created_at);
CREATE INDEX IF NOT EXISTS idx_services_deleted_at ON services (deleted_at) WHERE deleted_at IS NULL;

-- ADD CONSTRAINTS
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'chk_services_status'
    ) THEN
        ALTER TABLE services ADD CONSTRAINT chk_services_status
            CHECK (status IN ('active', 'maintenance', 'deprecated', 'inactive'));
    END IF;
END$$;
`
	return db.Exec(sql).Error
}
