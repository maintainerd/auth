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

		r.With(middleware.PermissionMiddleware([]string{"auth_client:create"})).
			Post("/", authClientHandler.Create)

		r.With(middleware.PermissionMiddleware([]string{"auth_client:update"})).
			Put("/{auth_client_uuid}", authClientHandler.Update)

		r.With(middleware.PermissionMiddleware([]string{"auth_client:update"})).
			Put("/{auth_client_uuid}/status", authClientHandler.SetStatus)

		r.With(middleware.PermissionMiddleware([]string{"auth_client:delete"})).
			Delete("/{auth_client_uuid}", authClientHandler.Delete)
	})
}
