package migration

import "gorm.io/gorm"

func CreateUserMFAWebAuthnCredentialsTable(db *gorm.DB) error {
	return db.Exec(`
CREATE TABLE IF NOT EXISTS user_mfa_webauthn_credentials (
    credential_id      BIGSERIAL    PRIMARY KEY,
    credential_uuid    UUID         NOT NULL UNIQUE DEFAULT gen_random_uuid(),
    user_id            BIGINT       NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    -- Credential ID as returned by the authenticator (base64url-encoded bytes).
    credential_key_id  TEXT         NOT NULL UNIQUE,
    -- CBOR-encoded COSE public key.
    public_key         BYTEA        NOT NULL,
    -- Authenticator Attestation GUID — identifies the authenticator model.
    aaguid             UUID,
    -- Monotonically increasing signature counter (replay protection).
    sign_count         BIGINT       NOT NULL DEFAULT 0,
    -- Comma-separated transport hints (usb, nfc, ble, hybrid, internal).
    transport          TEXT,
    -- Whether the authenticator supports backup eligibility / backup state.
    is_backup_eligible BOOLEAN      NOT NULL DEFAULT FALSE,
    is_backup_state    BOOLEAN      NOT NULL DEFAULT FALSE,
    -- Human-readable name for this credential (set by the user).
    name               VARCHAR(100) NOT NULL DEFAULT 'Security Key',
    last_used_at       TIMESTAMPTZ,
    created_at         TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ  NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_user_mfa_webauthn_credentials_user_id ON user_mfa_webauthn_credentials(user_id);
CREATE INDEX IF NOT EXISTS idx_user_webauthn_credential_key_id  ON user_mfa_webauthn_credentials(credential_key_id);

-- Keep users.is_webauthn_enabled / first_mfa_enrolled_at in sync. WebAuthn
-- credentials have no pending state (row exists only once registered, deleted
-- on removal), so INSERT/DELETE is sufficient. See sync_webauthn_flag() in
-- 024_create_users_table.go.
DROP TRIGGER IF EXISTS trg_sync_webauthn_flag ON user_mfa_webauthn_credentials;
CREATE TRIGGER trg_sync_webauthn_flag
    AFTER INSERT OR DELETE ON user_mfa_webauthn_credentials
    FOR EACH ROW EXECUTE FUNCTION sync_webauthn_flag();
`).Error
}
