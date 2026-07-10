package seeder

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	model "github.com/maintainerd/maintainerd-auth/internal/client"
	"github.com/maintainerd/maintainerd-auth/internal/platform/crypto"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func SeedClients(db *gorm.DB, tenantID int64, identityProviderID int64) error {
	// Derive the per-tenant frontend hosts from the tenant name/is_system so a
	// regular tenant (e.g. "acme") seeds acme.console.auth.maintainerd.local and
	// acme.auth.maintainerd.local, while the system tenant uses the bare host.
	tenantName, tenantIsSystem, err := loadTenantHostInfo(db, tenantID)
	if err != nil {
		return err
	}
	consoleHostName := shared.FrontendURL(shared.FrontendSurfaceConsole, tenantName, tenantIsSystem, "")
	identityHostName := shared.FrontendURL(shared.FrontendSurfaceIdentity, tenantName, tenantIsSystem, "")

	consoleID, err := crypto.GenerateIdentifier(32)
	if err != nil {
		return fmt.Errorf("failed to generate identifier: %w", err)
	}
	identityID, err := crypto.GenerateIdentifier(32)
	if err != nil {
		return fmt.Errorf("failed to generate identity identifier: %w", err)
	}

	clients := []model.Client{
		{
			ClientUUID:  uuid.New(),
			TenantID:    tenantID,
			Name:        shared.SystemClientNameAuthConsole,
			DisplayName: "Maintainerd Auth Console",
			ClientType:  shared.ClientTypeSPA,
			Domain:      strPtr(consoleHostName),
			Identifier:  strPtr(consoleID),
			SecretHash:  nil, // public client (TokenAuthMethodNone) — no secret
			Config: datatypes.JSON([]byte(`{
				"grant_types": ["authorization_code", "refresh_token"],
				"response_type": "code",
				"pkce": true
			}`)),
			Status:                  shared.StatusActive,
			IsDefault:               true,
			IsSystem:                true,
			TokenEndpointAuthMethod: model.TokenAuthMethodNone,
			GrantTypes:              pq.StringArray{model.GrantTypeAuthorizationCode, model.GrantTypeRefreshToken},
			ResponseTypes:           pq.StringArray{model.ResponseTypeCode},
			RequireConsent:          false,
			AllowedScopes:           pq.StringArray{},
			CreatedAt:               time.Now(),
			UpdatedAt:               time.Now(),
		},
		{
			ClientUUID:  uuid.New(),
			TenantID:    tenantID,
			Name:        shared.SystemClientNameAuthIdentity,
			DisplayName: "Maintainerd Auth Identity",
			ClientType:  shared.ClientTypeSPA,
			Domain:      strPtr(identityHostName),
			Identifier:  strPtr(identityID),
			SecretHash:  nil, // public client (TokenAuthMethodNone) — no secret
			Config: datatypes.JSON([]byte(`{
				"grant_types": ["authorization_code", "refresh_token"],
				"response_type": "code",
				"pkce": true
			}`)),
			Status:                  shared.StatusActive,
			IsDefault:               false,
			IsSystem:                true,
			TokenEndpointAuthMethod: model.TokenAuthMethodNone,
			GrantTypes:              pq.StringArray{model.GrantTypeAuthorizationCode, model.GrantTypeRefreshToken},
			ResponseTypes:           pq.StringArray{model.ResponseTypeCode},
			RequireConsent:          false,
			AllowedScopes:           pq.StringArray{},
			CreatedAt:               time.Now(),
			UpdatedAt:               time.Now(),
		},
	}

	for _, client := range clients {
		var existing model.Client
		err := db.
			Where("name = ? AND tenant_id = ?", client.Name, tenantID).
			First(&existing).Error

		if err == nil {
			// Update existing client - preserve existing IDs and UUID
			client.ClientID = existing.ClientID
			client.Identifier = existing.Identifier
			client.ClientUUID = existing.ClientUUID
			if err := db.Save(&client).Error; err != nil {
				return fmt.Errorf("failed to update auth client %q: %w", client.Name, err)
			}
			if err := seedClientIdentityProvider(db, existing.ClientID, tenantID, identityProviderID, client.IsDefault); err != nil {
				return err
			}
			slog.Info("Auth client updated", "name", client.Name)
			continue
		}

		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Create new client
			if err := db.Create(&client).Error; err != nil {
				return fmt.Errorf("failed to create auth client %q: %w", client.Name, err)
			}
			if err := seedClientIdentityProvider(db, client.ClientID, tenantID, identityProviderID, client.IsDefault); err != nil {
				return err
			}
			slog.Info("Auth client created", "name", client.Name)
			continue
		}

		// Unexpected error
		return fmt.Errorf("failed lookup for auth client %q: %w", client.Name, err)
	}

	return nil
}

func seedClientIdentityProvider(db *gorm.DB, clientID, tenantID, identityProviderID int64, isDefault bool) error {
	connection := model.ClientIdentityProvider{
		TenantID:           tenantID,
		ClientID:           clientID,
		IdentityProviderID: identityProviderID,
		IsDefault:          isDefault,
		Enabled:            true,
		DisplayOrder:       0,
	}

	var existing model.ClientIdentityProvider
	err := db.
		Where("client_id = ? AND identity_provider_id = ? AND deleted_at IS NULL", clientID, identityProviderID).
		First(&existing).Error
	if err == nil {
		existing.IsDefault = isDefault
		existing.Enabled = true
		existing.DisplayOrder = 0
		if err := db.Save(&existing).Error; err != nil {
			return fmt.Errorf("failed to update client identity provider connection: %w", err)
		}
		return nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		if err := db.Create(&connection).Error; err != nil {
			return fmt.Errorf("failed to create client identity provider connection: %w", err)
		}
		return nil
	}
	return fmt.Errorf("failed lookup for client identity provider connection: %w", err)
}

func strPtr(s string) *string {
	return &s
}

// loadTenantHostInfo returns the tenant's DNS slug (name) and whether it is the
// system tenant, so seeders can derive per-tenant frontend hosts.
func loadTenantHostInfo(db *gorm.DB, tenantID int64) (string, bool, error) {
	var t struct {
		Name     string
		IsSystem bool
	}
	if err := db.Table("tenants").
		Select("name", "is_system").
		Where("tenant_id = ?", tenantID).
		Scan(&t).Error; err != nil {
		return "", false, fmt.Errorf("failed to resolve tenant %d when seeding client hosts: %w", tenantID, err)
	}
	if t.Name == "" {
		return "", false, fmt.Errorf("tenant %d not found when seeding client hosts", tenantID)
	}
	return t.Name, t.IsSystem, nil
}
