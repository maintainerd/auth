package route

import (
	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/auth/internal/handler/resthandler"
	"github.com/maintainerd/auth/internal/middleware"
	"github.com/maintainerd/auth/internal/repository"
	"github.com/redis/go-redis/v9"
)

func UserRoute(
	r chi.Router,
	userHandler *resthandler.UserHandler,
	userRepo repository.UserRepository,
	redisClient *redis.Client,
) {
	r.Route("/users", func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Use(middleware.UserContextMiddleware(userRepo, redisClient))

		// Get users with pagination and filtering
		r.With(middleware.PermissionMiddleware([]string{"user:read"})).
			Get("/", userHandler.GetUsers)

		// Get user by UUID
		r.With(middleware.PermissionMiddleware([]string{"user:read"})).
			Get("/{user_uuid}", userHandler.GetUser)

		// Create user
		r.With(middleware.PermissionMiddleware([]string{"user:create"})).
			Post("/", userHandler.CreateUser)

		// Update user
		r.With(middleware.PermissionMiddleware([]string{"user:update"})).
			Put("/{user_uuid}", userHandler.UpdateUser)

		// Set user active status
		r.With(middleware.PermissionMiddleware([]string{"user:update"})).
			Patch("/{user_uuid}/status", userHandler.SetUserActiveStatus)

		// Delete user
		r.With(middleware.PermissionMiddleware([]string{"user:delete"})).
			Delete("/{user_uuid}", userHandler.DeleteUser)

		// Role management
		// Assign roles to user
		r.With(middleware.PermissionMiddleware([]string{"user:role:assign"})).
			Post("/{user_uuid}/roles", userHandler.AssignRoles)

		// Remove role from user
		r.With(middleware.PermissionMiddleware([]string{"user:role:remove"})).
			Delete("/{user_uuid}/roles/{role_uuid}", userHandler.RemoveRole)
	})
}
