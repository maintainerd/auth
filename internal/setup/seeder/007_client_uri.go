package seeder

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	model "github.com/maintainerd/maintainerd-auth/internal/client"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"gorm.io/gorm"
)

func SeedClientURIs(db *gorm.DB, tenantID int64, _ int64) error {
	// Derive the per-tenant frontend hosts from the tenant name/is_system.
	tenantName, tenantIsSystem, err := loadTenantHostInfo(db, tenantID)
	if err != nil {
		return err
	}
	consoleHostName := shared.FrontendURL(shared.FrontendSurfaceConsole, tenantName, tenantIsSystem, "")
	identityHostName := shared.FrontendURL(shared.FrontendSurfaceIdentity, tenantName, tenantIsSystem, "")

	// Map of client name -> URIs with their types.
	uris := map[string][]struct {
		URI  string
		Type string
	}{
		shared.SystemClientNameAuthConsole: {
			{URI: consoleHostName + "/auth/callback", Type: shared.ClientURITypeRedirect},
			{URI: consoleHostName, Type: shared.ClientURITypeOrigin},
			{URI: consoleHostName, Type: shared.ClientURITypeCORSOrigin},
			{URI: consoleHostName + "/logout", Type: shared.ClientURITypeLogout},
		},
		shared.SystemClientNameAuthIdentity: {
			{URI: identityHostName + "/callback", Type: shared.ClientURITypeRedirect},
			{URI: identityHostName, Type: shared.ClientURITypeOrigin},
			{URI: identityHostName, Type: shared.ClientURITypeCORSOrigin},
			{URI: identityHostName + "/logout", Type: shared.ClientURITypeLogout},
		},
	}

	for clientName, clientURIs := range uris {
		var client model.Client
		err := db.
			Where("name = ? AND tenant_id = ?", clientName, tenantID).
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

			if errors.Is(err, gorm.ErrRecordNotFound) {
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
