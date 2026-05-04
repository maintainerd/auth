package seeder

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/config"
	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

func SeedClientURIs(db *gorm.DB, tenantID int64, identityProviderID int64) error {
	appHostName := config.AppPrivateHostname

	// Map of client name -> URIs with their types. Only the auth-console SPA is
	// seeded; non-system clients (public identity, third-party apps) are
	// registered at runtime through the console.
	uris := map[string][]struct {
		URI  string
		Type string
	}{
		SystemClientNameAuthConsole: {
			{URI: "https://" + appHostName + "/callback", Type: model.ClientURITypeRedirect},
			{URI: "https://" + appHostName, Type: model.ClientURITypeOrigin},
			{URI: "https://" + appHostName, Type: model.ClientURITypeCORSOrigin},
			{URI: "https://" + appHostName + "/logout", Type: model.ClientURITypeLogout},
		},
	}

	for clientName, clientURIs := range uris {
		var client model.Client
		err := db.
			Where("name = ? AND identity_provider_id = ? AND tenant_id = ?", clientName, identityProviderID, tenantID).
			First(&client).Error
		if err != nil {
			return fmt.Errorf("auth client %q not found when seeding URIs: %w", clientName, err)
		}

		for _, uriData := range clientURIs {
			var existing model.ClientURI
			err := db.
				Where("client_id = ? AND uri = ? AND type = ?", client.ClientID, uriData.URI, uriData.Type).
				First(&existing).Error

			if err == nil {
				// Update existing URI
				existing.UpdatedAt = time.Now()
				if err := db.Save(&existing).Error; err != nil {
					return fmt.Errorf("failed to update URI %q (%s) for client %q: %w", uriData.URI, uriData.Type, clientName, err)
				}
				slog.Info("Client URI updated", "uri", uriData.URI, "type", uriData.Type, "client", clientName)
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
					return fmt.Errorf("failed to create URI %q (%s) for client %q: %w", uriData.URI, uriData.Type, clientName, err)
				}
				slog.Info("Client URI created", "uri", uriData.URI, "type", uriData.Type, "client", clientName)
				continue
			}

			// Unexpected error
			return fmt.Errorf("failed lookup for URI %q (%s) for client %q: %w", uriData.URI, uriData.Type, clientName, err)
		}
	}

	return nil
}
