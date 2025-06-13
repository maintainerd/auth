-- +goose Up

-- CREATE TABLE
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS profiles (
    profile_id      SERIAL PRIMARY KEY,
    profile_uuid    UUID NOT NULL UNIQUE,
    user_id         INTEGER NOT NULL,
    -- Personal Information
    first_name      VARCHAR(100) NOT NULL,
    middle_name     VARCHAR(100),
    last_name       VARCHAR(100),
    suffix          VARCHAR(50),
    birthdate       DATE,
    gender          VARCHAR(10), -- 'male', 'female'
    -- Contact Information
    phone           VARCHAR(20),
    email           VARCHAR(255),
    address         TEXT,
    -- Media
    avatar_url      TEXT,
    avatar_s3_key   TEXT,
    cover_url       TEXT,
    cover_s3_key    TEXT,
    -- Metadata
    created_at      TIMESTAMPTZ DEFAULT now(),
    updated_at      TIMESTAMPTZ DEFAULT now()
);
-- +goose StatementEnd

-- ADD CONSTRAINTS
-- +goose StatementBegin
ALTER TABLE profiles
    ADD CONSTRAINT fk_profiles_user_id FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE;
-- +goose StatementEnd

-- ADD INDEXES
-- +goose StatementBegin
CREATE INDEX idx_profiles_user_id ON profiles(user_id);
CREATE INDEX idx_profiles_profile_uuid ON profiles(profile_uuid);
CREATE INDEX idx_profiles_first_name ON profiles(first_name);
CREATE INDEX idx_profiles_last_name ON profiles(last_name);
-- +goose StatementEnd

-- +goose Down

-- DROP INDEXES
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_profiles_user_id;
DROP INDEX IF EXISTS idx_profiles_profile_uuid;
DROP INDEX IF EXISTS idx_profiles_first_name;
DROP INDEX IF EXISTS idx_profiles_last_name;
-- +goose StatementEnd

-- DROP CONSTRAINTS
-- +goose StatementBegin
ALTER TABLE profiles DROP CONSTRAINT IF EXISTS fk_profiles_user_id;
-- +goose StatementEnd

-- DROP TABLE
-- +goose StatementBegin
DROP TABLE IF EXISTS profiles;
-- +goose StatementEnd
