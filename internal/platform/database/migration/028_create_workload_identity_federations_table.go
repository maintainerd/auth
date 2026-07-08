package migration

import "gorm.io/gorm"

func CreateWorkloadIdentityFederationsTable(db *gorm.DB) error {
	return db.Exec(`
CREATE TABLE IF NOT EXISTS workload_identity_federations (
    workload_identity_federation_id     BIGSERIAL    PRIMARY KEY,
    workload_identity_federation_uuid   UUID         NOT NULL UNIQUE DEFAULT gen_random_uuid(),
    tenant_id                           BIGINT       NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
    client_id                           BIGINT       NOT NULL REFERENCES clients(client_id) ON DELETE CASCADE,
    name                                VARCHAR(100) NOT NULL,
    description                         TEXT,
    issuer_url                          VARCHAR(2048) NOT NULL,
    audience                            VARCHAR(512) NOT NULL,
    subject_claim                       VARCHAR(100) NOT NULL DEFAULT 'sub',
    subject_pattern                     VARCHAR(512) NOT NULL,
    allowed_scopes                      TEXT[]       NOT NULL DEFAULT '{}',
    attribute_mapping                   JSONB        NOT NULL DEFAULT '{}',
    is_active                           BOOLEAN      NOT NULL DEFAULT TRUE,
    created_by                          BIGINT,
    updated_by                          BIGINT,
    created_at                          TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at                          TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at                          TIMESTAMPTZ
);

-- Partial unique index: soft-delete-aware (inline UNIQUE constraint would reject
-- re-creating a WIF config with the same name after soft-deleting the old one).
CREATE UNIQUE INDEX IF NOT EXISTS uq_workload_identity_federations_tenant_name
    ON workload_identity_federations (tenant_id, name) WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_workload_identity_federations_tenant
    ON workload_identity_federations (tenant_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_workload_identity_federations_client
    ON workload_identity_federations (client_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_workload_identity_federations_issuer
    ON workload_identity_federations (issuer_url) WHERE deleted_at IS NULL;
`).Error
}
