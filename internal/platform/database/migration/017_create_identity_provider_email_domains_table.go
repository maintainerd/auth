package migration

import "gorm.io/gorm"

func CreateIdentityProviderEmailDomainsTable(db *gorm.DB) error {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS identity_provider_email_domains (
    identity_provider_email_domain_id BIGSERIAL PRIMARY KEY,
    identity_provider_email_domain_uuid UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),
    tenant_id            BIGINT NOT NULL,
    identity_provider_id BIGINT NOT NULL,
    domain               VARCHAR(255) NOT NULL,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at           TIMESTAMPTZ
);

-- ADD CONSTRAINTS (safe)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_idp_email_domains_tenant'
    ) THEN
        ALTER TABLE identity_provider_email_domains
            ADD CONSTRAINT fk_idp_email_domains_tenant FOREIGN KEY (tenant_id)
            REFERENCES tenants(tenant_id) ON DELETE CASCADE;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_idp_email_domains_idp'
    ) THEN
        ALTER TABLE identity_provider_email_domains
            ADD CONSTRAINT fk_idp_email_domains_idp FOREIGN KEY (identity_provider_id)
            REFERENCES identity_providers(identity_provider_id) ON DELETE CASCADE;
    END IF;
END$$;

-- ADD INDEXES
CREATE INDEX IF NOT EXISTS idx_idp_email_domains_uuid
    ON identity_provider_email_domains (identity_provider_email_domain_uuid);
-- one domain maps to exactly one IdP per tenant (home-realm discovery integrity)
CREATE UNIQUE INDEX IF NOT EXISTS uq_idp_email_domain
    ON identity_provider_email_domains (tenant_id, domain) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_idp_email_domain_provider
    ON identity_provider_email_domains (identity_provider_id);
`
	return db.Exec(sql).Error
}
