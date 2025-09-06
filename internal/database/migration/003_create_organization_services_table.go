package migration

import (
	"log"

	"gorm.io/gorm"
)

func CreateOrganizationServicesTable(db *gorm.DB) {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS organization_services (
    organization_service_id   	SERIAL PRIMARY KEY,
    organization_service_uuid		UUID NOT NULL UNIQUE,
    organization_id           	INTEGER NOT NULL,
    service_id                	INTEGER NOT NULL,
    created_at                	TIMESTAMPTZ DEFAULT now()
);

-- ADD CONSTRAINTS
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_organization_services_organization_id'
    ) THEN
        ALTER TABLE organization_services
            ADD CONSTRAINT fk_organization_services_organization_id FOREIGN KEY (organization_id)
            REFERENCES organizations(organization_id) ON DELETE CASCADE;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_organization_services_service_id'
    ) THEN
        ALTER TABLE organization_services
            ADD CONSTRAINT fk_organization_services_service_id FOREIGN KEY (service_id)
            REFERENCES services(service_id) ON DELETE CASCADE;
    END IF;
END$$;

-- ADD INDEXES
CREATE INDEX IF NOT EXISTS idx_organization_services_uuid ON organization_services (organization_service_uuid);
CREATE INDEX IF NOT EXISTS idx_organization_services_organization_id ON organization_services (organization_id);
CREATE INDEX IF NOT EXISTS idx_organization_services_service_id ON organization_services (service_id);
`

	if err := db.Exec(sql).Error; err != nil {
		log.Fatalf("❌ Failed to run migration 003_create_organization_services_table: %v", err)
	}

	log.Println("✅ Migration 003_create_organization_services_table executed")
}
