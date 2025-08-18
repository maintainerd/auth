package migration

import (
	"log"

	"gorm.io/gorm"
)

func CreateAuthLogTable(db *gorm.DB) {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS auth_logs (
    auth_log_id         SERIAL PRIMARY KEY,
    auth_log_uuid       UUID NOT NULL UNIQUE,
    user_id             INTEGER NOT NULL,
    event_type          VARCHAR(100) NOT NULL, -- e.g., 'login', 'logout', 'token_refresh', 'password_reset'
    description         TEXT,
    ip_address          VARCHAR(100),
    user_agent          TEXT,
    metadata            JSONB,
    auth_container_id   INTEGER NOT NULL,
    created_at          TIMESTAMPTZ DEFAULT now(),
    updated_at          TIMESTAMPTZ DEFAULT now()
);

-- ADD CONSTRAINTS (safe)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_auth_logs_user_id'
    ) THEN
        ALTER TABLE auth_logs
            ADD CONSTRAINT fk_auth_logs_user_id FOREIGN KEY (user_id)
            REFERENCES users(user_id) ON DELETE CASCADE;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_auth_logs_auth_container_id'
    ) THEN
        ALTER TABLE auth_logs
            ADD CONSTRAINT fk_auth_logs_auth_container_id FOREIGN KEY (auth_container_id)
            REFERENCES auth_containers(auth_container_id) ON DELETE CASCADE;
    END IF;
END$$;

-- ADD INDEXES
CREATE INDEX IF NOT EXISTS idx_auth_logs_user_id ON auth_logs (user_id);
CREATE INDEX IF NOT EXISTS idx_auth_logs_event_type ON auth_logs (event_type);
CREATE INDEX IF NOT EXISTS idx_auth_logs_auth_container_id ON auth_logs (auth_container_id);
CREATE INDEX IF NOT EXISTS idx_auth_logs_auth_log_uuid ON auth_logs (auth_log_uuid);
`
	if err := db.Exec(sql).Error; err != nil {
		log.Fatalf("❌ Failed to run migration 018_create_auth_logs_table: %v", err)
	}

	log.Println("✅ Migration 018_create_auth_logs_table executed")
}
