package runner

import (
	"log"

	"github.com/maintainerd/auth/db/seeder/v1/defaultseeder"
	"gorm.io/gorm"
)

func RunDefaultSeeders(db *gorm.DB, targetVersion string) {
	log.Println("🌱 Running default seeders...")

	switch targetVersion {
	case "v1":
		log.Println("🌱 Applying v1 default seeders...")
		defaultseeder.SeedRoles(db)
	default:
		log.Fatalf("❌ Unknown default seeder version: %s", targetVersion)
	}

	log.Println("✅ Default seeding process completed.")
}
