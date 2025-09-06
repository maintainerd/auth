package migration

import (
	"log"

	"gorm.io/gorm"
)

func CreateUserTokenTable(db *gorm.DB) {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS user_tokens (
    user_token_id				SERIAL PRIMARY KEY,
    user_token_uuid			UUID NOT NULL UNIQUE,
		user_id							INTEGER NOT NULL,
    token_type					VARCHAR(50) NOT NULL, -- 'refresh', 'api', 'reset_password'
    token								TEXT NOT NULL, -- hashed token string
    user_agent					TEXT,
    ip_address					VARCHAR(50),
    expires_at					TIMESTAMPTZ,
		is_revoked					BOOLEAN DEFAULT FALSE,
    created_at					TIMESTAMPTZ DEFAULT now(),
    updated_at					TIMESTAMPTZ DEFAULT now()
);

-- ADD CONSTRAINTS
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_user_tokens_user'
    ) THEN
        ALTER TABLE user_tokens
            ADD CONSTRAINT fk_user_tokens_user FOREIGN KEY (user_id)
            REFERENCES users(user_id) ON DELETE CASCADE;
    END IF;
END$$;

-- ADD INDEXES
CREATE INDEX IF NOT EXISTS idx_user_tokens_uuid ON user_tokens (user_token_uuid);
CREATE INDEX IF NOT EXISTS idx_user_tokens_user_id ON user_tokens (user_id);
CREATE INDEX IF NOT EXISTS idx_user_tokens_token_type ON user_tokens (token_type);
CREATE INDEX IF NOT EXISTS idx_user_tokens_token ON user_tokens (token);
CREATE INDEX IF NOT EXISTS idx_user_tokens_created_at ON user_tokens (created_at);
`
	if err := db.Exec(sql).Error; err != nil {
		log.Fatalf("❌ Failed to run migration 018_create_user_tokens_table: %v", err)
	}

	log.Println("✅ Migration 018_create_user_tokens_table executed")
}
