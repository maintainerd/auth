package seeder

import (
	"log/slog"

	model "github.com/maintainerd/auth/internal/secpolicy"
	"gorm.io/gorm"
)

// SeedSecuritySettings creates default security settings for a tenant.
func SeedSecuritySettings(db *gorm.DB, tenantID int64) error {
	// Check if security settings already exist for this tenant
	var existing model.SecuritySetting
	err := db.Where("tenant_id = ?", tenantID).First(&existing).Error
	if err == nil {
		if model.ApplySecuritySettingDefaults(&existing) {
			if err := db.Save(&existing).Error; err != nil {
				slog.Error("Failed to backfill security setting defaults", "tenant_id", tenantID, "error", err)
				return err
			}
		}
		slog.Info("Security settings already exist, skipping", "tenant_id", tenantID)
		return nil
	}

	securitySetting := model.NewDefaultSecuritySetting(tenantID)

	if err := db.Create(&securitySetting).Error; err != nil {
		slog.Error("Failed to create security settings", "tenant_id", tenantID, "error", err)
		return err
	}

	slog.Info("Security settings seeded", "tenant_id", tenantID)
	return nil
}
