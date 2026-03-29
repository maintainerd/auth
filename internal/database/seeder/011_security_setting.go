package seeder

import (
	"log/slog"

	"github.com/maintainerd/auth/internal/model"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// SeedSecuritySettings creates default security settings for a tenant
func SeedSecuritySettings(db *gorm.DB, tenantID int64) error {
	// Check if security settings already exist for this tenant
	var existing model.SecuritySetting
	err := db.Where("tenant_id = ?", tenantID).First(&existing).Error
	if err == nil {
		slog.Info("Security settings already exist, skipping", "tenant_id", tenantID)
		return nil
	}

	// Create default security settings with empty JSONB configs
	securitySetting := model.SecuritySetting{
		TenantID:       tenantID,
		GeneralConfig:  datatypes.JSON([]byte("{}")),
		PasswordConfig: datatypes.JSON([]byte("{}")),
		SessionConfig:  datatypes.JSON([]byte("{}")),
		ThreatConfig:   datatypes.JSON([]byte("{}")),
		IpConfig:       datatypes.JSON([]byte("{}")),
		Version:        1,
	}

	if err := db.Create(&securitySetting).Error; err != nil {
		slog.Error("Failed to create security settings", "tenant_id", tenantID, "error", err)
		return err
	}

	slog.Info("Security settings seeded", "tenant_id", tenantID)
	return nil
}
