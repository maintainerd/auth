package migration

import "gorm.io/gorm"

func CreateIdentityProviderAllowedAudiencesTable(db *gorm.DB) error {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS identity_provider_allowed_audiences (
    identity_provider_allowed_audience_id BIGSERIAL PRIMARY KEY,
    identity_provider_allowed_audience_uuid UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),
    tenant_id            BIGINT NOT NULL,
    identity_provider_id BIGINT NOT NULL,
    audience             VARCHAR(255) NOT NULL,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at           TIMESTAMPTZ
);

-- ADD CONSTRAINTS (safe)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_idp_allowed_audiences_tenant'
    ) THEN
        ALTER TABLE identity_provider_allowed_audiences
            ADD CONSTRAINT fk_idp_allowed_audiences_tenant FOREIGN KEY (tenant_id)
            REFERENCES tenants(tenant_id) ON DELETE CASCADE;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_idp_allowed_audiences_idp'
    ) THEN
        ALTER TABLE identity_provider_allowed_audiences
            ADD CONSTRAINT fk_idp_allowed_audiences_idp FOREIGN KEY (identity_provider_id)
            REFERENCES identity_providers(identity_provider_id) ON DELETE CASCADE;
    END IF;
END$$;

-- ADD INDEXES
CREATE INDEX IF NOT EXISTS idx_idp_allowed_audiences_uuid
    ON identity_provider_allowed_audiences (identity_provider_allowed_audience_uuid);
-- one audience per IdP, soft-delete aware
CREATE UNIQUE INDEX IF NOT EXISTS uq_idp_allowed_audience
    ON identity_provider_allowed_audiences (identity_provider_id, audience) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_idp_allowed_audiences_provider
    ON identity_provider_allowed_audiences (identity_provider_id);
`
	return db.Exec(sql).Error
}
