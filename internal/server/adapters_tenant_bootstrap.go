package server

import (
	"context"

	"github.com/maintainerd/maintainerd-auth/internal/client"
	"github.com/maintainerd/maintainerd-auth/internal/oauth"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"github.com/maintainerd/maintainerd-auth/internal/tenant"
)

// surfaceClientReaderAdapter satisfies tenant.SurfaceClientReader over the client
// service. It maps a resolved frontend surface to the tenant's seeded system
// client (console → auth-console, identity → auth-identity), returning only the
// public client fields. The tenant package stays decoupled from client: it
// declares the SurfaceClientReader interface and this app-layer adapter wires it.
type surfaceClientReaderAdapter struct {
	svc client.ClientService
}

// publicSurfaceClientResolver is the subset of the concrete client service that
// exposes the per-surface public client lookups. These methods are not on the
// client.ClientService interface, so we assert for them (mirroring the public
// client handler).
type publicSurfaceClientResolver interface {
	GetPublicConsoleByTenantIdentifier(context.Context, string) (*client.ClientPublicServiceDataResult, error)
	GetPublicIdentityByTenantIdentifier(context.Context, string) (*client.ClientPublicServiceDataResult, error)
}

func (a surfaceClientReaderAdapter) GetSurfaceClient(ctx context.Context, tenantName, surface string) (*tenant.SurfaceClient, error) {
	resolver, ok := a.svc.(publicSurfaceClientResolver)
	if !ok {
		return nil, nil
	}

	var (
		res *client.ClientPublicServiceDataResult
		err error
	)
	switch surface {
	case shared.FrontendSurfaceConsole:
		res, err = resolver.GetPublicConsoleByTenantIdentifier(ctx, tenantName)
	default:
		// identity is the default surface.
		res, err = resolver.GetPublicIdentityByTenantIdentifier(ctx, tenantName)
	}
	if err != nil || res == nil {
		return nil, err
	}
	return &tenant.SurfaceClient{
		ClientID:    res.ClientID,
		Name:        res.Name,
		DisplayName: res.DisplayName,
		ClientType:  res.ClientType,
	}, nil
}

// surfaceConnectionsReaderAdapter satisfies tenant.SurfaceConnectionsReader over
// the oauth connections service, so the bootstrap endpoint advertises exactly
// the same federated login options as GET /oauth/connections — one filtering
// rule, two entry points. The tenant package stays decoupled from oauth: it
// declares the interface and this app-layer adapter wires it.
type surfaceConnectionsReaderAdapter struct {
	svc oauth.OAuthConnectionsService
}

func (a surfaceConnectionsReaderAdapter) ListSurfaceConnections(ctx context.Context, clientIdentifier string) (tenant.SurfaceLoginMethods, error) {
	if a.svc == nil {
		return tenant.SurfaceLoginMethods{}, nil
	}
	res, err := a.svc.ListConnections(ctx, clientIdentifier)
	if err != nil || res == nil {
		return tenant.SurfaceLoginMethods{}, err
	}
	methods := tenant.SurfaceLoginMethods{MagicLinkEnabled: res.MagicLinkEnabled}
	out := make([]tenant.SurfaceConnection, 0, len(res.Connections))
	for _, c := range res.Connections {
		out = append(out, tenant.SurfaceConnection{
			Identifier:   c.Identifier,
			DisplayName:  c.DisplayName,
			Provider:     c.Provider,
			ProviderType: c.ProviderType,
			IsDefault:    c.IsDefault,
			DisplayOrder: c.DisplayOrder,
		})
	}
	methods.Connections = out
	return methods, nil
}
