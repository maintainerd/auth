package runner

import (
	"github.com/maintainerd/auth/internal/database/migration"
	"gorm.io/gorm"
)

func RunMigrations(db *gorm.DB) {
	migration.CreateOrganizationTable(db)
	migration.CreateServiceTable(db)
	migration.CreateOrganizationServicesTable(db)
	migration.CreatePoliciesTable(db)
	migration.CreateServicePoliciesTable(db)
	migration.CreateServiceLogsTable(db)
	migration.CreateAPITable(db)
	migration.CreatePermissionTable(db)
	migration.CreateAuthContainerTable(db)
	migration.CreateIdentityProviderTable(db)
	migration.CreateAuthClientTable(db)
	migration.CreateAuthClientPermissionTable(db)
	migration.CreateRoleTable(db)
	migration.CreateRolePermissionTable(db)
	migration.CreateUserTable(db)
	migration.CreateUserIdentitiesTable(db)
	migration.CreateUserRoleTable(db)
	migration.CreateUserTokenTable(db)
	migration.CreateProfileTable(db)
	migration.CreateOnboardingTable(db)
	migration.CreateOnboardingRoleTable(db)
	migration.CreateAuthLogTable(db)
	migration.CreateInvitesTable(db)
	migration.CreateInviteRolesTable(db)
	migration.CreateEmailTemplatesTable(db)
}
