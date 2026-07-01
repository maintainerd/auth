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
    client_id             BIGINT NOT NULL,
    identity_provider_id  BIGINT,
    sub                   VARCHAR(255) NOT NULL,
    provider              VARCHAR(100) NOT NULL,
    metadata              JSONB,
    created_at            TIMESTAMPTZ DEFAULT now(),
    updated_at            TIMESTAMPTZ DEFAULT now()
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
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_user_identities_client'
    ) THEN
        ALTER TABLE user_identities
            ADD CONSTRAINT fk_user_identities_client FOREIGN KEY (client_id)
            REFERENCES clients(client_id) ON DELETE CASCADE;
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

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'uq_user_identities_tenant_provider_sub'
    ) THEN
        ALTER TABLE user_identities
            ADD CONSTRAINT uq_user_identities_tenant_provider_sub UNIQUE (tenant_id, provider, sub);
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_user_identities_idp'
    ) THEN
        ALTER TABLE user_identities
            ADD CONSTRAINT fk_user_identities_idp FOREIGN KEY (identity_provider_id)
            REFERENCES identity_providers(identity_provider_id) ON DELETE SET NULL;
    END IF;
END$$;

-- ADD INDEXES (continued)
CREATE INDEX IF NOT EXISTS idx_user_identities_idp_id ON user_identities (identity_provider_id);

-- ADD INDEXES
CREATE INDEX IF NOT EXISTS idx_user_identities_uuid ON user_identities (user_identity_uuid);
CREATE INDEX IF NOT EXISTS idx_user_identities_user_id ON user_identities (user_id);
CREATE INDEX IF NOT EXISTS idx_user_identities_client_id ON user_identities (client_id);
CREATE INDEX IF NOT EXISTS idx_user_identities_tenant_id ON user_identities (tenant_id);
CREATE INDEX IF NOT EXISTS idx_user_identities_created_at ON user_identities (created_at);
`
	return db.Exec(sql).Error
}
