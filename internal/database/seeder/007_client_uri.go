package seeder

import (
	"log"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

func SeedClientURIs(db *gorm.DB, tenantID int64, identityProviderID int64) error {
	appHostName := os.Getenv("APP_PRIVATE_HOSTNAME")

	// Map of client name -> URIs with their types
	uris := map[string][]struct {
		URI  string
		Type string
	}{
		"traditional-default": {
			{URI: "http://" + appHostName + "/callback", Type: "redirect-uri"},
			{URI: "https://" + appHostName + "/callback", Type: "redirect-uri"},
			{URI: "http://" + appHostName, Type: "origin-uri"},
			{URI: "https://" + appHostName, Type: "origin-uri"},
			{URI: "http://" + appHostName + "/logout", Type: "logout-uri"},
			{URI: "https://" + appHostName + "/logout", Type: "logout-uri"},
		},
		"spa-default": {
			{URI: "http://localhost:3000/callback", Type: "redirect-uri"},
			{URI: "https://" + appHostName + "/callback", Type: "redirect-uri"},
			{URI: "http://localhost:3000", Type: "origin-uri"},
			{URI: "https://" + appHostName, Type: "origin-uri"},
			{URI: "http://localhost:3000", Type: "cors-origin-uri"},
			{URI: "https://" + appHostName, Type: "cors-origin-uri"},
		},
		"mobile-default": {
			{URI: "myapp://callback", Type: "redirect-uri"},
			{URI: "myapp://logout", Type: "logout-uri"},
		},
		"m2m-default": {}, // no URIs for m2m
	}

	for clientName, clientURIs := range uris {
		var client model.Client
		err := db.
			Where("name = ? AND identity_provider_id = ? AND tenant_id = ?", clientName, identityProviderID, tenantID).
			First(&client).Error
		if err != nil {
			log.Printf("⚠️ Auth client '%s' not found, skipping URIs", clientName)
			continue
		}

		for _, uriData := range clientURIs {
			var existing model.ClientURI
			err := db.
				Where("client_id = ? AND uri = ? AND type = ?", client.Identifier, uriData.URI, uriData.Type).
				First(&existing).Error

			if err == nil {
				// Update existing URI
				existing.UpdatedAt = time.Now()
				if err := db.Save(&existing).Error; err != nil {
					log.Printf("❌ Failed to update URI '%s' (%s) for client '%s': %v", uriData.URI, uriData.Type, clientName, err)
				} else {
					log.Printf("🔄 URI '%s' (%s) updated for client '%s'", uriData.URI, uriData.Type, clientName)
				}
				continue
			}

			if err == gorm.ErrRecordNotFound {
				// Create new URI
				uri := model.ClientURI{
					ClientURIUUID: uuid.New(),
					TenantID:      tenantID,
					ClientID:      client.ClientID,
					URI:           uriData.URI,
					Type:          uriData.Type,
					CreatedAt:     time.Now(),
					UpdatedAt:     time.Now(),
				}
				if err := db.Create(&uri).Error; err != nil {
					log.Printf("❌ Failed to create URI '%s' (%s) for client '%s': %v", uriData.URI, uriData.Type, clientName, err)
					continue
				}
				log.Printf("✅ URI '%s' (%s) created for client '%s'", uriData.URI, uriData.Type, clientName)
				continue
			}

			// Unexpected error
			log.Printf("❌ Failed lookup for URI '%s' (%s) for client '%s': %v", uriData.URI, uriData.Type, clientName, err)
		}
	}

	return nil
}
