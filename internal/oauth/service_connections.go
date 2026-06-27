package oauth

import (
	"context"
	"strings"

	"github.com/maintainerd/auth/internal/platform/apperror"
	"github.com/maintainerd/auth/internal/secpolicy"
	"github.com/maintainerd/auth/internal/shared"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"gorm.io/gorm"
)

// OAuthConnectionInfo is a single login option exposed to the hosted login page.
type OAuthConnectionInfo struct {
	Identifier   string
	DisplayName  string
	Provider     string
	ProviderType string
	IsDefault    bool
	DisplayOrder int
}

// OAuthConnectionsResult is the set of login options for a client: whether the
// in-house username/password form is available, whether self-registration is
// allowed, plus the connected OAuth2 providers (rendered as buttons), ordered
// by DisplayOrder.
type OAuthConnectionsResult struct {
	PasswordEnabled    bool
	RegistrationEnabled bool
	Connections        []OAuthConnectionInfo
}

// OAuthConnectionsService resolves the enabled identity-provider connections of a
// client so the hosted identity app can render its login page.
type OAuthConnectionsService interface {
	ListConnections(ctx context.Context, clientID string) (*OAuthConnectionsResult, error)
}

type oauthConnectionsService struct {
	db                  *gorm.DB
	clientRepo          ClientRepository
	securitySettingRepo secpolicy.SecuritySettingRepository // optional — nil-safe
}

func NewOAuthConnectionsService(db *gorm.DB, clientRepo ClientRepository, securitySettingRepo secpolicy.SecuritySettingRepository) OAuthConnectionsService {
	return &oauthConnectionsService{db: db, clientRepo: clientRepo, securitySettingRepo: securitySettingRepo}
}

func (s *oauthConnectionsService) ListConnections(ctx context.Context, clientID string) (*OAuthConnectionsResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "oauth_connections.list")
	defer span.End()

	client, err := s.clientRepo.FindByIdentifier(strings.TrimSpace(clientID))
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "client lookup failed")
		return nil, apperror.NewInternal("failed to load client", err)
	}
	allowedSystemClient := client != nil && client.IsSystem && client.Name == shared.SystemClientNameAuthConsole
	if client == nil || client.Status != shared.StatusActive || (client.IsSystem && !allowedSystemClient) {
		span.SetStatus(codes.Error, "client not found or inactive")
		return nil, apperror.NewNotFound("unknown or inactive client")
	}

	var connections []ClientIdentityProvider
	if err := s.db.
		Preload("IdentityProvider").
		Where("client_id = ? AND enabled = ?", client.ClientID, true).
		Order("display_order ASC").
		Find(&connections).Error; err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "connection lookup failed")
		return nil, apperror.NewInternal("failed to load connections", err)
	}

	result := &OAuthConnectionsResult{
		PasswordEnabled:    allowedSystemClient,
		RegistrationEnabled: allowedSystemClient,
		Connections:        make([]OAuthConnectionInfo, 0, len(connections)),
	}
	for _, conn := range connections {
		idp := conn.IdentityProvider
		if idp == nil || idp.Status != shared.StatusActive {
			continue
		}
		// The built-in (system) provider drives the username/password form rather
		// than an OAuth2 button.
		if idp.IsSystem || idp.ProviderType == shared.IDPTypeSystem {
			result.PasswordEnabled = true
			// RegistrationEnabled = tenant.self_registration_enabled
			//   && inHouseIdP.allow_registration
			//   && (auth_flow == nil || auth_flow.allow_registration)
			tenantAllows := true
			if cid := clientTenantID(client); cid > 0 && s.securitySettingRepo != nil {
				regPolicy := secpolicy.LoadRegistrationPolicy(s.securitySettingRepo, cid)
				tenantAllows = regPolicy.SelfRegistrationEnabled
			}
			// Auth_flow gate: look up the client's signup flow; no flow = unrestricted.
			afAllows := true
			var af struct{ AllowRegistration bool }
			if err := s.db.Table("auth_flows").Where("client_id = ? AND deleted_at IS NULL", client.ClientID).
				Select("allow_registration").Scan(&af).Error; err == nil && af.AllowRegistration {
				afAllows = true
			} else if err == nil {
				afAllows = af.AllowRegistration
			}
			result.RegistrationEnabled = tenantAllows && idp.AllowRegistration && afAllows
			continue
		}
		displayName := idp.DisplayName
		if displayName == "" {
			displayName = idp.Name
		}
		result.Connections = append(result.Connections, OAuthConnectionInfo{
			Identifier:   idp.Identifier,
			DisplayName:  displayName,
			Provider:     idp.Provider,
			ProviderType: idp.ProviderType,
			IsDefault:    conn.IsDefault,
			DisplayOrder: conn.DisplayOrder,
		})
	}

	span.SetStatus(codes.Ok, "")
	return result, nil
}

// clientTenantID returns the tenant ID for the OAuth client. Uses the client's
// own TenantID column (preferred) or its identity provider's tenant.
func clientTenantID(client *Client) int64 {
	if client == nil {
		return 0
	}
	if client.TenantID > 0 {
		return client.TenantID
	}
	if client.IdentityProvider != nil {
		if client.IdentityProvider.TenantID > 0 {
			return client.IdentityProvider.TenantID
		}
	}
	return 0
}
