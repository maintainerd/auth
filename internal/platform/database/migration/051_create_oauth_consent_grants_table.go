package migration

import (
	"gorm.io/gorm"
)

// CreateOAuthConsentGrantsTable creates the oauth_consent_grants table which
// stores user consent decisions per client. Each user-client pair has at most
// one row tracking the space-delimited scopes the user has approved.
func CreateOAuthConsentGrantsTable(db *gorm.DB) error {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS oauth_consent_grants (
    oauth_consent_grant_id   BIGSERIAL     PRIMARY KEY,
    oauth_consent_grant_uuid UUID          NOT NULL DEFAULT gen_random_uuid() UNIQUE,
    user_id                  BIGINT        NOT NULL,
    client_id                BIGINT        NOT NULL,
    tenant_id                BIGINT        NOT NULL,
    scopes                   TEXT          NOT NULL DEFAULT '',
    created_at               TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at               TIMESTAMPTZ   NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_oauth_consent_user_client UNIQUE (user_id, client_id)
);

-- ADD CONSTRAINTS (FOREIGN KEYS)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_oauth_consent_grants_user_id'
    ) THEN
        ALTER TABLE oauth_consent_grants
            ADD CONSTRAINT fk_oauth_consent_grants_user_id FOREIGN KEY (user_id)
            REFERENCES users(user_id) ON DELETE CASCADE;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_oauth_consent_grants_client_id'
    ) THEN
        ALTER TABLE oauth_consent_grants
            ADD CONSTRAINT fk_oauth_consent_grants_client_id FOREIGN KEY (client_id)
            REFERENCES clients(client_id) ON DELETE CASCADE;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_oauth_consent_grants_tenant_id'
    ) THEN
        ALTER TABLE oauth_consent_grants
            ADD CONSTRAINT fk_oauth_consent_grants_tenant_id FOREIGN KEY (tenant_id)
            REFERENCES tenants(tenant_id) ON DELETE CASCADE;
    END IF;
END$$;

-- ADD INDEXES
-- Consent lookups always carry (user_id, client_id), which the unique
-- constraint uq_oauth_consent_user_client already covers.
`
	return db.Exec(sql).Error
}
