package route

import (
	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/auth/internal/cache"
	"github.com/maintainerd/auth/internal/middleware"
	"github.com/maintainerd/auth/internal/rest/handler"
	"github.com/maintainerd/auth/internal/service"
)

// AuthEventRoute registers admin endpoints for querying auth events.
func AuthEventRoute(
	r chi.Router,
	authEventHandler *handler.AuthEventHandler,
	userService service.UserService,
	appCache *cache.Cache,
) {
	r.Route("/auth-events", func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Use(middleware.UserContextMiddleware(userService, appCache))

		r.With(middleware.PermissionMiddleware([]string{"auth_event:read"})).
			Get("/", authEventHandler.GetAll)
		r.With(middleware.PermissionMiddleware([]string{"auth_event:read"})).
			Get("/count", authEventHandler.CountByType)
		r.With(middleware.PermissionMiddleware([]string{"auth_event:read"})).
			Get("/{auth_event_uuid}", authEventHandler.Get)
	})
}
