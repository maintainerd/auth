package runner

import (
	"log"

	"github.com/maintainerd/auth/internal/seeder"
	"gorm.io/gorm"
)

func RunDefaultSeeders(db *gorm.DB, appVersion string) {
	log.Println("🏃 Running default seeders...")
	seeder.SeedRoles(db)
	seeder.SeedService(db, appVersion)
	log.Println("✅ Default seeding process completed.")
}
