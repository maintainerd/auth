package seeder

import (
	"log"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

func SeedAuthClientRedirectURIs(db *gorm.DB, identityProviderID int64) error {
	appHostName := os.Getenv("APP_PRIVATE_HOSTNAME")

	// Map of client name -> redirect URIs
	redirects := map[string][]string{
		"traditional-default": {
			"http://" + appHostName + "/callback",
			"https://" + appHostName + "/callback",
		},
		"spa-default": {
			"http://localhost:3000/callback",
			"https://" + appHostName + "/callback",
		},
		"mobile-default": {
			"myapp://callback",
		},
		"m2m-default": {}, // no redirects for m2m
	}

	for clientName, uris := range redirects {
		var client model.AuthClient
		err := db.
			Where("name = ? AND identity_provider_id = ?", clientName, identityProviderID).
			First(&client).Error
		if err != nil {
			log.Printf("⚠️ Auth client '%s' not found, skipping redirect URIs", clientName)
			continue
		}

		for _, uri := range uris {
			var existing model.AuthClientRedirectURI
			err := db.
				Where("auth_client_id = ? AND redirect_uri = ?", client.AuthClientID, uri).
				First(&existing).Error

			if err == nil {
				// Update existing redirect
				existing.UpdatedAt = time.Now()
				if err := db.Save(&existing).Error; err != nil {
					log.Printf("❌ Failed to update redirect URI '%s' for client '%s': %v", uri, clientName, err)
				} else {
					log.Printf("🔄 Redirect URI '%s' updated for client '%s'", uri, clientName)
				}
				continue
			}

			if err == gorm.ErrRecordNotFound {
				// Create new redirect
				redirect := model.AuthClientRedirectURI{
					AuthClientRedirectURIUUID: uuid.New(),
					AuthClientID:              client.AuthClientID,
					RedirectURI:               uri,
					CreatedAt:                 time.Now(),
					UpdatedAt:                 time.Now(),
				}
				if err := db.Create(&redirect).Error; err != nil {
					log.Printf("❌ Failed to create redirect URI '%s' for client '%s': %v", uri, clientName, err)
					continue
				}
				log.Printf("✅ Redirect URI '%s' created for client '%s'", uri, clientName)
				continue
			}

			// Unexpected error
			log.Printf("❌ Failed lookup for redirect URI '%s' for client '%s': %v", uri, clientName, err)
		}
	}

	return nil
}
