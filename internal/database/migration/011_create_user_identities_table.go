package migration

import (
	"log"

	"gorm.io/gorm"
)

func CreateUserIdentitiesTable(db *gorm.DB) {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS user_identities (
    user_identity_id    SERIAL PRIMARY KEY,
    user_identity_uuid  UUID NOT NULL UNIQUE,
    user_id             INTEGER NOT NULL,
    provider_name       VARCHAR(100) NOT NULL, -- 'google', 'cognito', 'microsoft'
    provider_user_id    VARCHAR(255) NOT NULL, -- external user
    email               VARCHAR(255),
    raw_profile         JSONB,
    created_at          TIMESTAMPTZ DEFAULT now()
);

-- ADD CONSTRAINTS (safe)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_user_identities_user'
    ) THEN
        ALTER TABLE user_identities
            ADD CONSTRAINT fk_user_identities_user FOREIGN KEY (user_id)
            REFERENCES users(user_id) ON DELETE CASCADE;
    END IF;
END$$;

-- ADD INDEXES (safe)
CREATE UNIQUE INDEX IF NOT EXISTS idx_user_identities_provider_combination
    ON user_identities (provider_name, provider_user_id);
`
	if err := db.Exec(sql).Error; err != nil {
		log.Fatalf("❌ Failed to run migration 011_create_user_identities_table: %v", err)
	}

	log.Println("✅ Migration 011_create_user_identities_table executed")
}
