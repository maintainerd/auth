package migration

import (
	"log"

	"gorm.io/gorm"
)

func CreateServicePoliciesTable(db *gorm.DB) {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS service_policies (
    service_policy_id   	SERIAL PRIMARY KEY,
    service_policy_uuid 	UUID NOT NULL UNIQUE,
    service_id          	INTEGER NOT NULL,
    policy_id           	INTEGER NOT NULL,
    created_at          	TIMESTAMPTZ DEFAULT now()
);

-- ADD CONSTRAINTS
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_service_policies_service_id'
    ) THEN
        ALTER TABLE service_policies
            ADD CONSTRAINT fk_service_policies_service_id FOREIGN KEY (service_id)
            REFERENCES services(service_id) ON DELETE CASCADE;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_service_policies_policy_id'
    ) THEN
        ALTER TABLE service_policies
            ADD CONSTRAINT fk_service_policies_policy_id FOREIGN KEY (policy_id)
            REFERENCES policies(policy_id) ON DELETE CASCADE;
    END IF;
END$$;

-- ADD INDEXES
CREATE INDEX IF NOT EXISTS idx_service_policies_uuid ON service_policies (service_policy_uuid);
CREATE INDEX IF NOT EXISTS idx_service_policies_service_id ON service_policies (service_id);
CREATE INDEX IF NOT EXISTS idx_service_policies_policy_id ON service_policies (policy_id);
`

	if err := db.Exec(sql).Error; err != nil {
		log.Fatalf("❌ Failed to run migration 005_create_service_policies_table: %v", err)
	}

	log.Println("✅ Migration 005_create_service_policies_table executed")
}
