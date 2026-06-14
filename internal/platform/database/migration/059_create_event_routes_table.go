package migration

import (
	"gorm.io/gorm"
)

// CreateEventRoutesTable creates the broker (RabbitMQ) routing table per tenant.
func CreateEventRoutesTable(db *gorm.DB) error {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS event_routes (
    event_route_id      BIGSERIAL PRIMARY KEY,
    event_route_uuid    UUID NOT NULL UNIQUE,
    tenant_id           BIGINT NOT NULL,
    event_type_id       BIGINT NOT NULL,
    channel             VARCHAR(50) NOT NULL DEFAULT 'rabbitmq',
    enabled             BOOLEAN NOT NULL DEFAULT true,
    created_at          TIMESTAMPTZ DEFAULT now(),
    updated_at          TIMESTAMPTZ DEFAULT now(),
    UNIQUE (tenant_id, event_type_id, channel)
);

-- ADD CONSTRAINTS
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_event_routes_tenant_id'
    ) THEN
        ALTER TABLE event_routes
            ADD CONSTRAINT fk_event_routes_tenant_id FOREIGN KEY (tenant_id)
            REFERENCES tenants(tenant_id) ON DELETE CASCADE;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_event_routes_event_type_id'
    ) THEN
        ALTER TABLE event_routes
            ADD CONSTRAINT fk_event_routes_event_type_id FOREIGN KEY (event_type_id)
            REFERENCES event_types(event_type_id) ON DELETE CASCADE;
    END IF;
END$$;

-- CREATE INDEXES
CREATE INDEX IF NOT EXISTS idx_event_routes_uuid ON event_routes (event_route_uuid);
CREATE INDEX IF NOT EXISTS idx_event_routes_tenant_id ON event_routes (tenant_id);
CREATE INDEX IF NOT EXISTS idx_event_routes_event_type_id ON event_routes (event_type_id);
CREATE INDEX IF NOT EXISTS idx_event_routes_enabled ON event_routes (enabled);
`

	return db.Exec(sql).Error
}
