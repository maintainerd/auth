package migration

import "gorm.io/gorm"

func CreateManagementAuditLogTable(db *gorm.DB) error {
	return db.Exec(`
CREATE TABLE IF NOT EXISTS management_audit_log (
    management_audit_log_id   BIGSERIAL    PRIMARY KEY,
    management_audit_log_uuid UUID         NOT NULL UNIQUE DEFAULT gen_random_uuid(),
    tenant_id                 BIGINT       NOT NULL,
    actor_user_id             BIGINT,
    actor_client_id           BIGINT,
    action                    VARCHAR(100) NOT NULL,
    resource_type             VARCHAR(100) NOT NULL,
    resource_id               VARCHAR(255) NOT NULL,
    resource_uuid             UUID,
    changes                   JSONB        NOT NULL DEFAULT '{}',
    ip_address                INET,
    user_agent                TEXT,
    trace_id                  VARCHAR(64),
    request_id                VARCHAR(255),
    outcome                   VARCHAR(20)  NOT NULL DEFAULT 'success',
    error_message             TEXT,
    created_at                TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT fk_management_audit_log_tenant FOREIGN KEY (tenant_id)
        REFERENCES tenants(tenant_id) ON DELETE CASCADE,
    CONSTRAINT fk_management_audit_log_actor_user FOREIGN KEY (actor_user_id)
        REFERENCES users(user_id) ON DELETE SET NULL,
    CONSTRAINT fk_management_audit_log_actor_client FOREIGN KEY (actor_client_id)
        REFERENCES clients(client_id) ON DELETE SET NULL,
    CONSTRAINT chk_management_audit_log_outcome CHECK (outcome IN ('success', 'failure', 'partial'))
);

CREATE OR REPLACE FUNCTION prevent_management_audit_log_mutation() RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'UPDATE' THEN
        RAISE EXCEPTION 'management_audit_log rows are immutable and cannot be updated';
    END IF;

    -- DELETE is permitted ONLY for sanctioned lifecycle operations (retention
    -- purge or full tenant deletion), signalled by a transaction-local GUC. This
    -- keeps the trail append-only in normal operation while allowing a tenant
    -- purge (GDPR/erasure) to cascade-delete the tenant's audit rows — without
    -- this exemption the ON DELETE CASCADE from tenants raises here and the whole
    -- purge transaction rolls back, so soft-deleted tenants can never be purged.
    IF TG_OP = 'DELETE'
       AND COALESCE(current_setting('maintainerd.allow_management_audit_log_delete', true), '') NOT IN ('retention', 'tenant_delete') THEN
        RAISE EXCEPTION 'management_audit_log rows are immutable and can only be deleted by retention or tenant deletion';
    END IF;

    RETURN OLD;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_management_audit_log_immutable ON management_audit_log;
CREATE TRIGGER trg_management_audit_log_immutable
    BEFORE UPDATE OR DELETE ON management_audit_log
    FOR EACH ROW EXECUTE FUNCTION prevent_management_audit_log_mutation();

CREATE INDEX IF NOT EXISTS idx_management_audit_log_tenant_created
    ON management_audit_log (tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_management_audit_log_actor_user
    ON management_audit_log (actor_user_id, created_at DESC) WHERE actor_user_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_management_audit_log_resource
    ON management_audit_log (resource_type, resource_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_management_audit_log_trace_id
    ON management_audit_log (trace_id) WHERE trace_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_management_audit_log_changes
    ON management_audit_log USING GIN (changes);
`).Error
}
