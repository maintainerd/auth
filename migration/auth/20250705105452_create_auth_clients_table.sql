-- +goose Up

-- CREATE TABLE
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS auth_clients (
    auth_client_id          SERIAL PRIMARY KEY,
    auth_client_uuid        UUID NOT NULL UNIQUE,
    client_name             VARCHAR(100) NOT NULL, -- 'default', 'google', 'facebook', 'github'
    display_name            TEXT NOT NULL,
    client_type             VARCHAR(100) NOT NULL, -- 'traditional', 'spa', 'native', 'm2m'
    domain                  TEXT, -- optional
    client_id               TEXT, -- optional
    client_secret           TEXT, -- optional
    redirect_uri            TEXT, -- optional
    config                  JSONB,
    is_active               BOOLEAN DEFAULT FALSE,
    is_default              BOOLEAN DEFAULT FALSE,
    identity_provider_id    INTEGER NOT NULL,
    auth_container_id       INTEGER NOT NULL,
    created_at              TIMESTAMPTZ DEFAULT now(),
    updated_at              TIMESTAMPTZ DEFAULT now()
);
-- +goose StatementEnd

-- ADD CONSTRAINTS
-- +goose StatementBegin
ALTER TABLE auth_clients
    ADD CONSTRAINT fk_auth_clients_identity_provider_id FOREIGN KEY (identity_provider_id) REFERENCES identity_providers(identity_provider_id) ON DELETE CASCADE;
ALTER TABLE auth_clients
    ADD CONSTRAINT fk_auth_clients_auth_container_id FOREIGN KEY (auth_container_id) REFERENCES auth_containers(auth_container_id) ON DELETE CASCADE;
-- +goose StatementEnd

-- ADD INDEXES
-- +goose StatementBegin
CREATE INDEX idx_auth_clients_identity_provider_id ON auth_clients (identity_provider_id);
CREATE INDEX idx_auth_clients_auth_container_id ON auth_clients (auth_container_id);
-- +goose StatementEnd

-- +goose Down

-- DROP INDEXES
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_auth_clients_identity_provider_id;
DROP INDEX IF EXISTS idx_auth_clients_auth_container_id;
-- +goose StatementEnd

-- DROP CONSTRAINTS
-- +goose StatementBegin
ALTER TABLE auth_clients DROP CONSTRAINT IF EXISTS fk_auth_clients_identity_provider_id;
ALTER TABLE auth_clients DROP CONSTRAINT IF EXISTS fk_auth_clients_auth_container_id;
-- +goose StatementEnd

-- DROP TABLE
-- +goose StatementBegin
DROP TABLE IF EXISTS auth_clients;
-- +goose StatementEnd
