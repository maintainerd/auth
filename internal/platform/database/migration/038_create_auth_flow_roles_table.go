package migration

import (
	"gorm.io/gorm"
)

func CreateAuthFlowRoleTable(db *gorm.DB) error {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS auth_flow_roles (
    auth_flow_role_id    BIGSERIAL PRIMARY KEY,
    auth_flow_role_uuid  UUID NOT NULL UNIQUE,
    auth_flow_id         BIGINT NOT NULL,
    role_id              BIGINT NOT NULL,
    created_at           TIMESTAMPTZ DEFAULT now()
);

-- ADD CONSTRAINTS
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_auth_flow_roles_auth_flow_id'
    ) THEN
        ALTER TABLE auth_flow_roles
            ADD CONSTRAINT fk_auth_flow_roles_auth_flow_id FOREIGN KEY (auth_flow_id)
            REFERENCES auth_flows(auth_flow_id) ON DELETE CASCADE;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_auth_flow_roles_role_id'
    ) THEN
        ALTER TABLE auth_flow_roles
            ADD CONSTRAINT fk_auth_flow_roles_role_id FOREIGN KEY (role_id)
            REFERENCES roles(role_id) ON DELETE CASCADE;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'uq_auth_flow_roles_flow_role'
    ) THEN
        ALTER TABLE auth_flow_roles
            ADD CONSTRAINT uq_auth_flow_roles_flow_role UNIQUE (auth_flow_id, role_id);
    END IF;
END$$;

-- ADD INDEXES
CREATE INDEX IF NOT EXISTS idx_auth_flow_roles_uuid ON auth_flow_roles (auth_flow_role_uuid);
CREATE INDEX IF NOT EXISTS idx_auth_flow_roles_auth_flow_id ON auth_flow_roles (auth_flow_id);
CREATE INDEX IF NOT EXISTS idx_auth_flow_roles_role_id ON auth_flow_roles (role_id);
`
	return db.Exec(sql).Error
}
