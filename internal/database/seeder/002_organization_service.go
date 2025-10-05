package seeder

import (
	"log"

	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

func SeedOrganizationService(db *gorm.DB, orgID, serviceID int64) (model.OrganizationService, error) {
	var orgService model.OrganizationService

	// Ensure valid IDs
	if orgID == 0 || serviceID == 0 {
		log.Printf("⚠️ Skipping OrganizationService seeding: missing IDs (orgID=%d, serviceID=%d)", orgID, serviceID)
		return orgService, nil
	}

	// Check if already linked
	err := db.Where("organization_id = ? AND service_id = ?", orgID, serviceID).
		First(&orgService).Error

	if err == nil {
		log.Printf("⚠️ OrganizationService already exists (ID: %d)", orgService.OrganizationServiceID)
		return orgService, nil
	}
	if err != gorm.ErrRecordNotFound {
		log.Printf("❌ Error checking OrganizationService: %v", err)
		return model.OrganizationService{}, err
	}

	orgService = model.OrganizationService{
		OrganizationID: orgID,
		ServiceID:      serviceID,
	}

	if err := db.Create(&orgService).Error; err != nil {
		log.Printf("❌ Failed to seed OrganizationService: %v", err)
		return model.OrganizationService{}, err
	}

	log.Printf("✅ OrganizationService seeded successfully (ID: %d)", orgService.OrganizationServiceID)
	return orgService, nil
}
