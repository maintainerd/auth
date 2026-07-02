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
    tenant_id                   BIGINT NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
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
    password_changed_at         TIMESTAMPTZ,
    temporary_password_expires_at TIMESTAMPTZ,
    pending_email               VARCHAR(255),
    email_change_otp            VARCHAR(10),
    email_change_otp_expires_at TIMESTAMPTZ,

    -- MFA status flags. is_totp_enabled / is_webauthn_enabled are denormalized
    -- caches (read on every login) kept in sync by the sync_totp_flag /
    -- sync_webauthn_flag triggers defined below. first_mfa_enrolled_at records
    -- the first-ever MFA enrollment: set once, never cleared.
    is_totp_enabled             BOOLEAN NOT NULL DEFAULT FALSE,
    is_webauthn_enabled         BOOLEAN NOT NULL DEFAULT FALSE,
    first_mfa_enrolled_at       TIMESTAMPTZ,

    created_at                  TIMESTAMPTZ DEFAULT now(),
    updated_at                  TIMESTAMPTZ DEFAULT now(),
    deleted_at                  TIMESTAMPTZ
);

-- ADD INDEXES
CREATE INDEX IF NOT EXISTS idx_users_uuid ON users (user_uuid);
CREATE INDEX IF NOT EXISTS idx_users_tenant_id ON users (tenant_id);
-- Tenant-scoped uniqueness: users are isolated per tenant, so the same email
-- or username may exist independently in different tenants ("separate worlds").
CREATE UNIQUE INDEX IF NOT EXISTS uq_users_tenant_username ON users (tenant_id, username) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS uq_users_tenant_email ON users (tenant_id, email) WHERE deleted_at IS NULL AND email IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_users_tenant_phone ON users (tenant_id, phone) WHERE deleted_at IS NULL AND phone IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_users_created_at ON users (created_at);
CREATE INDEX IF NOT EXISTS idx_users_deleted_at ON users (deleted_at) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_users_password_changed_at ON users(password_changed_at)
    WHERE password_changed_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_users_temporary_password_expires_at ON users(temporary_password_expires_at)
    WHERE temporary_password_expires_at IS NOT NULL;

-- Now that users exists, attach the audit FK constraints to all earlier
-- tables that declared created_by/updated_by columns. These can't be added
-- in their own migrations because users hadn't been created yet.
DO $$
DECLARE
    t TEXT;
    tables TEXT[] := ARRAY[
        'tenants', 'branding', 'email_config', 'sms_config',
        'services', 'policies', 'apis', 'permissions',
        'identity_providers', 'clients', 'api_keys', 'roles'
        -- webhook_endpoints is created AFTER users (migration 056, grouped with
        -- the event tables), so it attaches its own created_by/updated_by FKs there.
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

-- ──────────────────────────────────────────────────────────────────────────
-- MFA denormalization triggers
--
-- users.is_totp_enabled and users.is_webauthn_enabled are denormalized caches
-- read on the hot login path. These trigger functions recompute the flags from
-- the authoritative MFA factor tables so application code can never leave them
-- stale. first_mfa_enrolled_at is set once (on the first-ever enrollment) and
-- never cleared here.
--
-- The functions reference user_mfa_totp_secrets / user_mfa_webauthn_credentials
-- (created in migrations 032/033). PL/pgSQL bodies are parsed lazily at execution
-- time, so defining the functions before those tables exist is safe; the triggers
-- themselves are attached in 032/033 after each table is created.

-- sync_totp_flag: a TOTP secret row exists in a pending (is_enabled=false) state
-- between enrollment-begin and confirm, and enable/disable is an UPDATE (not a
-- row insert/delete). So the flag must track is_enabled=true, and the trigger
-- fires on INSERT/UPDATE/DELETE.
CREATE OR REPLACE FUNCTION sync_totp_flag() RETURNS TRIGGER AS $$
DECLARE
    uid BIGINT := COALESCE(NEW.user_id, OLD.user_id);
    has_totp BOOLEAN;
BEGIN
    SELECT EXISTS(
        SELECT 1 FROM user_mfa_totp_secrets WHERE user_id = uid AND is_enabled = true
    ) INTO has_totp;
    UPDATE users
        SET is_totp_enabled = has_totp,
            first_mfa_enrolled_at = CASE
                WHEN has_totp AND first_mfa_enrolled_at IS NULL THEN now()
                ELSE first_mfa_enrolled_at
            END
        WHERE user_id = uid;
    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

-- sync_webauthn_flag: WebAuthn credentials have no pending state — a row exists
-- only once fully registered and is deleted on removal — so plain row existence
-- is correct and the trigger fires on INSERT/DELETE.
CREATE OR REPLACE FUNCTION sync_webauthn_flag() RETURNS TRIGGER AS $$
DECLARE
    uid BIGINT := COALESCE(NEW.user_id, OLD.user_id);
    has_webauthn BOOLEAN;
BEGIN
    SELECT EXISTS(
        SELECT 1 FROM user_mfa_webauthn_credentials WHERE user_id = uid
    ) INTO has_webauthn;
    UPDATE users
        SET is_webauthn_enabled = has_webauthn,
            first_mfa_enrolled_at = CASE
                WHEN has_webauthn AND first_mfa_enrolled_at IS NULL THEN now()
                ELSE first_mfa_enrolled_at
            END
        WHERE user_id = uid;
    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;
`
	return db.Exec(sql).Error
}
