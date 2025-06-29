-- +goose Up

-- CREATE TABLE
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS users (
    user_id                 SERIAL PRIMARY KEY,
    user_uuid UUID          UNIQUE NOT NULL,
    username                VARCHAR(255) UNIQUE NOT NULL,
    email                   VARCHAR(255) UNIQUE NOT NULL,
    password                TEXT,
    is_email_verified       BOOLEAN DEFAULT FALSE,
    is_profile_completed    BOOLEAN DEFAULT FALSE,
    is_account_completed    BOOLEAN DEFAULT FALSE,
    is_active               BOOLEAN DEFAULT TRUE,
    created_at              TIMESTAMPTZ DEFAULT now(),
    updated_at              TIMESTAMPTZ
);
-- +goose StatementEnd

-- ADD INDEXES
-- +goose StatementBegin
CREATE INDEX idx_users_user_uuid ON users(user_uuid);
CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_email ON users(email);
-- +goose StatementEnd

-- +goose Down

-- DROP INDEXES
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_users_user_uuid;
DROP INDEX IF EXISTS idx_users_username;
DROP INDEX IF EXISTS idx_users_email;
-- +goose StatementEnd

-- DROP TABLE
-- +goose StatementBegin
DROP TABLE IF EXISTS users;
-- +goose StatementEnd
