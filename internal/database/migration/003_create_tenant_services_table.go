package migration

import (
	"log"

	"gorm.io/gorm"
)

func CreateTenantServicesTable(db *gorm.DB) {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS tenant_services (
    tenant_service_id   	SERIAL PRIMARY KEY,
    tenant_id           	INTEGER NOT NULL,
    service_id          	INTEGER NOT NULL,
    created_at          	TIMESTAMPTZ DEFAULT now(),
    updated_at          	TIMESTAMPTZ DEFAULT now()
);

-- ADD CONSTRAINTS
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_tenant_services_tenant_id'
    ) THEN
        ALTER TABLE tenant_services
            ADD CONSTRAINT fk_tenant_services_tenant_id FOREIGN KEY (tenant_id)
            REFERENCES tenants(tenant_id) ON DELETE CASCADE;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_tenant_services_service_id'
    ) THEN
        ALTER TABLE tenant_services
            ADD CONSTRAINT fk_tenant_services_service_id FOREIGN KEY (service_id)
            REFERENCES services(service_id) ON DELETE CASCADE;
    END IF;
END$$;

-- ADD INDEXES
CREATE INDEX IF NOT EXISTS idx_tenant_services_tenant_id ON tenant_services (tenant_id);
CREATE INDEX IF NOT EXISTS idx_tenant_services_service_id ON tenant_services (service_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_tenant_services_unique ON tenant_services (tenant_id, service_id);
`

	if err := db.Exec(sql).Error; err != nil {
		log.Fatalf("❌ Failed to run migration 003_create_tenant_services_table: %v", err)
	}

	log.Println("✅ Migration 003_create_tenant_services_table executed")
}
