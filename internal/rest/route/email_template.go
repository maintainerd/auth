package route

import (
	"github.com/maintainerd/auth/internal/rest/handler"
	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/auth/internal/middleware"
	"github.com/maintainerd/auth/internal/service"
	"github.com/maintainerd/auth/internal/cache"
)

func EmailTemplateRoute(
	r chi.Router,
	emailTemplateHandler *handler.EmailTemplateHandler,
	userService service.UserService,
	appCache *cache.Cache,
) {
	r.Route("/email_templates", func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Use(middleware.UserContextMiddleware(userService, appCache))

		// List email templates
		r.With(middleware.PermissionMiddleware([]string{"email-template:read"})).
			Get("/", emailTemplateHandler.GetAll)

		// Get single email template
		r.With(middleware.PermissionMiddleware([]string{"email-template:read"})).
			Get("/{email_template_uuid}", emailTemplateHandler.Get)

		// Create email template
		r.With(middleware.PermissionMiddleware([]string{"email-template:create"})).
			Post("/", emailTemplateHandler.Create)

		// Update email template
		r.With(middleware.PermissionMiddleware([]string{"email-template:update"})).
			Put("/{email_template_uuid}", emailTemplateHandler.Update)

		// Delete email template
		r.With(middleware.PermissionMiddleware([]string{"email-template:delete"})).
			Delete("/{email_template_uuid}", emailTemplateHandler.Delete)

		// Update email template status
		r.With(middleware.PermissionMiddleware([]string{"email-template:update"})).
			Patch("/{email_template_uuid}/status", emailTemplateHandler.UpdateStatus)
	})
}
