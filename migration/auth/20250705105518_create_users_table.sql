-- +goose Up

-- CREATE TABLE
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS users (
    user_id                 SERIAL PRIMARY KEY,
    user_uuid               UUID UNIQUE NOT NULL,
    username                VARCHAR(255) NOT NULL,
    email                   VARCHAR(255) NOT NULL,
    password                TEXT,
    is_email_verified       BOOLEAN DEFAULT FALSE,
    is_profile_completed    BOOLEAN DEFAULT FALSE,
    is_account_completed    BOOLEAN DEFAULT FALSE,
    is_active               BOOLEAN DEFAULT TRUE,
    organization_id         INTEGER NOT NULL,
    auth_container_id       INTEGER NOT NULL,
    created_at              TIMESTAMPTZ DEFAULT now(),
    updated_at              TIMESTAMPTZ DEFAULT now()
);
-- +goose StatementEnd

-- ADD CONSTRAINTS
-- +goose StatementBegin
ALTER TABLE users
    ADD CONSTRAINT fk_users_organization_id FOREIGN KEY (organization_id) REFERENCES organizations(organization_id) ON DELETE CASCADE;
ALTER TABLE users
    ADD CONSTRAINT fk_users_auth_container_id FOREIGN KEY (auth_container_id) REFERENCES auth_containers(auth_container_id) ON DELETE CASCADE;
-- +goose StatementEnd

-- ADD INDEXES
-- +goose StatementBegin
CREATE INDEX idx_users_user_uuid ON users(user_uuid);
CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_organization_id ON users(organization_id);
CREATE INDEX idx_users_auth_container_id ON users(auth_container_id);
-- +goose StatementEnd

-- +goose Down

-- DROP INDEXES
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_users_user_uuid;
DROP INDEX IF EXISTS idx_users_username;
DROP INDEX IF EXISTS idx_users_email;
DROP INDEX IF EXISTS idx_users_organization_id;
DROP INDEX IF EXISTS idx_users_auth_container_id;
-- +goose StatementEnd

-- DROP CONSTRAINTS
-- +goose StatementBegin
ALTER TABLE users DROP CONSTRAINT IF EXISTS fk_users_organization_id;
ALTER TABLE users DROP CONSTRAINT IF EXISTS fk_users_auth_container_id;
-- +goose StatementEnd

-- DROP TABLE
-- +goose StatementBegin
DROP TABLE IF EXISTS users;
-- +goose StatementEnd
