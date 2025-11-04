package seeder

import (
	"log"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/util"
	"gorm.io/gorm"
)

func SeedTenant(db *gorm.DB) (*model.Tenant, error) {
	var existing model.Tenant

	// Check if a default tenant already exists
	err := db.
		Where("is_default = ?", true).
		First(&existing).Error

	if err == nil {
		log.Printf("⚠️ Default tenant already exists (ID: %d)", existing.TenantID)
		return &existing, nil
	}

	if err != gorm.ErrRecordNotFound {
		log.Printf("❌ Error checking default tenant: %v", err)
		return nil, err
	}

	// Create a new default tenant
	tenant := model.Tenant{
		TenantUUID:  uuid.New(),
		Name:        "Default Tenant",
		Description: "This is the default tenant",
		Identifier:  util.GenerateIdentifier(15),
		IsActive:    true,
		IsPublic:    true,
		IsDefault:   true,
	}

	if err := db.Create(&tenant).Error; err != nil {
		log.Printf("❌ Failed to seed default tenant: %v", err)
		return nil, err
	}

	log.Printf("✅ Default tenant seeded successfully (ID: %d)", tenant.TenantID)
	return &tenant, nil
}
