package migration

import "gorm.io/gorm"

func CreateWebAuthnChallengesTable(db *gorm.DB) error {
	return db.Exec(`
CREATE TABLE IF NOT EXISTS webauthn_challenges (
    webauthn_challenge_id   BIGSERIAL    PRIMARY KEY,
    webauthn_challenge_uuid UUID         NOT NULL UNIQUE DEFAULT gen_random_uuid(),
    tenant_id               BIGINT       NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
    user_id                 BIGINT       REFERENCES users(user_id) ON DELETE CASCADE,
    challenge               VARCHAR(512) NOT NULL,
    operation               VARCHAR(20)  NOT NULL,
    rp_id                   VARCHAR(255) NOT NULL,
    expires_at              TIMESTAMPTZ  NOT NULL,
    used_at                 TIMESTAMPTZ,
    created_at              TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT chk_webauthn_challenges_operation CHECK (operation IN ('registration', 'authentication'))
);
CREATE INDEX IF NOT EXISTS idx_webauthn_challenges_challenge
    ON webauthn_challenges (challenge);
CREATE INDEX IF NOT EXISTS idx_webauthn_challenges_expires_at
    ON webauthn_challenges (expires_at) WHERE used_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_webauthn_challenges_user_id
    ON webauthn_challenges (user_id) WHERE user_id IS NOT NULL;
`).Error
}
