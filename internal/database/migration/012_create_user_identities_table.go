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
    auth_client_id      INTEGER NOT NULL, -- link to auth_clients
    provider            VARCHAR(100) NOT NULL, -- 'google', 'cognito', 'microsoft'
    sub                 VARCHAR(255) NOT NULL, -- external subject (user id at provider)
    user_data           JSONB,
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

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_user_identities_auth_client'
    ) THEN
        ALTER TABLE user_identities
            ADD CONSTRAINT fk_user_identities_auth_client FOREIGN KEY (auth_client_id)
            REFERENCES auth_clients(auth_client_id) ON DELETE CASCADE;
    END IF;
END$$;

-- ADD INDEXES (safe)
CREATE UNIQUE INDEX IF NOT EXISTS idx_user_identities_provider_sub
    ON user_identities (provider, sub);

CREATE INDEX IF NOT EXISTS idx_user_identities_auth_client_id
    ON user_identities (auth_client_id);
`
	if err := db.Exec(sql).Error; err != nil {
		log.Fatalf("❌ Failed to run migration 012_create_user_identities_table: %v", err)
	}

	log.Println("✅ Migration 012_create_user_identities_table executed")
}
