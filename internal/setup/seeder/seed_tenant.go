package seeder

import (
	"fmt"
	"log/slog"

	"gorm.io/gorm"
)

// seedServiceVersion is the version stamped on the per-tenant "auth" service
// record when a tenant is seeded.
const seedServiceVersion = "v0.1.0"

// SeedTenant seeds the full per-tenant baseline for an existing tenant:
// the tenant↔service link, API, permissions (+ api_permissions backfill),
// control policy, identity provider, client(s) and their URIs, roles
// (registered + super-admin), role-permission grants, email templates,
// security settings, and branding.
//
// It is idempotent and fully tenant-scoped, and is the single source of truth
// for what a tenant's baseline looks like. It is used both by first-run
// bootstrap (RunAll, for the system tenant) and by admin-side tenant creation
// (for new regular tenants).
//
// Everything SeedTenant creates is tenant-scoped, including the tenant's own
// "auth" service record. The only globally-shared seed is the integration
// event-type catalog, which is seeded per tenant separately by RunAll/callers.
func SeedTenant(db *gorm.DB, tenantID int64) error {
	// Seed this tenant's own "auth" service (services are tenant-scoped).
	service, err := SeedService(db, tenantID, seedServiceVersion)
	if err != nil {
		return fmt.Errorf("seed service: %w", err)
	}

	if _, err := SeedTenantService(db, tenantID, service.ServiceID); err != nil {
		return fmt.Errorf("seed tenant_service: %w", err)
	}

	api, err := SeedAPI(db, tenantID, service.ServiceID)
	if err != nil {
		return fmt.Errorf("seed api: %w", err)
	}

	if err := SeedPermissions(db, tenantID, api.APIID); err != nil {
		return fmt.Errorf("seed permissions: %w", err)
	}

	if err := SeedAPIPermissions(db, tenantID); err != nil {
		return fmt.Errorf("seed api permissions: %w", err)
	}

	if err := SeedControlPolicy(db, tenantID); err != nil {
		return fmt.Errorf("seed control policy: %w", err)
	}

	identityProvider, err := SeedIdentityProviders(db, tenantID)
	if err != nil {
		return fmt.Errorf("seed identity provider: %w", err)
	}

	if err := SeedClients(db, tenantID, identityProvider.IdentityProviderID); err != nil {
		return fmt.Errorf("seed clients: %w", err)
	}

	if err := SeedClientURIs(db, tenantID, identityProvider.IdentityProviderID); err != nil {
		return fmt.Errorf("seed client URIs: %w", err)
	}

	roles, err := SeedRoles(db, tenantID)
	if err != nil {
		return fmt.Errorf("seed roles: %w", err)
	}

	if err := SeedRolePermissions(db, roles); err != nil {
		return fmt.Errorf("seed role permissions: %w", err)
	}

	if err := SeedAuthFlows(db, tenantID); err != nil {
		return fmt.Errorf("seed auth flows: %w", err)
	}

	if err := SeedEmailTemplates(db, tenantID); err != nil {
		return fmt.Errorf("seed email templates: %w", err)
	}

	if err := SeedSMSTemplates(db, tenantID); err != nil {
		return fmt.Errorf("seed sms templates: %w", err)
	}

	if err := SeedSecuritySettings(db, tenantID); err != nil {
		return fmt.Errorf("seed security settings: %w", err)
	}

	if err := SeedBranding(db, tenantID); err != nil {
		return fmt.Errorf("seed branding: %w", err)
	}

	if err := SeedTenantSettings(db, tenantID); err != nil {
		return fmt.Errorf("seed tenant settings: %w", err)
	}

	// Event types are tenant-scoped: each tenant gets its own catalog copy.
	if err := SeedEventTypes(db, tenantID); err != nil {
		return fmt.Errorf("seed event types: %w", err)
	}

	slog.Info("Tenant baseline seeded", "tenant_id", tenantID)
	return nil
}
