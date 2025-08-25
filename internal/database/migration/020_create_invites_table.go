package migration

import (
	"log"

	"gorm.io/gorm"
)

func CreateInvitesTable(db *gorm.DB) {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS invites (
    invite_id           SERIAL PRIMARY KEY,
    invite_uuid         UUID NOT NULL UNIQUE,
    auth_client_id      INTEGER NOT NULL,
    invited_email       VARCHAR(255) NOT NULL,
    invited_by_user_id  INTEGER NOT NULL,
    invite_token        TEXT NOT NULL UNIQUE,
    status              VARCHAR(20) DEFAULT 'pending', -- pending, accepted, expired, revoked
    expires_at          TIMESTAMPTZ,
    used_at             TIMESTAMPTZ,
    created_at          TIMESTAMPTZ DEFAULT now(),
    updated_at          TIMESTAMPTZ DEFAULT now()
);

-- ADD CONSTRAINTS
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_invites_auth_client_id'
    ) THEN
        ALTER TABLE invites
            ADD CONSTRAINT fk_invites_auth_client_id FOREIGN KEY (auth_client_id)
            REFERENCES auth_clients(auth_client_id) ON DELETE CASCADE;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_invites_invited_by_user_id'
    ) THEN
        ALTER TABLE invites
            ADD CONSTRAINT fk_invites_invited_by_user_id FOREIGN KEY (invited_by_user_id)
            REFERENCES users(user_id) ON DELETE SET NULL;
    END IF;
END$$;

-- ADD INDEXES
CREATE INDEX IF NOT EXISTS idx_invites_invite_uuid ON invites (invite_uuid);
CREATE INDEX IF NOT EXISTS idx_invites_email ON invites (invited_email);
CREATE INDEX IF NOT EXISTS idx_invites_token ON invites (invite_token);
CREATE INDEX IF NOT EXISTS idx_invites_status ON invites (status);
CREATE INDEX IF NOT EXISTS idx_invites_auth_client_id ON invites (auth_client_id);
`
	if err := db.Exec(sql).Error; err != nil {
		log.Fatalf("❌ Failed to run migration 020_create_invites_table: %v", err)
	}

	log.Println("✅ Migration 020_create_invites_table executed")
}
