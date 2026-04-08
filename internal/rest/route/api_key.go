package route

import (
	"github.com/maintainerd/auth/internal/rest/handler"
	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/auth/internal/middleware"
	"github.com/maintainerd/auth/internal/service"
	"github.com/maintainerd/auth/internal/cache"
)

func APIKeyRoute(
	r chi.Router,
	apiKeyHandler *handler.APIKeyHandler,
	userService service.UserService,
	appCache *cache.Cache,
) {
	r.Route("/api_keys", func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Use(middleware.UserContextMiddleware(userService, appCache))

		// API Key CRUD operations
		r.With(middleware.PermissionMiddleware([]string{"api_key:read"})).
			Get("/", apiKeyHandler.Get)

		r.With(middleware.PermissionMiddleware([]string{"api_key:read"})).
			Get("/{api_key_uuid}", apiKeyHandler.GetByUUID)

		r.With(middleware.PermissionMiddleware([]string{"api_key:read"})).
			Get("/{api_key_uuid}/config", apiKeyHandler.GetConfigByUUID)

		r.With(middleware.PermissionMiddleware([]string{"api_key:create"})).
			Post("/", apiKeyHandler.Create)

		r.With(middleware.PermissionMiddleware([]string{"api_key:update"})).
			Put("/{api_key_uuid}", apiKeyHandler.Update)

		r.With(middleware.PermissionMiddleware([]string{"api_key:update"})).
			Put("/{api_key_uuid}/status", apiKeyHandler.SetStatus)

		r.With(middleware.PermissionMiddleware([]string{"api_key:delete"})).
			Delete("/{api_key_uuid}", apiKeyHandler.Delete)

		// API Key API operations
		r.Route("/{api_key_uuid}/apis", func(r chi.Router) {
			r.With(middleware.PermissionMiddleware([]string{"api_key:read"})).
				Get("/", apiKeyHandler.GetAPIs)

			r.With(middleware.PermissionMiddleware([]string{"api_key:update"})).
				Post("/", apiKeyHandler.AddAPIs)

			r.With(middleware.PermissionMiddleware([]string{"api_key:update"})).
				Delete("/{api_uuid}", apiKeyHandler.RemoveAPI)

			// API Key API Permission operations
			r.Route("/{api_uuid}/permissions", func(r chi.Router) {
				r.With(middleware.PermissionMiddleware([]string{"api_key:read"})).
					Get("/", apiKeyHandler.GetAPIPermissions)

				r.With(middleware.PermissionMiddleware([]string{"api_key:update"})).
					Post("/", apiKeyHandler.AddAPIPermissions)

				r.With(middleware.PermissionMiddleware([]string{"api_key:update"})).
					Delete("/{permission_uuid}", apiKeyHandler.RemoveAPIPermission)
			})
		})
	})
}
