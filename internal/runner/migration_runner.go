package runner

import (
	"github.com/maintainerd/auth/internal/database/migration"
	"gorm.io/gorm"
)

func RunMigrations(db *gorm.DB) {
	migration.CreateTenantTable(db)               // 001
	migration.CreateServiceTable(db)              // 002
	migration.CreateTenantServicesTable(db)       // 003
	migration.CreatePoliciesTable(db)             // 004
	migration.CreateServicePoliciesTable(db)      // 005
	migration.CreateServiceLogsTable(db)          // 006
	migration.CreateAPITable(db)                  // 007
	migration.CreatePermissionTable(db)           // 008
	migration.CreateApiPermissionTable(db)        // 009
	migration.CreateIdentityProviderTable(db)     // 010
	migration.CreateAuthClientTable(db)           // 011
	migration.CreateAuthClientUrisTable(db)       // 012
	migration.CreateAuthClientApiTable(db)        // 013
	migration.CreateAuthClientPermissionTable(db) // 014
	migration.CreateAPIKeysTable(db)              // 015
	migration.CreateAPIKeyApiTable(db)            // 016
	migration.CreateAPIKeyPermissionsTable(db)    // 017
	migration.CreateRoleTable(db)                 // 018
	migration.CreateRolePermissionTable(db)       // 019
	migration.CreateUserTable(db)                 // 020
	migration.CreateUserIdentitiesTable(db)       // 021
	migration.CreateUserRoleTable(db)             // 022
	migration.CreateUserTokenTable(db)            // 023
	migration.CreateProfileTable(db)              // 024
	migration.CreateUserSettingsTable(db)         // 025
	migration.CreateOnboardingTable(db)           // 026
	migration.CreateOnboardingRoleTable(db)       // 027
	migration.CreateAuthLogTable(db)              // 028
	migration.CreateInvitesTable(db)              // 029
	migration.CreateInviteRolesTable(db)          // 030
	migration.CreateEmailTemplatesTable(db)       // 031
}
