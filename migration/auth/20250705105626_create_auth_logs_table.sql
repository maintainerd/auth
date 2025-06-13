-- +goose Up

-- CREATE TABLE
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS auth_logs (
    auth_log_id         SERIAL PRIMARY KEY,
    auth_log_uuid       UUID NOT NULL UNIQUE,
    user_id             INTEGER NOT NULL,
    event_type          VARCHAR(100) NOT NULL, -- e.g., 'login', 'logout', 'token_refresh', 'password_reset'
    description         TEXT,
    ip_address          VARCHAR(100),
    user_agent          TEXT,
    metadata            JSONB,
    auth_container_id   INTEGER NOT NULL,
    created_at          TIMESTAMPTZ DEFAULT now(),
    updated_at          TIMESTAMPTZ DEFAULT now()
);
-- +goose StatementEnd

-- ADD CONSTRAINTS
-- +goose StatementBegin
ALTER TABLE auth_logs
    ADD CONSTRAINT fk_auth_logs_user_id FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE;

ALTER TABLE auth_logs
    ADD CONSTRAINT fk_auth_logs_auth_container_id FOREIGN KEY (auth_container_id) REFERENCES auth_containers(auth_container_id) ON DELETE CASCADE;
-- +goose StatementEnd

-- ADD INDEXES
-- +goose StatementBegin
CREATE INDEX idx_auth_logs_user_id ON auth_logs (user_id);
CREATE INDEX idx_auth_logs_event_type ON auth_logs (event_type);
CREATE INDEX idx_auth_logs_auth_container_id ON auth_logs (auth_container_id);
CREATE INDEX idx_auth_logs_auth_log_uuid ON auth_logs (auth_log_uuid);
-- +goose StatementEnd

-- +goose Down

-- DROP INDEXES
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_auth_logs_user_id;
DROP INDEX IF EXISTS idx_auth_logs_event_type;
DROP INDEX IF EXISTS idx_auth_logs_auth_container_id;
DROP INDEX IF EXISTS idx_auth_logs_auth_log_uuid;
-- +goose StatementEnd

-- DROP CONSTRAINTS
-- +goose StatementBegin
ALTER TABLE auth_logs DROP CONSTRAINT IF EXISTS fk_auth_logs_user_id;
ALTER TABLE auth_logs DROP CONSTRAINT IF EXISTS fk_auth_logs_auth_container_id;
-- +goose StatementEnd

-- DROP TABLE
-- +goose StatementBegin
DROP TABLE IF EXISTS auth_logs;
-- +goose StatementEnd
