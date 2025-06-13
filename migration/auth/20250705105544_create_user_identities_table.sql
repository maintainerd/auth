-- +goose Up

-- CREATE TABLE
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS user_identities (
    user_identity_id    SERIAL PRIMARY KEY,
    user_identity_uuid  UUID NOT NULL UNIQUE,
    user_id             INTEGER NOT NULL,
    provider_name       VARCHAR(100) NOT NULL, -- e.g., 'google', 'cognito', 'microsoft'
    provider_user_id    VARCHAR(255) NOT NULL, -- external user ID (e.g., sub in OIDC)
    email               VARCHAR(255),
    raw_profile         JSONB,
    created_at          TIMESTAMPTZ DEFAULT now()
);
-- +goose StatementEnd

-- ADD CONSTRAINTS
-- +goose StatementBegin
ALTER TABLE user_identities
    ADD CONSTRAINT fk_user_identities_user
    FOREIGN KEY (user_id) REFERENCES users(user_id);
-- +goose StatementEnd

-- ADD INDEXES
-- +goose StatementBegin
CREATE UNIQUE INDEX idx_user_identities_provider_combination
    ON user_identities (provider_name, provider_user_id);
-- +goose StatementEnd

-- +goose Down

-- DROP INDEXES
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_user_identities_provider_combination;
-- +goose StatementEnd

-- DROP CONSTRAINTS
-- +goose StatementBegin
ALTER TABLE user_identities
    DROP CONSTRAINT IF EXISTS fk_user_identities_user;
-- +goose StatementEnd

-- DROP TABLE
-- +goose StatementBegin
DROP TABLE IF EXISTS user_identities;
-- +goose StatementEnd
