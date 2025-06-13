package seeder

import (
	"log"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func SeedService(db *gorm.DB, appVersion string) (model.Service, error) {
	var service model.Service

	if appVersion == "" {
		log.Printf("⚠️ Skipping Service seeding: version is empty")
		return service, nil
	}

	err := db.Where("service_name = ?", "auth").First(&service).Error

	if err == gorm.ErrRecordNotFound {
		service = model.Service{
			ServiceUUID: uuid.New(),
			ServiceName: "auth",
			DisplayName: "Auth Service",
			Description: "Auth system service",
			ServiceType: "default",
			Version:     appVersion,
			Config:      datatypes.JSON([]byte(`{}`)),
			IsActive:    true,
			IsDefault:   true,
		}

		if err := db.Create(&service).Error; err != nil {
			log.Printf("❌ Failed to seed Default Service version '%s': %v", appVersion, err)
			return model.Service{}, err
		}

		log.Printf("✅ Default Service version '%s' seeded successfully", appVersion)
		return service, nil
	}

	if err != nil {
		log.Printf("❌ Error checking existing Default Service: %v", err)
		return model.Service{}, err
	}

	log.Printf("⚠️ Default Service already exists, skipping seeding")
	return service, nil
}
