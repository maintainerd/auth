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
	{"002_create_branding_table", migration.CreateBrandingTable},
	{"003_create_tenant_settings_table", migration.CreateTenantSettingsTable},
	{"004_create_email_config_table", migration.CreateEmailConfigTable},
	{"005_create_sms_config_table", migration.CreateSMSConfigTable},
	// Block 2: Services & policies
	{"006_create_services_table", migration.CreateServiceTable},
	{"007_create_tenant_services_table", migration.CreateTenantServicesTable},
	{"008_create_policies_table", migration.CreatePoliciesTable},
	{"009_create_service_policies_table", migration.CreateServicePoliciesTable},
	{"010_create_policy_version_history_table", migration.CreatePolicyVersionHistoryTable},
	// Block 3: APIs, permissions & roles
	{"011_create_apis_table", migration.CreateAPITable},
	{"012_create_permissions_table", migration.CreatePermissionTable},
	{"013_create_api_permissions_table", migration.CreateApiPermissionTable},
	{"014_create_roles_table", migration.CreateRoleTable},
	{"015_create_role_permissions_table", migration.CreateRolePermissionTable},
	// Block 4: Identity providers
	{"016_create_identity_providers_table", migration.CreateIdentityProviderTable},
	{"017_create_identity_provider_email_domains_table", migration.CreateIdentityProviderEmailDomainsTable},
	{"018_create_identity_provider_allowed_audiences_table", migration.CreateIdentityProviderAllowedAudiencesTable},
	// Block 5: Clients
	{"019_create_clients_table", migration.CreateClientTable},
	{"020_create_client_uris_table", migration.CreateClientURIsTable},
	{"021_create_client_apis_table", migration.CreateClientAPIsTable},
	{"022_create_client_permissions_table", migration.CreateClientPermissionTable},
	{"023_create_client_identity_providers_table", migration.CreateClientIdentityProvidersTable},
	{"024_create_client_roles_table", migration.CreateClientRolesTable},
	// Block 6: API keys
	{"025_create_api_keys_table", migration.CreateAPIKeysTable},
	{"026_create_api_key_apis_table", migration.CreateAPIKeyAPITable},
	{"027_create_api_key_permissions_table", migration.CreateAPIKeyPermissionsTable},
	// Block 7: Workload identity
	{"028_create_workload_identity_federations_table", migration.CreateWorkloadIdentityFederationsTable},
	// Block 8: Users — core + all user-scoped tables grouped together
	{"029_create_users_table", migration.CreateUserTable},
	{"030_create_user_identities_table", migration.CreateUserIdentityTable},
	{"031_create_user_roles_table", migration.CreateUserRoleTable},
	{"032_create_user_tokens_table", migration.CreateUserTokenTable},
	{"033_create_user_otps_table", migration.CreateUserOTPsTable},
	{"034_create_user_settings_table", migration.CreateUserSettingsTable},
	{"035_create_profiles_table", migration.CreateProfileTable},
	{"036_create_user_mfa_backup_codes_table", migration.CreateUserMFABackupCodesTable},
	{"037_create_user_mfa_totp_secrets_table", migration.CreateUserMFATOTPSecretsTable},
	{"038_create_user_mfa_webauthn_credentials_table", migration.CreateUserMFAWebAuthnCredentialsTable},
	{"039_create_webauthn_challenges_table", migration.CreateWebAuthnChallengesTable},
	{"040_create_user_mfa_phones_table", migration.CreateUserMFAPhonesTable},
	{"041_create_user_mfa_emails_table", migration.CreateUserMFAEmailsTable},
	{"042_create_user_password_history_table", migration.CreateUserPasswordHistoryTable},
	{"043_create_user_lockouts_table", migration.CreateUserLockoutsTable},
	{"044_create_user_consents_table", migration.CreateUserConsentsTable},
	{"045_create_user_trusted_devices_table", migration.CreateUserTrustedDevicesTable},
	{"046_create_user_sessions_table", migration.CreateUserSessionsTable},
	{"047_create_account_link_requests_table", migration.CreateAccountLinkRequestsTable},
	{"048_create_data_erasure_requests_table", migration.CreateDataErasureRequestsTable},
	// Block 9: Tenant organisation & flows
	{"049_create_tenant_members_table", migration.CreateTenantMembersTable},
	{"050_create_registration_flows_table", migration.CreateRegistrationFlowTable},
	{"051_create_registration_flow_roles_table", migration.CreateRegistrationFlowRoleTable},
	{"052_create_invites_table", migration.CreateInvitesTable},
	// Block 10: Security
	{"053_create_security_settings_table", migration.CreateSecuritySettingsTable},
	{"054_create_ip_restriction_rules_table", migration.CreateIPRestrictionRulesTable},
	{"055_create_security_settings_audit_table", migration.CreateSecuritySettingsAuditTable},
	{"056_create_management_audit_log_table", migration.CreateManagementAuditLogTable},
	// Block 11: Templates
	{"057_create_email_templates_table", migration.CreateEmailTemplatesTable},
	{"058_create_sms_templates_table", migration.CreateSMSTemplatesTable},
	// Block 12: Auth events
	{"059_create_auth_events_table", migration.CreateAuthEventsTable},
	// Block 13: OAuth
	{"060_create_oauth_authorization_codes_table", migration.CreateOAuthAuthorizationCodesTable},
	{"061_create_oauth_refresh_tokens_table", migration.CreateOAuthRefreshTokensTable},
	{"062_create_oauth_consent_grants_table", migration.CreateOAuthConsentGrantsTable},
	{"063_create_oauth_consent_challenges_table", migration.CreateOAuthConsentChallengesTable},
	{"064_create_oauth_par_requests_table", migration.CreateOAuthPARRequestsTable},
	{"065_create_oauth_device_codes_table", migration.CreateOAuthDeviceCodesTable},
	{"066_create_oauth_ciba_requests_table", migration.CreateOAuthCIBARequestsTable},
	{"067_create_oauth_broker_sessions_table", migration.CreateOAuthBrokerSessionsTable},
	{"068_create_oauth_authorize_requests_table", migration.CreateOAuthAuthorizeRequestsTable},
	{"069_create_oauth_token_revocations_table", migration.CreateOAuthTokenRevocationsTable},
	{"070_create_oauth_token_exchanges_table", migration.CreateOAuthTokenExchangesTable},
	{"071_create_oauth_dpop_nonces_table", migration.CreateOAuthDPoPNoncesTable},
	// Block 14: Signing keys
	{"072_create_signing_keys_table", migration.CreateSigningKeysTable},
	// Block 15: Webhooks & event-driven integration
	{"073_create_event_types_table", migration.CreateEventTypesTable},
	{"074_create_webhook_endpoints_table", migration.CreateWebhookEndpointsTable},
	{"075_create_webhook_endpoint_events_table", migration.CreateWebhookEndpointEventsTable},
	{"076_create_event_routes_table", migration.CreateEventRoutesTable},
	{"077_create_tenant_event_types_table", migration.CreateTenantEventTypesTable},
	{"078_create_integration_event_outbox_table", migration.CreateIntegrationEventOutboxTable},
	{"079_create_webhook_delivery_history_table", migration.CreateWebhookDeliveryHistoryTable},
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
