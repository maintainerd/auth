package migration

import (
	"log"

	"gorm.io/gorm"
)

func CreateOnboardingRouteRoleTable(db *gorm.DB) {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS onboarding_route_roles (
    onboarding_route_role_id SERIAL PRIMARY KEY,
    onboarding_route_id      INTEGER NOT NULL,
    role_id                    INTEGER NOT NULL,
    created_at                 TIMESTAMPTZ DEFAULT now()
);

-- ADD CONSTRAINTS (safe)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_onboarding_route_roles_route_id'
    ) THEN
        ALTER TABLE onboarding_route_roles
            ADD CONSTRAINT fk_onboarding_route_roles_route_id FOREIGN KEY (onboarding_route_id)
            REFERENCES onboarding_routes(onboarding_route_id) ON DELETE CASCADE;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_onboarding_route_roles_role_id'
    ) THEN
        ALTER TABLE onboarding_route_roles
            ADD CONSTRAINT fk_onboarding_route_roles_role_id FOREIGN KEY (role_id)
            REFERENCES roles(role_id) ON DELETE CASCADE;
    END IF;
END$$;

-- ADD INDEXES
CREATE INDEX IF NOT EXISTS idx_onboarding_route_roles_route_id ON onboarding_route_roles (onboarding_route_id);
CREATE INDEX IF NOT EXISTS idx_onboarding_route_roles_role_id ON onboarding_route_roles (role_id);
`
	if err := db.Exec(sql).Error; err != nil {
		log.Fatalf("❌ Failed to run migration 017_create_onboarding_route_roles_table: %v", err)
	}

	log.Println("✅ Migration 017_create_onboarding_route_roles_table executed")
}
