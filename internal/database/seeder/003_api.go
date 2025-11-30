package seeder

import (
	"fmt"

	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/util"
	"gorm.io/gorm"
)

func SeedAPI(db *gorm.DB, serviceID int64) (*model.API, error) {
	var existing model.API
	err := db.Where("name = ?", "auth").First(&existing).Error

	if err == nil {
		fmt.Println("⚠️ API 'auth' already exists, skipping seeding")
		return &existing, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("failed to check existing API: %w", err)
	}

	api := &model.API{
		Name:        "auth",
		DisplayName: "Auth API",
		APIType:     "rest",
		Description: "API for authentication",
		Identifier:  fmt.Sprintf("api-%s", util.GenerateIdentifier(12)),
		Status:      "active",
		IsDefault:   true,
		IsSystem:    true,
		ServiceID:   serviceID,
	}

	if err := db.Create(api).Error; err != nil {
		return nil, fmt.Errorf("failed to seed API: %w", err)
	}

	fmt.Println("✅ Auth API seeded successfully")
	return api, nil
}
