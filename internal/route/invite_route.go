package route

import (
	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/auth/internal/handler/resthandler"
	"github.com/maintainerd/auth/internal/middleware"
	"github.com/maintainerd/auth/internal/repository"
	"github.com/redis/go-redis/v9"
)

func InviteRoute(
	r chi.Router,
	inviteHandler *resthandler.InviteHandler,
	userRepo repository.UserRepository,
	redisClient *redis.Client,
) {
	r.Route("/internal/invite", func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Use(middleware.UserContextMiddleware(userRepo, redisClient))

		r.Post("/", inviteHandler.SendPrivate)
	})
}
