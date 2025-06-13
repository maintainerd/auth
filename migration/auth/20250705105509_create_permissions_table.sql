-- +goose Up

-- CREATE TABLE
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS permissions (
    permission_id       SERIAL PRIMARY KEY,
    permission_uuid     UUID UNIQUE NOT NULL,
    name                VARCHAR(255) UNIQUE NOT NULL,
    description         TEXT NOT NULL,
    is_active           BOOLEAN DEFAULT FALSE,
    is_default          BOOLEAN DEFAULT FALSE,
    auth_container_id   INTEGER NOT NULL,
    created_at          TIMESTAMPTZ DEFAULT now(),
    updated_at          TIMESTAMPTZ DEFAULT now()
);
-- +goose StatementEnd

-- ADD CONSTRAINTS
-- +goose StatementBegin
ALTER TABLE permissions
    ADD CONSTRAINT fk_permissions_auth_container_id FOREIGN KEY (auth_container_id) REFERENCES auth_containers(auth_container_id) ON DELETE CASCADE;
-- +goose StatementEnd

-- ADD INDEXES
-- +goose StatementBegin
CREATE INDEX idx_permissions_permission_uuid ON permissions(permission_uuid);
CREATE INDEX idx_permissions_name ON permissions(name);
CREATE INDEX idx_permissions_auth_container_id ON permissions(auth_container_id);
-- +goose StatementEnd

-- +goose Down

-- DROP INDEXES
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_permissions_permission_uuid;
DROP INDEX IF EXISTS idx_permissions_name;
DROP INDEX IF EXISTS idx_permissions_auth_container_id;
-- +goose StatementEnd

-- DROP CONSTRAINTS
-- +goose StatementBegin
ALTER TABLE permissions DROP CONSTRAINT IF EXISTS fk_permissions_auth_container_id;
-- +goose StatementEnd

-- DROP TABLE
-- +goose StatementBegin
DROP TABLE IF EXISTS permissions;
-- +goose StatementEnd
