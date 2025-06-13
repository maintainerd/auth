-- +goose Up

-- CREATE TABLE
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS auth_containers (
    auth_container_id   SERIAL PRIMARY KEY,
    auth_container_uuid UUID NOT NULL UNIQUE,
    name                VARCHAR(255) NOT NULL,
    description         TEXT NOT NULL,
    is_active           BOOLEAN DEFAULT FALSE,
    is_default          BOOLEAN DEFAULT FALSE,
    organization_id     INTEGER NOT NULL,
    created_at          TIMESTAMPTZ DEFAULT now(),
    updated_at          TIMESTAMPTZ
);
-- +goose StatementEnd

-- ADD INDEXES
-- +goose StatementBegin
CREATE INDEX idx_auth_containers_auth_container_uuid ON auth_containers(auth_container_uuid);
CREATE INDEX idx_auth_containers_name ON auth_containers(name);
CREATE INDEX idx_auth_containers_is_active ON auth_containers(is_active);
CREATE INDEX idx_auth_containers_is_default ON auth_containers(is_default);
-- +goose StatementEnd

-- +goose Down

-- DROP INDEXES
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_auth_containers_auth_container_uuid;
DROP INDEX IF EXISTS idx_auth_containers_name;
DROP INDEX IF EXISTS idx_auth_containers_is_active;
DROP INDEX IF EXISTS idx_auth_containers_is_default;
-- +goose StatementEnd

-- DROP TABLE
-- +goose StatementBegin
DROP TABLE IF EXISTS auth_containers;
-- +goose StatementEnd
