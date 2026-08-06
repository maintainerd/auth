package client

import (
	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/maintainerd-auth/internal/platform/cache"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
)

func ClientPublicRoute(r chi.Router, handler *ClientHandler) {
	r.Get("/client", handler.GetPublic)
	r.Get("/client/console", handler.GetPublicConsole)
}

func ClientRoute(
	r chi.Router,
	ClientHandler *ClientHandler,
	userService middleware.UserContextProvider,
	appCache *cache.Cache,
	rateLimitMiddleware ...middleware.Middleware,
) {
	r.Route("/clients", func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Use(middleware.UserContextMiddleware(userService, appCache))
		r.Use(middleware.OptionalMiddleware(rateLimitMiddleware...))

		r.With(middleware.PermissionMiddleware([]string{"client:read"})).
			Get("/", ClientHandler.Get)

		r.With(middleware.PermissionMiddleware([]string{"client:read"})).
			Get("/{client_uuid}", ClientHandler.GetByUUID)

		// There is deliberately no GET /{client_uuid}/secret. Secrets are bcrypt
		// hashed at rest, so nothing can read one back; the route that used to sit
		// here answered 410 unconditionally and existed only to make the seeded
		// client:secret:read permission look like it granted something. Rotation
		// below is the only way to obtain a secret after creation.
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

		// Auth Client identity provider connections
		r.With(middleware.PermissionMiddleware([]string{"client:identity_provider:read"})).
			Get("/{client_uuid}/identity_providers", ClientHandler.GetConnections)

		r.With(middleware.PermissionMiddleware([]string{"client:identity_provider:create"}), middleware.RequireStepUp).
			Post("/{client_uuid}/identity_providers", ClientHandler.AddConnection)

		r.With(middleware.PermissionMiddleware([]string{"client:identity_provider:update"}), middleware.RequireStepUp).
			Put("/{client_uuid}/identity_providers/{client_identity_provider_uuid}", ClientHandler.UpdateConnection)

		r.With(middleware.PermissionMiddleware([]string{"client:identity_provider:delete"}), middleware.RequireStepUp).
			Delete("/{client_uuid}/identity_providers/{client_identity_provider_uuid}", ClientHandler.RemoveConnection)

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

		r.With(middleware.PermissionMiddleware([]string{"client:api:permission:create"}), middleware.RequireStepUp).
			Post("/{client_uuid}/apis/{api_uuid}/permissions", ClientHandler.AddAPIPermissions)

		r.With(middleware.PermissionMiddleware([]string{"client:api:permission:delete"}), middleware.RequireStepUp).
			Delete("/{client_uuid}/apis/{api_uuid}/permissions/{permission_uuid}", ClientHandler.RemoveAPIPermission)

		// Auth Client Role Assignment
		r.With(middleware.PermissionMiddleware([]string{"client:role:read"})).
			Get("/{client_uuid}/roles", ClientHandler.ListRoles)

		r.With(middleware.PermissionMiddleware([]string{"client:role:create"}), middleware.RequireStepUp).
			Post("/{client_uuid}/roles", ClientHandler.AssignRole)

		r.With(middleware.PermissionMiddleware([]string{"client:role:delete"}), middleware.RequireStepUp).
			Delete("/{client_uuid}/roles/{role_uuid}", ClientHandler.RemoveRole)
	})
}
