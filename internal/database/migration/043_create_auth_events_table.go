package migration

import (
	"gorm.io/gorm"
)

// CreateAuthEventsTable creates the auth_events table which stores security
// events following the OWASP Logging Vocabulary standard. This replaces the
// former auth_logs table with a standards-compliant schema.
func CreateAuthEventsTable(db *gorm.DB) error {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS auth_events (
    auth_event_id     BIGSERIAL     PRIMARY KEY,
    auth_event_uuid   UUID          NOT NULL DEFAULT gen_random_uuid() UNIQUE,
    tenant_id         BIGINT        NOT NULL,

    -- WHO
    actor_user_id     BIGINT,
    target_user_id    BIGINT,
    ip_address        VARCHAR(45)   NOT NULL,
    user_agent        TEXT,

    -- WHAT  (OWASP Logging Vocabulary)
    category          VARCHAR(20)   NOT NULL,
    event_type        VARCHAR(60)   NOT NULL,
    severity          VARCHAR(10)   NOT NULL DEFAULT 'INFO',
    result            VARCHAR(10)   NOT NULL,
    description       TEXT,
    error_reason      VARCHAR(255),

    -- CONTEXT
    trace_id          VARCHAR(32),
    metadata          JSONB         DEFAULT '{}',

    -- WHEN  (immutable — no updated_at)
    created_at        TIMESTAMPTZ   NOT NULL DEFAULT NOW(),

    -- CONSTRAINTS
    CONSTRAINT chk_auth_events_category CHECK (category IN (
        'AUTHN', 'AUTHZ', 'SESSION', 'USER', 'SYSTEM'
    )),
    CONSTRAINT chk_auth_events_severity CHECK (severity IN (
        'INFO', 'WARN', 'CRITICAL'
    )),
    CONSTRAINT chk_auth_events_result CHECK (result IN (
        'success', 'failure'
    ))
);

-- ADD CONSTRAINTS (FOREIGN KEYS)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_auth_events_tenant_id'
    ) THEN
        ALTER TABLE auth_events
            ADD CONSTRAINT fk_auth_events_tenant_id FOREIGN KEY (tenant_id)
            REFERENCES tenants(tenant_id) ON DELETE CASCADE;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_auth_events_actor_user_id'
    ) THEN
        ALTER TABLE auth_events
            ADD CONSTRAINT fk_auth_events_actor_user_id FOREIGN KEY (actor_user_id)
            REFERENCES users(user_id) ON DELETE SET NULL;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_auth_events_target_user_id'
    ) THEN
        ALTER TABLE auth_events
            ADD CONSTRAINT fk_auth_events_target_user_id FOREIGN KEY (target_user_id)
            REFERENCES users(user_id) ON DELETE SET NULL;
    END IF;
END$$;

-- PRIMARY QUERY PATTERN INDEXES
CREATE INDEX IF NOT EXISTS idx_auth_events_tenant_created ON auth_events (tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_auth_events_actor ON auth_events (actor_user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_auth_events_event_type ON auth_events (event_type, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_auth_events_category ON auth_events (category, created_at DESC);

-- COMPLIANCE-FOCUSED PARTIAL INDEXES
CREATE INDEX IF NOT EXISTS idx_auth_events_failures ON auth_events (result, created_at DESC)
    WHERE result = 'failure';
CREATE INDEX IF NOT EXISTS idx_auth_events_critical ON auth_events (severity, created_at DESC)
    WHERE severity IN ('WARN', 'CRITICAL');
`
	return db.Exec(sql).Error
}
