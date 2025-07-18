package seeder

import (
	"log"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

func SeedOrganization(db *gorm.DB) (*model.Organization, error) {
	var existing model.Organization

	// Check if a default organization already exists
	err := db.
		Where("is_default = ?", true).
		First(&existing).Error

	if err == nil {
		log.Printf("⚠️ Default organization already exists (ID: %d)", existing.OrganizationID)
		return &existing, nil
	}

	if err != gorm.ErrRecordNotFound {
		log.Printf("❌ Error checking default organization: %v", err)
		return nil, err
	}

	org := model.Organization{
		OrganizationUUID:    uuid.New(),
		Name:                "Default Organization",
		Description:         strPtr("This is the default organization used for initial setup."),
		Email:               strPtr("admin@example.com"),
		PhoneNumber:         strPtr("000-000-0000"),
		WebsiteURL:          strPtr("https://example.com"),
		LogoURL:             nil,
		ExternalReferenceID: nil,
		IsDefault:           true,
		IsActive:            true,
	}

	if err := db.Create(&org).Error; err != nil {
		log.Printf("❌ Failed to seed default organization: %v", err)
		return nil, err
	}

	log.Printf("✅ Default organization seeded successfully (ID: %d)", org.OrganizationID)
	return &org, nil
}

func strPtr(s string) *string {
	return &s
}
