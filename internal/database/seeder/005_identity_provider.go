package seeder

import (
	"log/slog"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/util"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func SeedIdentityProviders(db *gorm.DB, tenantID int64) (*model.IdentityProvider, error) {
	var existing model.IdentityProvider

	// Check if a default identity provider already exists
	err := db.
		Where("name = ? AND tenant_id = ?", "default", tenantID).
		First(&existing).Error

	if err == nil {
		slog.Info("Default identity provider already exists, skipping", "id", existing.IdentityProviderID)
		return &existing, nil
	}

	if err != gorm.ErrRecordNotFound {
		slog.Error("Error checking default identity provider", "error", err)
		return nil, err
	}

	// Create a new default identity provider
	provider := model.IdentityProvider{
		IdentityProviderUUID: uuid.New(),
		Name:                 "default",
		DisplayName:          "Built-in Authentication System",
		Provider:             "internal",
		ProviderType:         "identity",
		Identifier:           util.GenerateIdentifier(15),
		Config: datatypes.JSON([]byte(`{
			"allow_registration": true,
			"allow_login": true,
			"allow_password_reset": true,
			"require_email_verify": false,
			"require_phone_verify": false,
			"require_mfa": false,
			"allow_mfa_enrollment": true,
			"session_timeout_min": 60,
			"refresh_timeout_min": 1440,
			"max_login_attempts": 5,
			"lockout_duration_min": 15
		}`)),
		TenantID:  tenantID,
		Status:    "active",
		IsDefault: true,
		IsSystem:  true,
	}

	if err := db.Create(&provider).Error; err != nil {
		slog.Error("Failed to seed default identity provider", "error", err)
		return nil, err
	}

	slog.Info("Default identity provider seeded", "id", provider.IdentityProviderID)
	return &provider, nil
}
