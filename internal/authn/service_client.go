package authn

import (
	"context"
	"errors"

	"github.com/maintainerd/maintainerd-auth/internal/shared"
)

// TenantResolver resolves a subdomain tenant slug (the DNS name) to its tenant
// ID on the public auth surface. It is implemented by an app-layer adapter over
// the tenant service so the authn package does not import internal/tenant
// (avoiding a cross-domain dependency). This mirrors the tenant.SurfaceClientReader
// decoupling pattern: authn declares the interface and internal/server wires the
// concrete adapter via SetTenantResolver.
type TenantResolver interface {
	ResolveTenantIDByName(ctx context.Context, name string) (tenantID int64, isSystem bool, err error)
}

// publicTenantResolver is the injected resolver that makes the subdomain tenant
// authoritative on the public auth surface. It is set once at wiring time via
// SetTenantResolver. When nil (e.g. unit tests that do not exercise the subdomain
// path), resolvePublicClient preserves the legacy behavior so the change is
// strictly additive: a public request carrying a tenant_id but no wired resolver
// behaves exactly as before (tenant_id ignored).
var publicTenantResolver TenantResolver

// SetTenantResolver injects the subdomain tenant resolver for the public auth
// surface. Mirrors tenant.SetSurfaceClientReader: an app-layer adapter over the
// tenant service is wired in internal/server.
func SetTenantResolver(r TenantResolver) { publicTenantResolver = r }

// ErrClientTenantMismatch is returned when a public auth request carries an
// authoritative subdomain tenant context and a client_id that resolves to a
// DIFFERENT tenant. It is a hard reject with NO fallback to the system tenant.
var ErrClientTenantMismatch = errors.New("client does not belong to tenant")

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
//
// Two modes, selected by whether a subdomain tenant context is present:
//
//   - SUBDOMAIN AUTHORITATIVE (tenant_id = slug, resolver wired): the subdomain
//     tenant is authoritative. The slug is resolved to a tenant ID; if a
//     client_id is also present it MUST belong to that tenant (hard reject on
//     mismatch, no system fallback); if no client_id is present, the acting
//     tenant is the subdomain tenant, driven by its seeded identity client.
//
//   - LEGACY / CLIENT-SCOPED (no tenant_id, or no resolver wired): behavior is
//     unchanged — external apps identify themselves with client_id, tenant_id is
//     ignored, and seeded system clients are not accepted on public routes
//     (except the allowed hosted-login clients).
//
// The system-client rejection and the impersonation guard (an external client_id
// caller cannot act as another tenant) are preserved in both modes.
func resolvePublicClient(
	ctx context.Context,
	clientRepo ClientRepository,
	clientID *string,
	tenantID *string,
) (*Client, error) {
	if tenantID != nil && *tenantID != "" {
		// No resolver wired → preserve the exact legacy behavior (tenant_id is
		// ignored on the public surface). Keeps the change strictly additive.
		if publicTenantResolver == nil {
			return nil, nil
		}

		// The subdomain tenant is AUTHORITATIVE. Resolve the slug first; an
		// unknown slug is rejected (no client is resolved).
		subTenantID, _, err := publicTenantResolver.ResolveTenantIDByName(ctx, *tenantID)
		if err != nil {
			return nil, err
		}
		if subTenantID == 0 {
			return nil, nil
		}

		if clientID != nil && *clientID != "" {
			client, err := clientRepo.FindByIdentifier(*clientID)
			if err != nil || client == nil {
				return client, err
			}
			// System clients are never valid public client_ids (except the allowed
			// hosted-login clients). Preserve this before the tenant binding check.
			if client.IsSystem && !isPublicAuthSystemClientAllowed(client) {
				return nil, nil
			}
			// SECURITY-CRITICAL: the client_id must belong to the authoritative
			// subdomain tenant. Mismatch is a hard reject with NO fallback to system.
			if client.TenantID != subTenantID {
				return nil, ErrClientTenantMismatch
			}
			return client, nil
		}

		// Direct-nav first-party login (no client_id): the acting tenant is the
		// subdomain tenant, driven by its seeded identity client. The lookup is
		// scoped to the subdomain slug and to is_system/active, so it cannot select
		// another tenant's client.
		return clientRepo.FindSystemByTenantIdentifierAndName(*tenantID, shared.SystemClientNameAuthIdentity)
	}

	// No subdomain tenant context: LEGACY behavior — client_id alone determines
	// the tenant.
	if clientID != nil && *clientID != "" {
		client, err := clientRepo.FindByIdentifier(*clientID)
		if err != nil || client == nil {
			return client, err
		}
		if client.IsSystem && !isPublicAuthSystemClientAllowed(client) {
			return nil, nil
		}
		return client, nil
	}
	return nil, nil
}

// isPublicAuthSystemClientAllowed reports whether a seeded system client may
// drive the public auth surface. The first-party SPA login clients
// (auth-console, auth-identity) front the hosted identity UI and must be
// accepted on public auth routes even though they are system clients; all other
// system clients stay rejected. This mirrors isPublicOAuthSystemClientAllowed on
// the OAuth authorize endpoint so /oauth/authorize and /login agree on which
// client may complete the hosted login flow.
func isPublicAuthSystemClientAllowed(client *Client) bool {
	if client == nil {
		return false
	}
	return client.Name == shared.SystemClientNameAuthConsole ||
		client.Name == shared.SystemClientNameAuthIdentity
}

func resolveClientForContext(
	ctx context.Context,
	clientRepo ClientRepository,
	clientID *string,
	tenantID *string,
) (*Client, error) {
	if publicAuthSurfaceFromContext(ctx) {
		return resolvePublicClient(ctx, clientRepo, clientID, tenantID)
	}
	return resolveClient(clientRepo, clientID, tenantID)
}

func stringPtrOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
