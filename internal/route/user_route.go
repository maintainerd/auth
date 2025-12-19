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
	profileHandler *resthandler.ProfileHandler,
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

		// Set user status
		r.With(middleware.PermissionMiddleware([]string{"user:update"})).
			Patch("/{user_uuid}/status", userHandler.SetUserStatus)

		// Delete user
		r.With(middleware.PermissionMiddleware([]string{"user:delete"})).
			Delete("/{user_uuid}", userHandler.DeleteUser)

		// Role management
		// Get user roles
		r.With(middleware.PermissionMiddleware([]string{"user:read"})).
			Get("/{user_uuid}/roles", userHandler.GetUserRoles)

		// Get user identities
		r.With(middleware.PermissionMiddleware([]string{"user:read"})).
			Get("/{user_uuid}/identities", userHandler.GetUserIdentities)

		// Assign roles to user
		r.With(middleware.PermissionMiddleware([]string{"user:create"})).
			Post("/{user_uuid}/roles", userHandler.AssignRoles)

		// Remove role from user
		r.With(middleware.PermissionMiddleware([]string{"user:create"})).
			Delete("/{user_uuid}/roles/{role_uuid}", userHandler.RemoveRole)

		// Profile management (admin access to user profiles)
		// Get all profiles for a user
		r.With(middleware.PermissionMiddleware([]string{"user:read"})).
			Get("/{user_uuid}/profiles", profileHandler.AdminGetAllProfiles)

		// Create new profile for a user
		r.With(middleware.PermissionMiddleware([]string{"user:update"})).
			Post("/{user_uuid}/profiles", profileHandler.AdminCreateProfile)

		// Get specific profile by UUID
		r.With(middleware.PermissionMiddleware([]string{"user:read"})).
			Get("/{user_uuid}/profiles/{profile_uuid}", profileHandler.AdminGetProfile)

		// Update specific profile by UUID
		r.With(middleware.PermissionMiddleware([]string{"user:update"})).
			Put("/{user_uuid}/profiles/{profile_uuid}", profileHandler.AdminUpdateProfile)

		// Set specific profile as default (admin)
		r.With(middleware.PermissionMiddleware([]string{"user:update"})).
			Put("/{user_uuid}/profiles/{profile_uuid}/set-default", profileHandler.AdminSetDefaultProfile)

		// Delete specific profile by UUID
		r.With(middleware.PermissionMiddleware([]string{"user:delete"})).
			Delete("/{user_uuid}/profiles/{profile_uuid}", profileHandler.AdminDeleteProfile)
	})
}
