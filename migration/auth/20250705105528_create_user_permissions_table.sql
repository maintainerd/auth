-- +goose Up

-- CREATE TABLE
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS user_permissions (
    user_permission_id      SERIAL PRIMARY KEY,
    user_permission_uuid    UUID UNIQUE NOT NULL,
    user_id                 INTEGER NOT NULL,
    permission_id           INTEGER NOT NULL,
    is_default              BOOLEAN DEFAULT FALSE,
    created_at              TIMESTAMPTZ DEFAULT now(),
    updated_at              TIMESTAMPTZ
);
-- +goose StatementEnd

-- ADD CONSTRAINTS
-- +goose StatementBegin
ALTER TABLE user_permissions
    ADD CONSTRAINT fk_user_permissions_user_id FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE;
ALTER TABLE user_permissions
    ADD CONSTRAINT fk_user_permissions_permission_id FOREIGN KEY (permission_id) REFERENCES permissions(permission_id) ON DELETE CASCADE;
ALTER TABLE user_permissions
    ADD CONSTRAINT unique_user_permissions_user_id_permission_id UNIQUE (user_id, permission_id);
-- +goose StatementEnd

-- ADD INDEXES
-- +goose StatementBegin
CREATE INDEX idx_user_permissions_uuid ON user_permissions(user_permission_uuid);
CREATE INDEX idx_user_permissions_user_id ON user_permissions(user_id);
CREATE INDEX idx_user_permissions_permission_id ON user_permissions(permission_id);
-- +goose StatementEnd

-- +goose Down

-- DROP INDEXES
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_user_permissions_uuid;
DROP INDEX IF EXISTS idx_user_permissions_user_id;
DROP INDEX IF EXISTS idx_user_permissions_permission_id;
-- +goose StatementEnd

-- DROP CONSTRAINTS
-- +goose StatementBegin
ALTER TABLE user_permissions DROP CONSTRAINT IF EXISTS fk_user_permissions_user_id;
ALTER TABLE user_permissions DROP CONSTRAINT IF EXISTS fk_user_permissions_permission_id;
ALTER TABLE user_permissions DROP CONSTRAINT IF EXISTS unique_user_permissions_user_id_permission_id;
-- +goose StatementEnd

-- DROP TABLE
-- +goose StatementBegin
DROP TABLE IF EXISTS user_permissions;
-- +goose StatementEnd
