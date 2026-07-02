package migration

import (
	"gorm.io/gorm"
)

func CreateInvitesTable(db *gorm.DB) error {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS invites (
    invite_id           BIGSERIAL PRIMARY KEY,
    invite_uuid         UUID NOT NULL UNIQUE,
    tenant_id           BIGINT NOT NULL,
    client_id           BIGINT NOT NULL,
    invited_email       VARCHAR(255) NOT NULL,
    invited_by_user_id  BIGINT,
    invite_token        TEXT NOT NULL UNIQUE,
    registration_flow_id        BIGINT,
    callback_url        TEXT,
    status              VARCHAR(20) NOT NULL DEFAULT 'pending',
    expires_at          TIMESTAMPTZ,
    used_at             TIMESTAMPTZ,
    created_by          BIGINT,
    updated_by          BIGINT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at          TIMESTAMPTZ
);

-- ADD CONSTRAINTS
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_invites_tenant_id'
    ) THEN
        ALTER TABLE invites
            ADD CONSTRAINT fk_invites_tenant_id FOREIGN KEY (tenant_id)
            REFERENCES tenants(tenant_id) ON DELETE CASCADE;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_invites_client_id'
    ) THEN
        ALTER TABLE invites
            ADD CONSTRAINT fk_invites_client_id FOREIGN KEY (client_id)
            REFERENCES clients(client_id) ON DELETE CASCADE;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_invites_registration_flow_id'
    ) THEN
        ALTER TABLE invites
            ADD CONSTRAINT fk_invites_registration_flow_id FOREIGN KEY (registration_flow_id)
            REFERENCES registration_flows(registration_flow_id) ON DELETE SET NULL;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_invites_invited_by_user_id'
    ) THEN
        ALTER TABLE invites
            ADD CONSTRAINT fk_invites_invited_by_user_id FOREIGN KEY (invited_by_user_id)
            REFERENCES users(user_id) ON DELETE SET NULL;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_invites_created_by'
    ) THEN
        ALTER TABLE invites
            ADD CONSTRAINT fk_invites_created_by FOREIGN KEY (created_by)
            REFERENCES users(user_id) ON DELETE SET NULL;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_invites_updated_by'
    ) THEN
        ALTER TABLE invites
            ADD CONSTRAINT fk_invites_updated_by FOREIGN KEY (updated_by)
            REFERENCES users(user_id) ON DELETE SET NULL;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'chk_invites_status'
    ) THEN
        ALTER TABLE invites
            ADD CONSTRAINT chk_invites_status
            CHECK (status IN ('pending', 'accepted', 'expired', 'revoked'));
    END IF;
END$$;

-- ADD INDEXES
CREATE INDEX IF NOT EXISTS idx_invites_uuid ON invites (invite_uuid);
CREATE INDEX IF NOT EXISTS idx_invites_tenant_id ON invites (tenant_id);
CREATE INDEX IF NOT EXISTS idx_invites_client_id ON invites (client_id);
CREATE INDEX IF NOT EXISTS idx_invites_registration_flow_id ON invites (registration_flow_id);
CREATE INDEX IF NOT EXISTS idx_invites_tenant_email ON invites (tenant_id, invited_email) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_invites_invited_by_user_id ON invites (invited_by_user_id);
CREATE INDEX IF NOT EXISTS idx_invites_created_at ON invites (created_at);
CREATE INDEX IF NOT EXISTS idx_invites_deleted_at ON invites (deleted_at) WHERE deleted_at IS NULL;
`
	return db.Exec(sql).Error
}
