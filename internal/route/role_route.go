package route

import (
	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/auth/internal/handler/resthandler"
	"github.com/maintainerd/auth/internal/middleware"
	"github.com/maintainerd/auth/internal/repository"
	"github.com/redis/go-redis/v9"
)

func RoleRoute(
	r chi.Router,
	roleHandler *resthandler.RoleHandler,
	userRepo repository.UserRepository,
	redisClient *redis.Client,
) {
	r.Route("/roles", func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Use(middleware.UserContextMiddleware(userRepo, redisClient))

		r.With(middleware.PermissionMiddleware([]string{"role:read"})).
			Get("/", roleHandler.Get)

		r.With(middleware.PermissionMiddleware([]string{"role:read"})).
			Get("/{role_uuid}", roleHandler.GetByUUID)

		r.With(middleware.PermissionMiddleware([]string{"role:create"})).
			Post("/", roleHandler.Create)

		r.With(middleware.PermissionMiddleware([]string{"role:update"})).
			Put("/{role_uuid}", roleHandler.Update)

		r.With(middleware.PermissionMiddleware([]string{"role:update"})).
			Put("/{role_uuid}/status", roleHandler.SetStatus)

		r.With(middleware.PermissionMiddleware([]string{"role:delete"})).
			Delete("/{role_uuid}", roleHandler.Delete)
	})
}
