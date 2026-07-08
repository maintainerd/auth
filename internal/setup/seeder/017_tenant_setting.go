package seeder

import (
	"log/slog"

	model "github.com/maintainerd/maintainerd-auth/internal/tenant"
	"gorm.io/gorm"
)

func SeedTenantSettings(db *gorm.DB, tenantID int64) error {
	var existing model.TenantSetting
	err := db.Where("tenant_id = ?", tenantID).First(&existing).Error
	if err == nil {
		slog.Info("TenantSettings already exist, skipping", "tenant_id", tenantID)
		return nil
	}
	if err != gorm.ErrRecordNotFound {
		slog.Error("Error checking TenantSettings", "error", err)
		return err
	}

	setting := model.NewDefaultTenantSetting(tenantID)
	if err := db.Create(&setting).Error; err != nil {
		slog.Error("Failed to seed TenantSettings", "error", err)
		return err
	}

	slog.Info("TenantSettings seeded", "tenant_id", tenantID)
	return nil
}
