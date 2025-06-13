-- +goose Up

-- CREATE TABLE
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS user_roles (
    user_role_id      SERIAL PRIMARY KEY,
    user_role_uuid    UUID UNIQUE NOT NULL,
    user_id           INTEGER NOT NULL,
    role_id           INTEGER NOT NULL,
    is_default        BOOLEAN DEFAULT FALSE,
    created_at        TIMESTAMPTZ DEFAULT now(),
    updated_at        TIMESTAMPTZ
);
-- +goose StatementEnd

-- ADD CONSTRAINTS
-- +goose StatementBegin
ALTER TABLE user_roles
    ADD CONSTRAINT fk_user_roles_user_id FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE;
ALTER TABLE user_roles
    ADD CONSTRAINT fk_user_roles_role_id FOREIGN KEY (role_id) REFERENCES roles(role_id) ON DELETE CASCADE;
ALTER TABLE user_roles
    ADD CONSTRAINT unique_user_roles_user_id_role_id UNIQUE (user_id, role_id);
-- +goose StatementEnd

-- ADD INDEXES
-- +goose StatementBegin
CREATE INDEX idx_user_roles_uuid ON user_roles(user_role_uuid);
CREATE INDEX idx_user_roles_user_id ON user_roles(user_id);
CREATE INDEX idx_user_roles_role_id ON user_roles(role_id);
-- +goose StatementEnd

-- +goose Down

-- DROP INDEXES
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_user_roles_uuid;
DROP INDEX IF EXISTS idx_user_roles_user_id;
DROP INDEX IF EXISTS idx_user_roles_role_id;
-- +goose StatementEnd

-- DROP CONSTRAINTS
-- +goose StatementBegin
ALTER TABLE user_roles DROP CONSTRAINT IF EXISTS fk_user_roles_user_id;
ALTER TABLE user_roles DROP CONSTRAINT IF EXISTS fk_user_roles_role_id;
ALTER TABLE user_roles DROP CONSTRAINT IF EXISTS unique_user_roles_user_id_role_id;
-- +goose StatementEnd

-- DROP TABLE
-- +goose StatementBegin
DROP TABLE IF EXISTS user_roles;
-- +goose StatementEnd
