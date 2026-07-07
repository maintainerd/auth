package scim

import (
	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/maintainerd-auth/internal/platform/cache"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
)

func SCIMConfigurationRoute(
	r chi.Router,
	handler *SCIMConfigurationHandler,
	userService middleware.UserContextProvider,
	appCache *cache.Cache,
	rateLimitMiddleware ...middleware.Middleware,
) {
	r.Route("/scim-configurations", func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Use(middleware.UserContextMiddleware(userService, appCache))
		r.Use(middleware.OptionalMiddleware(rateLimitMiddleware...))

		r.With(middleware.PermissionMiddleware([]string{"scim-configuration:read"})).
			Get("/", handler.GetAll)
		r.With(middleware.PermissionMiddleware([]string{"scim-configuration:read"})).
			Get("/{scim_configuration_uuid}", handler.Get)
		r.With(middleware.PermissionMiddleware([]string{"scim-configuration:create"})).
			Post("/", handler.Create)
		r.With(middleware.PermissionMiddleware([]string{"scim-configuration:update"})).
			Put("/{scim_configuration_uuid}", handler.Update)
		r.With(middleware.PermissionMiddleware([]string{"scim-configuration:delete"})).
			Delete("/{scim_configuration_uuid}", handler.Delete)
	})
}

func SCIMProtocolRoute(
	r chi.Router,
	userHandler *SCIMUserHandler,
	scimConfigRepo SCIMConfigurationRepository,
) {
	scimMW := NewSCIMBearerMiddleware(scimConfigRepo)

	r.Route("/scim/v2", func(r chi.Router) {
		r.Use(scimMW)

		r.Get("/ServiceProviderConfig", HandleServiceProviderConfig)
		r.Get("/Schemas", HandleSchemas)
		r.Get("/Users", userHandler.List)
		r.Post("/Users", userHandler.Create)
		r.Get("/Users/{userID}", userHandler.Get)
		r.Put("/Users/{userID}", userHandler.Update)
		r.Patch("/Users/{userID}", userHandler.Patch)
		r.Delete("/Users/{userID}", userHandler.Delete)
	})
}
