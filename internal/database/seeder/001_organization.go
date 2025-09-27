package seeder

import (
	"log"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

func SeedOrganization(db *gorm.DB) (*model.Organization, error) {
	// Seed root organization first
	var rootOrg model.Organization
	err := db.Where("is_root = ?", true).First(&rootOrg).Error
	if err == gorm.ErrRecordNotFound {
		// Create root organization
		rootOrg = model.Organization{
			OrganizationUUID: uuid.New(),
			Name:             "Maintainerd Root",
			Description:      strPtr("Root organization for Maintainerd"),
			Email:            strPtr("root@maintainerd.com"),
			Phone:            strPtr("+1234567890"),
			IsRoot:           true,
			IsDefault:        false,
			IsActive:         true,
		}

		if err := db.Create(&rootOrg).Error; err != nil {
			log.Printf("❌ Failed to seed root organization: %v", err)
			return nil, err
		}
		log.Printf("✅ Root organization seeded successfully (ID: %d)", rootOrg.OrganizationID)
	} else if err != nil {
		log.Printf("❌ Error checking root organization: %v", err)
		return nil, err
	} else {
		log.Printf("⚠️ Root organization already exists (ID: %d)", rootOrg.OrganizationID)
	}

	// Seed default organization
	var defaultOrg model.Organization
	err = db.Where("is_default = ?", true).First(&defaultOrg).Error
	if err == gorm.ErrRecordNotFound {
		// Create default organization
		defaultOrg = model.Organization{
			OrganizationUUID: uuid.New(),
			Name:             "Maintainerd Default",
			Description:      strPtr("Default organization for Maintainerd"),
			Email:            strPtr("admin@maintainerd.com"),
			Phone:            strPtr("+1234567891"),
			IsRoot:           false,
			IsDefault:        true,
			IsActive:         true,
		}

		if err := db.Create(&defaultOrg).Error; err != nil {
			log.Printf("❌ Failed to seed default organization: %v", err)
			return nil, err
		}
		log.Printf("✅ Default organization seeded successfully (ID: %d)", defaultOrg.OrganizationID)
	} else if err != nil {
		log.Printf("❌ Error checking default organization: %v", err)
		return nil, err
	} else {
		log.Printf("⚠️ Default organization already exists (ID: %d)", defaultOrg.OrganizationID)
	}

	// Return the default organization for backward compatibility
	return &defaultOrg, nil
}

func strPtr(s string) *string {
	return &s
}
