package user

import (
	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/auth/internal/platform/cache"
	"github.com/maintainerd/auth/internal/platform/middleware"
)

// AccountRoute mounts authenticated self-service account management endpoints.
func AccountRoute(
	r chi.Router,
	accountHandler *AccountHandler,
	userService middleware.UserContextProvider,
	appCache *cache.Cache,
) {
	r.Route("/account", func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Use(middleware.UserContextMiddleware(userService, appCache))

		// Email change flow
		r.Post("/email/change", accountHandler.InitiateEmailChange)
		r.Post("/email/verify", accountHandler.VerifyEmailChange)

		// Username change
		r.Put("/username", accountHandler.ChangeUsername)

		// Account deletion
		r.With(middleware.RequireStepUp).Delete("/", accountHandler.DeleteAccount)

		// Account data export (GDPR / data portability)
		r.Get("/export", accountHandler.ExportAccountData)

		// Backup codes — generate and store securely
		r.With(middleware.RequireStepUp).Post("/backup-codes", accountHandler.GenerateBackupCodes)

		// Session management
		r.Get("/sessions", accountHandler.ListSessions)
		r.With(middleware.RequireStepUp).Delete("/sessions", accountHandler.RevokeAllSessions)
		r.Delete("/sessions/{session_uuid}", accountHandler.RevokeSession)
	})
}

// RecoveryRoute mounts unauthenticated account recovery endpoints.
func RecoveryRoute(
	r chi.Router,
	accountHandler *AccountHandler,
) {
	r.Route("/recovery", func(r chi.Router) {
		// Backup-code based account recovery (issues tokens)
		r.Post("/backup-code", accountHandler.VerifyBackupCode)
	})
}

func ProfileRoute(
	r chi.Router,
	profileHandler *ProfileHandler,
	userService middleware.UserContextProvider,
	appCache *cache.Cache,
) {
	// /profile - Default profile operations (shortcut for convenience)
	r.Route("/profile", func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Use(middleware.UserContextMiddleware(userService, appCache))

		// Get default profile
		r.With(middleware.PermissionMiddleware([]string{"account:profile:read:self"})).
			Get("/", profileHandler.Get)

		// Create or update default profile (combined for convenience)
		r.With(middleware.PermissionMiddleware([]string{"account:profile:update:self"})).
			Post("/", profileHandler.CreateOrUpdate)

		// Update default profile
		r.With(middleware.PermissionMiddleware([]string{"account:profile:update:self"})).
			Put("/", profileHandler.CreateOrUpdate)

		// Delete default profile
		r.With(middleware.PermissionMiddleware([]string{"account:profile:delete:self"})).
			Delete("/", profileHandler.Delete)
	})

	// /profiles - All profiles operations (including default, with full CRUD)
	r.Route("/profiles", func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Use(middleware.UserContextMiddleware(userService, appCache))

		// Get all profiles with pagination and filtering
		r.With(middleware.PermissionMiddleware([]string{"account:profile:read:self"})).
			Get("/", profileHandler.GetAll)

		// Create new profile (auto-generate UUID)
		r.With(middleware.PermissionMiddleware([]string{"account:profile:update:self"})).
			Post("/", profileHandler.CreateProfile)

		// Get specific profile by UUID
		r.With(middleware.PermissionMiddleware([]string{"account:profile:read:self"})).
			Get("/{profile_uuid}", profileHandler.GetByUUID)

		// Update specific profile by UUID
		r.With(middleware.PermissionMiddleware([]string{"account:profile:update:self"})).
			Put("/{profile_uuid}", profileHandler.UpdateProfile)

		// Set specific profile as default
		r.With(middleware.PermissionMiddleware([]string{"account:profile:update:self"})).
			Patch("/{profile_uuid}/set-default", profileHandler.SetDefaultProfile)

		// Delete specific profile by UUID
		r.With(middleware.PermissionMiddleware([]string{"account:profile:delete:self"})).
			Delete("/{profile_uuid}", profileHandler.DeleteByUUID)
	})
}

func UserRoute(
	r chi.Router,
	userHandler *UserHandler,
	profileHandler *ProfileHandler,
	userService middleware.UserContextProvider,
	appCache *cache.Cache,
) {
	r.Route("/users", func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Use(middleware.UserContextMiddleware(userService, appCache))

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
		r.With(middleware.PermissionMiddleware([]string{"user:update"}), middleware.RequireStepUp).
			Patch("/{user_uuid}/status", userHandler.SetUserStatus)

		// Verify email (also marks account as completed)
		r.With(middleware.PermissionMiddleware([]string{"user:update"})).
			Patch("/{user_uuid}/verify-email", userHandler.VerifyEmail)

		// Verify phone
		r.With(middleware.PermissionMiddleware([]string{"user:update"})).
			Patch("/{user_uuid}/verify-phone", userHandler.VerifyPhone)

		// Mark account as completed
		r.With(middleware.PermissionMiddleware([]string{"user:update"})).
			Patch("/{user_uuid}/complete-account", userHandler.CompleteAccount)

		// Delete user
		r.With(middleware.PermissionMiddleware([]string{"user:delete"}), middleware.RequireStepUp).
			Delete("/{user_uuid}", userHandler.DeleteUser)

		// Force password change on next login
		r.With(middleware.PermissionMiddleware([]string{"user:update"}), middleware.RequireStepUp).
			Put("/{user_uuid}/force-password-change", userHandler.ForcePasswordChange)

		// Role management
		// Get user roles
		r.With(middleware.PermissionMiddleware([]string{"user:read"})).
			Get("/{user_uuid}/roles", userHandler.GetUserRoles)

		// Get user identities
		r.With(middleware.PermissionMiddleware([]string{"user:read"})).
			Get("/{user_uuid}/identities", userHandler.GetUserIdentities)

		// Assign roles to user
		r.With(middleware.PermissionMiddleware([]string{"user:create"}), middleware.RequireStepUp).
			Post("/{user_uuid}/roles", userHandler.AssignRoles)

		// Remove role from user
		r.With(middleware.PermissionMiddleware([]string{"user:create"}), middleware.RequireStepUp).
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

// UserPoolRoute mounts tenant-scoped user pool management endpoints.
func UserPoolRoute(
	r chi.Router,
	userPoolHandler *UserPoolHandler,
	userService middleware.UserContextProvider,
	appCache *cache.Cache,
) {
	r.Route("/user-pools", func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Use(middleware.UserContextMiddleware(userService, appCache))

		// List user pools
		r.With(middleware.PermissionMiddleware([]string{"user-pool:read"})).
			Get("/", userPoolHandler.GetUserPools)

		// Get user pool by UUID
		r.With(middleware.PermissionMiddleware([]string{"user-pool:read"})).
			Get("/{user_pool_uuid}", userPoolHandler.GetUserPool)

		// Create user pool
		r.With(middleware.PermissionMiddleware([]string{"user-pool:create"})).
			Post("/", userPoolHandler.CreateUserPool)

		// Update user pool
		r.With(middleware.PermissionMiddleware([]string{"user-pool:update"})).
			Put("/{user_pool_uuid}", userPoolHandler.UpdateUserPool)

		// Update user pool status
		r.With(middleware.PermissionMiddleware([]string{"user-pool:update"})).
			Patch("/{user_pool_uuid}/status", userPoolHandler.SetStatus)

		// Delete user pool
		r.With(middleware.PermissionMiddleware([]string{"user-pool:delete"}), middleware.RequireStepUp).
			Delete("/{user_pool_uuid}", userPoolHandler.DeleteUserPool)
	})
}

func UserSettingRoute(
	r chi.Router,
	userSettingHandler *UserSettingHandler,
	userService middleware.UserContextProvider,
	appCache *cache.Cache,
) {
	r.Route("/user-settings", func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Use(middleware.UserContextMiddleware(userService, appCache))

		// Create or update user settings - requires settings update permission
		r.With(middleware.PermissionMiddleware([]string{"settings:update:self"})).
			Post("/", userSettingHandler.CreateOrUpdate)

		// Get user settings - requires settings read permission
		r.With(middleware.PermissionMiddleware([]string{"settings:read:self"})).
			Get("/", userSettingHandler.Get)

		// Delete user settings - requires settings update permission (since it's modifying settings)
		r.With(middleware.PermissionMiddleware([]string{"settings:update:self"})).
			Delete("/", userSettingHandler.Delete)
	})
}
