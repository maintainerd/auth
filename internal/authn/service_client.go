package authn

import (
	"context"

	"github.com/maintainerd/auth/internal/shared"
)

// resolveClient resolves the auth client for register/login/invite operations.
//
// HTTP handlers enforce the surface contract. This resolver retains the
// default for trusted/internal callers that do not originate at those routes.
func resolveClient(
	clientRepo ClientRepository,
	clientID *string,
	tenantID *string,
) (*Client, error) {
	if clientID != nil && *clientID != "" {
		return clientRepo.FindByIdentifier(*clientID)
	}
	if tenantID != nil && *tenantID != "" {
		return clientRepo.FindSystemByTenantIdentifierAndName(*tenantID, shared.SystemClientNameAuthConsole)
	}
	return clientRepo.FindSystem()
}

// resolvePublicClient resolves client context for the public identity surface.
// Public auth is client-scoped: external apps identify themselves with
// client_id, tenant_id is rejected by the HTTP layer, and seeded system clients
// are not accepted on public routes.
func resolvePublicClient(
	clientRepo ClientRepository,
	clientID *string,
	tenantID *string,
) (*Client, error) {
	if tenantID != nil && *tenantID != "" {
		return nil, nil
	}
	if clientID != nil && *clientID != "" {
		client, err := clientRepo.FindByIdentifier(*clientID)
		if err != nil || client == nil {
			return client, err
		}
		if client.IsSystem {
			return nil, nil
		}
		return client, nil
	}
	return nil, nil
}

func resolveClientForContext(
	ctx context.Context,
	clientRepo ClientRepository,
	clientID *string,
	tenantID *string,
) (*Client, error) {
	if publicAuthSurfaceFromContext(ctx) {
		return resolvePublicClient(clientRepo, clientID, tenantID)
	}
	return resolveClient(clientRepo, clientID, tenantID)
}

func stringPtrOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
