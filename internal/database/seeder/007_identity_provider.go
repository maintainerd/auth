package seeder

import (
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/util"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func SeedIdentityProviders(db *gorm.DB, authContainerID int64) (*model.IdentityProvider, error) {
	var existing model.IdentityProvider

	// Check if default provider already exists
	err := db.
		Where("name = ? AND auth_container_id = ?", "default", authContainerID).
		First(&existing).Error

	if err == nil {
		log.Printf("⚠️ Default identity provider already exists (ID: %d)", existing.AuthContainerID)
		return &existing, nil
	}

	if err != gorm.ErrRecordNotFound {
		log.Printf("❌ Error checking default auth container: %v", err)
		return nil, err
	}

	// Create new default provider
	provider := model.IdentityProvider{
		IdentityProviderUUID: uuid.New(),
		Name:                 "default",
		DisplayName:          "Default",
		ProviderType:         "default",
		Identifier:           fmt.Sprintf("auth-api-%s", util.GenerateIdentifier(12)),
		Config:               datatypes.JSON([]byte(`{}`)),
		IsActive:             true,
		IsDefault:            true,
		AuthContainerID:      authContainerID,
		CreatedAt:            time.Now(),
		UpdatedAt:            time.Now(),
	}

	if err := db.Create(&provider).Error; err != nil {
		log.Printf("❌ Failed to seed default identity provider: %v", err)
		return nil, err
	}

	log.Printf("✅ Default auth container seeded successfully (ID: %d)", provider.IdentityProviderID)
	return &provider, nil
}
