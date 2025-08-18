package migration

import (
	"log"

	"gorm.io/gorm"
)

func CreateProfileTable(db *gorm.DB) {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS profiles (
    profile_id      SERIAL PRIMARY KEY,
    profile_uuid    UUID NOT NULL UNIQUE,
    user_id         INTEGER NOT NULL,
    first_name      VARCHAR(100) NOT NULL,
    middle_name     VARCHAR(100),
    last_name       VARCHAR(100),
    suffix          VARCHAR(50),
    birthdate       DATE,
    gender          VARCHAR(10), -- 'male', 'female'
    phone           VARCHAR(20),
    email           VARCHAR(255),
    address         TEXT,
    avatar_url      TEXT,
    avatar_s3_key   TEXT,
    cover_url       TEXT,
    cover_s3_key    TEXT,
    created_at      TIMESTAMPTZ DEFAULT now(),
    updated_at      TIMESTAMPTZ DEFAULT now()
);

-- ADD CONSTRAINTS (safe)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_profiles_user_id'
    ) THEN
        ALTER TABLE profiles
            ADD CONSTRAINT fk_profiles_user_id FOREIGN KEY (user_id)
            REFERENCES users(user_id) ON DELETE CASCADE;
    END IF;
END$$;

-- ADD INDEXES
CREATE INDEX IF NOT EXISTS idx_profiles_user_id ON profiles(user_id);
CREATE INDEX IF NOT EXISTS idx_profiles_profile_uuid ON profiles(profile_uuid);
CREATE INDEX IF NOT EXISTS idx_profiles_first_name ON profiles(first_name);
CREATE INDEX IF NOT EXISTS idx_profiles_last_name ON profiles(last_name);
`
	if err := db.Exec(sql).Error; err != nil {
		log.Fatalf("❌ Failed to run migration 014_create_profiles_table: %v", err)
	}

	log.Println("✅ Migration 014_create_profiles_table executed")
}
