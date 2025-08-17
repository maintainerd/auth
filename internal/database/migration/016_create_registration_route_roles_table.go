package migration

import (
	"log"

	"gorm.io/gorm"
)

func CreateRegistrationRouteRoleTable(db *gorm.DB) {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS registration_route_roles (
    registration_route_role_id SERIAL PRIMARY KEY,
    registration_route_id      INTEGER NOT NULL,
    role_id                    INTEGER NOT NULL,
    created_at                 TIMESTAMPTZ DEFAULT now()
);

-- ADD CONSTRAINTS
ALTER TABLE registration_route_roles
    ADD CONSTRAINT fk_registration_route_roles_route_id FOREIGN KEY (registration_route_id) REFERENCES registration_routes(registration_route_id) ON DELETE CASCADE;
ALTER TABLE registration_route_roles
    ADD CONSTRAINT fk_registration_route_roles_role_id FOREIGN KEY (role_id) REFERENCES roles(role_id) ON DELETE CASCADE;

-- ADD INDEXES
CREATE INDEX idx_registration_route_roles_route_id ON registration_route_roles (registration_route_id);
CREATE INDEX idx_registration_route_roles_role_id ON registration_route_roles (role_id);
`
	if err := db.Exec(sql).Error; err != nil {
		log.Fatalf("❌ Failed to run migration 016_create_registration_route_roles_table: %v", err)
	}

	log.Println("✅ Migration 016_create_registration_route_roles_table executed")
}
