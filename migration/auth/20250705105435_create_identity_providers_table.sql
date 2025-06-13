-- +goose Up

-- CREATE TABLE
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS identity_providers (
    identity_provider_id    SERIAL PRIMARY KEY,
    identity_provider_uuid  UUID NOT NULL UNIQUE,
    provider_name           VARCHAR(100) NOT NULL, -- 'default', 'cognito', 'auth0'
    display_name            TEXT NOT NULL,
    provider_type           VARCHAR(100) NOT NULL, -- 'primary', 'oauth2'
    identifier              TEXT,
    config                  JSONB,
    is_active               BOOLEAN DEFAULT FALSE,
    is_default              BOOLEAN DEFAULT FALSE,
    auth_container_id       INTEGER NOT NULL,
    created_at              TIMESTAMPTZ DEFAULT now(),
    updated_at              TIMESTAMPTZ DEFAULT now()
);
-- +goose StatementEnd

-- ADD CONSTRAINTS
-- +goose StatementBegin
ALTER TABLE identity_providers
    ADD CONSTRAINT fk_identity_providers_auth_container_id FOREIGN KEY (auth_container_id) REFERENCES auth_containers(auth_container_id) ON DELETE CASCADE;
-- +goose StatementEnd

-- ADD INDEXES
-- +goose StatementBegin
CREATE INDEX idx_identity_providers_auth_container_id ON identity_providers (auth_container_id);
CREATE INDEX idx_identity_providers_provider_name ON identity_providers (provider_name);
-- +goose StatementEnd

-- +goose Down

-- DROP INDEXES
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_identity_providers_auth_container_id;
DROP INDEX IF EXISTS idx_identity_providers_provider_name;
-- +goose StatementEnd

-- DROP CONSTRAINTS
-- +goose StatementBegin
ALTER TABLE identity_providers DROP CONSTRAINT IF EXISTS fk_identity_providers_auth_container_id;
-- +goose StatementEnd

-- DROP TABLE
-- +goose StatementBegin
DROP TABLE IF EXISTS identity_providers;
-- +goose StatementEnd
