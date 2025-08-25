package route

import (
	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/auth/internal/handler/resthandler"
	"github.com/maintainerd/auth/internal/middleware"
	"github.com/maintainerd/auth/internal/repository"
	"github.com/redis/go-redis/v9"
)

func ProfileRoute(
	r chi.Router,
	profileHandler *resthandler.ProfileHandler,
	userRepo repository.UserRepository,
	redisClient *redis.Client,
) {
	r.Route("/profiles", func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Use(middleware.UserContextMiddleware(userRepo, redisClient))

		r.Post("/", profileHandler.CreateOrUpdate)
		r.Get("/", profileHandler.Get)
		r.Delete("/", profileHandler.Delete)
	})
}
