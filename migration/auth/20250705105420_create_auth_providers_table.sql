-- +goose Up

-- CREATE TABLE
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS auth_providers (
    auth_provider_id    SERIAL PRIMARY KEY,
    auth_provider_uuid  UUID NOT NULL UNIQUE,
    provider_name       VARCHAR(100) NOT NULL, -- 'default', 'cognito', 'auth0', 'facebook'
    display_name        TEXT NOT NULL,
    provider_type       TEXT NOT NULL, -- 'default', 'identity', 'oauth'
    description         TEXT NOT NULL,
    logo_url            TEXT,
    documentation_url   TEXT,
    is_default          BOOLEAN DEFAULT FALSE,
    created_at          TIMESTAMPTZ DEFAULT now(),
    updated_at          TIMESTAMPTZ
);
-- +goose StatementEnd

-- ADD INDEXES
-- +goose StatementBegin
CREATE INDEX idx_auth_providers_provider_name ON auth_providers (provider_name);
CREATE INDEX idx_auth_providers_provider_type ON auth_providers (provider_type);
-- +goose StatementEnd

-- +goose Down

-- DROP INDEXES
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_auth_providers_provider_name;
DROP INDEX IF EXISTS idx_auth_providers_provider_type;
-- +goose StatementEnd

-- DROP TABLE
-- +goose StatementBegin
DROP TABLE IF EXISTS auth_providers;
-- +goose StatementEnd
