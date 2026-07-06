package migration

import "gorm.io/gorm"

func CreatePolicyVersionHistoryTable(db *gorm.DB) error {
	return db.Exec(`
CREATE TABLE IF NOT EXISTS policy_version_history (
    policy_version_history_id   BIGSERIAL    PRIMARY KEY,
    policy_version_history_uuid UUID         NOT NULL UNIQUE DEFAULT gen_random_uuid(),
    tenant_id                   BIGINT       NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
    policy_id                   BIGINT       NOT NULL REFERENCES policies(policy_id) ON DELETE RESTRICT,
    version_number              INT          NOT NULL,
    name                        VARCHAR(255) NOT NULL,
    description                 TEXT,
    document                    JSONB        NOT NULL DEFAULT '{}',
    policy_version              VARCHAR(20)  NOT NULL,
    changed_by_user_id          BIGINT       REFERENCES users(user_id) ON DELETE SET NULL,
    changed_by_client_id        BIGINT       REFERENCES clients(client_id) ON DELETE SET NULL,
    change_reason               TEXT,
    snapshot_at                 TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT uq_policy_version_history_policy_version UNIQUE (policy_id, version_number)
);

-- Immutability trigger: policy version history rows must never be modified.
CREATE OR REPLACE FUNCTION prevent_policy_version_history_mutation() RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'policy_version_history rows are immutable';
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_policy_version_history_immutable ON policy_version_history;
CREATE TRIGGER trg_policy_version_history_immutable
    BEFORE UPDATE OR DELETE ON policy_version_history
    FOR EACH ROW EXECUTE FUNCTION prevent_policy_version_history_mutation();

CREATE INDEX IF NOT EXISTS idx_policy_version_history_policy_id
    ON policy_version_history (policy_id, version_number DESC);
CREATE INDEX IF NOT EXISTS idx_policy_version_history_tenant_created
    ON policy_version_history (tenant_id, snapshot_at DESC);
CREATE INDEX IF NOT EXISTS idx_policy_version_history_changed_by
    ON policy_version_history (changed_by_user_id, snapshot_at DESC)
    WHERE changed_by_user_id IS NOT NULL;
`).Error
}
