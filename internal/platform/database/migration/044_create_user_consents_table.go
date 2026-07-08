package migration

import "gorm.io/gorm"

func CreateUserConsentsTable(db *gorm.DB) error {
	return db.Exec(`
CREATE TABLE IF NOT EXISTS user_consents (
    user_consent_id   BIGSERIAL    PRIMARY KEY,
    user_consent_uuid UUID         NOT NULL UNIQUE DEFAULT gen_random_uuid(),
    user_id           BIGINT       NOT NULL,
    tenant_id         BIGINT       NOT NULL,
    consent_type      VARCHAR(50)  NOT NULL,
    policy_version    VARCHAR(50)  NOT NULL,
    accepted          BOOLEAN      NOT NULL,
    ip_address        INET,
    user_agent        TEXT,
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT fk_user_consents_user FOREIGN KEY (user_id)
        REFERENCES users(user_id) ON DELETE CASCADE,
    CONSTRAINT fk_user_consents_tenant FOREIGN KEY (tenant_id)
        REFERENCES tenants(tenant_id) ON DELETE CASCADE,
    CONSTRAINT chk_user_consents_type CHECK (consent_type IN (
        'terms_of_service', 'privacy_policy', 'data_processing'
    ))
);
CREATE INDEX IF NOT EXISTS idx_user_consents_user_id ON user_consents (user_id);
CREATE INDEX IF NOT EXISTS idx_user_consents_user_type ON user_consents (user_id, consent_type, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_user_consents_created_at ON user_consents (created_at);
CREATE UNIQUE INDEX IF NOT EXISTS uq_user_consents_accepted_version
    ON user_consents (user_id, consent_type, policy_version)
    WHERE accepted = TRUE;
`).Error
}
