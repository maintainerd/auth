-- +goose Up

-- CREATE TABLE
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS organizations (
    organization_id       SERIAL PRIMARY KEY,
    organization_uuid     UUID NOT NULL UNIQUE,
    name                  VARCHAR(255) NOT NULL,
    description           TEXT,
    email                 VARCHAR(255),
    phone_number          VARCHAR(50),
    website_url           TEXT,
    logo_url              TEXT,
    external_reference_id VARCHAR(255), -- Optional for external integrations
    is_default            BOOLEAN DEFAULT FALSE,
    is_active             BOOLEAN DEFAULT TRUE,
    created_at            TIMESTAMPTZ DEFAULT now(),
    updated_at            TIMESTAMPTZ DEFAULT now()
);
-- +goose StatementEnd

-- ADD INDEXES
-- +goose StatementBegin
CREATE INDEX idx_organizations_organization_uuid ON organizations (organization_uuid);
CREATE INDEX idx_organizations_name ON organizations (name);
CREATE INDEX idx_organizations_email ON organizations (email);
CREATE INDEX idx_organizations_phone_number ON organizations (phone_number);
CREATE INDEX idx_organizations_is_active ON organizations (is_active);
CREATE INDEX idx_organizations_is_default ON organizations (is_default);
CREATE INDEX idx_organizations_external_reference_id ON organizations (external_reference_id);
-- +goose StatementEnd

-- +goose Down

-- DROP INDEXES
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_organizations_organization_uuid;
DROP INDEX IF EXISTS idx_organizations_name;
DROP INDEX IF EXISTS idx_organizations_email;
DROP INDEX IF EXISTS idx_organizations_phone_number;
DROP INDEX IF EXISTS idx_organizations_is_active;
DROP INDEX IF EXISTS idx_organizations_is_default;
DROP INDEX IF EXISTS idx_organizations_external_reference_id;
-- +goose StatementEnd

-- DROP TABLE
-- +goose StatementBegin
DROP TABLE IF EXISTS organizations;
-- +goose StatementEnd
