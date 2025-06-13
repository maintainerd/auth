-- +goose Up

-- CREATE TABLE
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS services (
    service_id      SERIAL PRIMARY KEY,
    service_uuid    UUID NOT NULL UNIQUE,
    service_name    VARCHAR(100) NOT NULL, -- 'default', 'auth', 'your-custom-service'
    display_name    TEXT NOT NULL,
    description     TEXT NOT NULL,
    service_type    TEXT NOT NULL, -- 'default', 'custom'
    version         VARCHAR(20) NOT NULL,
    config          JSONB,
    is_active       BOOLEAN DEFAULT FALSE,
    is_default      BOOLEAN DEFAULT FALSE,
    created_at      TIMESTAMPTZ DEFAULT now(),
    updated_at      TIMESTAMPTZ DEFAULT now()
);
-- +goose StatementEnd

-- ADD INDEXES
-- +goose StatementBegin
CREATE INDEX idx_services_service_name ON services (service_name);
CREATE INDEX idx_services_display_name ON services (display_name);
CREATE INDEX idx_services_service_type ON services (service_type);
CREATE INDEX idx_services_service_uuid ON services (service_uuid);
-- +goose StatementEnd

-- +goose Down

-- DROP INDEXES
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_services_service_name;
DROP INDEX IF EXISTS idx_services_display_name;
DROP INDEX IF EXISTS idx_services_service_type;
DROP INDEX IF EXISTS idx_services_service_uuid;
-- +goose StatementEnd

-- DROP TABLE
-- +goose StatementBegin
DROP TABLE IF EXISTS services;
-- +goose StatementEnd
