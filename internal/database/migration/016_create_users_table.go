package migration

import (
	"log"

	"gorm.io/gorm"
)

func CreateUserTable(db *gorm.DB) {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS users (
    user_id                 SERIAL PRIMARY KEY,
    user_uuid               UUID NOT NULL UNIQUE,
    username                VARCHAR(255) NOT NULL,
    email                   VARCHAR(255),
    phone                   VARCHAR(20),
    password                TEXT,
    is_email_verified       BOOLEAN DEFAULT FALSE,
		is_phone_verified       BOOLEAN DEFAULT FALSE,
    is_profile_completed    BOOLEAN DEFAULT FALSE,
    is_account_completed    BOOLEAN DEFAULT FALSE,
    is_active               BOOLEAN DEFAULT FALSE,
    tenant_id               INTEGER NOT NULL,
    created_at              TIMESTAMPTZ DEFAULT now(),
    updated_at              TIMESTAMPTZ DEFAULT now()
);

-- ADD CONSTRAINTS
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_users_tenant_id'
    ) THEN
        ALTER TABLE users
            ADD CONSTRAINT fk_users_tenant_id FOREIGN KEY (tenant_id)
            REFERENCES tenants(tenant_id) ON DELETE CASCADE;
    END IF;
END$$;

-- ADD INDEXES
CREATE INDEX IF NOT EXISTS idx_users_uuid ON users (user_uuid);
CREATE INDEX IF NOT EXISTS idx_users_username ON users (username);
CREATE INDEX IF NOT EXISTS idx_users_email ON users (email);
CREATE INDEX IF NOT EXISTS idx_users_phone ON users (phone);
CREATE INDEX IF NOT EXISTS idx_users_tenant_id ON users (tenant_id);
CREATE INDEX IF NOT EXISTS idx_users_created_at ON users (created_at);
`
	if err := db.Exec(sql).Error; err != nil {
		log.Fatalf("❌ Failed to run migration 016_create_users_table: %v", err)
	}

	log.Println("✅ Migration 016_create_users_table executed")
}
