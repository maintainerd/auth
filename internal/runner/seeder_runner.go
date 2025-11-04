package runner

import (
	"log"

	"github.com/maintainerd/auth/internal/database/seeder"
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

	// 002: Seed tenant
	tenant, err := seeder.SeedTenant(db)
	if err != nil {
		log.Printf("❌ Failed to seed tenant: %v", err)
		return err
	}

	// 003: Link tenant to service
	_, err = seeder.SeedTenantService(db, tenant.TenantID, service.ServiceID)
	if err != nil {
		log.Printf("❌ Failed to seed tenant_service: %v", err)
		return err
	}

	// 004: Seed API
	api, err := seeder.SeedAPI(db, service.ServiceID)
	if err != nil {
		log.Printf("❌ Failed to seed api: %v", err)
		return err
	}

	// 005: Seed permissions
	if err := seeder.SeedPermissions(db, api.APIID); err != nil {
		log.Printf("❌ Failed to seed permissions: %v", err)
		return err
	}

	// 006: Seed identity providers
	identityProvider, err := seeder.SeedIdentityProviders(db, tenant.TenantID)
	if err != nil {
		log.Printf("❌ Failed to seed identity provider: %v", err)
		return err
	}

	// 007: Seed auth clients
	if err := seeder.SeedAuthClients(db, identityProvider.IdentityProviderID); err != nil {
		log.Printf("❌ Failed to seed auth clients: %v", err)
		return err
	}

	// 008: Seed auth client redirect URIs
	if err := seeder.SeedAuthClientRedirectURIs(db, identityProvider.IdentityProviderID); err != nil {
		log.Printf("❌ Failed to seed auth client redirect URIs: %v", err)
		return err
	}

	// 009: Seed roles
	roles, err := seeder.SeedRoles(db, tenant.TenantID)
	if err != nil {
		log.Printf("❌ Failed to seed roles: %v", err)
		return err
	}

	// 010: Seed role permissions
	if err := seeder.SeedRolePermissions(db, roles); err != nil {
		log.Printf("❌ Failed to seed role permissions: %v", err)
		return err
	}

	// 011: Seed email templates
	if err := seeder.SeedEmailTemplates(db); err != nil {
		log.Printf("❌ Failed to seed email templates: %v", err)
		return err
	}

	log.Println("✅ Default seeding process completed.")
	return nil
}
