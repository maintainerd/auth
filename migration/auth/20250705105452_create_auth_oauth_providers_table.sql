-- +goose Up

-- CREATE TABLE
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS auth_oauth_providers (
    auth_oauth_provider_id      SERIAL PRIMARY KEY,
    auth_oauth_provider_uuid    UUID NOT NULL UNIQUE,
    provider_name               VARCHAR(100) NOT NULL, -- 'default', 'google', 'facebook', 'github'
    display_name                TEXT NOT NULL,
    application_type            VARCHAR(100) NOT NULL, -- 'native', 'spa', 'traditional', 'm2m'
    client_id                   TEXT,
    client_secret               TEXT,
    redirect_uri                TEXT,
    metadata                    JSONB, -- OIDC discovery URL
    is_active                   BOOLEAN DEFAULT FALSE,
    is_default                  BOOLEAN DEFAULT FALSE,
    auth_container_id           INTEGER NOT NULL,
    created_at                  TIMESTAMPTZ DEFAULT now(),
    updated_at                  TIMESTAMPTZ DEFAULT now()
);
-- +goose StatementEnd

-- ADD CONSTRAINTS
-- +goose StatementBegin
ALTER TABLE auth_oauth_providers
    ADD CONSTRAINT fk_auth_oauth_providers_auth_container_id FOREIGN KEY (auth_container_id) REFERENCES auth_containers(auth_container_id) ON DELETE CASCADE;
-- +goose StatementEnd

-- ADD INDEXES
-- +goose StatementBegin
CREATE INDEX idx_auth_oauth_providers_provider_name ON auth_oauth_providers (provider_name);
CREATE INDEX idx_auth_oauth_providers_application_type ON auth_oauth_providers (application_type);
CREATE INDEX idx_auth_oauth_providers_auth_container_id ON auth_oauth_providers (auth_container_id);
-- +goose StatementEnd

-- +goose Down

-- DROP INDEXES
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_auth_oauth_providers_provider_name;
DROP INDEX IF EXISTS idx_auth_oauth_providers_application_type;
DROP INDEX IF EXISTS idx_auth_oauth_providers_auth_container_id;
-- +goose StatementEnd

-- DROP CONSTRAINTS
-- +goose StatementBegin
ALTER TABLE auth_oauth_providers DROP CONSTRAINT IF EXISTS fk_auth_oauth_providers_auth_container_id;
-- +goose StatementEnd

-- DROP TABLE
-- +goose StatementBegin
DROP TABLE IF EXISTS auth_oauth_providers;
-- +goose StatementEnd
