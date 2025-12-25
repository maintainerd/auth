package runner

import (
	"github.com/maintainerd/auth/internal/database/migration"
	"gorm.io/gorm"
)

func RunMigrations(db *gorm.DB) {
	migration.CreateTenantTable(db)                // 001
	migration.CreateServiceTable(db)               // 002
	migration.CreateTenantServicesTable(db)        // 003
	migration.CreatePoliciesTable(db)              // 004
	migration.CreateServicePoliciesTable(db)       // 005
	migration.CreateServiceLogsTable(db)           // 006
	migration.CreateAPITable(db)                   // 007
	migration.CreatePermissionTable(db)            // 008
	migration.CreateApiPermissionTable(db)         // 009
	migration.CreateIdentityProviderTable(db)      // 010
	migration.CreateAuthClientTable(db)            // 011
	migration.CreateAuthClientUrisTable(db)        // 012
	migration.CreateAuthClientApiTable(db)         // 013
	migration.CreateAuthClientPermissionTable(db)  // 014
	migration.CreateAPIKeysTable(db)               // 015
	migration.CreateAPIKeyApiTable(db)             // 016
	migration.CreateAPIKeyPermissionsTable(db)     // 017
	migration.CreateRoleTable(db)                  // 018
	migration.CreateRolePermissionTable(db)        // 019
	migration.CreateUserTable(db)                  // 020
	migration.CreateUserIdentitiesTable(db)        // 021
	migration.CreateUserRoleTable(db)              // 022
	migration.CreateUserTokenTable(db)             // 023
	migration.CreateUserSettingsTable(db)          // 024
	migration.CreateProfileTable(db)               // 025
	migration.CreateSignupFlowTable(db)            // 026
	migration.CreateSignupFlowRoleTable(db)        // 027
	migration.CreateInvitesTable(db)               // 028
	migration.CreateInviteRolesTable(db)           // 029
	migration.CreateSecuritySettingsTable(db)      // 030
	migration.CreateIpRestrictionRulesTable(db)    // 031
	migration.CreateSecuritySettingsAuditTable(db) // 032
	migration.CreateLoginTemplatesTable(db)        // 033
	migration.CreateEmailTemplatesTable(db)        // 034
	migration.CreateSmsTemplatesTable(db)          // 035
	migration.CreateAuthLogTable(db)               // 036
}
