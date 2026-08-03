package middleware

import (
	"context"
	"net/http"
	"strings"

	resp "github.com/maintainerd/maintainerd-auth/internal/platform/response"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
)

// TenantStatusResolver resolves a subdomain slug to the tenant's lifecycle
// status. Implemented by an app-layer adapter over the tenant service, mirroring
// TenantSlugResolver.
type TenantStatusResolver interface {
	ResolveTenantStatusBySlug(ctx context.Context, slug string) (status string, ok bool, err error)
}

// AuthEndpointTenantStatusMiddleware refuses end-user authentication for a
// tenant that is not active.
//
// Suspending or deactivating a tenant in the console previously changed nothing:
// no code path compared a tenant's status before authenticating, so login,
// registration and token issuance all kept working. This is the enforcement for
// that switch, and it sits pre-auth for the same reason the maintenance gate
// does — a suspended tenant must not be able to mint a session at all.
//
// Scope matches the maintenance gate deliberately: only the identity surface is
// gated, so operators can always still reach the console for the tenant they
// need to un-suspend. A slug that cannot be resolved is passed through — an
// unknown host is the routing layer's problem, not an auth decision — but a
// resolver error fails CLOSED, because "we could not read the tenant's status"
// must not silently mean "active".
func AuthEndpointTenantStatusMiddleware(resolver TenantStatusResolver) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if resolver == nil {
				next.ServeHTTP(w, r)
				return
			}

			rt := ResolveRequestTenantTrusted(r)
			if !rt.OK || rt.Surface != shared.FrontendSurfaceIdentity {
				next.ServeHTTP(w, r)
				return
			}

			status, ok, err := resolver.ResolveTenantStatusBySlug(r.Context(), rt.Slug)
			if err != nil {
				resp.ErrorWithCode(w, http.StatusForbidden, "tenant_unavailable", tenantUnavailableMessage)
				return
			}
			if !ok {
				next.ServeHTTP(w, r)
				return
			}
			if !strings.EqualFold(strings.TrimSpace(status), shared.StatusActive) {
				resp.ErrorWithCode(w, http.StatusForbidden, "tenant_unavailable", tenantUnavailableMessage)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// Deliberately does not distinguish suspended from inactive or unknown: the
// end user cannot act on the difference, and the distinction would disclose
// tenant lifecycle state to anonymous callers.
const tenantUnavailableMessage = "This organization is not available. Please contact your administrator."
