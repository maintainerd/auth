-- +goose Up

-- CREATE TABLE
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS permissions (
    permission_id       SERIAL PRIMARY KEY,
    permission_uuid     UUID UNIQUE NOT NULL,
    name VARCHAR(255)   UNIQUE NOT NULL,
    description         TEXT,
    is_default          BOOLEAN DEFAULT FALSE,
    created_at          TIMESTAMPTZ DEFAULT now(),
    updated_at          TIMESTAMPTZ
);
-- +goose StatementEnd

-- ADD INDEXES
-- +goose StatementBegin
CREATE INDEX idx_permissions_permission_uuid ON permissions(permission_uuid);
CREATE INDEX idx_permissions_name ON permissions(name);
-- +goose StatementEnd

-- +goose Down

-- DROP INDEXES
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_permissions_permission_uuid;
DROP INDEX IF EXISTS idx_permissions_name;
-- +goose StatementEnd

-- DROP TABLE
-- +goose StatementBegin
DROP TABLE IF EXISTS permissions;
-- +goose StatementEnd
