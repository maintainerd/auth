package oauth

import (
	"context"
	"strings"

	"github.com/maintainerd/maintainerd-auth/internal/branding"
	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	"github.com/maintainerd/maintainerd-auth/internal/secpolicy"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
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
	PasswordEnabled     bool
	RegistrationEnabled bool
	// MagicLinkEnabled reports whether this client offers passwordless email
	// sign-in. Off unless an operator turns it on for the client.
	MagicLinkEnabled bool
	Branding         *branding.ClientBrandingResponse
	Connections      []OAuthConnectionInfo
}

type OAuthConnectionsService interface {
	ListConnections(ctx context.Context, clientID string) (*OAuthConnectionsResult, error)
}

type oauthConnectionsService struct {
	db                  *gorm.DB
	clientRepo          ClientRepository
	securitySettingRepo secpolicy.SecuritySettingRepository
	brandingResolver    *branding.ClientBrandingResolver
}

func NewOAuthConnectionsService(db *gorm.DB, clientRepo ClientRepository, securitySettingRepo secpolicy.SecuritySettingRepository, brandingResolver *branding.ClientBrandingResolver) OAuthConnectionsService {
	return &oauthConnectionsService{db: db, clientRepo: clientRepo, securitySettingRepo: securitySettingRepo, brandingResolver: brandingResolver}
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
	allowedSystemClient := isSeededSurfaceClient(client)
	if client == nil || client.Status != shared.StatusActive || (client.IsSystem && !allowedSystemClient) {
		span.SetStatus(codes.Error, "client not found or inactive")
		return nil, apperror.NewNotFound("unknown or inactive client")
	}

	var resolvedBranding *branding.ClientBrandingResponse
	if s.brandingResolver != nil && client != nil {
		resolvedBranding = s.brandingResolver.ResolveForClient(client.BrandingID, client.TenantID)
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
		PasswordEnabled:     allowedSystemClient,
		RegistrationEnabled: allowedSystemClient,
		MagicLinkEnabled:    client.AllowMagicLink,
		Branding:            resolvedBranding,
		Connections:         make([]OAuthConnectionInfo, 0, len(connections)),
	}
	for _, conn := range connections {
		idp := conn.IdentityProvider
		if idp == nil || idp.Status != shared.StatusActive {
			continue
		}
		if idp.IsSystem || idp.ProviderType == shared.IDPTypeSystem {
			result.PasswordEnabled = true
			tenantAllows := true
			if cid := clientTenantID(client); cid > 0 && s.securitySettingRepo != nil {
				regPolicy := secpolicy.LoadRegistrationPolicy(s.securitySettingRepo, cid)
				tenantAllows = regPolicy.SelfRegistrationEnabled
			}
			result.RegistrationEnabled = tenantAllows && client.AllowRegistration && idp.AllowRegistration
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

// isSeededSurfaceClient reports whether a client is one of the two seeded
// first-party surface clients (auth-console, auth-identity). Those are the only
// system clients that render a hosted login page, so they are the only ones
// allowed to list their connections; every other system client is internal and
// must not expose them. Non-system clients are never "seeded surface" clients —
// callers gate on client.IsSystem separately.
func isSeededSurfaceClient(client *Client) bool {
	if client == nil || !client.IsSystem {
		return false
	}
	return client.Name == shared.SystemClientNameAuthConsole ||
		client.Name == shared.SystemClientNameAuthIdentity
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
