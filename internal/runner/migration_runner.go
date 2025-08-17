package runner

import (
	"github.com/maintainerd/auth/internal/database/migration"
	"gorm.io/gorm"
)

func RunMigrations(db *gorm.DB) {
	migration.CreateServiceTable(db)
	migration.CreateOrganizationTable(db)
	migration.CreateAuthContainerTable(db)
	migration.CreateAPITable(db)
	migration.CreatePermissionTable(db)
	migration.CreateIdentityProviderTable(db)
	migration.CreateAuthClientTable(db)
	migration.CreateRoleTable(db)
	migration.CreateRolePermissionTable(db)
	migration.CreateUserTable(db)
	migration.CreateUserIdentitiesTable(db)
	migration.CreateUserRoleTable(db)
	migration.CreateUserTokenTable(db)
	migration.CreateProfileTable(db)
	migration.CreateRegistrationRouteTable(db)
	migration.CreateRegistrationRouteRoleTable(db)
	migration.CreateLoginAttemptTable(db)
	migration.CreateAuthLogTable(db)
}
