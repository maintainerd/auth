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

		r.Post("/", roleHandler.Create)
		r.Get("/", roleHandler.GetAll)
		r.Get("/{role_uuid}", roleHandler.GetByUUID)
		r.Put("/{role_uuid}", roleHandler.Update)
		r.Delete("/{role_uuid}", roleHandler.Delete)
	})
}
