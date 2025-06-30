-- +goose Up

-- CREATE TABLE
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS roles (
    role_id             SERIAL PRIMARY KEY,
    role_uuid           UUID UNIQUE NOT NULL,
    name VARCHAR(255)   UNIQUE NOT NULL,
    description         TEXT,
    is_default          BOOLEAN DEFAULT FALSE,
    created_at          TIMESTAMPTZ DEFAULT now(),
    updated_at          TIMESTAMPTZ
);
-- +goose StatementEnd

-- ADD INDEXES
-- +goose StatementBegin
CREATE INDEX idx_roles_role_uuid ON roles(role_uuid);
CREATE INDEX idx_roles_name ON roles(name);
-- +goose StatementEnd

-- +goose Down

-- DROP INDEXES
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_roles_role_uuid;
DROP INDEX IF EXISTS idx_roles_name;
-- +goose StatementEnd

-- DROP TABLE
-- +goose StatementBegin
DROP TABLE IF EXISTS roles;
-- +goose StatementEnd
