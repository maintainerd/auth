package migration

import (
	"gorm.io/gorm"
)

func CreateAPIKeysTable(db *gorm.DB) error {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS api_keys (
    api_key_id      BIGSERIAL PRIMARY KEY,
    api_key_uuid    UUID NOT NULL UNIQUE,
    tenant_id       BIGINT NOT NULL,
    name            VARCHAR(100) NOT NULL,
    description     TEXT,
    key_hash        TEXT NOT NULL UNIQUE,
    key_prefix      VARCHAR(20) NOT NULL,
    config          JSONB,
    expires_at      TIMESTAMPTZ,
    status          TEXT DEFAULT 'active' CHECK (status IN ('active', 'inactive')),
    created_by      BIGINT,
    updated_by      BIGINT,
    created_at      TIMESTAMPTZ DEFAULT now(),
    updated_at      TIMESTAMPTZ DEFAULT now(),
    deleted_at      TIMESTAMPTZ
);

-- ADD CONSTRAINTS (safe)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_api_keys_tenant_id'
    ) THEN
        ALTER TABLE api_keys
            ADD CONSTRAINT fk_api_keys_tenant_id FOREIGN KEY (tenant_id)
            REFERENCES tenants(tenant_id) ON DELETE CASCADE;
    END IF;
END$$;

-- ADD INDEXES
CREATE INDEX IF NOT EXISTS idx_api_keys_uuid ON api_keys (api_key_uuid);
CREATE INDEX IF NOT EXISTS idx_api_keys_tenant_id ON api_keys (tenant_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_api_keys_tenant_name ON api_keys (tenant_id, name) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_api_keys_key_hash ON api_keys (key_hash);
CREATE INDEX IF NOT EXISTS idx_api_keys_key_prefix ON api_keys (key_prefix);
CREATE INDEX IF NOT EXISTS idx_api_keys_status ON api_keys (status);
CREATE INDEX IF NOT EXISTS idx_api_keys_expires_at ON api_keys (expires_at);
CREATE INDEX IF NOT EXISTS idx_api_keys_created_at ON api_keys (created_at);
CREATE INDEX IF NOT EXISTS idx_api_keys_deleted_at ON api_keys (deleted_at) WHERE deleted_at IS NULL;
`

	return db.Exec(sql).Error
}
