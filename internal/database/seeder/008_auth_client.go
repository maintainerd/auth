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

func SeedAuthClients(db *gorm.DB, identityProviderID int64, authContainerID int64) {
	appHostName := os.Getenv("APP_PRIVATE_HOSTNAME")
	accountHostName := os.Getenv("ACCOUNT_HOSTNAME")

	clients := []model.AuthClient{
		{
			AuthClientUUID: uuid.New(),
			Name:           "traditional-default",
			DisplayName:    "Traditional Web App Default",
			ClientType:     "traditional",
			Domain:         strPtr(appHostName),
			ClientID:       strPtr(util.GenerateIdentifier(32)),
			ClientSecret:   strPtr(util.GenerateIdentifier(64)),
			RedirectURI:    strPtr(accountHostName + "/callback"),
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
			RedirectURI:    strPtr(accountHostName + "/callback"),
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
			RedirectURI:    strPtr(accountHostName + "://callback"),
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
			RedirectURI:    nil,
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
			Where("name = ? AND auth_container_id = ?", client.Name, authContainerID).
			First(&existing).Error

		if err == nil {
			log.Printf("⚠️ Auth client '%s' already exists, skipping", client.Name)
			continue
		}

		if err := db.Create(&client).Error; err != nil {
			log.Printf("❌ Failed to seed auth client '%s': %v", client.Name, err)
			continue
		}

		log.Printf("✅ Auth client '%s' seeded successfully", client.Name)
	}
}
