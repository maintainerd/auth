package seeder

import (
	"log/slog"

	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

func SeedTenantService(db *gorm.DB, tenantID, serviceID int64) (model.TenantService, error) {
	var tenantService model.TenantService

	// Ensure valid IDs
	if tenantID == 0 || serviceID == 0 {
		slog.Warn("Skipping TenantService seeding: missing IDs", "tenant_id", tenantID, "service_id", serviceID)
		return tenantService, nil
	}

	// Check if already linked
	err := db.Where("tenant_id = ? AND service_id = ?", tenantID, serviceID).
		First(&tenantService).Error

	if err == nil {
		slog.Info("TenantService already exists, skipping", "id", tenantService.TenantServiceID)
		return tenantService, nil
	}
	if err != gorm.ErrRecordNotFound {
		slog.Error("Error checking TenantService", "error", err)
		return model.TenantService{}, err
	}

	tenantService = model.TenantService{
		TenantID:  tenantID,
		ServiceID: serviceID,
	}

	if err := db.Create(&tenantService).Error; err != nil {
		slog.Error("Failed to seed TenantService", "error", err)
		return model.TenantService{}, err
	}

	slog.Info("TenantService seeded", "id", tenantService.TenantServiceID)
	return tenantService, nil
}
