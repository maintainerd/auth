-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS auth_oauth_providers (
    auth_oauth_provider_id      SERIAL PRIMARY KEY,
    auth_oauth_provider_uuid    UUID NOT NULL UNIQUE,
    provider_name               VARCHAR(100) NOT NULL, -- e.g., 'google', 'facebook', 'github'
    client_id                   TEXT,
    client_secret               TEXT,
    redirect_uri                TEXT,
    metadata                    JSONB, -- OIDC discovery URL
    is_active                   BOOLEAN DEFAULT TRUE,
    created_at                  TIMESTAMPTZ DEFAULT now(),
    updated_at                  TIMESTAMPTZ
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_auth_oauth_providers_provider_name
    ON auth_oauth_providers (provider_name);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_auth_oauth_providers_provider_name;
-- +goose StatementEnd

-- +goose StatementBegin
DROP TABLE IF EXISTS auth_oauth_providers;
-- +goose StatementEnd
