package federation

import (
	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/maintainerd-auth/internal/platform/cache"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
)

// WorkloadIdentityFederationRoute registers workload identity federation
// management routes on the internal port (8080). All endpoints require a
// tenant context and the appropriate IAM permission.
func WorkloadIdentityFederationRoute(
	r chi.Router,
	handler *WorkloadIdentityFederationHandler,
	userService middleware.UserContextProvider,
	appCache *cache.Cache,
	rateLimitMiddleware ...middleware.Middleware,
) {
	r.Route("/workload-identity-federations", func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Use(middleware.UserContextMiddleware(userService, appCache))
		r.Use(middleware.OptionalMiddleware(rateLimitMiddleware...))

		r.With(middleware.PermissionMiddleware([]string{"workload-identity-federation:read"})).
			Get("/", handler.GetAll)

		r.With(middleware.PermissionMiddleware([]string{"workload-identity-federation:read"})).
			Get("/{workload_identity_federation_uuid}", handler.Get)

		r.With(middleware.PermissionMiddleware([]string{"workload-identity-federation:create"})).
			Post("/", handler.Create)

		r.With(middleware.PermissionMiddleware([]string{"workload-identity-federation:update"})).
			Put("/{workload_identity_federation_uuid}", handler.Update)

		r.With(middleware.PermissionMiddleware([]string{"workload-identity-federation:delete"})).
			Delete("/{workload_identity_federation_uuid}", handler.Delete)
	})
}
