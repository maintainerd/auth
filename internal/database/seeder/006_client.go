package seeder

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/maintainerd/auth/internal/config"
	"github.com/maintainerd/auth/internal/crypto"
	"github.com/maintainerd/auth/internal/model"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// SystemClientNameAuthConsole is the seeded system client used by the
// maintainerd-auth-console SPA frontend during initial bootstrap. Additional
// clients (public identity, mobile, m2m, third-party apps) are created at
// runtime through the console and are not part of the system seed.
const SystemClientNameAuthConsole = "auth-console"

func SeedClients(db *gorm.DB, tenantID int64, identityProviderID int64) error {
	appHostName := config.AppPrivateHostname

	consoleID, err := crypto.GenerateIdentifier(32)
	if err != nil {
		return fmt.Errorf("failed to generate identifier: %w", err)
	}

	clients := []model.Client{
		{
			ClientUUID:  uuid.New(),
			TenantID:    tenantID,
			Name:        SystemClientNameAuthConsole,
			DisplayName: "Maintainerd Auth Console",
			ClientType:  model.ClientTypeSPA,
			Domain:      strPtr(appHostName),
			Identifier:  strPtr(consoleID),
			Secret:      nil,
			Config: datatypes.JSON([]byte(`{
				"grant_types": ["authorization_code", "refresh_token"],
				"response_type": "code",
				"pkce": true
			}`)),
			Status:                  model.StatusActive,
			IsDefault:               true,
			IsSystem:                true,
			IdentityProviderID:      identityProviderID,
			TokenEndpointAuthMethod: model.TokenAuthMethodNone,
			GrantTypes:              pq.StringArray{model.GrantTypeAuthorizationCode, model.GrantTypeRefreshToken},
			ResponseTypes:           pq.StringArray{model.ResponseTypeCode},
			RequireConsent:          false,
			CreatedAt:               time.Now(),
			UpdatedAt:               time.Now(),
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
				return fmt.Errorf("failed to update auth client %q: %w", client.Name, err)
			}
			slog.Info("Auth client updated", "name", client.Name)
			continue
		}

		if err == gorm.ErrRecordNotFound {
			// Create new client
			if err := db.Create(&client).Error; err != nil {
				return fmt.Errorf("failed to create auth client %q: %w", client.Name, err)
			}
			slog.Info("Auth client created", "name", client.Name)
			continue
		}

		// Unexpected error
		return fmt.Errorf("failed lookup for auth client %q: %w", client.Name, err)
	}

	return nil
}

func strPtr(s string) *string {
	return &s
}
