package route

import (
	"github.com/maintainerd/auth/internal/rest/handler"
	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/auth/internal/middleware"
	"github.com/maintainerd/auth/internal/service"
	"github.com/redis/go-redis/v9"
)

func APIRoute(
	r chi.Router,
	apiHandler *handler.APIHandler,
	userService service.UserService,
	redisClient *redis.Client,
) {
	r.Route("/apis", func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Use(middleware.UserContextMiddleware(userService, redisClient))

		r.With(middleware.PermissionMiddleware([]string{"api:read"})).
			Get("/", apiHandler.Get)

		r.With(middleware.PermissionMiddleware([]string{"api:read"})).
			Get("/{api_uuid}", apiHandler.GetByUUID)

		r.With(middleware.PermissionMiddleware([]string{"api:create"})).
			Post("/", apiHandler.Create)

		r.With(middleware.PermissionMiddleware([]string{"api:update"})).
			Put("/{api_uuid}", apiHandler.Update)

		r.With(middleware.PermissionMiddleware([]string{"api:update"})).
			Put("/{api_uuid}/status", apiHandler.SetStatus)

		r.With(middleware.PermissionMiddleware([]string{"api:delete"})).
			Delete("/{api_uuid}", apiHandler.Delete)
	})
}
