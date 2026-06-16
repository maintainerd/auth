package seeder

import (
	"errors"
	"log/slog"

	"github.com/google/uuid"
	model "github.com/maintainerd/auth/internal/iam"
	"gorm.io/gorm"
)

func SeedService(db *gorm.DB, tenantID int64, appVersion string) (*model.Service, error) {
	var service model.Service

	if appVersion == "" {
		slog.Warn("Skipping Service seeding: version is empty")
		return &service, nil
	}

	// Services are tenant-scoped: look up (and seed) the "auth" service for THIS
	// tenant. The same service name exists independently in every tenant.
	err := db.Where("name = ? AND tenant_id = ?", "auth", tenantID).First(&service).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		service = model.Service{
			ServiceUUID: uuid.New(),
			TenantID:    tenantID,
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

		slog.Info("Default Service seeded", "tenant_id", tenantID, "version", appVersion)
		return &service, nil
	}
	if err != nil {
		slog.Error("Error checking existing Default Service", "error", err)
		return nil, err
	}

	slog.Info("Default Service already exists, skipping")
	return &service, nil
}
