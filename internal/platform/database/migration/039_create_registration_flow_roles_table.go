package migration

import (
	"gorm.io/gorm"
)

func CreateRegistrationFlowRoleTable(db *gorm.DB) error {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS registration_flow_roles (
    registration_flow_role_id    BIGSERIAL PRIMARY KEY,
    registration_flow_role_uuid  UUID NOT NULL UNIQUE,
    registration_flow_id         BIGINT NOT NULL,
    role_id              BIGINT NOT NULL,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ADD CONSTRAINTS
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_registration_flow_roles_registration_flow_id'
    ) THEN
        ALTER TABLE registration_flow_roles
            ADD CONSTRAINT fk_registration_flow_roles_registration_flow_id FOREIGN KEY (registration_flow_id)
            REFERENCES registration_flows(registration_flow_id) ON DELETE CASCADE;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_registration_flow_roles_role_id'
    ) THEN
        ALTER TABLE registration_flow_roles
            ADD CONSTRAINT fk_registration_flow_roles_role_id FOREIGN KEY (role_id)
            REFERENCES roles(role_id) ON DELETE CASCADE;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'uq_registration_flow_roles_flow_role'
    ) THEN
        ALTER TABLE registration_flow_roles
            ADD CONSTRAINT uq_registration_flow_roles_flow_role UNIQUE (registration_flow_id, role_id);
    END IF;
END$$;

-- ADD INDEXES
CREATE INDEX IF NOT EXISTS idx_registration_flow_roles_uuid ON registration_flow_roles (registration_flow_role_uuid);
CREATE INDEX IF NOT EXISTS idx_registration_flow_roles_registration_flow_id ON registration_flow_roles (registration_flow_id);
CREATE INDEX IF NOT EXISTS idx_registration_flow_roles_role_id ON registration_flow_roles (role_id);
`
	return db.Exec(sql).Error
}
