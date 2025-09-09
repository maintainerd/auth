package route

import (
	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/auth/internal/handler/resthandler"
	"github.com/maintainerd/auth/internal/middleware"
	"github.com/maintainerd/auth/internal/repository"
	"github.com/redis/go-redis/v9"
)

func AuthContainerRoute(
	r chi.Router,
	authContainerHandler *resthandler.AuthContainerHandler,
	userRepo repository.UserRepository,
	redisClient *redis.Client,
) {
	r.Route("/auth_containers", func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Use(middleware.UserContextMiddleware(userRepo, redisClient))

		r.With(middleware.PermissionMiddleware([]string{"auth_container:read"})).
			Get("/", authContainerHandler.Get)

		r.With(middleware.PermissionMiddleware([]string{"auth_container:read"})).
			Get("/{auth_container_uuid}", authContainerHandler.GetByUUID)

		r.With(middleware.PermissionMiddleware([]string{"auth_container:create"})).
			Post("/", authContainerHandler.Create)

		r.With(middleware.PermissionMiddleware([]string{"auth_container:update"})).
			Put("/{auth_container_uuid}", authContainerHandler.Update)

		r.With(middleware.PermissionMiddleware([]string{"auth_container:update"})).
			Put("/{auth_container_uuid}/status", authContainerHandler.SetStatus)

		r.With(middleware.PermissionMiddleware([]string{"auth_container:update"})).
			Put("/{auth_container_uuid}/public", authContainerHandler.SetPublic)

		r.With(middleware.PermissionMiddleware([]string{"auth_container:delete"})).
			Delete("/{auth_container_uuid}", authContainerHandler.Delete)
	})
}
