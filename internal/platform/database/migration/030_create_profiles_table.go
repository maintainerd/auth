package migration

import (
	"gorm.io/gorm"
)

// CreateProfileTable creates the profiles table.
// Removed columns (vs prior versions):
//   - email     → use users.email (auth identifier)
//   - timezone  → use user_settings.timezone (preference)
//   - language  → use user_settings.locale (preference, BCP-47)
func CreateProfileTable(db *gorm.DB) error {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS profiles (
    profile_id      BIGSERIAL PRIMARY KEY,
    profile_uuid    UUID NOT NULL UNIQUE,
    user_id         BIGINT NOT NULL,
    -- Basic Identity Information
    first_name      VARCHAR(100) NOT NULL,
    middle_name     VARCHAR(100),
    last_name       VARCHAR(100),
    suffix          VARCHAR(50),
    display_name    VARCHAR(150),
    bio             TEXT,
    -- Personal Information
    birthdate       DATE,
    gender          VARCHAR(25),
    phone           VARCHAR(20),
    address         VARCHAR(500),
    city            VARCHAR(100),
    country         VARCHAR(2),     -- ISO 3166-1 alpha-2 code
    -- Media & Assets (auth-centric)
    profile_url     VARCHAR(2048),
    -- Profile Flags
    is_default      BOOLEAN NOT NULL DEFAULT false,
    -- Extended data
    metadata        JSONB DEFAULT '{}',
    -- Audit
    created_by      BIGINT,
    updated_by      BIGINT,
    -- System Fields
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at      TIMESTAMPTZ
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

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_profiles_created_by'
    ) THEN
        ALTER TABLE profiles
            ADD CONSTRAINT fk_profiles_created_by FOREIGN KEY (created_by)
            REFERENCES users(user_id) ON DELETE SET NULL;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_profiles_updated_by'
    ) THEN
        ALTER TABLE profiles
            ADD CONSTRAINT fk_profiles_updated_by FOREIGN KEY (updated_by)
            REFERENCES users(user_id) ON DELETE SET NULL;
    END IF;

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
CREATE INDEX IF NOT EXISTS idx_profiles_display_name ON profiles (display_name);
CREATE INDEX IF NOT EXISTS idx_profiles_created_at ON profiles (created_at);
CREATE INDEX IF NOT EXISTS idx_profiles_deleted_at ON profiles (deleted_at) WHERE deleted_at IS NULL;
`
	return db.Exec(sql).Error
}
