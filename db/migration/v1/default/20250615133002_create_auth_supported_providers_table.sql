-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS auth_supported_providers (
    auth_supported_provider_id      SERIAL PRIMARY KEY,
    auth_supported_provider_uuid    UUID NOT NULL UNIQUE,
    provider_name                   VARCHAR(100) NOT NULL, -- e.g., 'cognito', 'auth0', 'facebook'
    display_name                    TEXT NOT NULL,
    provider_type                   TEXT NOT NULL CHECK (provider_type IN ('identity_provider', 'oauth_social')),
    description                     TEXT,
    logo_url                        TEXT,
    documentation_url               TEXT,
    created_at                      TIMESTAMPTZ DEFAULT now(),
    updated_at                      TIMESTAMPTZ
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_auth_supported_providers_provider_name
    ON auth_supported_providers (provider_name);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_auth_supported_providers_provider_name;
-- +goose StatementEnd

-- +goose StatementBegin
DROP TABLE IF EXISTS auth_supported_providers;
-- +goose StatementEnd
