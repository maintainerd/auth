package migration

import (
	"log"

	"gorm.io/gorm"
)

func CreateUserTokenTable(db *gorm.DB) {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS user_tokens (
    token_id        SERIAL PRIMARY KEY,
    token_uuid      UUID NOT NULL UNIQUE,
    user_id         INTEGER NOT NULL,
    token_type      VARCHAR(50) NOT NULL, -- e.g. "refresh", "api", "reset_password"
    token           TEXT NOT NULL,        -- hashed token string
    user_agent      TEXT,
    ip_address      VARCHAR(50),
    is_revoked      BOOLEAN DEFAULT FALSE,
    expires_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ DEFAULT now(),
    updated_at      TIMESTAMPTZ
);

-- ADD CONSTRAINTS
ALTER TABLE user_tokens
    ADD CONSTRAINT fk_user_tokens_user FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE;

-- ADD INDEXES
CREATE INDEX idx_user_tokens_user_id ON user_tokens(user_id);
CREATE INDEX idx_user_tokens_token_uuid ON user_tokens(token_uuid);
CREATE INDEX idx_user_tokens_token_type ON user_tokens(token_type);
`
	if err := db.Exec(sql).Error; err != nil {
		log.Fatalf("❌ Failed to run migration 013_create_user_tokens_table: %v", err)
	}

	log.Println("✅ Migration 013_create_user_tokens_table executed")
}
