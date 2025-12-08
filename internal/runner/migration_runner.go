package runner

import (
	"github.com/maintainerd/auth/internal/database/migration"
	"gorm.io/gorm"
)

func RunMigrations(db *gorm.DB) {
	migration.CreateTenantTable(db)                         // 001
	migration.CreateServiceTable(db)                        // 002
	migration.CreateTenantServicesTable(db)                 // 003
	migration.CreatePoliciesTable(db)                       // 004
	migration.CreateServicePoliciesTable(db)                // 005
	migration.CreateServiceLogsTable(db)                    // 006
	migration.CreateAPITable(db)                            // 007
	migration.CreatePermissionTable(db)                     // 008
	migration.CreateApiPermissionTable(db)                  // 009
	migration.CreateIdentityProviderTable(db)               // 010
	migration.CreateAuthClientTable(db)                     // 011
	migration.CreateAuthClientUrisTable(db)                 // 012
	migration.CreateAuthClientApiTable(db)                  // 013
	migration.CreateAuthClientPermissionTable(db)           // 014
	migration.CreateRoleTable(db)                           // 015
	migration.CreateRolePermissionTable(db)                 // 016
	migration.CreateUserTable(db)                           // 017
	migration.CreateUserIdentitiesTable(db)                 // 018
	migration.CreateUserRoleTable(db)                       // 019
	migration.CreateUserTokenTable(db)                      // 020
	migration.CreateProfileTable(db)                        // 021
	migration.CreateUserSettingsTable(db)                   // 022
	migration.CreateOnboardingTable(db)                     // 023
	migration.CreateOnboardingRoleTable(db)                 // 024
	migration.CreateAuthLogTable(db)                        // 025
	migration.CreateInvitesTable(db)                        // 026
	migration.CreateInviteRolesTable(db)                    // 027
	migration.CreateEmailTemplatesTable(db)                 // 028
	migration.AddUniqueConstraintsAuthClientApis(db)        // 029
	migration.AddUniqueConstraintsAuthClientPermissions(db) // 030
}
