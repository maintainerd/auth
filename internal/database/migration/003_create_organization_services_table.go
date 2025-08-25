package migration

import (
	"log"

	"gorm.io/gorm"
)

func CreateOrganizationServicesTable(db *gorm.DB) {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS organization_services (
    organization_service_id			SERIAL PRIMARY KEY,
		organization_service_uuid		UUID NOT NULL UNIQUE,
    organization_id         		INT NOT NULL REFERENCES organizations(organization_id) ON DELETE CASCADE,
    service_id              		INT NOT NULL REFERENCES services(service_id) ON DELETE CASCADE,
    created_at              		TIMESTAMPTZ DEFAULT now(),
    updated_at              		TIMESTAMPTZ DEFAULT now(),
    UNIQUE (organization_id, service_id)
);

-- ADD INDEXES
CREATE INDEX IF NOT EXISTS idx_org_services_organization_uuid ON organization_services (organization_service_uuid);
CREATE INDEX IF NOT EXISTS idx_org_services_organization_id ON organization_services (organization_id);
CREATE INDEX IF NOT EXISTS idx_org_services_service_id ON organization_services (service_id);
`
	if err := db.Exec(sql).Error; err != nil {
		log.Fatalf("❌ Failed to run migration 003_create_organization_services_table: %v", err)
	}

	log.Println("✅ Migration 003_create_organization_services_table executed")
}
