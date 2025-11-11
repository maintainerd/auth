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
	r.Route("/profile", func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Use(middleware.UserContextMiddleware(userRepo, redisClient))

		// Create or update profile - requires profile update permission
		r.With(middleware.PermissionMiddleware([]string{"account:profile:update:self"})).
			Post("/", profileHandler.CreateOrUpdate)

		// Get profile - requires profile read permission
		r.With(middleware.PermissionMiddleware([]string{"account:profile:read:self"})).
			Get("/", profileHandler.Get)

		// Delete profile - requires profile delete permission
		r.With(middleware.PermissionMiddleware([]string{"account:profile:delete:self"})).
			Delete("/", profileHandler.Delete)
	})
}
