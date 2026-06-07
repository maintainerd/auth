package seeder

import (
	"log/slog"

	"github.com/maintainerd/auth/internal/tenant"
	"gorm.io/gorm"
)

func RunAll(db *gorm.DB, appVersion string) error {
	slog.Info("Running default seeders")

	service, err := SeedService(db, appVersion)
	if err != nil {
		slog.Error("Failed to seed service", "error", err)
		return err
	}

	var sysTenant tenant.Tenant
	err = db.Where("is_system = ?", true).First(&sysTenant).Error
	if err != nil {
		slog.Error("Failed to find system tenant", "error", err)
		return err
	}
	slog.Info("Found default tenant", "tenant_id", sysTenant.TenantID)

	_, err = SeedTenantService(db, sysTenant.TenantID, service.ServiceID)
	if err != nil {
		slog.Error("Failed to seed tenant_service", "error", err)
		return err
	}

	api, err := SeedAPI(db, sysTenant.TenantID, service.ServiceID)
	if err != nil {
		slog.Error("Failed to seed api", "error", err)
		return err
	}

	if err := SeedPermissions(db, sysTenant.TenantID, api.APIID); err != nil {
		slog.Error("Failed to seed permissions", "error", err)
		return err
	}

	if err := SeedAPIPermissions(db, sysTenant.TenantID); err != nil {
		slog.Error("Failed to seed api permissions", "error", err)
		return err
	}

	if err := SeedControlPolicy(db, sysTenant.TenantID); err != nil {
		slog.Error("Failed to seed control policy", "error", err)
		return err
	}

	identityProvider, err := SeedIdentityProviders(db, sysTenant.TenantID)
	if err != nil {
		slog.Error("Failed to seed identity provider", "error", err)
		return err
	}

	if err := SeedClients(db, sysTenant.TenantID, identityProvider.IdentityProviderID); err != nil {
		slog.Error("Failed to seed auth clients", "error", err)
		return err
	}

	if err := SeedClientURIs(db, sysTenant.TenantID, identityProvider.IdentityProviderID); err != nil {
		slog.Error("Failed to seed auth client URIs", "error", err)
		return err
	}

	roles, err := SeedRoles(db, sysTenant.TenantID)
	if err != nil {
		slog.Error("Failed to seed roles", "error", err)
		return err
	}

	if err := SeedRolePermissions(db, roles); err != nil {
		slog.Error("Failed to seed role permissions", "error", err)
		return err
	}

	if err := SeedEmailTemplates(db, sysTenant.TenantID); err != nil {
		slog.Error("Failed to seed email templates", "error", err)
		return err
	}

	if err := SeedSecuritySettings(db, sysTenant.TenantID); err != nil {
		slog.Error("Failed to seed security settings", "error", err)
		return err
	}

	// Seed integration event types catalog
	if err := SeedEventTypes(db); err != nil {
		slog.Error("Failed to seed event types", "error", err)
		return err
	}

	slog.Info("Default seeding process completed")
	return nil
}
