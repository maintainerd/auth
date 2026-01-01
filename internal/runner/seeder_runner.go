package runner

import (
	"log"

	"github.com/maintainerd/auth/internal/database/seeder"
	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

func RunSeeders(db *gorm.DB, appVersion string) error {
	log.Println("🏃 Running default seeders...")

	// 001: Seed service
	service, err := seeder.SeedService(db, appVersion)
	if err != nil {
		log.Printf("❌ Failed to seed service: %v", err)
		return err
	}

	// 002: Get existing tenant (created by setup service)
	var tenant model.Tenant
	err = db.Where("is_default = ?", true).First(&tenant).Error
	if err != nil {
		log.Printf("❌ Failed to find default tenant: %v", err)
		return err
	}
	log.Printf("✅ Found default tenant (ID: %d)", tenant.TenantID)

	// 002: Link tenant to service
	_, err = seeder.SeedTenantService(db, tenant.TenantID, service.ServiceID)
	if err != nil {
		log.Printf("❌ Failed to seed tenant_service: %v", err)
		return err
	}

	// 003: Seed API
	api, err := seeder.SeedAPI(db, tenant.TenantID, service.ServiceID)
	if err != nil {
		log.Printf("❌ Failed to seed api: %v", err)
		return err
	}

	// 004: Seed permissions
	if err := seeder.SeedPermissions(db, tenant.TenantID, api.APIID); err != nil {
		log.Printf("❌ Failed to seed permissions: %v", err)
		return err
	}

	// 005: Seed identity providers
	identityProvider, err := seeder.SeedIdentityProviders(db, tenant.TenantID)
	if err != nil {
		log.Printf("❌ Failed to seed identity provider: %v", err)
		return err
	}

	// 006: Seed auth clients
	if err := seeder.SeedAuthClients(db, tenant.TenantID, identityProvider.IdentityProviderID); err != nil {
		log.Printf("❌ Failed to seed auth clients: %v", err)
		return err
	}

	// 007: Seed auth client URIs
	if err := seeder.SeedAuthClientURIs(db, tenant.TenantID, identityProvider.IdentityProviderID); err != nil {
		log.Printf("❌ Failed to seed auth client URIs: %v", err)
		return err
	}

	// 008: Seed roles
	roles, err := seeder.SeedRoles(db, tenant.TenantID)
	if err != nil {
		log.Printf("❌ Failed to seed roles: %v", err)
		return err
	}

	// 009: Seed role permissions
	if err := seeder.SeedRolePermissions(db, roles); err != nil {
		log.Printf("❌ Failed to seed role permissions: %v", err)
		return err
	}

	// 010: Seed email templates
	if err := seeder.SeedEmailTemplates(db); err != nil {
		log.Printf("❌ Failed to seed email templates: %v", err)
		return err
	}

	// 011: Seed security settings
	if err := seeder.SeedSecuritySettings(db, tenant.TenantID); err != nil {
		log.Printf("❌ Failed to seed security settings: %v", err)
		return err
	}

	log.Println("✅ Default seeding process completed.")
	return nil
}
