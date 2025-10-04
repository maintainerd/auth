package seeder

import (
	"log"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

func SeedOrganization(db *gorm.DB) (*model.Organization, error) {
	// Seed organization
	var org model.Organization
	err := db.Where("name = ?", "Maintainerd").First(&org).Error
	if err == gorm.ErrRecordNotFound {
		// Create organization
		org = model.Organization{
			OrganizationUUID: uuid.New(),
			Name:             "Maintainerd",
			Description:      strPtr("Maintainerd organization"),
			Email:            strPtr("admin@maintainerd.com"),
			Phone:            strPtr("+1234567890"),
			IsActive:         true,
		}

		if err := db.Create(&org).Error; err != nil {
			log.Printf("❌ Failed to seed organization: %v", err)
			return nil, err
		}
		log.Printf("✅ Organization seeded successfully (ID: %d)", org.OrganizationID)
	} else if err != nil {
		log.Printf("❌ Error checking organization: %v", err)
		return nil, err
	} else {
		log.Printf("⚠️ Organization already exists (ID: %d)", org.OrganizationID)
	}

	return &org, nil
}

func strPtr(s string) *string {
	return &s
}
