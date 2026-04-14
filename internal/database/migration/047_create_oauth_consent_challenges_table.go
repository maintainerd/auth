package migration

import (
	"gorm.io/gorm"
)

// CreateOAuthConsentChallengesTable creates the oauth_consent_challenges table
// which stores pending consent challenges. Created when the authorization
// endpoint determines consent is needed, consumed when the frontend submits
// the user's decision. Short-lived (10 minutes).
func CreateOAuthConsentChallengesTable(db *gorm.DB) error {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS oauth_consent_challenges (
    oauth_consent_challenge_id   BIGSERIAL     PRIMARY KEY,
    oauth_consent_challenge_uuid UUID          NOT NULL DEFAULT gen_random_uuid() UNIQUE,
    client_id                    INTEGER       NOT NULL,
    user_id                      INTEGER       NOT NULL,
    tenant_id                    INTEGER       NOT NULL,
    redirect_uri                 TEXT          NOT NULL,
    scope                        TEXT          NOT NULL DEFAULT '',
    state                        TEXT,
    nonce                        TEXT,
    code_challenge               TEXT          NOT NULL,
    code_challenge_method        VARCHAR(10)   NOT NULL DEFAULT 'S256',
    response_type                VARCHAR(10)   NOT NULL DEFAULT 'code',
    expires_at                   TIMESTAMPTZ   NOT NULL,
    created_at                   TIMESTAMPTZ   NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_oauth_consent_challenge_method CHECK (code_challenge_method IN ('S256'))
);

-- ADD CONSTRAINTS (FOREIGN KEYS)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_oauth_consent_challenges_client_id'
    ) THEN
        ALTER TABLE oauth_consent_challenges
            ADD CONSTRAINT fk_oauth_consent_challenges_client_id FOREIGN KEY (client_id)
            REFERENCES clients(client_id) ON DELETE CASCADE;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_oauth_consent_challenges_user_id'
    ) THEN
        ALTER TABLE oauth_consent_challenges
            ADD CONSTRAINT fk_oauth_consent_challenges_user_id FOREIGN KEY (user_id)
            REFERENCES users(user_id) ON DELETE CASCADE;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_oauth_consent_challenges_tenant_id'
    ) THEN
        ALTER TABLE oauth_consent_challenges
            ADD CONSTRAINT fk_oauth_consent_challenges_tenant_id FOREIGN KEY (tenant_id)
            REFERENCES tenants(tenant_id) ON DELETE CASCADE;
    END IF;
END$$;

-- ADD INDEXES
CREATE INDEX IF NOT EXISTS idx_oauth_consent_challenges_uuid ON oauth_consent_challenges (oauth_consent_challenge_uuid);
CREATE INDEX IF NOT EXISTS idx_oauth_consent_challenges_expires ON oauth_consent_challenges (expires_at);
`
	return db.Exec(sql).Error
}
