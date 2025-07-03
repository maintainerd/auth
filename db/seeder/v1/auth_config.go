package seeder

import (
	"log"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

func SeedAuthConfigs(db *gorm.DB, appVersion string) {
	if appVersion == "" {
		log.Printf("⚠️ Skipping AuthConfig seeding: version is empty")
		return
	}

	var existing model.AuthConfig
	err := db.Where("version = ?", appVersion).First(&existing).Error

	if err == gorm.ErrRecordNotFound {
		authConfig := model.AuthConfig{
			AuthConfigUUID: uuid.New(),
			Version:        appVersion,
			IsActive:       true,
			IsApplied:      true,
		}

		if err := db.Create(&authConfig).Error; err != nil {
			log.Printf("❌ Failed to seed AuthConfig version '%s': %v", appVersion, err)
			return
		}

		log.Printf("✅ AuthConfig version '%s' seeded successfully", appVersion)
		return
	}

	if err != nil {
		log.Printf("❌ Error checking existing AuthConfig version '%s': %v", appVersion, err)
		return
	}

	log.Printf("⚠️ AuthConfig version '%s' already exists, skipping seeding", appVersion)
}
