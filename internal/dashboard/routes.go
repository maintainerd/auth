package dashboard

import (
	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/maintainerd-auth/internal/platform/cache"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
)

func DashboardRoute(
	r chi.Router,
	handler *Handler,
	userService middleware.UserContextProvider,
	appCache *cache.Cache,
	rateLimitMiddleware ...middleware.Middleware,
) {
	r.Route("/dashboard", func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Use(middleware.UserContextMiddleware(userService, appCache))
		r.Use(middleware.OptionalMiddleware(rateLimitMiddleware...))

		// tenant:read is REQUIRED here. /summary returns tenant-wide aggregates —
		// user, service, client, identity-provider and role counts plus auth-event
		// volume — and authentication alone used to be enough to read them. That
		// made this the one management route in the internal router with no
		// permission guard, so any principal in the tenant (an invited member with
		// a single self-service role, a machine client) could size the whole
		// tenant. tenant:read is the same permission the tenant-wide reads in
		// tenant/routes.go require, which is the scope this aggregate covers.
		r.With(middleware.PermissionMiddleware([]string{"tenant:read"})).
			Get("/summary", handler.GetSummary)
	})
}
