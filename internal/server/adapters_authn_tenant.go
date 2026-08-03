package server

import (
	"context"

	"github.com/maintainerd/maintainerd-auth/internal/tenant"
)

// authnTenantResolverAdapter satisfies authn.TenantResolver over the tenant
// service. It resolves a subdomain tenant slug (the DNS name) to its tenant ID
// so the public auth surface can treat the subdomain as the authoritative
// tenant. The authn package stays decoupled from tenant: it declares the
// TenantResolver interface and this app-layer adapter wires it, mirroring
// surfaceClientReaderAdapter.
type authnTenantResolverAdapter struct {
	svc tenant.TenantService
}

// ResolveTenantIDByName returns the tenant ID and is_system flag for the given
// subdomain slug. A slug that does not resolve returns the service error (which
// the caller treats as a hard reject), so the public auth surface never falls
// back to another tenant.
func (a authnTenantResolverAdapter) ResolveTenantIDByName(ctx context.Context, name string) (int64, bool, error) {
	res, err := a.svc.GetByName(ctx, name)
	if err != nil {
		return 0, false, err
	}
	if res == nil {
		return 0, false, nil
	}
	return res.TenantID, res.IsSystem, nil
}

// ResolveSystemTenantID returns the ID of the unique system tenant
// (is_system = true). It lets the OAuth authorize endpoint bind a client_id to
// the system tenant when the request is on the bare system host, so a regular
// tenant's client cannot be driven on the system surface. Decoupling is
// preserved: oauth declares the consumer interface and this app-layer adapter
// wires the tenant service.
func (a authnTenantResolverAdapter) ResolveSystemTenantID(ctx context.Context) (int64, error) {
	res, err := a.svc.GetSystem(ctx)
	if err != nil {
		return 0, err
	}
	if res == nil {
		return 0, nil
	}
	return res.TenantID, nil
}
