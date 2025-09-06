package migration

import (
	"log"

	"gorm.io/gorm"
)

func CreateUserRoleTable(db *gorm.DB) {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS user_roles (
    user_role_id      SERIAL PRIMARY KEY,
    user_role_uuid    UUID NOT NULL UNIQUE,
    user_id           INTEGER NOT NULL,
    role_id           INTEGER NOT NULL,
    created_at        TIMESTAMPTZ DEFAULT now()
);

-- ADD CONSTRAINTS (safe)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_user_roles_user_id'
    ) THEN
        ALTER TABLE user_roles
            ADD CONSTRAINT fk_user_roles_user_id FOREIGN KEY (user_id)
            REFERENCES users(user_id) ON DELETE CASCADE;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_user_roles_role_id'
    ) THEN
        ALTER TABLE user_roles
            ADD CONSTRAINT fk_user_roles_role_id FOREIGN KEY (role_id)
            REFERENCES roles(role_id) ON DELETE CASCADE;
    END IF;
END$$;

-- ADD INDEXES
CREATE INDEX IF NOT EXISTS idx_user_roles_uuid ON user_roles (user_role_uuid);
CREATE INDEX IF NOT EXISTS idx_user_roles_user_id ON user_roles (user_id);
CREATE INDEX IF NOT EXISTS idx_user_roles_role_id ON user_roles (role_id);
`
	if err := db.Exec(sql).Error; err != nil {
		log.Fatalf("❌ Failed to run migration 017_create_user_roles_table: %v", err)
	}

	log.Println("✅ Migration 017_create_user_roles_table executed")
}
