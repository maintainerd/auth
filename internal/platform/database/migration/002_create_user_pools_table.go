package migration

import "gorm.io/gorm"

// CreateUserPoolTable creates the user_pools table.
// A user pool is the isolation boundary for users, roles, clients, and
// settings within a single tenant deployment. Each tenant can have one or
// more user pools, each representing an independent application's user
// namespace — analogous to AWS Cognito User Pools.
func CreateUserPoolTable(db *gorm.DB) error {
	sql := `
CREATE TABLE IF NOT EXISTS user_pools (
    user_pool_id   BIGSERIAL PRIMARY KEY,
    user_pool_uuid UUID        NOT NULL UNIQUE DEFAULT gen_random_uuid(),
    tenant_id      BIGINT      NOT NULL,
    name           VARCHAR(255) NOT NULL,
    display_name   VARCHAR(255),
    identifier     VARCHAR(255) NOT NULL,
    is_default     BOOLEAN     NOT NULL DEFAULT FALSE,
    is_system      BOOLEAN     NOT NULL DEFAULT FALSE,
    status         VARCHAR(16) NOT NULL DEFAULT 'active',
    metadata       JSONB       NOT NULL DEFAULT '{}',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at     TIMESTAMPTZ,

    CONSTRAINT fk_user_pools_tenant
        FOREIGN KEY (tenant_id) REFERENCES tenants (tenant_id)
        ON DELETE RESTRICT,

    CONSTRAINT uq_user_pools_tenant_identifier
        UNIQUE (tenant_id, identifier)
);

CREATE INDEX IF NOT EXISTS idx_user_pools_tenant_id ON user_pools (tenant_id);
CREATE INDEX IF NOT EXISTS idx_user_pools_is_default ON user_pools (tenant_id, is_default)
    WHERE is_default = TRUE;
CREATE INDEX IF NOT EXISTS idx_user_pools_deleted_at ON user_pools (deleted_at)
    WHERE deleted_at IS NULL;
`
	return db.Exec(sql).Error
}
