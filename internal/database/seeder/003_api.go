package seeder

import (
	"fmt"
	"log/slog"

	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/util"
	"gorm.io/gorm"
)

func SeedAPI(db *gorm.DB, tenantID, serviceID int64) (*model.API, error) {
	var existing model.API
	err := db.Where("name = ? AND tenant_id = ?", "auth", tenantID).First(&existing).Error

	if err == nil {
		slog.Info("API 'auth' already exists, skipping")
		return &existing, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("failed to check existing API: %w", err)
	}

	api := &model.API{
		TenantID:    tenantID,
		Name:        "auth",
		DisplayName: "Auth API",
		APIType:     "rest",
		Description: "API for authentication",
		Identifier:  fmt.Sprintf("api-%s", util.GenerateIdentifier(12)),
		Status:      "active",
		IsSystem:    true,
		ServiceID:   serviceID,
	}

	if err := db.Create(api).Error; err != nil {
		return nil, fmt.Errorf("failed to seed API: %w", err)
	}

	slog.Info("Auth API seeded successfully")
	return api, nil
}
