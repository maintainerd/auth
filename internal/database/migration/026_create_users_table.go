package migration

import (
	"gorm.io/gorm"
)

func CreateUserTable(db *gorm.DB) error {
	sql := `
-- CREATE TABLE
-- Note: ` + "`fullname`" + ` was removed. Use Profile.first_name/last_name/display_name instead.
CREATE TABLE IF NOT EXISTS users (
    user_id                     BIGSERIAL PRIMARY KEY,
    user_uuid                   UUID NOT NULL UNIQUE,
    username                    VARCHAR(255) NOT NULL,
    email                       VARCHAR(255),
    phone                       VARCHAR(20),
    password                    TEXT,
    is_email_verified           BOOLEAN DEFAULT FALSE,
    is_phone_verified           BOOLEAN DEFAULT FALSE,
    is_profile_completed        BOOLEAN DEFAULT FALSE,
    is_account_completed        BOOLEAN DEFAULT FALSE,
    status                      VARCHAR(20) DEFAULT 'active',
    metadata                    JSONB DEFAULT '{}'::jsonb,
    force_password_change       BOOLEAN NOT NULL DEFAULT FALSE,
    pending_email               VARCHAR(255),
    email_change_otp            VARCHAR(10),
    email_change_otp_expires_at TIMESTAMPTZ,
    created_at                  TIMESTAMPTZ DEFAULT now(),
    updated_at                  TIMESTAMPTZ DEFAULT now(),
    deleted_at                  TIMESTAMPTZ
);

-- ADD INDEXES
CREATE INDEX IF NOT EXISTS idx_users_uuid ON users (user_uuid);
CREATE UNIQUE INDEX IF NOT EXISTS uq_users_username ON users (username) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS uq_users_email ON users (email) WHERE deleted_at IS NULL AND email IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_users_phone ON users (phone);
CREATE INDEX IF NOT EXISTS idx_users_created_at ON users (created_at);
CREATE INDEX IF NOT EXISTS idx_users_deleted_at ON users (deleted_at) WHERE deleted_at IS NULL;

-- Now that users exists, attach the audit FK constraints to all earlier
-- tables that declared created_by/updated_by columns. These can't be added
-- in their own migrations because users hadn't been created yet.
DO $$
DECLARE
    t TEXT;
    tables TEXT[] := ARRAY[
        'tenants', 'branding', 'email_config', 'sms_config', 'webhook_endpoints',
        'services', 'policies', 'apis', 'permissions', 'identity_providers',
        'clients', 'api_keys', 'roles'
    ];
BEGIN
    FOREACH t IN ARRAY tables LOOP
        IF NOT EXISTS (
            SELECT 1 FROM pg_constraint WHERE conname = 'fk_' || t || '_created_by'
        ) THEN
            EXECUTE format(
                'ALTER TABLE %I ADD CONSTRAINT %I FOREIGN KEY (created_by) REFERENCES users(user_id) ON DELETE SET NULL',
                t, 'fk_' || t || '_created_by'
            );
        END IF;
        IF NOT EXISTS (
            SELECT 1 FROM pg_constraint WHERE conname = 'fk_' || t || '_updated_by'
        ) THEN
            EXECUTE format(
                'ALTER TABLE %I ADD CONSTRAINT %I FOREIGN KEY (updated_by) REFERENCES users(user_id) ON DELETE SET NULL',
                t, 'fk_' || t || '_updated_by'
            );
        END IF;
    END LOOP;
END$$;
`
	return db.Exec(sql).Error
}
