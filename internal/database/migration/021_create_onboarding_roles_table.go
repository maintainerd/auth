package migration

import (
	"log"

	"gorm.io/gorm"
)

func CreateOnboardingRoleTable(db *gorm.DB) {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS onboarding_roles (
    onboarding_role_id			SERIAL PRIMARY KEY,
		onboarding_role_uuid		UUID NOT NULL UNIQUE,
    onboarding_id						INTEGER NOT NULL,
    role_id									INTEGER NOT NULL,
    created_at							TIMESTAMPTZ DEFAULT now()
);

-- ADD CONSTRAINTS
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_onboarding_roles_onboarding_id'
    ) THEN
        ALTER TABLE onboarding_roles
            ADD CONSTRAINT fk_onboarding_roles_onboarding_id FOREIGN KEY (onboarding_id)
            REFERENCES onboardings(onboarding_id) ON DELETE CASCADE;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_onboarding_roles_role_id'
    ) THEN
        ALTER TABLE onboarding_roles
            ADD CONSTRAINT fk_onboarding_roles_role_id FOREIGN KEY (role_id)
            REFERENCES roles(role_id) ON DELETE CASCADE;
    END IF;
END$$;

-- ADD INDEXES
CREATE INDEX IF NOT EXISTS idx_onboarding_roles_uuid ON onboarding_roles (onboarding_role_uuid);
CREATE INDEX IF NOT EXISTS idx_onboarding_roles_onboarding_id ON onboarding_roles (onboarding_id);
CREATE INDEX IF NOT EXISTS idx_onboarding_roles_role_id ON onboarding_roles (role_id);
`
	if err := db.Exec(sql).Error; err != nil {
		log.Fatalf("❌ Failed to run migration 021_create_onboarding_roles_table: %v", err)
	}

	log.Println("✅ Migration 021_create_onboarding_roles_table executed")
}
