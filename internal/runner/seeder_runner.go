package runner

import (
	"log"

	"github.com/maintainerd/auth/internal/database/seeder"
	"gorm.io/gorm"
)

func RunSeeders(db *gorm.DB, appVersion string) {
	log.Println("🏃 Running default seeders...")

	// Seed service
	service, err := seeder.SeedService(db, appVersion)
	if err != nil {
		log.Fatal("❌ Failed to seed service:", err)
	}

	// Seed organization
	org, err := seeder.SeedOrganization(db)
	if err != nil {
		log.Fatal("❌ Failed to seed organization:", err)
	}

	// Seed auth container
	authContainer, err := seeder.SeedAuthContainer(db, org.OrganizationID)
	if err != nil {
		log.Fatal("❌ Failed to seed auth container:", err)
	}

	// Seed API
	api, err := seeder.SeedAPI(db, service.ServiceID, authContainer.AuthContainerID)
	if err != nil {
		log.Fatal("❌ Failed to seed api:", err)
	}

	// Seed permissions
	seeder.SeedPermissions(db, api.APIID, authContainer.AuthContainerID)

	// Seed identity providers
	identityProvider, err := seeder.SeedIdentityProviders(db, authContainer.AuthContainerID)
	if err != nil {
		log.Fatal("❌ Failed to seed identity provider:", err)
	}

	// Seed auth clients
	seeder.SeedAuthClients(db, identityProvider.IdentityProviderID, authContainer.AuthContainerID)

	// Seed roles
	roles, err := seeder.SeedRoles(db, authContainer.AuthContainerID)
	if err != nil {
		log.Fatal("❌ Failed to seed roles:", err)
	}

	// Seed role permissions
	seeder.SeedRolePermissions(db, roles, authContainer.AuthContainerID)

	log.Println("✅ Default seeding process completed.")
}
