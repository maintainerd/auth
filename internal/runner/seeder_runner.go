package runner

import (
	"log"

	"github.com/maintainerd/auth/internal/database/seeder"
	"gorm.io/gorm"
)

func RunSeeders(db *gorm.DB, appVersion string, organizationID int64) error {
	log.Println("🏃 Running default seeders...")

	// Organization is now created via API, not seeded
	// Use the provided organizationID from the setup service

	// 001: Seed service
	service, err := seeder.SeedService(db, appVersion)
	if err != nil {
		log.Printf("❌ Failed to seed service: %v", err)
		return err
	}

	// 002: Link organization to service
	_, err = seeder.SeedOrganizationService(db, organizationID, service.ServiceID)
	if err != nil {
		log.Printf("❌ Failed to seed organization_service: %v", err)
		return err
	}

	// 003: Seed API
	api, err := seeder.SeedAPI(db, service.ServiceID)
	if err != nil {
		log.Printf("❌ Failed to seed api: %v", err)
		return err
	}

	// 004: Seed permissions
	if err := seeder.SeedPermissions(db, api.APIID); err != nil {
		log.Printf("❌ Failed to seed permissions: %v", err)
		return err
	}

	// 005: Seed auth container
	authContainer, err := seeder.SeedAuthContainer(db, organizationID)
	if err != nil {
		log.Printf("❌ Failed to seed auth container: %v", err)
		return err
	}

	// 006: Seed identity providers
	identityProvider, err := seeder.SeedIdentityProviders(db, authContainer.AuthContainerID)
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
	roles, err := seeder.SeedRoles(db, authContainer.AuthContainerID)
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
