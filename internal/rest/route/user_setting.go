package route

import (
	"github.com/maintainerd/auth/internal/rest/handler"
	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/auth/internal/middleware"
	"github.com/maintainerd/auth/internal/service"
	"github.com/maintainerd/auth/internal/cache"
)

func UserSettingRoute(
	r chi.Router,
	userSettingHandler *handler.UserSettingHandler,
	userService service.UserService,
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
