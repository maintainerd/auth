-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS auth_audit_logs (
    auth_audit_log_id       SERIAL PRIMARY KEY,
    auth_audit_log_uuid     UUID NOT NULL UNIQUE,
    user_id                 INTEGER,
    event_type              VARCHAR(100) NOT NULL, -- e.g., 'login', 'logout', 'token_refresh', 'password_reset'
    description             TEXT,
    ip_address              VARCHAR(100),
    user_agent              TEXT,
    metadata                JSONB,
    created_at              TIMESTAMPTZ DEFAULT now()
);
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE auth_audit_logs
    ADD CONSTRAINT fk_auth_audit_logs_user
    FOREIGN KEY (user_id) REFERENCES users(user_id);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_auth_audit_logs_user_id
    ON auth_audit_logs (user_id);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_auth_audit_logs_event_type
    ON auth_audit_logs (event_type);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_auth_audit_logs_event_type;
-- +goose StatementEnd

-- +goose StatementBegin
DROP INDEX IF EXISTS idx_auth_audit_logs_user_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE auth_audit_logs
    DROP CONSTRAINT IF EXISTS fk_auth_audit_logs_user;
-- +goose StatementEnd

-- +goose StatementBegin
DROP TABLE IF EXISTS auth_audit_logs;
-- +goose StatementEnd
