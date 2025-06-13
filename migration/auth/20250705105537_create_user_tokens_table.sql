-- +goose Up

-- CREATE TABLE
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS user_tokens (
    token_id        SERIAL PRIMARY KEY,
    token_uuid      UUID NOT NULL UNIQUE,
    user_id         INTEGER NOT NULL,
    token_type      VARCHAR(50) NOT NULL, -- e.g. "refresh", "api", "reset_password"
    token           TEXT NOT NULL,        -- hashed token string
    user_agent      TEXT,
    ip_address      VARCHAR(50),
    is_revoked      BOOLEAN DEFAULT FALSE,
    expires_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ DEFAULT now(),
    updated_at      TIMESTAMPTZ
);
-- +goose StatementEnd

-- ADD CONSTRAINTS
-- +goose StatementBegin
ALTER TABLE user_tokens
    ADD CONSTRAINT fk_user_tokens_user
    FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE;
-- +goose StatementEnd

-- ADD INDEXES
-- +goose StatementBegin
CREATE INDEX idx_user_tokens_user_id ON user_tokens(user_id);
CREATE INDEX idx_user_tokens_token_uuid ON user_tokens(token_uuid);
CREATE INDEX idx_user_tokens_token_type ON user_tokens(token_type);
-- +goose StatementEnd

-- +goose Down

-- DROP INDEXES
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_user_tokens_token_type;
DROP INDEX IF EXISTS idx_user_tokens_token_uuid;
DROP INDEX IF EXISTS idx_user_tokens_user_id;
-- +goose StatementEnd

-- DROP CONSTRAINTS
-- +goose StatementBegin
ALTER TABLE user_tokens DROP CONSTRAINT IF EXISTS fk_user_tokens_user;
-- +goose StatementEnd

-- DROP TABLE
-- +goose StatementBegin
DROP TABLE IF EXISTS user_tokens;
-- +goose StatementEnd
