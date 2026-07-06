package migration

import (
	"gorm.io/gorm"
)

func CreateClientURIsTable(db *gorm.DB) error {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS client_uris (
    client_uri_id   BIGSERIAL PRIMARY KEY,
    client_uri_uuid UUID NOT NULL UNIQUE,
    tenant_id       BIGINT NOT NULL,
    client_id       BIGINT NOT NULL,
    uri             VARCHAR(2048) NOT NULL,
    type            VARCHAR(20) NOT NULL DEFAULT 'redirect_uri',
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
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_client_uris_tenant_id'
    ) THEN
        ALTER TABLE client_uris
            ADD CONSTRAINT fk_client_uris_tenant_id FOREIGN KEY (tenant_id)
            REFERENCES tenants(tenant_id) ON DELETE CASCADE;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_client_uris_client_id'
    ) THEN
        ALTER TABLE client_uris
            ADD CONSTRAINT fk_client_uris_client_id FOREIGN KEY (client_id)
            REFERENCES clients(client_id) ON DELETE CASCADE;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_client_uris_created_by'
    ) THEN
        ALTER TABLE client_uris ADD CONSTRAINT fk_client_uris_created_by
            FOREIGN KEY (created_by) REFERENCES users(user_id) ON DELETE SET NULL;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_client_uris_updated_by'
    ) THEN
        ALTER TABLE client_uris ADD CONSTRAINT fk_client_uris_updated_by
            FOREIGN KEY (updated_by) REFERENCES users(user_id) ON DELETE SET NULL;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'chk_client_uris_type'
    ) THEN
        ALTER TABLE client_uris
            ADD CONSTRAINT chk_client_uris_type CHECK (type IN ('redirect_uri', 'origin_uri', 'logout_uri', 'login_uri', 'cors-origin_uri'));
    END IF;
END$$;

-- ADD INDEXES
CREATE INDEX IF NOT EXISTS idx_client_uris_uuid ON client_uris (client_uri_uuid);
CREATE INDEX IF NOT EXISTS idx_client_uris_tenant_id ON client_uris (tenant_id);
CREATE INDEX IF NOT EXISTS idx_client_uris_client_id ON client_uris (client_id);
CREATE INDEX IF NOT EXISTS idx_client_uris_uri ON client_uris (uri);
CREATE INDEX IF NOT EXISTS idx_client_uris_type ON client_uris (type);
CREATE INDEX IF NOT EXISTS idx_client_uris_client_id_type ON client_uris (client_id, type);
CREATE INDEX IF NOT EXISTS idx_client_uris_deleted_at ON client_uris (deleted_at) WHERE deleted_at IS NULL;
`
	return db.Exec(sql).Error
}
