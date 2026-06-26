package seeder

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	model "github.com/maintainerd/auth/internal/client"
	"github.com/maintainerd/auth/internal/platform/config"
	"github.com/maintainerd/auth/internal/shared"
	"gorm.io/gorm"
)

func SeedClientURIs(db *gorm.DB, tenantID int64, _ int64) error {
	privateHostName := normalizedHTTPSBaseURL(config.AppPrivateHostname)
	identityHostName := normalizedHTTPSBaseURL(config.AppFrontendIdentityHostname)

	// Map of client name -> URIs with their types.
	uris := map[string][]struct {
		URI  string
		Type string
	}{
		shared.SystemClientNameAuthConsole: {
			{URI: privateHostName + "/callback", Type: shared.ClientURITypeRedirect},
			{URI: privateHostName, Type: shared.ClientURITypeOrigin},
			{URI: privateHostName, Type: shared.ClientURITypeCORSOrigin},
			{URI: privateHostName + "/logout", Type: shared.ClientURITypeLogout},
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

func normalizedHTTPSBaseURL(hostname string) string {
	base := strings.TrimRight(strings.TrimSpace(hostname), "/")
	if strings.HasPrefix(base, "http://") || strings.HasPrefix(base, "https://") {
		return base
	}
	return "https://" + base
}
