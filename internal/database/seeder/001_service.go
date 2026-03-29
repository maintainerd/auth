package seeder

import (
	"log/slog"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

func SeedService(db *gorm.DB, appVersion string) (*model.Service, error) {
	var service model.Service

	if appVersion == "" {
		slog.Warn("Skipping Service seeding: version is empty")
		return &service, nil
	}

	err := db.Where("name = ?", "auth").First(&service).Error
	if err == gorm.ErrRecordNotFound {
		service = model.Service{
			ServiceUUID: uuid.New(),
			Name:        "auth",
			DisplayName: "Auth Service",
			Description: "Auth system service",
			Version:     appVersion,
			Status:      "active",
			IsSystem:    true,
		}

		if err := db.Create(&service).Error; err != nil {
			slog.Error("Failed to seed Default Service", "version", appVersion, "error", err)
			return nil, err
		}

		slog.Info("Default Service seeded", "version", appVersion)
		return &service, nil
	}
	if err != nil {
		slog.Error("Error checking existing Default Service", "error", err)
		return nil, err
	}

	slog.Info("Default Service already exists, skipping")
	return &service, nil
}
