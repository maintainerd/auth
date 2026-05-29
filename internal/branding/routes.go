package branding

import (
	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/auth/internal/platform/cache"
	"github.com/maintainerd/auth/internal/platform/middleware"
)

// BrandingRoute registers branding configuration endpoints.
func BrandingRoute(
	r chi.Router,
	brandingHandler *BrandingHandler,
	userService UserService,
	appCache *cache.Cache,
) {
	r.Route("/branding", func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Use(middleware.UserContextMiddleware(userService, appCache))

		r.With(middleware.PermissionMiddleware([]string{"branding:read"})).
			Get("/", brandingHandler.Get)
		r.With(middleware.PermissionMiddleware([]string{"branding:update"})).
			Put("/", brandingHandler.Update)
	})
}
