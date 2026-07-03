package authevent

import (
	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/maintainerd-auth/internal/platform/cache"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
)

// AuthEventRoute registers admin endpoints for querying auth events.
func AuthEventRoute(
	r chi.Router,
	authEventHandler *AuthEventHandler,
	userService middleware.UserContextProvider,
	appCache *cache.Cache,
	rateLimitMiddleware ...middleware.Middleware,
) {
	r.Route("/auth-events", func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Use(middleware.UserContextMiddleware(userService, appCache))
		r.Use(middleware.OptionalMiddleware(rateLimitMiddleware...))

		r.With(middleware.PermissionMiddleware([]string{"auth_event:read"})).
			Get("/", authEventHandler.GetAll)
		r.With(middleware.PermissionMiddleware([]string{"auth_event:read"})).
			Get("/count", authEventHandler.CountByType)
		r.With(middleware.PermissionMiddleware([]string{"auth_event:read"})).
			Get("/export", authEventHandler.Export)
		r.With(middleware.PermissionMiddleware([]string{"auth_event:read"})).
			Get("/{auth_event_uuid}", authEventHandler.Get)
	})
}
