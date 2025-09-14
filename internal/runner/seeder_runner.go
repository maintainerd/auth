package runner

import (
	"log"

	"github.com/maintainerd/auth/internal/database/seeder"
	"gorm.io/gorm"
)

func RunSeeders(db *gorm.DB, appVersion string) {
	log.Println("🏃 Running default seeders...")

	// Seed organization
	org, err := seeder.SeedOrganization(db)
	if err != nil {
		log.Fatal("❌ Failed to seed organization:", err)
	}

	// Seed service
	service, err := seeder.SeedService(db, appVersion)
	if err != nil {
		log.Fatal("❌ Failed to seed service:", err)
	}

	// Link organization to service
	_, err = seeder.SeedOrganizationService(db, org.OrganizationID, service.ServiceID)
	if err != nil {
		log.Fatal("❌ Failed to seed organization_service:", err)
	}

	// Seed API
	api, err := seeder.SeedAPI(db, service.ServiceID)
	if err != nil {
		log.Fatal("❌ Failed to seed api:", err)
	}

	// Seed permissions
	seeder.SeedPermissions(db, api.APIID)

	// Seed auth container
	authContainer, err := seeder.SeedAuthContainer(db, org.OrganizationID)
	if err != nil {
		log.Fatal("❌ Failed to seed auth container:", err)
	}

	// Seed identity providers
	identityProvider, err := seeder.SeedIdentityProviders(db, authContainer.AuthContainerID)
	if err != nil {
		log.Fatal("❌ Failed to seed identity provider:", err)
	}

	// Seed auth clients
	seeder.SeedAuthClients(db, identityProvider.IdentityProviderID)

	// Seed auth client redirect URIs
	seeder.SeedAuthClientRedirectURIs(db, identityProvider.IdentityProviderID)

	// Seed roles
	roles, err := seeder.SeedRoles(db, authContainer.AuthContainerID)
	if err != nil {
		log.Fatal("❌ Failed to seed roles:", err)
	}

	// Seed role permissions
	seeder.SeedRolePermissions(db, roles)

	// Seed email templates
	seeder.SeedEmailTemplates(db)

	log.Println("✅ Default seeding process completed.")
}
