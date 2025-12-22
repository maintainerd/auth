package seeder

import (
	"log"

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
		log.Printf("✅ Security settings already exist for tenant ID: %d", tenantID)
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
		log.Printf("❌ Failed to create security settings for tenant ID %d: %v", tenantID, err)
		return err
	}

	log.Printf("✅ Created security settings for tenant ID: %d", tenantID)
	return nil
}
