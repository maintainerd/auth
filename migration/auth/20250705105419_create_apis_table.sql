-- +goose Up

-- CREATE TABLE
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS apis (
    api_id              SERIAL PRIMARY KEY,
    api_uuid            UUID NOT NULL UNIQUE,
    api_name            VARCHAR(100) NOT NULL, -- 'default', 'your-custom-api'
    display_name        TEXT NOT NULL,
    api_type            TEXT NOT NULL, -- 'default', 'custom'
    description         TEXT NOT NULL,
    identifier          TEXT NOT NULL, -- 'http://api.example.com'
    is_active           BOOLEAN DEFAULT FALSE,
    is_default          BOOLEAN DEFAULT FALSE,
    service_id          INTEGER NOT NULL,
    auth_container_id   INTEGER NOT NULL,
    created_at          TIMESTAMPTZ DEFAULT now(),
    updated_at          TIMESTAMPTZ DEFAULT now()
);
-- +goose StatementEnd

-- ADD CONSTRAINTS
-- +goose StatementBegin
ALTER TABLE apis
    ADD CONSTRAINT fk_apis_service_id FOREIGN KEY (service_id) REFERENCES services(service_id) ON DELETE CASCADE;
ALTER TABLE apis
    ADD CONSTRAINT fk_apis_auth_container_id FOREIGN KEY (auth_container_id) REFERENCES auth_containers(auth_container_id) ON DELETE CASCADE;
-- +goose StatementEnd

-- ADD INDEXES
-- +goose StatementBegin
CREATE INDEX idx_apis_api_name ON apis (api_name);
CREATE INDEX idx_apis_api_type ON apis (api_type);
CREATE INDEX idx_apis_identifier ON apis (identifier);
CREATE INDEX idx_apis_service_id ON apis (service_id);
CREATE INDEX idx_apis_auth_container_id ON apis (auth_container_id);
-- +goose StatementEnd

-- +goose Down

-- DROP INDEXES
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_apis_api_name;
DROP INDEX IF EXISTS idx_apis_api_type;
DROP INDEX IF EXISTS idx_apis_identifier;
DROP INDEX IF EXISTS idx_apis_service_id;
DROP INDEX IF EXISTS idx_apis_auth_container_id;
-- +goose StatementEnd

-- DROP CONSTRAINTS
-- +goose StatementBegin
ALTER TABLE apis DROP CONSTRAINT IF EXISTS fk_apis_service_id;
ALTER TABLE apis DROP CONSTRAINT IF EXISTS fk_apis_auth_container_id;
-- +goose StatementEnd

-- DROP TABLE
-- +goose StatementBegin
DROP TABLE IF EXISTS apis;
-- +goose StatementEnd
