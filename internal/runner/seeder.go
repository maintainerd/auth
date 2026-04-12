package runner

import (
	"log/slog"

	"github.com/maintainerd/auth/internal/database/seeder"
	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

var RunSeeders = runSeeders

func runSeeders(db *gorm.DB, appVersion string) error {
	slog.Info("Running default seeders")

	// 001: Seed service
	service, err := seeder.SeedService(db, appVersion)
	if err != nil {
		slog.Error("Failed to seed service", "error", err)
		return err
	}

	// 002: Get existing tenant (created by setup service)
	var tenant model.Tenant
	err = db.Where("is_system = ?", true).First(&tenant).Error
	if err != nil {
		slog.Error("Failed to find system tenant", "error", err)
		return err
	}
	slog.Info("Found default tenant", "tenant_id", tenant.TenantID)

	// 002: Link tenant to service
	_, err = seeder.SeedTenantService(db, tenant.TenantID, service.ServiceID)
	if err != nil {
		slog.Error("Failed to seed tenant_service", "error", err)
		return err
	}

	// 003: Seed API
	api, err := seeder.SeedAPI(db, tenant.TenantID, service.ServiceID)
	if err != nil {
		slog.Error("Failed to seed api", "error", err)
		return err
	}

	// 004: Seed permissions
	if err := seeder.SeedPermissions(db, tenant.TenantID, api.APIID); err != nil {
		slog.Error("Failed to seed permissions", "error", err)
		return err
	}

	// 005: Seed identity providers
	identityProvider, err := seeder.SeedIdentityProviders(db, tenant.TenantID)
	if err != nil {
		slog.Error("Failed to seed identity provider", "error", err)
		return err
	}

	// 006: Seed auth clients
	if err := seeder.SeedClients(db, tenant.TenantID, identityProvider.IdentityProviderID); err != nil {
		slog.Error("Failed to seed auth clients", "error", err)
		return err
	}

	// 007: Seed auth client URIs
	if err := seeder.SeedClientURIs(db, tenant.TenantID, identityProvider.IdentityProviderID); err != nil {
		slog.Error("Failed to seed auth client URIs", "error", err)
		return err
	}

	// 008: Seed roles
	roles, err := seeder.SeedRoles(db, tenant.TenantID)
	if err != nil {
		slog.Error("Failed to seed roles", "error", err)
		return err
	}

	// 009: Seed role permissions
	if err := seeder.SeedRolePermissions(db, roles); err != nil {
		slog.Error("Failed to seed role permissions", "error", err)
		return err
	}

	// 010: Seed email templates
	if err := seeder.SeedEmailTemplates(db, tenant.TenantID); err != nil {
		slog.Error("Failed to seed email templates", "error", err)
		return err
	}

	// 011: Seed security settings
	if err := seeder.SeedSecuritySettings(db, tenant.TenantID); err != nil {
		slog.Error("Failed to seed security settings", "error", err)
		return err
	}

	slog.Info("Default seeding process completed")
	return nil
}
