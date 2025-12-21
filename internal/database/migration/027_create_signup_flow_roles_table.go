package migration

import (
	"log"

	"gorm.io/gorm"
)

func CreateSignupFlowRoleTable(db *gorm.DB) {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS signup_flow_roles (
    signup_flow_role_id			SERIAL PRIMARY KEY,
		signup_flow_role_uuid		UUID NOT NULL UNIQUE,
    signup_flow_id					INTEGER NOT NULL,
    role_id									INTEGER NOT NULL,
    created_at							TIMESTAMPTZ DEFAULT now()
);

-- ADD CONSTRAINTS
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_signup_flow_roles_signup_flow_id'
    ) THEN
        ALTER TABLE signup_flow_roles
            ADD CONSTRAINT fk_signup_flow_roles_signup_flow_id FOREIGN KEY (signup_flow_id)
            REFERENCES signup_flows(signup_flow_id) ON DELETE CASCADE;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_signup_flow_roles_role_id'
    ) THEN
        ALTER TABLE signup_flow_roles
            ADD CONSTRAINT fk_signup_flow_roles_role_id FOREIGN KEY (role_id)
            REFERENCES roles(role_id) ON DELETE CASCADE;
    END IF;
END$$;

-- ADD INDEXES
CREATE INDEX IF NOT EXISTS idx_signup_flow_roles_uuid ON signup_flow_roles (signup_flow_role_uuid);
CREATE INDEX IF NOT EXISTS idx_signup_flow_roles_signup_flow_id ON signup_flow_roles (signup_flow_id);
CREATE INDEX IF NOT EXISTS idx_signup_flow_roles_role_id ON signup_flow_roles (role_id);
`
	if err := db.Exec(sql).Error; err != nil {
		log.Fatalf("❌ Failed to run migration 027_create_signup_flow_roles_table: %v", err)
	}

	log.Println("✅ Migration 027_create_signup_flow_roles_table executed")
}
