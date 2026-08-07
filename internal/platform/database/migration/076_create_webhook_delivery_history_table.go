package migration

import (
	"gorm.io/gorm"
)

// CreateWebhookDeliveryHistoryTable creates the durable delivery tracking table.
func CreateWebhookDeliveryHistoryTable(db *gorm.DB) error {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS webhook_delivery_history (
    delivery_history_id     BIGSERIAL PRIMARY KEY,
    delivery_history_uuid   UUID NOT NULL UNIQUE,
    webhook_endpoint_id     BIGINT NOT NULL,
    event_id                UUID NOT NULL,
    event_type              VARCHAR(100) NOT NULL,
    tenant_id               BIGINT NOT NULL,
    attempt_count           INTEGER NOT NULL DEFAULT 1,
    response_status         INTEGER,
    response_summary        TEXT,
    error_reason            TEXT,
    next_retry_time         TIMESTAMPTZ,
    final_status            VARCHAR(20) NOT NULL DEFAULT 'pending',
    is_replay               BOOLEAN NOT NULL DEFAULT false,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ADD CONSTRAINTS
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_webhook_delivery_history_webhook_endpoint_id'
    ) THEN
        ALTER TABLE webhook_delivery_history
            ADD CONSTRAINT fk_webhook_delivery_history_webhook_endpoint_id FOREIGN KEY (webhook_endpoint_id)
            REFERENCES webhook_endpoints(webhook_endpoint_id) ON DELETE CASCADE;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'chk_webhook_delivery_history_final_status'
    ) THEN
        ALTER TABLE webhook_delivery_history
            ADD CONSTRAINT chk_webhook_delivery_history_final_status CHECK (final_status IN ('pending', 'success', 'failed', 'dead_letter'));
    END IF;
END$$;

-- CREATE INDEXES
CREATE INDEX IF NOT EXISTS idx_webhook_delivery_history_uuid ON webhook_delivery_history (delivery_history_uuid);
CREATE INDEX IF NOT EXISTS idx_webhook_delivery_history_webhook_endpoint_id ON webhook_delivery_history (webhook_endpoint_id);
CREATE INDEX IF NOT EXISTS idx_webhook_delivery_history_event_id ON webhook_delivery_history (event_id);
CREATE INDEX IF NOT EXISTS idx_webhook_delivery_history_tenant_id ON webhook_delivery_history (tenant_id);
CREATE INDEX IF NOT EXISTS idx_webhook_delivery_history_next_retry ON webhook_delivery_history (next_retry_time) WHERE final_status = 'pending';
CREATE INDEX IF NOT EXISTS idx_webhook_delivery_history_created_at ON webhook_delivery_history (created_at);
`

	return db.Exec(sql).Error
}
