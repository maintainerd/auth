package seeder

import (
	"log/slog"

	"github.com/maintainerd/auth/internal/model"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// SeedSecuritySettings creates default security settings for a tenant
func SeedSecuritySettings(db *gorm.DB, userPoolID int64) error {
	// Check if security settings already exist for this tenant
	var existing model.SecuritySetting
	err := db.Where("user_pool_id = ?", userPoolID).First(&existing).Error
	if err == nil {
		slog.Info("Security settings already exist, skipping", "user_pool_id", userPoolID)
		return nil
	}

	// Create default security settings with empty JSONB configs
	securitySetting := model.SecuritySetting{
		UserPoolID:         userPoolID,
		MFAConfig:          datatypes.JSON([]byte("{}")),
		PasswordConfig:     datatypes.JSON([]byte("{}")),
		SessionConfig:      datatypes.JSON([]byte("{}")),
		ThreatConfig:       datatypes.JSON([]byte("{}")),
		LockoutConfig:      datatypes.JSON([]byte("{}")),
		RegistrationConfig: datatypes.JSON([]byte("{}")),
		TokenConfig:        datatypes.JSON([]byte("{}")),
		Version:            1,
	}

	if err := db.Create(&securitySetting).Error; err != nil {
		slog.Error("Failed to create security settings", "user_pool_id", userPoolID, "error", err)
		return err
	}

	slog.Info("Security settings seeded", "user_pool_id", userPoolID)
	return nil
}
