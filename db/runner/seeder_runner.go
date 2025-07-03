package runner

import (
	"log"

	"github.com/maintainerd/auth/db/seeder/v1"
	"gorm.io/gorm"
)

func RunDefaultSeeders(db *gorm.DB, appVersion string) {
	log.Println("🏃 Running default seeders...")
	seeder.SeedRoles(db)
	seeder.SeedAuthConfigs(db, appVersion)
	log.Println("✅ Default seeding process completed.")
}
