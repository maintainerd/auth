-- +goose Up

-- CREATE TABLE registration_route_roles
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS registration_route_roles (
    registration_route_role_id SERIAL PRIMARY KEY,
    registration_route_id      INTEGER NOT NULL,
    role_id                    INTEGER NOT NULL,
    created_at                 TIMESTAMPTZ DEFAULT now()
);
-- +goose StatementEnd

-- ADD CONSTRAINTS
-- +goose StatementBegin
ALTER TABLE registration_route_roles
    ADD CONSTRAINT fk_registration_route_roles_route_id FOREIGN KEY (registration_route_id) REFERENCES registration_routes(registration_route_id) ON DELETE CASCADE;
ALTER TABLE registration_route_roles
    ADD CONSTRAINT fk_registration_route_roles_role_id FOREIGN KEY (role_id) REFERENCES roles(role_id) ON DELETE CASCADE;
-- +goose StatementEnd

-- ADD INDEXES
-- +goose StatementBegin
CREATE INDEX idx_registration_route_roles_route_id ON registration_route_roles (registration_route_id);
CREATE INDEX idx_registration_route_roles_role_id ON registration_route_roles (role_id);
-- +goose StatementEnd

-- +goose Down

-- DROP INDEXES
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_registration_route_roles_route_id;
DROP INDEX IF EXISTS idx_registration_route_roles_role_id;
-- +goose StatementEnd

-- DROP CONSTRAINTS
-- +goose StatementBegin
ALTER TABLE registration_route_roles DROP CONSTRAINT IF EXISTS fk_registration_route_roles_route_id;
ALTER TABLE registration_route_roles DROP CONSTRAINT IF EXISTS fk_registration_route_roles_role_id;
-- +goose StatementEnd

-- DROP TABLE
-- +goose StatementBegin
DROP TABLE IF EXISTS registration_route_roles;
-- +goose StatementEnd
