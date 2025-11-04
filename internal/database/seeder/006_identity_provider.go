package seeder

import (
	"log"

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
		log.Printf("⚠️ Default identity provider already exists (ID: %d)", existing.IdentityProviderID)
		return &existing, nil
	}

	if err != gorm.ErrRecordNotFound {
		log.Printf("❌ Error checking default identity provider: %v", err)
		return nil, err
	}

	// Create a new default identity provider
	provider := model.IdentityProvider{
		IdentityProviderUUID: uuid.New(),
		Name:                 "default",
		DisplayName:          "Default Identity Provider",
		ProviderType:         "default",
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
		IsActive:  true,
		IsDefault: true,
	}

	if err := db.Create(&provider).Error; err != nil {
		log.Printf("❌ Failed to seed default identity provider: %v", err)
		return nil, err
	}

	log.Printf("✅ Default identity provider seeded successfully (ID: %d)", provider.IdentityProviderID)
	return &provider, nil
}
