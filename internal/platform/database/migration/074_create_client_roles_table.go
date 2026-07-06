package migration

import "gorm.io/gorm"

func CreateClientRolesTable(db *gorm.DB) error {
	return db.Exec(`
CREATE TABLE IF NOT EXISTS client_roles (
    client_role_id   BIGSERIAL   PRIMARY KEY,
    client_role_uuid UUID        NOT NULL UNIQUE DEFAULT gen_random_uuid(),
    client_id        BIGINT      NOT NULL,
    role_id          BIGINT      NOT NULL,
    created_by       BIGINT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT fk_client_roles_client FOREIGN KEY (client_id)
        REFERENCES clients(client_id) ON DELETE CASCADE,
    CONSTRAINT fk_client_roles_role FOREIGN KEY (role_id)
        REFERENCES roles(role_id) ON DELETE CASCADE,
    CONSTRAINT fk_client_roles_created_by FOREIGN KEY (created_by)
        REFERENCES users(user_id) ON DELETE SET NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS uq_client_roles_client_role
    ON client_roles (client_id, role_id);
CREATE INDEX IF NOT EXISTS idx_client_roles_client_id
    ON client_roles (client_id);
CREATE INDEX IF NOT EXISTS idx_client_roles_role_id
    ON client_roles (role_id);
`).Error
}
