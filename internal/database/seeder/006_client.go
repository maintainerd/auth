package seeder

import (
	"log"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/util"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func SeedClients(db *gorm.DB, tenantID int64, identityProviderID int64) error {
	appHostName := os.Getenv("APP_PRIVATE_HOSTNAME")

	clients := []model.Client{
		{
			ClientUUID:   uuid.New(),
			TenantID:     tenantID,
			Name:         "traditional-default",
			DisplayName:  "Traditional Web App Default",
			ClientType:   "traditional",
			Domain:       strPtr(appHostName),
			Identifier:   strPtr(util.GenerateIdentifier(32)),
			Secret:       strPtr(util.GenerateIdentifier(64)),
			Config: datatypes.JSON([]byte(`{
				"grant_types": ["authorization_code"],
				"response_type": "code",
				"pkce": false
			}`)),
			Status:             "active",
			IsDefault:          true,
			IsSystem:           true,
			IdentityProviderID: identityProviderID,
			CreatedAt:          time.Now(),
			UpdatedAt:          time.Now(),
		},
		{
			ClientUUID:   uuid.New(),
			TenantID:     tenantID,
			Name:         "spa-default",
			DisplayName:  "Single Page App Default",
			ClientType:   "spa",
			Domain:       strPtr(appHostName),
			Identifier:   strPtr(util.GenerateIdentifier(32)),
			Secret:       nil,
			Config: datatypes.JSON([]byte(`{
				"grant_types": ["authorization_code"],
				"response_type": "code",
				"pkce": true
			}`)),
			Status:             "active",
			IsDefault:          true,
			IsSystem:           true,
			IdentityProviderID: identityProviderID,
			CreatedAt:          time.Now(),
			UpdatedAt:          time.Now(),
		},
		{
			ClientUUID:   uuid.New(),
			TenantID:     tenantID,
			Name:         "mobile-default",
			DisplayName:  "Mobile App Default",
			ClientType:   "mobile",
			Domain:       strPtr(appHostName),
			Identifier:   strPtr(util.GenerateIdentifier(32)),
			Secret:       nil,
			Config: datatypes.JSON([]byte(`{
				"grant_types": ["authorization_code"],
				"response_type": "code",
				"pkce": true
			}`)),
			Status:             "active",
			IsDefault:          true,
			IsSystem:           true,
			IdentityProviderID: identityProviderID,
			CreatedAt:          time.Now(),
			UpdatedAt:          time.Now(),
		},
		{
			ClientUUID:   uuid.New(),
			TenantID:     tenantID,
			Name:         "m2m-default",
			DisplayName:  "Machine to Machine Default",
			ClientType:   "m2m",
			Domain:       strPtr(appHostName),
			Identifier:   strPtr(util.GenerateIdentifier(32)),
			Secret:       strPtr(util.GenerateIdentifier(64)),
			Config: datatypes.JSON([]byte(`{
				"grant_types": ["client_credentials"]
			}`)),
			Status:             "active",
			IsDefault:          true,
			IsSystem:           true,
			IdentityProviderID: identityProviderID,
			CreatedAt:          time.Now(),
			UpdatedAt:          time.Now(),
		},
	}

	for _, client := range clients {
		var existing model.Client
		err := db.
			Where("name = ? AND identity_provider_id = ? AND tenant_id = ?", client.Name, identityProviderID, tenantID).
			First(&existing).Error

		if err == nil {
			// Update existing client - preserve existing IDs and UUID
			client.Identifier = existing.Identifier
			client.ClientUUID = existing.ClientUUID
			if err := db.Save(&client).Error; err != nil {
				log.Printf("❌ Failed to update auth client '%s': %v", client.Name, err)
			} else {
				log.Printf("🔄 Auth client '%s' updated", client.Name)
			}
			continue
		}

		if err == gorm.ErrRecordNotFound {
			// Create new client
			if err := db.Create(&client).Error; err != nil {
				log.Printf("❌ Failed to create auth client '%s': %v", client.Name, err)
				continue
			}
			log.Printf("✅ Auth client '%s' created", client.Name)
			continue
		}

		// Unexpected error
		log.Printf("❌ Failed lookup for auth client '%s': %v", client.Name, err)
	}

	return nil
}

func strPtr(s string) *string {
	return &s
}
