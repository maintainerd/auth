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
	r.Route("/auth_clients", func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Use(middleware.UserContextMiddleware(userRepo, redisClient))

		r.With(middleware.PermissionMiddleware([]string{"auth_client:read"})).
			Get("/", authClientHandler.Get)

		r.With(middleware.PermissionMiddleware([]string{"auth_client:read"})).
			Get("/{auth_client_uuid}", authClientHandler.GetByUUID)

		r.With(middleware.PermissionMiddleware([]string{"auth_client:secret:read"})).
			Get("/{auth_client_uuid}/secret", authClientHandler.GetSecretByUUID)

		r.With(middleware.PermissionMiddleware([]string{"auth_client:config:read"})).
			Get("/{auth_client_uuid}/config", authClientHandler.GetConfigByUUID)

		r.With(middleware.PermissionMiddleware([]string{"auth_client:create"})).
			Post("/", authClientHandler.Create)

		r.With(middleware.PermissionMiddleware([]string{"auth_client:update"})).
			Put("/{auth_client_uuid}", authClientHandler.Update)

		r.With(middleware.PermissionMiddleware([]string{"auth_client:update"})).
			Put("/{auth_client_uuid}/status", authClientHandler.SetStatus)

		r.With(middleware.PermissionMiddleware([]string{"auth_client:delete"})).
			Delete("/{auth_client_uuid}", authClientHandler.Delete)

		r.With(middleware.PermissionMiddleware([]string{"auth_client:redirect_uri:create"})).
			Post("/{auth_client_uuid}/redirect_uris", authClientHandler.CreateRedirectURI)

		r.With(middleware.PermissionMiddleware([]string{"auth_client:redirect_uri:update"})).
			Put("/{auth_client_uuid}/redirect_uris/{auth_client_redirect_uri_uuid}", authClientHandler.UpdateRedirectURI)

		r.With(middleware.PermissionMiddleware([]string{"auth_client:redirect_uri:delete"})).
			Delete("/{auth_client_uuid}/redirect_uris/{auth_client_redirect_uri_uuid}", authClientHandler.DeleteRedirectURI)
	})
}
