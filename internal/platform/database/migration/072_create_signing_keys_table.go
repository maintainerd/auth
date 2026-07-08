package migration

import "gorm.io/gorm"

func CreateSigningKeysTable(db *gorm.DB) error {
	return db.Exec(`
CREATE TABLE IF NOT EXISTS signing_keys (
    signing_key_id          BIGSERIAL    PRIMARY KEY,
    signing_key_uuid        UUID         NOT NULL UNIQUE DEFAULT gen_random_uuid(),
    tenant_id               BIGINT       REFERENCES tenants(tenant_id) ON DELETE CASCADE,
    kid                     VARCHAR(128) NOT NULL UNIQUE,
    algorithm               VARCHAR(20)  NOT NULL,
    use                     VARCHAR(10)  NOT NULL,
    status                  VARCHAR(20)  NOT NULL DEFAULT 'active',
    public_key_pem          TEXT         NOT NULL,
    private_key_encrypted   BYTEA        NOT NULL,
    key_encryption_key_id   VARCHAR(255) NOT NULL,
    rotated_at              TIMESTAMPTZ,
    expires_at              TIMESTAMPTZ,
    created_by              BIGINT       REFERENCES users(user_id) ON DELETE SET NULL,
    created_at              TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at              TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT chk_signing_keys_algorithm CHECK (algorithm IN ('RS256', 'RS384', 'RS512', 'ES256', 'ES384', 'ES512', 'EdDSA')),
    CONSTRAINT chk_signing_keys_use CHECK (use IN ('sig', 'enc')),
    CONSTRAINT chk_signing_keys_status CHECK (status IN ('active', 'retired', 'compromised'))
);
CREATE INDEX IF NOT EXISTS idx_signing_keys_tenant_status
    ON signing_keys (tenant_id, status) WHERE status = 'active';
CREATE INDEX IF NOT EXISTS idx_signing_keys_expires_at
    ON signing_keys (expires_at) WHERE expires_at IS NOT NULL AND status = 'active';
`).Error
}
