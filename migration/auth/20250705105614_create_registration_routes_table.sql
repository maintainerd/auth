-- +goose Up

-- CREATE TABLE registration_routes
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS registration_routes (
    registration_route_id   SERIAL PRIMARY KEY,
    registration_route_uuid UUID NOT NULL UNIQUE,
    name                    VARCHAR(100) NOT NULL,
    identifier              VARCHAR(255) NOT NULL UNIQUE,
    description             TEXT NOT NULL,
    auth_container_id       INTEGER NOT NULL,
    is_active               BOOLEAN DEFAULT TRUE,
    created_at              TIMESTAMPTZ DEFAULT now(),
    updated_at              TIMESTAMPTZ DEFAULT now()
);
-- +goose StatementEnd

-- ADD CONSTRAINTS
-- +goose StatementBegin
ALTER TABLE registration_routes
    ADD CONSTRAINT fk_registration_routes_auth_container_id FOREIGN KEY (auth_container_id) REFERENCES auth_containers(auth_container_id) ON DELETE CASCADE;
-- +goose StatementEnd

-- ADD INDEXES
-- +goose StatementBegin
CREATE INDEX idx_registration_routes_registration_route_uuid ON registration_routes (registration_route_uuid);
CREATE INDEX idx_registration_routes_identifier ON registration_routes (identifier);
CREATE INDEX idx_registration_routes_is_active ON registration_routes (is_active);
-- +goose StatementEnd

-- +goose Down

-- DROP INDEXES
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_registration_routes_registration_route_uuid;
DROP INDEX IF EXISTS idx_registration_routes_identifier;
DROP INDEX IF EXISTS idx_registration_routes_is_active;
-- +goose StatementEnd

-- DROP CONSTRAINTS
-- +goose StatementBegin
ALTER TABLE registration_routes DROP CONSTRAINT IF EXISTS fk_registration_routes_auth_container_id;
-- +goose StatementEnd

-- DROP TABLE
-- +goose StatementBegin
DROP TABLE IF EXISTS registration_routes;
-- +goose StatementEnd
