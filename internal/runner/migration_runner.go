package runner

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/maintainerd/auth/internal/database/migration"
	"gorm.io/gorm"
)

// advisoryLockKey is a fixed 32-bit integer used as a PostgreSQL session-level
// advisory lock key. This guarantees that only one pod runs migrations at a time
// when multiple instances start against the same database simultaneously.
const advisoryLockKey = 7316949

// migrationEntry pairs a unique version string with its migration function.
// The version string is what gets written to schema_migrations — it must never
// be changed after a migration has been applied.
type migrationEntry struct {
	Version string
	Fn      func(db *gorm.DB) error
}

// migrations is the ordered list of all migrations. Add new entries at the
// bottom only — never reorder or remove existing entries.
var migrations = []migrationEntry{
	{"001_create_tenants_table", migration.CreateTenantTable},
	{"002_create_services_table", migration.CreateServiceTable},
	{"003_create_tenant_services_table", migration.CreateTenantServicesTable},
	{"004_create_policies_table", migration.CreatePoliciesTable},
	{"005_create_service_policies_table", migration.CreateServicePoliciesTable},
	{"006_create_service_logs_table", migration.CreateServiceLogsTable},
	{"007_create_apis_table", migration.CreateAPITable},
	{"008_create_permissions_table", migration.CreatePermissionTable},
	{"009_create_api_permissions_table", migration.CreateApiPermissionTable},
	{"010_create_identity_providers_table", migration.CreateIdentityProviderTable},
	{"011_create_clients_table", migration.CreateClientTable},
	{"012_create_client_uris_table", migration.CreateClientURIsTable},
	{"013_create_client_apis_table", migration.CreateClientAPIsTable},
	{"014_create_client_permissions_table", migration.CreateClientPermissionTable},
	{"015_create_api_keys_table", migration.CreateAPIKeysTable},
	{"016_create_api_key_apis_table", migration.CreateAPIKeyApiTable},
	{"017_create_api_key_permissions_table", migration.CreateAPIKeyPermissionsTable},
	{"018_create_roles_table", migration.CreateRoleTable},
	{"019_create_role_permissions_table", migration.CreateRolePermissionTable},
	{"020_create_users_table", migration.CreateUserTable},
	{"021_create_user_identities_table", migration.CreateUserIdentityTable},
	{"022_create_user_roles_table", migration.CreateUserRoleTable},
	{"023_create_user_tokens_table", migration.CreateUserTokenTable},
	{"024_create_user_settings_table", migration.CreateUserSettingsTable},
	{"025_create_profiles_table", migration.CreateProfileTable},
	{"026_create_tenant_users_table", migration.CreateTenantUsersTable},
	{"027_create_tenant_members_table", migration.CreateTenantMembersTable},
	{"028_create_signup_flows_table", migration.CreateSignupFlowTable},
	{"029_create_signup_flow_roles_table", migration.CreateSignupFlowRoleTable},
	{"030_create_invites_table", migration.CreateInvitesTable},
	{"031_create_invite_roles_table", migration.CreateInviteRolesTable},
	{"032_create_security_settings_table", migration.CreateSecuritySettingsTable},
	{"033_create_ip_restriction_rules_table", migration.CreateIpRestrictionRulesTable},
	{"034_create_security_settings_audit_table", migration.CreateSecuritySettingsAuditTable},
	{"035_create_login_templates_table", migration.CreateLoginTemplatesTable},
	{"036_create_email_templates_table", migration.CreateEmailTemplatesTable},
	{"037_create_sms_templates_table", migration.CreateSmsTemplatesTable},
	{"038_create_auth_logs_table", migration.CreateAuthLogTable},
}

// RunMigrations bootstraps the schema_migrations tracking table, acquires a
// PostgreSQL session-level advisory lock so only one pod runs migrations at a
// time, then applies every unapplied migration in order.
func RunMigrations(db *gorm.DB) error {
	ctx := context.Background()

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("migration: get underlying sql.DB: %w", err)
	}

	// Bootstrap the tracking table first. This is the one call that is always
	// idempotent — IF NOT EXISTS makes it safe to run on every startup.
	if err := bootstrapTrackingTable(db); err != nil {
		return err
	}

	// Acquire a session-level advisory lock. pg_advisory_lock blocks until the
	// lock is free, so concurrent pods will queue up here rather than racing.
	if _, err := sqlDB.ExecContext(ctx, "SELECT pg_advisory_lock($1)", advisoryLockKey); err != nil {
		return fmt.Errorf("migration: acquire advisory lock: %w", err)
	}
	defer sqlDB.ExecContext(ctx, "SELECT pg_advisory_unlock($1)", advisoryLockKey) //nolint:errcheck

	slog.Info("migration: advisory lock acquired")

	for _, m := range migrations {
		applied, err := isMigrationApplied(db, m.Version)
		if err != nil {
			return err
		}
		if applied {
			slog.Debug("migration: already applied, skipping", "version", m.Version)
			continue
		}

		start := time.Now()
		if err := m.Fn(db); err != nil {
			return fmt.Errorf("migration: %s failed: %w", m.Version, err)
		}
		if err := recordMigration(db, m.Version); err != nil {
			return err
		}
		slog.Info("migration: applied", "version", m.Version, "duration_ms", time.Since(start).Milliseconds())
	}

	slog.Info("migration: all migrations complete")
	return nil
}

// bootstrapTrackingTable creates the schema_migrations table if it does not
// already exist. This runs before the advisory lock is acquired because it must
// succeed for any migration logic to work, and CREATE TABLE IF NOT EXISTS is
// itself safe to run concurrently in PostgreSQL.
func bootstrapTrackingTable(db *gorm.DB) error {
	sql := `
CREATE TABLE IF NOT EXISTS schema_migrations (
    version    VARCHAR(255) PRIMARY KEY,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
);`
	if err := db.Exec(sql).Error; err != nil {
		return fmt.Errorf("migration: bootstrap schema_migrations table: %w", err)
	}
	return nil
}

func isMigrationApplied(db *gorm.DB, version string) (bool, error) {
	var count int64
	if err := db.Raw("SELECT COUNT(1) FROM schema_migrations WHERE version = ?", version).Scan(&count).Error; err != nil {
		return false, fmt.Errorf("migration: check applied status for %s: %w", version, err)
	}
	return count > 0, nil
}

func recordMigration(db *gorm.DB, version string) error {
	if err := db.Exec("INSERT INTO schema_migrations (version) VALUES (?)", version).Error; err != nil {
		return fmt.Errorf("migration: record %s in schema_migrations: %w", version, err)
	}
	return nil
}
