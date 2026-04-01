package route

import (
	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/auth/internal/handler/resthandler"
	"github.com/maintainerd/auth/internal/middleware"
	"github.com/maintainerd/auth/internal/repository"
	"github.com/redis/go-redis/v9"
)

func ClientRoute(
	r chi.Router,
	ClientHandler *resthandler.ClientHandler,
	userRepo repository.UserRepository,
	redisClient *redis.Client,
) {
	r.Route("/clients", func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Use(middleware.UserContextMiddleware(userRepo, redisClient))

		r.With(middleware.PermissionMiddleware([]string{"client:read"})).
			Get("/", ClientHandler.Get)

		r.With(middleware.PermissionMiddleware([]string{"client:read"})).
			Get("/{client_uuid}", ClientHandler.GetByUUID)

		r.With(middleware.PermissionMiddleware([]string{"client:secret:read"})).
			Get("/{client_uuid}/secret", ClientHandler.GetSecretByUUID)

		r.With(middleware.PermissionMiddleware([]string{"client:config:read"})).
			Get("/{client_uuid}/config", ClientHandler.GetConfigByUUID)

		r.With(middleware.PermissionMiddleware([]string{"client:create"})).
			Post("/", ClientHandler.Create)

		r.With(middleware.PermissionMiddleware([]string{"client:update"})).
			Put("/{client_uuid}", ClientHandler.Update)

		r.With(middleware.PermissionMiddleware([]string{"client:update"})).
			Put("/{client_uuid}/status", ClientHandler.SetStatus)

		r.With(middleware.PermissionMiddleware([]string{"client:delete"})).
			Delete("/{client_uuid}", ClientHandler.Delete)

		r.With(middleware.PermissionMiddleware([]string{"client:uri:read"})).
			Get("/{client_uuid}/uris", ClientHandler.GetURIs)

		r.With(middleware.PermissionMiddleware([]string{"client:uri:create"})).
			Post("/{client_uuid}/uris", ClientHandler.CreateURI)

		r.With(middleware.PermissionMiddleware([]string{"client:uri:update"})).
			Put("/{client_uuid}/uris/{client_uri_uuid}", ClientHandler.UpdateURI)

		r.With(middleware.PermissionMiddleware([]string{"client:uri:delete"})).
			Delete("/{client_uuid}/uris/{client_uri_uuid}", ClientHandler.DeleteURI)

		// Auth Client APIs Management
		r.With(middleware.PermissionMiddleware([]string{"client:api:read"})).
			Get("/{client_uuid}/apis", ClientHandler.GetAPIs)

		r.With(middleware.PermissionMiddleware([]string{"client:api:create"})).
			Post("/{client_uuid}/apis", ClientHandler.AddAPIs)

		r.With(middleware.PermissionMiddleware([]string{"client:api:delete"})).
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
