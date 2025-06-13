-- +goose Up

-- CREATE TABLE
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS api_permissions (
    api_permission_id       SERIAL PRIMARY KEY,
    api_permission_uuid     UUID UNIQUE NOT NULL,
    api_id                  INTEGER NOT NULL,
    permission_id           INTEGER NOT NULL,
    is_default              BOOLEAN DEFAULT FALSE,
    created_at              TIMESTAMPTZ DEFAULT now(),
    updated_at              TIMESTAMPTZ DEFAULT now()
);
-- +goose StatementEnd

-- ADD CONSTRAINTS
-- +goose StatementBegin
ALTER TABLE api_permissions
    ADD CONSTRAINT fk_api_permissions_api_id FOREIGN KEY (api_id) REFERENCES apis(api_id) ON DELETE CASCADE;
ALTER TABLE api_permissions
    ADD CONSTRAINT fk_api_permissions_permission_id FOREIGN KEY (permission_id) REFERENCES permissions(permission_id) ON DELETE CASCADE;
ALTER TABLE api_permissions
    ADD CONSTRAINT unique_api_permissions_api_id_permission_id UNIQUE (api_id, permission_id);
-- +goose StatementEnd

-- ADD INDEXES
-- +goose StatementBegin
CREATE INDEX idx_api_permissions_uuid ON api_permissions(api_permission_uuid);
CREATE INDEX idx_api_permissions_api_id ON api_permissions(api_id);
CREATE INDEX idx_api_permissions_permission_id ON api_permissions(permission_id);
-- +goose StatementEnd

-- +goose Down

-- DROP INDEXES
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_api_permissions_uuid;
DROP INDEX IF EXISTS idx_api_permissions_api_id;
DROP INDEX IF EXISTS idx_api_permissions_permission_id;
-- +goose StatementEnd

-- DROP CONSTRAINTS
-- +goose StatementBegin
ALTER TABLE api_permissions DROP CONSTRAINT IF EXISTS fk_api_permissions_api_id;
ALTER TABLE api_permissions DROP CONSTRAINT IF EXISTS fk_api_permissions_permission_id;
ALTER TABLE api_permissions DROP CONSTRAINT IF EXISTS unique_api_permissions_api_id_permission_id;
-- +goose StatementEnd

-- DROP TABLE
-- +goose StatementBegin
DROP TABLE IF EXISTS api_permissions;
-- +goose StatementEnd
