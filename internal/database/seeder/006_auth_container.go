package seeder

import (
	"log"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/util"
	"gorm.io/gorm"
)

func SeedAuthContainer(db *gorm.DB, organizationID int64) (*model.AuthContainer, error) {
	var existing model.AuthContainer

	// Check if a default auth container already exists
	err := db.
		Where("is_default = ? AND organization_id = ?", true, organizationID).
		First(&existing).Error

	if err == nil {
		log.Printf("⚠️ Default auth container already exists (ID: %d)", existing.AuthContainerID)
		return &existing, nil
	}

	if err != gorm.ErrRecordNotFound {
		log.Printf("❌ Error checking default auth container: %v", err)
		return nil, err
	}

	// Create a new default auth container
	container := model.AuthContainer{
		AuthContainerUUID: uuid.New(),
		Name:              "Default Container",
		Description:       "This is the default authentication container",
		Identifier:        util.GenerateIdentifier(15),
		OrganizationID:    organizationID,
		IsActive:          true,
		IsPublic:          true,
		IsDefault:         true,
	}

	if err := db.Create(&container).Error; err != nil {
		log.Printf("❌ Failed to seed default auth container: %v", err)
		return nil, err
	}

	log.Printf("✅ Default auth container seeded successfully (ID: %d)", container.AuthContainerID)
	return &container, nil
}
