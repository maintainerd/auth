-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS login_attempts (
    login_attempt_id    SERIAL PRIMARY KEY,
    login_attempt_uuid  UUID NOT NULL UNIQUE,
    user_id             INTEGER, -- nullable: user might not exist or match
    email               VARCHAR(255), -- the email/username used in the attempt
    ip_address          VARCHAR(100),
    user_agent          TEXT,
    is_success          BOOLEAN DEFAULT FALSE,
    attempted_at        TIMESTAMPTZ DEFAULT now()
);
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE login_attempts
    ADD CONSTRAINT fk_login_attempts_user
    FOREIGN KEY (user_id) REFERENCES users(user_id);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_login_attempts_user_id
    ON login_attempts (user_id);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_login_attempts_email
    ON login_attempts (email);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_login_attempts_email;
-- +goose StatementEnd

-- +goose StatementBegin
DROP INDEX IF EXISTS idx_login_attempts_user_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE login_attempts
    DROP CONSTRAINT IF EXISTS fk_login_attempts_user;
-- +goose StatementEnd

-- +goose StatementBegin
DROP TABLE IF EXISTS login_attempts;
-- +goose StatementEnd
