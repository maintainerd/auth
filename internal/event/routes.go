package event

import (
	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/maintainerd-auth/internal/platform/cache"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
)

// ConfigRoute registers event configuration routes.
func ConfigRoute(
	r chi.Router,
	configHandler *ConfigHandler,
	managementHandler *ManagementHandler,
	userService middleware.UserContextProvider,
	appCache *cache.Cache,
	rateLimitMiddleware ...middleware.Middleware,
) {
	r.Route("/event-types", func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Use(middleware.UserContextMiddleware(userService, appCache))
		r.Use(middleware.OptionalMiddleware(rateLimitMiddleware...))

		r.With(middleware.PermissionMiddleware([]string{"webhook-endpoint:read"})).
			Get("/", configHandler.ListEventTypes)
	})

	r.Route("/tenant-event-types", func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Use(middleware.UserContextMiddleware(userService, appCache))
		r.Use(middleware.OptionalMiddleware(rateLimitMiddleware...))

		r.With(middleware.PermissionMiddleware([]string{"webhook-endpoint:read"})).
			Get("/", configHandler.GetTenantEventTypes)

		r.With(middleware.PermissionMiddleware([]string{"webhook-endpoint:update"})).
			Put("/", configHandler.SetTenantEventType)
	})

	r.Route("/event-routes", func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Use(middleware.UserContextMiddleware(userService, appCache))
		r.Use(middleware.OptionalMiddleware(rateLimitMiddleware...))

		r.With(middleware.PermissionMiddleware([]string{"webhook-endpoint:read"})).
			Get("/", managementHandler.ListEventRoutes)

		r.With(middleware.PermissionMiddleware([]string{"webhook-endpoint:read"})).
			Get("/{event_route_uuid}", managementHandler.GetEventRoute)

		r.With(middleware.PermissionMiddleware([]string{"webhook-endpoint:create"})).
			Post("/", managementHandler.CreateEventRoute)

		r.With(middleware.PermissionMiddleware([]string{"webhook-endpoint:update"})).
			Put("/{event_route_uuid}", managementHandler.UpdateEventRoute)

		r.With(middleware.PermissionMiddleware([]string{"webhook-endpoint:delete"})).
			Delete("/{event_route_uuid}", managementHandler.DeleteEventRoute)
	})
}
