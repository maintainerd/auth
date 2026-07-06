package migration

import "gorm.io/gorm"

func CreateDataErasureRequestsTable(db *gorm.DB) error {
	return db.Exec(`
CREATE TABLE IF NOT EXISTS data_erasure_requests (
    data_erasure_request_id     BIGSERIAL    PRIMARY KEY,
    data_erasure_request_uuid   UUID         NOT NULL UNIQUE DEFAULT gen_random_uuid(),
    tenant_id                   BIGINT       NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
    user_id                     BIGINT       NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    requested_by_user_id        BIGINT       REFERENCES users(user_id) ON DELETE SET NULL,
    requested_by_admin_id       BIGINT       REFERENCES users(user_id) ON DELETE SET NULL,
    status                      VARCHAR(30)  NOT NULL DEFAULT 'pending',
    reason                      TEXT         NOT NULL DEFAULT '',
    rejection_reason            TEXT,
    legal_hold                  BOOLEAN      NOT NULL DEFAULT FALSE,
    legal_hold_reason           TEXT,
    scheduled_at                TIMESTAMPTZ  NOT NULL,
    started_at                  TIMESTAMPTZ,
    completed_at                TIMESTAMPTZ,
    created_at                  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at                  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT chk_data_erasure_requests_status CHECK (
        status IN ('pending', 'in_progress', 'completed', 'rejected', 'on_hold')
    )
);

CREATE INDEX IF NOT EXISTS idx_data_erasure_requests_tenant_status
    ON data_erasure_requests (tenant_id, status) WHERE status IN ('pending', 'in_progress');
CREATE INDEX IF NOT EXISTS idx_data_erasure_requests_user
    ON data_erasure_requests (user_id);
CREATE INDEX IF NOT EXISTS idx_data_erasure_requests_scheduled_at
    ON data_erasure_requests (scheduled_at) WHERE status = 'pending';
`).Error
}
