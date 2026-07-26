package migration

import (
	"gorm.io/gorm"
)

// CreateIntegrationEventOutboxTable creates the durable transactional outbox
// for integration events.
func CreateIntegrationEventOutboxTable(db *gorm.DB) error {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS integration_event_outbox (
    outbox_id           BIGSERIAL PRIMARY KEY,
    outbox_uuid         UUID NOT NULL UNIQUE,
    event_id            UUID NOT NULL,
    event_type          VARCHAR(100) NOT NULL,
    event_version       INTEGER NOT NULL DEFAULT 1,
    tenant_id           BIGINT NOT NULL,
    actor_user_id       BIGINT,
    subject_uuid        UUID,
    subject_type        VARCHAR(50),
    changed_fields      JSONB NOT NULL DEFAULT '[]',
    payload             JSONB NOT NULL DEFAULT '{}',
    occurred_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    trace_id            VARCHAR(255),
    request_id          VARCHAR(255),
    is_published        BOOLEAN NOT NULL DEFAULT false,
    published_at        TIMESTAMPTZ,
    -- Per-arm delivery state. The relay hands each event to two independent arms
    -- (webhook fan-out, broker publish); tracking them separately means a failure
    -- in one arm never re-runs the other on re-claim. is_published flips true only
    -- once both arms are done. NULL = that arm not yet completed.
    webhook_delivered_at TIMESTAMPTZ,
    broker_published_at  TIMESTAMPTZ,
    claimed_at          TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ADD CONSTRAINTS
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_integration_event_outbox_tenant_id'
    ) THEN
        ALTER TABLE integration_event_outbox
            ADD CONSTRAINT fk_integration_event_outbox_tenant_id FOREIGN KEY (tenant_id)
            REFERENCES tenants(tenant_id) ON DELETE CASCADE;
    END IF;

    -- event_id is the producer-side idempotency / consumer dedup key; enforce uniqueness.
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'uq_outbox_event_id'
    ) THEN
        ALTER TABLE integration_event_outbox
            ADD CONSTRAINT uq_outbox_event_id UNIQUE (event_id);
    END IF;
END$$;

-- CREATE INDEXES
CREATE INDEX IF NOT EXISTS idx_outbox_uuid ON integration_event_outbox (outbox_uuid);
CREATE INDEX IF NOT EXISTS idx_outbox_tenant_id ON integration_event_outbox (tenant_id);
CREATE INDEX IF NOT EXISTS idx_outbox_event_type ON integration_event_outbox (event_type);
CREATE INDEX IF NOT EXISTS idx_outbox_is_published ON integration_event_outbox (is_published);
CREATE INDEX IF NOT EXISTS idx_outbox_unpublished ON integration_event_outbox (is_published, created_at) WHERE NOT is_published;
-- supports the relay's FOR UPDATE SKIP LOCKED claim (unpublished + claim-expiry ordering)
CREATE INDEX IF NOT EXISTS idx_outbox_claim ON integration_event_outbox (claimed_at, created_at) WHERE NOT is_published;
CREATE INDEX IF NOT EXISTS idx_outbox_tenant_unpublished ON integration_event_outbox (tenant_id, is_published) WHERE NOT is_published;
CREATE INDEX IF NOT EXISTS idx_outbox_created_at ON integration_event_outbox (created_at);
CREATE INDEX IF NOT EXISTS idx_outbox_subject_uuid ON integration_event_outbox (subject_uuid);
`

	return db.Exec(sql).Error
}
