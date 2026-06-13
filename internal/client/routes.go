package client

import (
	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/auth/internal/platform/cache"
	"github.com/maintainerd/auth/internal/platform/middleware"
)

func APIKeyRoute(
	r chi.Router,
	apiKeyHandler *APIKeyHandler,
	userService middleware.UserContextProvider,
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

		r.With(middleware.PermissionMiddleware([]string{"api_key:create"}), middleware.RequireStepUp).
			Post("/", apiKeyHandler.Create)

		r.With(middleware.PermissionMiddleware([]string{"api_key:update"}), middleware.RequireStepUp).
			Put("/{api_key_uuid}", apiKeyHandler.Update)

		r.With(middleware.PermissionMiddleware([]string{"api_key:update"}), middleware.RequireStepUp).
			Put("/{api_key_uuid}/status", apiKeyHandler.SetStatus)

		r.With(middleware.PermissionMiddleware([]string{"api_key:delete"}), middleware.RequireStepUp).
			Delete("/{api_key_uuid}", apiKeyHandler.Delete)

		// API Key API operations
		r.Route("/{api_key_uuid}/apis", func(r chi.Router) {
			r.With(middleware.PermissionMiddleware([]string{"api_key:read"})).
				Get("/", apiKeyHandler.GetAPIs)

			r.With(middleware.PermissionMiddleware([]string{"api_key:update"}), middleware.RequireStepUp).
				Post("/", apiKeyHandler.AddAPIs)

			r.With(middleware.PermissionMiddleware([]string{"api_key:update"}), middleware.RequireStepUp).
				Delete("/{api_uuid}", apiKeyHandler.RemoveAPI)

			// API Key API Permission operations
			r.Route("/{api_uuid}/permissions", func(r chi.Router) {
				r.With(middleware.PermissionMiddleware([]string{"api_key:read"})).
					Get("/", apiKeyHandler.GetAPIPermissions)

				r.With(middleware.PermissionMiddleware([]string{"api_key:update"}), middleware.RequireStepUp).
					Post("/", apiKeyHandler.AddAPIPermissions)

				r.With(middleware.PermissionMiddleware([]string{"api_key:update"}), middleware.RequireStepUp).
					Delete("/{permission_uuid}", apiKeyHandler.RemoveAPIPermission)
			})
		})
	})
}

func ClientRoute(
	r chi.Router,
	ClientHandler *ClientHandler,
	userService middleware.UserContextProvider,
	appCache *cache.Cache,
) {
	r.Route("/clients", func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Use(middleware.UserContextMiddleware(userService, appCache))

		r.With(middleware.PermissionMiddleware([]string{"client:read"})).
			Get("/", ClientHandler.Get)

		r.With(middleware.PermissionMiddleware([]string{"client:read"})).
			Get("/{client_uuid}", ClientHandler.GetByUUID)

		r.With(middleware.PermissionMiddleware([]string{"client:secret:read"}), middleware.RequireStepUp).
			Get("/{client_uuid}/secret", ClientHandler.GetSecretByUUID)

		r.With(middleware.PermissionMiddleware([]string{"client:secret:rotate"}), middleware.RequireStepUp).
			Post("/{client_uuid}/rotate-secret", ClientHandler.RotateSecret)

		r.With(middleware.PermissionMiddleware([]string{"client:config:read"})).
			Get("/{client_uuid}/config", ClientHandler.GetConfigByUUID)

		r.With(middleware.PermissionMiddleware([]string{"client:create"})).
			Post("/", ClientHandler.Create)

		r.With(middleware.PermissionMiddleware([]string{"client:update"}), middleware.RequireStepUp).
			Put("/{client_uuid}", ClientHandler.Update)

		r.With(middleware.PermissionMiddleware([]string{"client:update"}), middleware.RequireStepUp).
			Put("/{client_uuid}/status", ClientHandler.SetStatus)

		r.With(middleware.PermissionMiddleware([]string{"client:delete"}), middleware.RequireStepUp).
			Delete("/{client_uuid}", ClientHandler.Delete)

		r.With(middleware.PermissionMiddleware([]string{"client:uri:read"})).
			Get("/{client_uuid}/uris", ClientHandler.GetURIs)

		r.With(middleware.PermissionMiddleware([]string{"client:uri:create"}), middleware.RequireStepUp).
			Post("/{client_uuid}/uris", ClientHandler.CreateURI)

		r.With(middleware.PermissionMiddleware([]string{"client:uri:update"}), middleware.RequireStepUp).
			Put("/{client_uuid}/uris/{client_uri_uuid}", ClientHandler.UpdateURI)

		r.With(middleware.PermissionMiddleware([]string{"client:uri:delete"}), middleware.RequireStepUp).
			Delete("/{client_uuid}/uris/{client_uri_uuid}", ClientHandler.DeleteURI)

		// Auth Client APIs Management
		r.With(middleware.PermissionMiddleware([]string{"client:api:read"})).
			Get("/{client_uuid}/apis", ClientHandler.GetAPIs)

		r.With(middleware.PermissionMiddleware([]string{"client:api:create"}), middleware.RequireStepUp).
			Post("/{client_uuid}/apis", ClientHandler.AddAPIs)

		r.With(middleware.PermissionMiddleware([]string{"client:api:delete"}), middleware.RequireStepUp).
			Delete("/{client_uuid}/apis/{api_uuid}", ClientHandler.RemoveAPI)

		// Auth Client API Permissions Management (nested under APIs)
		r.With(middleware.PermissionMiddleware([]string{"client:api:permission:read"})).
			Get("/{client_uuid}/apis/{api_uuid}/permissions", ClientHandler.GetAPIPermissions)

		r.With(middleware.PermissionMiddleware([]string{"client:api:permission:create"})).
			Post("/{client_uuid}/apis/{api_uuid}/permissions", ClientHandler.AddAPIPermissions)

		r.With(middleware.PermissionMiddleware([]string{"client:api:permission:delete"})).
			Delete("/{client_uuid}/apis/{api_uuid}/permissions/{permission_uuid}", ClientHandler.RemoveAPIPermission)
	})
}
