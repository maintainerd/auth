package route

import (
	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/auth/internal/handler/resthandler"
	"github.com/maintainerd/auth/internal/middleware"
	"github.com/maintainerd/auth/internal/repository"
	"github.com/redis/go-redis/v9"
)

func PermissionRoute(
	r chi.Router,
	oermissionHandler *resthandler.PermissionHandler,
	userRepo repository.UserRepository,
	redisClient *redis.Client,
) {
	r.Route("/permissions", func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Use(middleware.UserContextMiddleware(userRepo, redisClient))

		r.With(middleware.PermissionMiddleware([]string{"permission:read"})).
			Get("/", oermissionHandler.Get)

		r.With(middleware.PermissionMiddleware([]string{"permission:read"})).
			Get("/{permission_uuid}", oermissionHandler.GetByUUID)

		r.With(middleware.PermissionMiddleware([]string{"permission:create"})).
			Post("/", oermissionHandler.Create)

		r.With(middleware.PermissionMiddleware([]string{"permission:update"})).
			Put("/{permission_uuid}", oermissionHandler.Update)

		r.With(middleware.PermissionMiddleware([]string{"permission:update"})).
			Put("/{permission_uuid}/status", oermissionHandler.SetStatus)

		r.With(middleware.PermissionMiddleware([]string{"permission:delete"})).
			Delete("/{permission_uuid}", oermissionHandler.Delete)
	})
}
