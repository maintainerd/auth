package migration

import (
	"log"

	"gorm.io/gorm"
)

func CreateRolePermissionTable(db *gorm.DB) {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS role_permissions (
    role_permission_id      SERIAL PRIMARY KEY,
    role_permission_uuid    UUID NOT NULL UNIQUE,
    role_id                 INTEGER NOT NULL,
    permission_id           INTEGER NOT NULL,
    created_at              TIMESTAMPTZ DEFAULT now()
);

-- ADD CONSTRAINTS
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_role_permissions_role_id'
    ) THEN
        ALTER TABLE role_permissions
            ADD CONSTRAINT fk_role_permissions_role_id FOREIGN KEY (role_id)
            REFERENCES roles(role_id) ON DELETE CASCADE;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_role_permissions_permission_id'
    ) THEN
        ALTER TABLE role_permissions
            ADD CONSTRAINT fk_role_permissions_permission_id FOREIGN KEY (permission_id)
            REFERENCES permissions(permission_id) ON DELETE CASCADE;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'uq_role_permissions_role_permission'
    ) THEN
        ALTER TABLE role_permissions
            ADD CONSTRAINT uq_role_permissions_role_permission UNIQUE (role_id, permission_id);
    END IF;
END$$;

-- ADD INDEXES
CREATE INDEX IF NOT EXISTS idx_role_permissions_uuid ON role_permissions (role_permission_uuid);
CREATE INDEX IF NOT EXISTS idx_role_permissions_role_id ON role_permissions (role_id);
CREATE INDEX IF NOT EXISTS idx_role_permissions_permission_id ON role_permissions (permission_id);
`
	if err := db.Exec(sql).Error; err != nil {
		log.Fatalf("❌ Failed to run migration 019_create_role_permissions_table: %v", err)
	}

	log.Println("✅ Migration 019_create_role_permissions_table executed")
}
