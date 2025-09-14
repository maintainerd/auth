package migration

import (
	"log"

	"gorm.io/gorm"
)

func CreateInviteRolesTable(db *gorm.DB) {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS invite_roles (
    invite_role_id			SERIAL PRIMARY KEY,
		invite_role_uuid		UUID NOT NULL UNIQUE,
    invite_id						INTEGER NOT NULL,
    role_id							INTEGER NOT NULL,
    created_at					TIMESTAMPTZ DEFAULT now()
);

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_invite_roles_invite_id'
    ) THEN
        ALTER TABLE invite_roles
            ADD CONSTRAINT fk_invite_roles_invite_id FOREIGN KEY (invite_id)
            REFERENCES invites(invite_id) ON DELETE CASCADE;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_invite_roles_role_id'
    ) THEN
        ALTER TABLE invite_roles
            ADD CONSTRAINT fk_invite_roles_role_id FOREIGN KEY (role_id)
            REFERENCES roles(role_id) ON DELETE CASCADE;
    END IF;
END$$;

-- ADD INDEXES
CREATE INDEX IF NOT EXISTS idx_invite_roles_invite_role_uuid ON invite_roles (invite_role_uuid);
CREATE INDEX IF NOT EXISTS idx_invite_roles_invite_id ON invite_roles (invite_id);
CREATE INDEX IF NOT EXISTS idx_invite_roles_role_id ON invite_roles (role_id);
`
	if err := db.Exec(sql).Error; err != nil {
		log.Fatalf("❌ Failed to run migration 025_create_invite_roles_table: %v", err)
	}

	log.Println("✅ Migration 025_create_invite_roles_table executed")
}
