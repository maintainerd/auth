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

func SeedClients(db *gorm.DB, tenantID int64, identityProviderID int64) error {
	appHostName := config.AppPrivateHostname

	// Pre-generate all client identifiers and secrets
	traditionalID, err := crypto.GenerateIdentifier(32)
	if err != nil {
		return fmt.Errorf("failed to generate identifier: %w", err)
	}
	traditionalSecret, err := crypto.GenerateIdentifier(64)
	if err != nil {
		return fmt.Errorf("failed to generate secret: %w", err)
	}
	spaID, err := crypto.GenerateIdentifier(32)
	if err != nil {
		return fmt.Errorf("failed to generate identifier: %w", err)
	}
	mobileID, err := crypto.GenerateIdentifier(32)
	if err != nil {
		return fmt.Errorf("failed to generate identifier: %w", err)
	}
	m2mID, err := crypto.GenerateIdentifier(32)
	if err != nil {
		return fmt.Errorf("failed to generate identifier: %w", err)
	}
	m2mSecret, err := crypto.GenerateIdentifier(64)
	if err != nil {
		return fmt.Errorf("failed to generate secret: %w", err)
	}

	clients := []model.Client{
		{
			ClientUUID:  uuid.New(),
			TenantID:    tenantID,
			Name:        "traditional-default",
			DisplayName: "Traditional Web App Default",
			ClientType:  "traditional",
			Domain:      strPtr(appHostName),
			Identifier:  strPtr(traditionalID),
			Secret:      strPtr(traditionalSecret),
			Config: datatypes.JSON([]byte(`{
				"grant_types": ["authorization_code"],
				"response_type": "code",
				"pkce": false
			}`)),
			Status:                  "active",
			IsDefault:               true,
			IsSystem:                true,
			IdentityProviderID:      identityProviderID,
			TokenEndpointAuthMethod: model.TokenAuthMethodSecretBasic,
			GrantTypes:              pq.StringArray{model.GrantTypeAuthorizationCode, model.GrantTypeRefreshToken},
			ResponseTypes:           pq.StringArray{model.ResponseTypeCode},
			RequireConsent:          false,
			CreatedAt:               time.Now(),
			UpdatedAt:               time.Now(),
		},
		{
			ClientUUID:  uuid.New(),
			TenantID:    tenantID,
			Name:        "spa-default",
			DisplayName: "Single Page App Default",
			ClientType:  "spa",
			Domain:      strPtr(appHostName),
			Identifier:  strPtr(spaID),
			Secret:      nil,
			Config: datatypes.JSON([]byte(`{
				"grant_types": ["authorization_code"],
				"response_type": "code",
				"pkce": true
			}`)),
			Status:                  "active",
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
		{
			ClientUUID:  uuid.New(),
			TenantID:    tenantID,
			Name:        "mobile-default",
			DisplayName: "Mobile App Default",
			ClientType:  "mobile",
			Domain:      strPtr(appHostName),
			Identifier:  strPtr(mobileID),
			Secret:      nil,
			Config: datatypes.JSON([]byte(`{
				"grant_types": ["authorization_code"],
				"response_type": "code",
				"pkce": true
			}`)),
			Status:                  "active",
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
		{
			ClientUUID:  uuid.New(),
			TenantID:    tenantID,
			Name:        "m2m-default",
			DisplayName: "Machine to Machine Default",
			ClientType:  "m2m",
			Domain:      strPtr(appHostName),
			Identifier:  strPtr(m2mID),
			Secret:      strPtr(m2mSecret),
			Config: datatypes.JSON([]byte(`{
				"grant_types": ["client_credentials"]
			}`)),
			Status:                  "active",
			IsDefault:               true,
			IsSystem:                true,
			IdentityProviderID:      identityProviderID,
			TokenEndpointAuthMethod: model.TokenAuthMethodSecretBasic,
			GrantTypes:              pq.StringArray{model.GrantTypeClientCredentials},
			ResponseTypes:           pq.StringArray{},
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
