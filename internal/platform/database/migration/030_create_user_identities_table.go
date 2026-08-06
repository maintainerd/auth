package migration

import (
	"gorm.io/gorm"
)

func CreateUserIdentityTable(db *gorm.DB) error {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS user_identities (
    user_identity_id      BIGSERIAL PRIMARY KEY,
    user_identity_uuid    UUID NOT NULL UNIQUE,
    tenant_id             BIGINT NOT NULL,
    user_id               BIGINT NOT NULL,
    -- No client_id: identities are per identity provider, not per application.
    -- Storing one made the same human a different subject on every client, and
    -- let a client keep authenticating a user after its IdP connection was
    -- disabled. Client access is resolved through client_identity_providers.
    identity_provider_id  BIGINT NOT NULL,
    sub                   VARCHAR(255) NOT NULL,
    provider              VARCHAR(100) NOT NULL,
    metadata              JSONB NOT NULL DEFAULT '{}',
    jit_provisioned_at    TIMESTAMPTZ,
    provisioning_source   VARCHAR(50)
        CONSTRAINT chk_user_identities_provisioning_source CHECK (
            provisioning_source IS NULL OR provisioning_source IN (
                'jit', 'scim', 'manual', 'invite', 'import'
            )
        ),
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ADD CONSTRAINTS
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_user_identities_user'
    ) THEN
        ALTER TABLE user_identities
            ADD CONSTRAINT fk_user_identities_user FOREIGN KEY (user_id)
            REFERENCES users(user_id) ON DELETE CASCADE;
    END IF;


    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_user_identities_tenant'
    ) THEN
        ALTER TABLE user_identities
            ADD CONSTRAINT fk_user_identities_tenant FOREIGN KEY (tenant_id)
            REFERENCES tenants(tenant_id) ON DELETE CASCADE;
    END IF;

    -- Drop the legacy two-column unique so multi-tenant federation works
    -- (two tenants sharing the same Google will have different tenant_id).
    IF EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'uq_user_identities_provider_sub'
    ) THEN
        ALTER TABLE user_identities DROP CONSTRAINT uq_user_identities_provider_sub;
    END IF;

    -- Superseded by the stronger (tenant_id, sub) key below.
    IF EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'uq_user_identities_tenant_provider_sub'
    ) THEN
        ALTER TABLE user_identities DROP CONSTRAINT uq_user_identities_tenant_provider_sub;
    END IF;

    -- The sub column is unique per TENANT, not per (tenant, provider). The tenant is the
    -- OIDC issuer, and OIDC Core §2 requires sub to be unique per issuer — every
    -- lookup that resolves a token back to a user matches on sub alone.
    --
    -- With the provider slug in the key, a provider whose subject an attacker chooses
    -- (their own OIDC/SAML connection, or a settable NameID) could JIT-provision
    -- an identity reusing a victim's sub under a different provider slug. The
    -- insert succeeded, and the sub lookup then returned whichever row sorted
    -- first — the victim's account. This makes that insert fail instead.
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'uq_user_identities_tenant_sub'
    ) THEN
        ALTER TABLE user_identities
            ADD CONSTRAINT uq_user_identities_tenant_sub UNIQUE (tenant_id, sub);
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_user_identities_idp'
    ) THEN
        ALTER TABLE user_identities
            ADD CONSTRAINT fk_user_identities_idp FOREIGN KEY (identity_provider_id)
            REFERENCES identity_providers(identity_provider_id) ON DELETE RESTRICT;
    END IF;
END$$;

-- ADD INDEXES (continued)
CREATE INDEX IF NOT EXISTS idx_user_identities_idp_id ON user_identities (identity_provider_id);

-- ADD INDEXES
CREATE INDEX IF NOT EXISTS idx_user_identities_uuid ON user_identities (user_identity_uuid);
CREATE INDEX IF NOT EXISTS idx_user_identities_user_id ON user_identities (user_id);
CREATE INDEX IF NOT EXISTS idx_user_identities_tenant_id ON user_identities (tenant_id);
CREATE INDEX IF NOT EXISTS idx_user_identities_created_at ON user_identities (created_at);
CREATE INDEX IF NOT EXISTS idx_user_identities_idp_provider ON user_identities (identity_provider_id, provider);
`
	return db.Exec(sql).Error
}
