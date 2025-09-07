package seeder

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/util"
	"gorm.io/gorm"
)

func SeedAPI(db *gorm.DB, serviceID, authContainerID int64) (*model.API, error) {
	var existing model.API
	err := db.Where("name = ?", "auth-api").First(&existing).Error

	if err == nil {
		fmt.Println("⚠️ API 'auth-api' already exists, skipping seeding")
		return &existing, nil
	}

	if err != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("failed to check existing API: %w", err)
	}

	api := &model.API{
		APIUUID:     uuid.New(),
		Name:        "auth",
		DisplayName: "Auth API",
		APIType:     "default",
		Description: "API for authentication",
		Identifier:  fmt.Sprintf("auth-api-%s", util.GenerateIdentifier(12)),
		IsActive:    true,
		IsDefault:   true,
		ServiceID:   serviceID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := db.Create(api).Error; err != nil {
		return nil, fmt.Errorf("failed to seed API: %w", err)
	}

	fmt.Println("✅ Auth API seeded successfully")
	return api, nil
}
