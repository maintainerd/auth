package notifier

import (
	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/auth/internal/platform/cache"
	"github.com/maintainerd/auth/internal/platform/middleware"
)

// EmailConfigRoute registers email delivery configuration endpoints.
func EmailConfigRoute(
	r chi.Router,
	emailConfigHandler *EmailConfigHandler,
	userService UserService,
	appCache *cache.Cache,
) {
	r.Route("/email-config", func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Use(middleware.UserContextMiddleware(userService, appCache))

		r.With(middleware.PermissionMiddleware([]string{"email-config:read"})).
			Get("/", emailConfigHandler.Get)
		r.With(middleware.PermissionMiddleware([]string{"email-config:update"})).
			Put("/", emailConfigHandler.Update)
	})
}
