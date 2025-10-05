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

func SeedAuthClients(db *gorm.DB, identityProviderID int64) error {
	appHostName := os.Getenv("APP_PRIVATE_HOSTNAME")

	clients := []model.AuthClient{
		{
			AuthClientUUID: uuid.New(),
			Name:           "traditional-default",
			DisplayName:    "Traditional Web App Default",
			ClientType:     "traditional",
			Domain:         strPtr(appHostName),
			ClientID:       strPtr(util.GenerateIdentifier(32)),
			ClientSecret:   strPtr(util.GenerateIdentifier(64)),
			Config: datatypes.JSON([]byte(`{
				"grant_types": ["authorization_code"],
				"response_type": "code",
				"pkce": false
			}`)),
			IsActive:           true,
			IsDefault:          true,
			IdentityProviderID: identityProviderID,
			CreatedAt:          time.Now(),
			UpdatedAt:          time.Now(),
		},
		{
			AuthClientUUID: uuid.New(),
			Name:           "spa-default",
			DisplayName:    "Single Page App Default",
			ClientType:     "spa",
			Domain:         strPtr(appHostName),
			ClientID:       strPtr(util.GenerateIdentifier(32)),
			ClientSecret:   nil,
			Config: datatypes.JSON([]byte(`{
				"grant_types": ["authorization_code"],
				"response_type": "code",
				"pkce": true
			}`)),
			IsActive:           true,
			IsDefault:          true,
			IdentityProviderID: identityProviderID,
			CreatedAt:          time.Now(),
			UpdatedAt:          time.Now(),
		},
		{
			AuthClientUUID: uuid.New(),
			Name:           "mobile-default",
			DisplayName:    "Native Mobile App Default",
			ClientType:     "native",
			Domain:         strPtr(appHostName),
			ClientID:       strPtr(util.GenerateIdentifier(32)),
			ClientSecret:   nil,
			Config: datatypes.JSON([]byte(`{
				"grant_types": ["authorization_code"],
				"response_type": "code",
				"pkce": true
			}`)),
			IsActive:           true,
			IsDefault:          true,
			IdentityProviderID: identityProviderID,
			CreatedAt:          time.Now(),
			UpdatedAt:          time.Now(),
		},
		{
			AuthClientUUID: uuid.New(),
			Name:           "m2m-default",
			DisplayName:    "Machine to Machine Default",
			ClientType:     "m2m",
			Domain:         strPtr(appHostName),
			ClientID:       strPtr(util.GenerateIdentifier(32)),
			ClientSecret:   strPtr(util.GenerateIdentifier(64)),
			Config: datatypes.JSON([]byte(`{
				"grant_types": ["client_credentials"]
			}`)),
			IsActive:           true,
			IsDefault:          true,
			IdentityProviderID: identityProviderID,
			CreatedAt:          time.Now(),
			UpdatedAt:          time.Now(),
		},
	}

	for _, client := range clients {
		var existing model.AuthClient
		err := db.
			Where("name = ? AND identity_provider_id = ?", client.Name, identityProviderID).
			First(&existing).Error

		if err == nil {
			// Update existing client - preserve existing IDs and UUID
			client.AuthClientID = existing.AuthClientID
			client.AuthClientUUID = existing.AuthClientUUID
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
