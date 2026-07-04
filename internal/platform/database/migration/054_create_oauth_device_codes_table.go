package migration

import "gorm.io/gorm"

// CreateOAuthDeviceCodesTable creates the oauth_device_codes table which stores
// device authorization requests per RFC 8628. A device_code is issued to the
// client; the corresponding user_code is displayed to the user who then approves
// or denies access at the verification URI.
func CreateOAuthDeviceCodesTable(db *gorm.DB) error {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS oauth_device_codes (
    oauth_device_code_id   BIGSERIAL    PRIMARY KEY,
    oauth_device_code_uuid UUID         NOT NULL DEFAULT gen_random_uuid() UNIQUE,
    device_code_hash       TEXT         NOT NULL UNIQUE,
    user_code              VARCHAR(9)   NOT NULL UNIQUE,
    client_id              BIGINT       NOT NULL,
    tenant_id              BIGINT       NOT NULL,
    user_id                BIGINT,
    auth_acr               VARCHAR(32),
    auth_amr               JSONB        NOT NULL DEFAULT '[]'::jsonb,
    scope                  TEXT[]       NOT NULL DEFAULT '{}',
    status                 VARCHAR(20)  NOT NULL DEFAULT 'pending',
    interval               SMALLINT     NOT NULL DEFAULT 5,
    last_poll_at           TIMESTAMPTZ,
    expires_at             TIMESTAMPTZ  NOT NULL,
    created_at             TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_oauth_device_codes_status CHECK (status IN ('pending', 'approved', 'denied', 'expired'))
);

-- ADD CONSTRAINTS (FOREIGN KEYS)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_oauth_device_codes_client_id'
    ) THEN
        ALTER TABLE oauth_device_codes
            ADD CONSTRAINT fk_oauth_device_codes_client_id FOREIGN KEY (client_id)
            REFERENCES clients(client_id) ON DELETE CASCADE;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_oauth_device_codes_tenant_id'
    ) THEN
        ALTER TABLE oauth_device_codes
            ADD CONSTRAINT fk_oauth_device_codes_tenant_id FOREIGN KEY (tenant_id)
            REFERENCES tenants(tenant_id) ON DELETE CASCADE;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_oauth_device_codes_user_id'
    ) THEN
        ALTER TABLE oauth_device_codes
            ADD CONSTRAINT fk_oauth_device_codes_user_id FOREIGN KEY (user_id)
            REFERENCES users(user_id) ON DELETE SET NULL;
    END IF;
END$$;

-- ADD INDEXES
CREATE INDEX IF NOT EXISTS idx_oauth_device_codes_uuid       ON oauth_device_codes (oauth_device_code_uuid);
CREATE INDEX IF NOT EXISTS idx_oauth_device_codes_user_code  ON oauth_device_codes (user_code);
CREATE INDEX IF NOT EXISTS idx_oauth_device_codes_client_id  ON oauth_device_codes (client_id);
CREATE INDEX IF NOT EXISTS idx_oauth_device_codes_expires_at ON oauth_device_codes (expires_at);
CREATE INDEX IF NOT EXISTS idx_oauth_device_codes_status     ON oauth_device_codes (status);
CREATE INDEX IF NOT EXISTS idx_oauth_device_codes_scope      ON oauth_device_codes USING GIN (scope);
`
	return db.Exec(sql).Error
}
