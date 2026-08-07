package migration

import "gorm.io/gorm"

// CreateOAuthBrokerSessionsTable creates the oauth_broker_sessions table which
// correlates the two OAuth2 legs of a brokered login: the downstream app's
// /oauth/authorize request (OAuth #1) and maintainerd's request to the upstream
// identity provider (OAuth #2). Each row stores the original app request (to
// resume after the provider callback) plus the per-attempt state and PKCE
// verifier used against the provider. Rows are single-use and short-lived.
func CreateOAuthBrokerSessionsTable(db *gorm.DB) error {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS oauth_broker_sessions (
    oauth_broker_session_id   BIGSERIAL    PRIMARY KEY,
    oauth_broker_session_uuid UUID         NOT NULL DEFAULT gen_random_uuid() UNIQUE,
    tenant_id                 BIGINT       NOT NULL,
    client_id                 BIGINT       NOT NULL,
    identity_provider_id      BIGINT       NOT NULL,
    identity_provider_identifier VARCHAR(512)      NOT NULL,

    -- purpose discriminates the two flows that share this table. 'login' is the
    -- brokered sign-in (app -> maintainerd -> provider); 'link' attaches a
    -- provider identity to an ALREADY signed-in account. They must never be
    -- interchangeable: a login session replayed against the link callback would
    -- attach an arbitrary identity to the caller's account, and a link session
    -- replayed against the login callback would mint a session from a request
    -- that was never a sign-in.
    purpose                   VARCHAR(16)  NOT NULL DEFAULT 'login',
    -- The account a 'link' request belongs to. NULL for 'login', where no user
    -- is known yet. Binding the request to the user at START is what prevents
    -- account-linking CSRF: the callback can only ever attach to the account
    -- that began the flow.
    user_id                   BIGINT,

    -- The login broker resumes an original app authorize request; a link flow
    -- has none, so these are nullable.
    app_redirect_uri          VARCHAR(2048),
    app_state                 TEXT,
    app_scope                 TEXT[],
    app_nonce                 TEXT,
    app_code_challenge        TEXT,
    app_code_challenge_method VARCHAR(10),
    CONSTRAINT chk_oauth_broker_sessions_challenge_method
        CHECK (app_code_challenge_method IS NULL OR app_code_challenge_method IN ('S256')),
    CONSTRAINT chk_oauth_broker_sessions_purpose
        CHECK (purpose IN ('login', 'link')),
    -- Enforce the shape of each flow in the schema, so a bug in the service
    -- layer cannot persist a half-formed request: a link must name its user, a
    -- login must carry the app redirect it will resume.
    CONSTRAINT chk_oauth_broker_sessions_purpose_shape CHECK (
        (purpose = 'link'  AND user_id IS NOT NULL AND app_redirect_uri IS NULL) OR
        (purpose = 'login' AND user_id IS NULL     AND app_redirect_uri IS NOT NULL)
    ),

    idp_state                 TEXT         NOT NULL UNIQUE,
    idp_pkce_verifier         TEXT         NOT NULL,
    idp_nonce                 TEXT,

    expires_at                TIMESTAMPTZ  NOT NULL,
    consumed_at               TIMESTAMPTZ,
    created_at                TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- ADD CONSTRAINTS (FOREIGN KEYS)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_oauth_broker_sessions_user_id'
    ) THEN
        ALTER TABLE oauth_broker_sessions
            ADD CONSTRAINT fk_oauth_broker_sessions_user_id FOREIGN KEY (user_id)
            REFERENCES users(user_id) ON DELETE CASCADE;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_oauth_broker_sessions_tenant_id'
    ) THEN
        ALTER TABLE oauth_broker_sessions
            ADD CONSTRAINT fk_oauth_broker_sessions_tenant_id FOREIGN KEY (tenant_id)
            REFERENCES tenants(tenant_id) ON DELETE CASCADE;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_oauth_broker_sessions_client_id'
    ) THEN
        ALTER TABLE oauth_broker_sessions
            ADD CONSTRAINT fk_oauth_broker_sessions_client_id FOREIGN KEY (client_id)
            REFERENCES clients(client_id) ON DELETE CASCADE;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_oauth_broker_sessions_idp_id'
    ) THEN
        ALTER TABLE oauth_broker_sessions
            ADD CONSTRAINT fk_oauth_broker_sessions_idp_id FOREIGN KEY (identity_provider_id)
            REFERENCES identity_providers(identity_provider_id) ON DELETE CASCADE;
    END IF;
END$$;

-- ADD INDEXES
CREATE INDEX IF NOT EXISTS idx_oauth_broker_sessions_uuid          ON oauth_broker_sessions (oauth_broker_session_uuid);
CREATE UNIQUE INDEX IF NOT EXISTS uq_oauth_broker_sessions_idp_state ON oauth_broker_sessions (idp_state);
CREATE INDEX IF NOT EXISTS idx_oauth_broker_sessions_expires_at    ON oauth_broker_sessions (expires_at);
CREATE INDEX IF NOT EXISTS idx_oauth_broker_sessions_client_id     ON oauth_broker_sessions (client_id);
CREATE INDEX IF NOT EXISTS idx_oauth_broker_sessions_app_scope ON oauth_broker_sessions USING GIN (app_scope) WHERE app_scope IS NOT NULL;
`
	return db.Exec(sql).Error
}
