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
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
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

    -- created_by / updated_by FKs on client_uris reference users, but users
    -- is created later (migration 024). They are attached via the deferred
    -- FK loop in 024_create_users_table.go instead of here.

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'chk_client_uris_type'
    ) THEN
        ALTER TABLE client_uris
            ADD CONSTRAINT chk_client_uris_type CHECK (type IN ('redirect_uri', 'origin_uri', 'logout_uri', 'login_uri', 'cors_origin_uri'));
    END IF;
END$$;

-- ADD INDEXES
-- client_uri_uuid is declared UNIQUE above, which already indexes it.
CREATE INDEX IF NOT EXISTS idx_client_uris_tenant_id ON client_uris (tenant_id);
CREATE INDEX IF NOT EXISTS idx_client_uris_client_id ON client_uris (client_id);
-- A redirect-URI allowlist with duplicates is a review hazard, and nothing
-- de-duplicated on write. Unique per (client, type, uri) among live rows.
CREATE UNIQUE INDEX IF NOT EXISTS uq_client_uris_client_type_uri ON client_uris (client_id, type, uri) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_client_uris_type ON client_uris (type);
CREATE INDEX IF NOT EXISTS idx_client_uris_client_id_type ON client_uris (client_id, type);
CREATE INDEX IF NOT EXISTS idx_client_uris_deleted_at ON client_uris (deleted_at) WHERE deleted_at IS NULL;
`
	return db.Exec(sql).Error
}
