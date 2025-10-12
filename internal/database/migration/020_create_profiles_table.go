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

    -- Basic Identity Information
    first_name      VARCHAR(100) NOT NULL,
    middle_name     VARCHAR(100),
    last_name       VARCHAR(100),
    suffix          VARCHAR(50),
    display_name    VARCHAR(150),

    -- Personal Information
    birthdate       DATE,
    gender          VARCHAR(25), -- 'male', 'female', 'other', 'prefer_not_to_say'
    bio             TEXT,

    -- Contact Information
    phone           VARCHAR(20),
    email           VARCHAR(255),

    -- Address Information
    address_line_1  TEXT,
    address_line_2  TEXT,
    city            VARCHAR(100),
    state_province  VARCHAR(100),
    postal_code     VARCHAR(20),
    country         VARCHAR(2),     -- ISO 3166-1 alpha-2 code
    country_name    VARCHAR(100),   -- Full country name

    -- Professional Information
    company         VARCHAR(200),
    job_title       VARCHAR(150),
    department      VARCHAR(150),
    industry        VARCHAR(100),
    website_url     TEXT,

    -- Media & Assets
    avatar_url      TEXT,
    avatar_s3_key   TEXT,
    cover_url       TEXT,
    cover_s3_key    TEXT,

    -- System Fields
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

-- ADD CHECK CONSTRAINTS FOR DATA INTEGRITY
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'chk_profiles_gender'
    ) THEN
        ALTER TABLE profiles
            ADD CONSTRAINT chk_profiles_gender
            CHECK (gender IN ('male', 'female', 'other', 'prefer_not_to_say'));
    END IF;
END$$;



-- ADD INDEXES
CREATE INDEX IF NOT EXISTS idx_profiles_uuid ON profiles (profile_uuid);
CREATE INDEX IF NOT EXISTS idx_profiles_user_id ON profiles (user_id);
CREATE INDEX IF NOT EXISTS idx_profiles_first_name ON profiles (first_name);
CREATE INDEX IF NOT EXISTS idx_profiles_last_name ON profiles (last_name);
CREATE INDEX IF NOT EXISTS idx_profiles_email ON profiles (email);
CREATE INDEX IF NOT EXISTS idx_profiles_display_name ON profiles (display_name);
CREATE INDEX IF NOT EXISTS idx_profiles_company ON profiles (company);
CREATE INDEX IF NOT EXISTS idx_profiles_created_at ON profiles (created_at);
`
	if err := db.Exec(sql).Error; err != nil {
		log.Fatalf("❌ Failed to run migration 020_create_profiles_table: %v", err)
	}

	log.Println("✅ Migration 020_create_profiles_table executed")
}
