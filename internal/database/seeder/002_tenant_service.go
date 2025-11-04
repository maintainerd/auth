package seeder

import (
	"log"

	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

func SeedTenantService(db *gorm.DB, tenantID, serviceID int64) (model.TenantService, error) {
	var tenantService model.TenantService

	// Ensure valid IDs
	if tenantID == 0 || serviceID == 0 {
		log.Printf("⚠️ Skipping TenantService seeding: missing IDs (tenantID=%d, serviceID=%d)", tenantID, serviceID)
		return tenantService, nil
	}

	// Check if already linked
	err := db.Where("tenant_id = ? AND service_id = ?", tenantID, serviceID).
		First(&tenantService).Error

	if err == nil {
		log.Printf("⚠️ TenantService already exists (ID: %d)", tenantService.TenantServiceID)
		return tenantService, nil
	}
	if err != gorm.ErrRecordNotFound {
		log.Printf("❌ Error checking TenantService: %v", err)
		return model.TenantService{}, err
	}

	tenantService = model.TenantService{
		TenantID:  tenantID,
		ServiceID: serviceID,
	}

	if err := db.Create(&tenantService).Error; err != nil {
		log.Printf("❌ Failed to seed TenantService: %v", err)
		return model.TenantService{}, err
	}

	log.Printf("✅ TenantService seeded successfully (ID: %d)", tenantService.TenantServiceID)
	return tenantService, nil
}
