package branding

import (
	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/auth/internal/platform/cache"
	"github.com/maintainerd/auth/internal/platform/middleware"
)

func LoginTemplateRoute(
	r chi.Router,
	loginTemplateHandler *LoginTemplateHandler,
	userService UserService,
	appCache *cache.Cache,
) {
	r.Route("/login_templates", func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Use(middleware.UserContextMiddleware(userService, appCache))

		// List login templates
		r.With(middleware.PermissionMiddleware([]string{"login-template:read"})).
			Get("/", loginTemplateHandler.GetAll)

		// Get single login template
		r.With(middleware.PermissionMiddleware([]string{"login-template:read"})).
			Get("/{login_template_uuid}", loginTemplateHandler.Get)

		// Create login template
		r.With(middleware.PermissionMiddleware([]string{"login-template:create"})).
			Post("/", loginTemplateHandler.Create)

		// Update login template
		r.With(middleware.PermissionMiddleware([]string{"login-template:update"})).
			Put("/{login_template_uuid}", loginTemplateHandler.Update)

		// Delete login template
		r.With(middleware.PermissionMiddleware([]string{"login-template:delete"})).
			Delete("/{login_template_uuid}", loginTemplateHandler.Delete)

		// Update login template status
		r.With(middleware.PermissionMiddleware([]string{"login-template:update"})).
			Patch("/{login_template_uuid}/status", loginTemplateHandler.UpdateStatus)
	})
}
