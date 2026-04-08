package route

import (
	"github.com/maintainerd/auth/internal/rest/handler"
	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/auth/internal/middleware"
	"github.com/maintainerd/auth/internal/service"
	"github.com/maintainerd/auth/internal/cache"
)

func ProfileRoute(
	r chi.Router,
	profileHandler *handler.ProfileHandler,
	userService service.UserService,
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
