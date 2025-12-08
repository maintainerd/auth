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
    auth_client_id      INTEGER NOT NULL,
		sub                 VARCHAR(255) NOT NULL, -- external subject (user identifier from provider)
    provider            VARCHAR(100) NOT NULL, -- 'google', 'cognito', 'microsoft'
    metadata           	JSONB,
    created_at          TIMESTAMPTZ DEFAULT now(),
		updated_at          TIMESTAMPTZ DEFAULT now()
);

-- ADD CONSTRAINTS
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

-- ADD INDEXES
CREATE INDEX IF NOT EXISTS idx_user_identities_uuid ON user_identities (user_identity_uuid);
CREATE INDEX IF NOT EXISTS idx_user_identities_user_id ON user_identities (user_id);
CREATE INDEX IF NOT EXISTS idx_user_identities_auth_client_id ON user_identities (auth_client_id);
CREATE INDEX IF NOT EXISTS idx_user_identities_sub ON user_identities (sub);
CREATE INDEX IF NOT EXISTS idx_user_identities_provider ON user_identities (provider);
CREATE INDEX IF NOT EXISTS idx_user_identities_created_at ON user_identities (created_at);
`
	if err := db.Exec(sql).Error; err != nil {
		log.Fatalf("❌ Failed to run migration 017_create_user_identities_table: %v", err)
	}

	log.Println("✅ Migration 017_create_user_identities_table executed")
}
