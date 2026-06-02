package runner

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/maintainerd/auth/internal/platform/database/migration"
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
	// Block 1: Tenant core
	{"001_create_tenants_table", migration.CreateTenantTable},
	{"002_create_user_pools_table", migration.CreateUserPoolTable},
	{"003_create_branding_table", migration.CreateBrandingTable},
	{"004_create_tenant_settings_table", migration.CreateTenantSettingsTable},
	{"005_create_email_config_table", migration.CreateEmailConfigTable},
	{"006_create_sms_config_table", migration.CreateSMSConfigTable},
	{"007_create_webhook_endpoints_table", migration.CreateWebhookEndpointsTable},
	// Block 2: Services & policies
	{"008_create_services_table", migration.CreateServiceTable},
	{"009_create_tenant_services_table", migration.CreateTenantServicesTable},
	{"010_create_policies_table", migration.CreatePoliciesTable},
	{"011_create_service_policies_table", migration.CreateServicePoliciesTable},
	// Block 3: APIs & permissions
	{"012_create_apis_table", migration.CreateAPITable},
	{"013_create_permissions_table", migration.CreatePermissionTable},
	{"014_create_api_permissions_table", migration.CreateApiPermissionTable},
	// Block 4: Identity providers & clients
	{"015_create_identity_providers_table", migration.CreateIdentityProviderTable},
	{"016_create_clients_table", migration.CreateClientTable},
	{"017_create_client_uris_table", migration.CreateClientURIsTable},
	{"018_create_client_apis_table", migration.CreateClientAPIsTable},
	{"019_create_client_permissions_table", migration.CreateClientPermissionTable},
	// Block 5: API keys
	{"020_create_api_keys_table", migration.CreateAPIKeysTable},
	{"021_create_api_key_apis_table", migration.CreateAPIKeyAPITable},
	{"022_create_api_key_permissions_table", migration.CreateAPIKeyPermissionsTable},
	// Block 6: Roles
	{"023_create_roles_table", migration.CreateRoleTable},
	{"024_create_role_permissions_table", migration.CreateRolePermissionTable},
	// Block 7: Users — core + all user-scoped tables grouped together
	{"025_create_users_table", migration.CreateUserTable},
	{"026_create_user_identities_table", migration.CreateUserIdentityTable},
	{"027_create_user_roles_table", migration.CreateUserRoleTable},
	{"028_create_user_tokens_table", migration.CreateUserTokenTable},
	{"029_create_user_settings_table", migration.CreateUserSettingsTable},
	{"030_create_profiles_table", migration.CreateProfileTable},
	{"031_create_user_backup_codes_table", migration.CreateUserBackupCodesTable},
	{"032_create_user_totp_secrets_table", migration.CreateUserTOTPSecretsTable},
	{"033_create_user_webauthn_credentials_table", migration.CreateUserWebAuthnCredentialsTable},
	// Block 8: Tenant organisation & flows
	{"034_create_tenant_members_table", migration.CreateTenantMembersTable},
	{"035_create_signup_flows_table", migration.CreateSignupFlowTable},
	{"036_create_signup_flow_roles_table", migration.CreateSignupFlowRoleTable},
	{"037_create_invites_table", migration.CreateInvitesTable},
	{"038_create_invite_roles_table", migration.CreateInviteRolesTable},
	// Block 9: Security
	{"039_create_security_settings_table", migration.CreateSecuritySettingsTable},
	{"040_create_ip_restriction_rules_table", migration.CreateIPRestrictionRulesTable},
	{"041_create_security_settings_audit_table", migration.CreateSecuritySettingsAuditTable},
	// Block 10: Templates
	{"042_create_login_templates_table", migration.CreateLoginTemplatesTable},
	{"043_create_email_templates_table", migration.CreateEmailTemplatesTable},
	{"044_create_sms_templates_table", migration.CreateSMSTemplatesTable},
	// Block 11: Auth events
	{"045_create_auth_events_table", migration.CreateAuthEventsTable},
	// Block 12: OAuth
	{"046_create_oauth_authorization_codes_table", migration.CreateOAuthAuthorizationCodesTable},
	{"047_create_oauth_refresh_tokens_table", migration.CreateOAuthRefreshTokensTable},
	{"048_create_oauth_consent_grants_table", migration.CreateOAuthConsentGrantsTable},
	{"049_create_oauth_consent_challenges_table", migration.CreateOAuthConsentChallengesTable},
	{"050_create_oauth_par_requests_table", migration.CreateOAuthPARRequestsTable},
	{"051_create_oauth_device_codes_table", migration.CreateOAuthDeviceCodesTable},
	{"052_create_oauth_ciba_requests_table", migration.CreateOAuthCIBARequestsTable},
	// Block 13: SMS OTP
	{"053_create_sms_otps_table", migration.CreateSMSOtpsTable},
	// Block 14: Password policy
	{"054_create_user_password_history_table", migration.CreateUserPasswordHistoryTable},
	{"055_add_password_changed_at_to_users", migration.AddPasswordChangedAtToUsers},
	// Block 15: Audit hardening
	{"056_auth_events_append_only", migration.CreateAuthEventsAppendOnlyRule},
	// Block 16: OAuth client auth hardening
	{"057_add_client_secret_encrypted", migration.AddClientSecretEncrypted},
	// Block 17: SMS OTP hardening
	{"058_add_sms_otp_failed_attempts", migration.AddSMSOTPFailedAttempts},
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
