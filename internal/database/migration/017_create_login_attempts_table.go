package migration

import (
	"log"

	"gorm.io/gorm"
)

func CreateLoginAttemptTable(db *gorm.DB) {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS login_attempts (
    login_attempt_id    SERIAL PRIMARY KEY,
    login_attempt_uuid  UUID NOT NULL UNIQUE,
    user_id             INTEGER, -- nullable: user might not exist or match
    email               VARCHAR(255), -- the email/username used in the attempt
    ip_address          VARCHAR(100),
    user_agent          TEXT,
    is_success          BOOLEAN DEFAULT FALSE,
    attempted_at        TIMESTAMPTZ DEFAULT now(),
    auth_container_id   INTEGER NOT NULL,
    created_at          TIMESTAMPTZ DEFAULT now(),
    updated_at          TIMESTAMPTZ DEFAULT now()
);

-- ADD CONSTRAINTS
ALTER TABLE login_attempts
    ADD CONSTRAINT fk_login_attempts_user_id FOREIGN KEY (user_id) REFERENCES users(user_id);
ALTER TABLE login_attempts
    ADD CONSTRAINT fk_login_attempts_auth_container_id FOREIGN KEY (auth_container_id) REFERENCES auth_containers(auth_container_id) ON DELETE CASCADE;

-- ADD INDEXES
CREATE INDEX idx_login_attempts_user_id ON login_attempts (user_id);
CREATE INDEX idx_login_attempts_email ON login_attempts (email);
CREATE INDEX idx_login_attempts_auth_container_id ON login_attempts (auth_container_id);
`
	if err := db.Exec(sql).Error; err != nil {
		log.Fatalf("❌ Failed to run migration 017_create_login_attempts_table: %v", err)
	}

	log.Println("✅ Migration 017_create_login_attempts_table executed")
}
