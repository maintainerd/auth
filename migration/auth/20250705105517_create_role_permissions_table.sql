-- +goose Up

-- CREATE TABLE
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS role_permissions (
    role_permission_id      SERIAL PRIMARY KEY,
    role_permission_uuid    UUID UNIQUE NOT NULL,
    role_id                 INTEGER NOT NULL,
    permission_id           INTEGER NOT NULL,
    is_default              BOOLEAN DEFAULT FALSE,
    created_at              TIMESTAMPTZ DEFAULT now(),
    updated_at              TIMESTAMPTZ
);
-- +goose StatementEnd

-- ADD CONSTRAINTS
-- +goose StatementBegin
ALTER TABLE role_permissions
    ADD CONSTRAINT fk_role_permissions_role_id FOREIGN KEY (role_id) REFERENCES roles(role_id) ON DELETE CASCADE;
ALTER TABLE role_permissions
    ADD CONSTRAINT fk_role_permissions_permission_id FOREIGN KEY (permission_id) REFERENCES permissions(permission_id) ON DELETE CASCADE;
ALTER TABLE role_permissions
    ADD CONSTRAINT unique_role_permissions_role_id_permission_id UNIQUE (role_id, permission_id);
-- +goose StatementEnd

-- ADD INDEXES
-- +goose StatementBegin
CREATE INDEX idx_role_permissions_uuid ON role_permissions(role_permission_uuid);
CREATE INDEX idx_role_permissions_role_id ON role_permissions(role_id);
CREATE INDEX idx_role_permissions_permission_id ON role_permissions(permission_id);
-- +goose StatementEnd

-- +goose Down

-- DROP INDEXES
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_role_permissions_uuid;
DROP INDEX IF EXISTS idx_role_permissions_role_id;
DROP INDEX IF EXISTS idx_role_permissions_permission_id;
-- +goose StatementEnd

-- DROP CONSTRAINTS
-- +goose StatementBegin
ALTER TABLE role_permissions DROP CONSTRAINT IF EXISTS fk_role_permissions_role_id;
ALTER TABLE role_permissions DROP CONSTRAINT IF EXISTS fk_role_permissions_permission_id;
ALTER TABLE role_permissions DROP CONSTRAINT IF EXISTS unique_role_permissions_role_id_permission_id;
-- +goose StatementEnd

-- DROP TABLE
-- +goose StatementBegin
DROP TABLE IF EXISTS role_permissions;
-- +goose StatementEnd
