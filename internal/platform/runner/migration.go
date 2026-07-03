package runner

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/maintainerd/maintainerd-auth/internal/platform/database/migration"
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

// migrations is the ordered list of all migrations.
//
// Schema policy (pre-release / not deployed) — see
// docs/contributing/database-migrations.md: migrations are CREATE-ONLY, one
// canonical create migration per table. To change a table, edit its create
// migration in place; do NOT add *_add_/*_alter_/*_drop_ migrations. Only
// brand-new TABLES get a new entry appended at the bottom.
// This policy freezes at first production deployment (then: forward-only).
var migrations = []migrationEntry{
	// Block 1: Tenant core
	{"001_create_tenants_table", migration.CreateTenantTable},
	{"003_create_branding_table", migration.CreateBrandingTable},
	{"004_create_tenant_settings_table", migration.CreateTenantSettingsTable},
	{"005_create_email_config_table", migration.CreateEmailConfigTable},
	{"006_create_sms_config_table", migration.CreateSMSConfigTable},
	// Block 2: Services & policies
	{"007_create_services_table", migration.CreateServiceTable},
	{"008_create_tenant_services_table", migration.CreateTenantServicesTable},
	{"009_create_policies_table", migration.CreatePoliciesTable},
	{"010_create_service_policies_table", migration.CreateServicePoliciesTable},
	// Block 3: APIs & permissions
	{"011_create_apis_table", migration.CreateAPITable},
	{"012_create_permissions_table", migration.CreatePermissionTable},
	{"013_create_api_permissions_table", migration.CreateApiPermissionTable},
	// Block 4: Identity providers & clients
	{"014_create_identity_providers_table", migration.CreateIdentityProviderTable},
	{"015_create_clients_table", migration.CreateClientTable},
	{"016_create_client_uris_table", migration.CreateClientURIsTable},
	{"017_create_client_apis_table", migration.CreateClientAPIsTable},
	{"018_create_client_permissions_table", migration.CreateClientPermissionTable},
	// Block 5: API keys
	{"019_create_api_keys_table", migration.CreateAPIKeysTable},
	{"020_create_api_key_apis_table", migration.CreateAPIKeyAPITable},
	{"021_create_api_key_permissions_table", migration.CreateAPIKeyPermissionsTable},
	// Block 6: Roles
	{"022_create_roles_table", migration.CreateRoleTable},
	{"023_create_role_permissions_table", migration.CreateRolePermissionTable},
	// Block 7: Users — core + all user-scoped tables grouped together
	{"024_create_users_table", migration.CreateUserTable},
	{"025_create_user_identities_table", migration.CreateUserIdentityTable},
	{"026_create_user_roles_table", migration.CreateUserRoleTable},
	{"027_create_user_tokens_table", migration.CreateUserTokenTable},
	{"028_create_user_otps_table", migration.CreateUserOTPsTable},
	{"029_create_user_settings_table", migration.CreateUserSettingsTable},
	{"030_create_profiles_table", migration.CreateProfileTable},
	{"031_create_user_mfa_backup_codes_table", migration.CreateUserMFABackupCodesTable},
	{"032_create_user_mfa_totp_secrets_table", migration.CreateUserMFATOTPSecretsTable},
	{"033_create_user_mfa_webauthn_credentials_table", migration.CreateUserMFAWebAuthnCredentialsTable},
	{"034_create_user_mfa_phones_table", migration.CreateUserMFAPhonesTable},
	{"035_create_user_mfa_emails_table", migration.CreateUserMFAEmailsTable},
	{"036_create_user_password_history_table", migration.CreateUserPasswordHistoryTable},
	// Block 8: Tenant organisation & flows
	{"037_create_tenant_members_table", migration.CreateTenantMembersTable},
	{"038_create_registration_flows_table", migration.CreateRegistrationFlowTable},
	{"039_create_registration_flow_roles_table", migration.CreateRegistrationFlowRoleTable},
	{"041_create_invites_table", migration.CreateInvitesTable},
	// Block 9: Security
	{"042_create_security_settings_table", migration.CreateSecuritySettingsTable},
	{"043_create_ip_restriction_rules_table", migration.CreateIPRestrictionRulesTable},
	{"044_create_security_settings_audit_table", migration.CreateSecuritySettingsAuditTable},
	// Block 10: Templates
	{"046_create_email_templates_table", migration.CreateEmailTemplatesTable},
	{"047_create_sms_templates_table", migration.CreateSMSTemplatesTable},
	// Block 11: Auth events
	{"048_create_auth_events_table", migration.CreateAuthEventsTable},
	// Block 12: OAuth
	{"049_create_oauth_authorization_codes_table", migration.CreateOAuthAuthorizationCodesTable},
	{"050_create_oauth_refresh_tokens_table", migration.CreateOAuthRefreshTokensTable},
	{"051_create_oauth_consent_grants_table", migration.CreateOAuthConsentGrantsTable},
	{"052_create_oauth_consent_challenges_table", migration.CreateOAuthConsentChallengesTable},
	{"053_create_oauth_par_requests_table", migration.CreateOAuthPARRequestsTable},
	{"054_create_oauth_device_codes_table", migration.CreateOAuthDeviceCodesTable},
	{"055_create_oauth_ciba_requests_table", migration.CreateOAuthCIBARequestsTable},
	// Block 14: Webhooks & event-driven integration
	{"056_create_event_types_table", migration.CreateEventTypesTable},
	{"057_create_webhook_endpoints_table", migration.CreateWebhookEndpointsTable},
	{"058_create_webhook_endpoint_events_table", migration.CreateWebhookEndpointEventsTable},
	{"059_create_event_routes_table", migration.CreateEventRoutesTable},
	{"060_create_tenant_event_types_table", migration.CreateTenantEventTypesTable},
	{"061_create_integration_event_outbox_table", migration.CreateIntegrationEventOutboxTable},
	{"062_create_webhook_delivery_history_table", migration.CreateWebhookDeliveryHistoryTable},
	{"063_create_client_identity_providers_table", migration.CreateClientIdentityProvidersTable},
	{"064_create_oauth_broker_sessions_table", migration.CreateOAuthBrokerSessionsTable},
	{"065_create_identity_provider_email_domains_table", migration.CreateIdentityProviderEmailDomainsTable},
	{"066_create_identity_provider_allowed_audiences_table", migration.CreateIdentityProviderAllowedAudiencesTable},
	{"067_create_oauth_authorize_requests_table", migration.CreateOAuthAuthorizeRequestsTable},
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
