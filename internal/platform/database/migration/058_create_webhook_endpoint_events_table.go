package migration

import (
	"gorm.io/gorm"
)

// CreateWebhookEndpointEventsTable creates the M:N junction for
// webhook endpoint ↔ event type subscriptions.
func CreateWebhookEndpointEventsTable(db *gorm.DB) error {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS webhook_endpoint_events (
    webhook_endpoint_event_id   BIGSERIAL PRIMARY KEY,
    webhook_endpoint_id         BIGINT NOT NULL,
    event_type_id               BIGINT NOT NULL,
    created_at                  TIMESTAMPTZ DEFAULT now(),
    UNIQUE (webhook_endpoint_id, event_type_id)
);

-- ADD CONSTRAINTS
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_wee_webhook_endpoint_id'
    ) THEN
        ALTER TABLE webhook_endpoint_events
            ADD CONSTRAINT fk_wee_webhook_endpoint_id FOREIGN KEY (webhook_endpoint_id)
            REFERENCES webhook_endpoints(webhook_endpoint_id) ON DELETE CASCADE;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_wee_event_type_id'
    ) THEN
        ALTER TABLE webhook_endpoint_events
            ADD CONSTRAINT fk_wee_event_type_id FOREIGN KEY (event_type_id)
            REFERENCES event_types(event_type_id) ON DELETE CASCADE;
    END IF;
END$$;

-- CREATE INDEXES
CREATE INDEX IF NOT EXISTS idx_wee_webhook_endpoint_id ON webhook_endpoint_events (webhook_endpoint_id);
CREATE INDEX IF NOT EXISTS idx_wee_event_type_id ON webhook_endpoint_events (event_type_id);
`

	return db.Exec(sql).Error
}
