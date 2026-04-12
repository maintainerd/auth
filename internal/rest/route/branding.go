package route

import (
	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/auth/internal/cache"
	"github.com/maintainerd/auth/internal/middleware"
	"github.com/maintainerd/auth/internal/rest/handler"
	"github.com/maintainerd/auth/internal/service"
)

// BrandingRoute registers branding configuration endpoints.
func BrandingRoute(
	r chi.Router,
	brandingHandler *handler.BrandingHandler,
	userService service.UserService,
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
