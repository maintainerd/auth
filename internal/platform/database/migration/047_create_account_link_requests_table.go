package migration

import "gorm.io/gorm"

func CreateAccountLinkRequestsTable(db *gorm.DB) error {
	return db.Exec(`
CREATE TABLE IF NOT EXISTS account_link_requests (
    account_link_request_id     BIGSERIAL    PRIMARY KEY,
    account_link_request_uuid   UUID         NOT NULL UNIQUE DEFAULT gen_random_uuid(),
    tenant_id                   BIGINT       NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
    existing_user_id            BIGINT       NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    -- The identity this request will attach belongs to an identity provider,
    -- not to a client. Carried through the request so confirmation writes the
    -- same identity_provider_id the collision was detected under, instead of
    -- re-resolving a provider by name (ambiguous when a tenant configures two
    -- providers of the same type).
    identity_provider_id        BIGINT       NOT NULL REFERENCES identity_providers(identity_provider_id) ON DELETE CASCADE,
    provider_name               VARCHAR(100) NOT NULL,
    provider_subject            VARCHAR(512) NOT NULL,
    provider_email              VARCHAR(255),
    provider_claims             JSONB        NOT NULL DEFAULT '{}',
    status                      VARCHAR(20)  NOT NULL DEFAULT 'pending',
    confirmation_token          VARCHAR(255) NOT NULL UNIQUE,
    ip_address                  INET,
    expires_at                  TIMESTAMPTZ  NOT NULL,
    confirmed_at                TIMESTAMPTZ,
    rejected_at                 TIMESTAMPTZ,
    created_at                  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT chk_account_link_requests_status CHECK (status IN ('pending', 'confirmed', 'rejected', 'expired'))
);

CREATE INDEX IF NOT EXISTS idx_account_link_requests_token
    ON account_link_requests (confirmation_token) WHERE status = 'pending';
CREATE INDEX IF NOT EXISTS idx_account_link_requests_existing_user
    ON account_link_requests (existing_user_id, status);
CREATE INDEX IF NOT EXISTS idx_account_link_requests_expires_at
    ON account_link_requests (expires_at) WHERE status = 'pending';
`).Error
}
