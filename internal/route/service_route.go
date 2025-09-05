package route

import (
	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/auth/internal/handler/resthandler"
	"github.com/maintainerd/auth/internal/middleware"
	"github.com/maintainerd/auth/internal/repository"
	"github.com/redis/go-redis/v9"
)

func ServiceRoute(
	r chi.Router,
	serviceHandler *resthandler.ServiceHandler,
	userRepo repository.UserRepository,
	redisClient *redis.Client,
) {
	r.Route("/services", func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Use(middleware.UserContextMiddleware(userRepo, redisClient))

		r.With(middleware.PermissionMiddleware([]string{"service:read"})).
			Get("/", serviceHandler.Get)

		r.With(middleware.PermissionMiddleware([]string{"service:read"})).
			Get("/{service_uuid}", serviceHandler.GetByUUID)

		r.With(middleware.PermissionMiddleware([]string{"service:create"})).
			Post("/", serviceHandler.Create)

		r.With(middleware.PermissionMiddleware([]string{"service:update"})).
			Put("/{service_uuid}", serviceHandler.Update)

		r.With(middleware.PermissionMiddleware([]string{"service:delete"})).
			Delete("/{service_uuid}", serviceHandler.Delete)
	})
}
