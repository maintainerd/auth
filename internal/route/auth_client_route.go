package route

import (
	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/auth/internal/handler/resthandler"
	"github.com/maintainerd/auth/internal/middleware"
	"github.com/maintainerd/auth/internal/repository"
	"github.com/redis/go-redis/v9"
)

func AuthClientRoute(
	r chi.Router,
	authClientHandler *resthandler.AuthClientHandler,
	userRepo repository.UserRepository,
	redisClient *redis.Client,
) {
	r.Route("/clients", func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Use(middleware.UserContextMiddleware(userRepo, redisClient))

		r.With(middleware.PermissionMiddleware([]string{"client:read"})).
			Get("/", authClientHandler.Get)

		r.With(middleware.PermissionMiddleware([]string{"client:read"})).
			Get("/{auth_client_uuid}", authClientHandler.GetByUUID)

		r.With(middleware.PermissionMiddleware([]string{"client:secret:read"})).
			Get("/{auth_client_uuid}/secret", authClientHandler.GetSecretByUUID)

		r.With(middleware.PermissionMiddleware([]string{"client:config:read"})).
			Get("/{auth_client_uuid}/config", authClientHandler.GetConfigByUUID)

		r.With(middleware.PermissionMiddleware([]string{"client:create"})).
			Post("/", authClientHandler.Create)

		r.With(middleware.PermissionMiddleware([]string{"client:update"})).
			Put("/{auth_client_uuid}", authClientHandler.Update)

		r.With(middleware.PermissionMiddleware([]string{"client:update"})).
			Put("/{auth_client_uuid}/status", authClientHandler.SetStatus)

		r.With(middleware.PermissionMiddleware([]string{"client:delete"})).
			Delete("/{auth_client_uuid}", authClientHandler.Delete)

		r.With(middleware.PermissionMiddleware([]string{"client:uri:read"})).
			Get("/{auth_client_uuid}/uris", authClientHandler.GetURIs)

		r.With(middleware.PermissionMiddleware([]string{"client:uri:create"})).
			Post("/{auth_client_uuid}/uris", authClientHandler.CreateURI)

		r.With(middleware.PermissionMiddleware([]string{"client:uri:update"})).
			Put("/{auth_client_uuid}/uris/{auth_client_uri_uuid}", authClientHandler.UpdateURI)

		r.With(middleware.PermissionMiddleware([]string{"client:uri:delete"})).
			Delete("/{auth_client_uuid}/uris/{auth_client_uri_uuid}", authClientHandler.DeleteURI)

		// Auth Client APIs Management
		r.With(middleware.PermissionMiddleware([]string{"client:api:read"})).
			Get("/{auth_client_uuid}/apis", authClientHandler.GetApis)

		r.With(middleware.PermissionMiddleware([]string{"client:api:create"})).
			Post("/{auth_client_uuid}/apis", authClientHandler.AddApis)

		r.With(middleware.PermissionMiddleware([]string{"client:api:delete"})).
			Delete("/{auth_client_uuid}/apis/{api_uuid}", authClientHandler.RemoveApi)

		// Auth Client API Permissions Management (nested under APIs)
		r.With(middleware.PermissionMiddleware([]string{"client:api:permission:read"})).
			Get("/{auth_client_uuid}/apis/{api_uuid}/permissions", authClientHandler.GetApiPermissions)

		r.With(middleware.PermissionMiddleware([]string{"client:api:permission:create"})).
			Post("/{auth_client_uuid}/apis/{api_uuid}/permissions", authClientHandler.AddApiPermissions)

		r.With(middleware.PermissionMiddleware([]string{"client:api:permission:delete"})).
			Delete("/{auth_client_uuid}/apis/{api_uuid}/permissions/{permission_uuid}", authClientHandler.RemoveApiPermission)
	})
}
