package migration

import (
	"gorm.io/gorm"
)

func CreateTenantMembersTable(db *gorm.DB) error {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS tenant_members (
    tenant_member_id   BIGSERIAL PRIMARY KEY,
    tenant_member_uuid UUID NOT NULL UNIQUE,
    tenant_id          BIGINT NOT NULL,
    user_id            BIGINT NOT NULL,
    role               VARCHAR(32) NOT NULL DEFAULT 'member',
    created_by         BIGINT,
    updated_by         BIGINT,
    created_at         TIMESTAMPTZ DEFAULT now(),
    updated_at         TIMESTAMPTZ DEFAULT now(),
    deleted_at         TIMESTAMPTZ
);

-- ADD CONSTRAINTS (safe)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'chk_tenant_members_role'
    ) THEN
        ALTER TABLE tenant_members
            ADD CONSTRAINT chk_tenant_members_role
            CHECK (role IN ('owner', 'member'));
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_tenant_members_tenant_id'
    ) THEN
        ALTER TABLE tenant_members
            ADD CONSTRAINT fk_tenant_members_tenant_id FOREIGN KEY (tenant_id)
            REFERENCES tenants(tenant_id) ON DELETE CASCADE;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_tenant_members_user_id'
    ) THEN
        ALTER TABLE tenant_members
            ADD CONSTRAINT fk_tenant_members_user_id FOREIGN KEY (user_id)
            REFERENCES users(user_id) ON DELETE CASCADE;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_tenant_members_created_by'
    ) THEN
        ALTER TABLE tenant_members
            ADD CONSTRAINT fk_tenant_members_created_by FOREIGN KEY (created_by)
            REFERENCES users(user_id) ON DELETE SET NULL;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_tenant_members_updated_by'
    ) THEN
        ALTER TABLE tenant_members
            ADD CONSTRAINT fk_tenant_members_updated_by FOREIGN KEY (updated_by)
            REFERENCES users(user_id) ON DELETE SET NULL;
    END IF;
END$$;

-- ADD INDEXES
CREATE INDEX IF NOT EXISTS idx_tenant_members_uuid ON tenant_members (tenant_member_uuid);
CREATE INDEX IF NOT EXISTS idx_tenant_members_tenant_id ON tenant_members (tenant_id);
CREATE INDEX IF NOT EXISTS idx_tenant_members_user_id ON tenant_members (user_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_tenant_members_tenant_user ON tenant_members (tenant_id, user_id) WHERE deleted_at IS NULL;
-- A live tenant can have at most one owner. The service layer additionally
-- prevents removing/demoting that owner without an atomic ownership transfer.
CREATE UNIQUE INDEX IF NOT EXISTS uq_tenant_members_one_owner ON tenant_members (tenant_id) WHERE role = 'owner' AND deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_tenant_members_created_at ON tenant_members (created_at);
CREATE INDEX IF NOT EXISTS idx_tenant_members_deleted_at ON tenant_members (deleted_at) WHERE deleted_at IS NULL;
`
	return db.Exec(sql).Error
}
