package migration

import (
	"gorm.io/gorm"
)

// CreateWebhookEndpointsTable creates the webhook_endpoints table for
// tenant-level outbound event notification subscriptions.
func CreateWebhookEndpointsTable(db *gorm.DB) error {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS webhook_endpoints (
    webhook_endpoint_id     BIGSERIAL PRIMARY KEY,
    webhook_endpoint_uuid   UUID NOT NULL UNIQUE,
    tenant_id               BIGINT NOT NULL,
    url                     VARCHAR(2048) NOT NULL,
    secret_encrypted        TEXT,
    subscribe_all           BOOLEAN NOT NULL DEFAULT false,
    max_retries             INTEGER NOT NULL DEFAULT 3,
    timeout_seconds         INTEGER NOT NULL DEFAULT 30,
    status                  VARCHAR(20) NOT NULL DEFAULT 'active',
    consecutive_failures    INTEGER NOT NULL DEFAULT 0,
    description             TEXT,
    metadata                JSONB DEFAULT '{}',
    last_triggered_at       TIMESTAMPTZ,
    created_by              BIGINT,
    updated_by              BIGINT,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at              TIMESTAMPTZ
);

-- ADD CONSTRAINTS
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_webhook_endpoints_tenant_id'
    ) THEN
        ALTER TABLE webhook_endpoints
            ADD CONSTRAINT fk_webhook_endpoints_tenant_id FOREIGN KEY (tenant_id)
            REFERENCES tenants(tenant_id) ON DELETE CASCADE;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'chk_webhook_endpoints_status'
    ) THEN
        ALTER TABLE webhook_endpoints
            ADD CONSTRAINT chk_webhook_endpoints_status CHECK (status IN ('active', 'inactive', 'quarantined'));
    END IF;

    -- Audit FKs to users. webhook_endpoints is created after the users table
    -- (migration 024), so unlike the pre-users tables it attaches these itself.
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_webhook_endpoints_created_by'
    ) THEN
        ALTER TABLE webhook_endpoints
            ADD CONSTRAINT fk_webhook_endpoints_created_by FOREIGN KEY (created_by)
            REFERENCES users(user_id) ON DELETE SET NULL;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_webhook_endpoints_updated_by'
    ) THEN
        ALTER TABLE webhook_endpoints
            ADD CONSTRAINT fk_webhook_endpoints_updated_by FOREIGN KEY (updated_by)
            REFERENCES users(user_id) ON DELETE SET NULL;
    END IF;
END$$;

-- CREATE INDEXES
CREATE INDEX IF NOT EXISTS idx_webhook_endpoints_uuid ON webhook_endpoints (webhook_endpoint_uuid);
CREATE INDEX IF NOT EXISTS idx_webhook_endpoints_tenant_id ON webhook_endpoints (tenant_id);
CREATE INDEX IF NOT EXISTS idx_webhook_endpoints_status ON webhook_endpoints (status);
CREATE INDEX IF NOT EXISTS idx_webhook_endpoints_created_at ON webhook_endpoints (created_at);
CREATE INDEX IF NOT EXISTS idx_webhook_endpoints_deleted_at ON webhook_endpoints (deleted_at) WHERE deleted_at IS NULL;
`

	return db.Exec(sql).Error
}
