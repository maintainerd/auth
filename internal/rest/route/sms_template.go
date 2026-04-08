package route

import (
	"github.com/maintainerd/auth/internal/rest/handler"
	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/auth/internal/middleware"
	"github.com/maintainerd/auth/internal/service"
	"github.com/maintainerd/auth/internal/cache"
)

func SMSTemplateRoute(
	r chi.Router,
	smsTemplateHandler *handler.SMSTemplateHandler,
	userService service.UserService,
	appCache *cache.Cache,
) {
	r.Route("/sms_templates", func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Use(middleware.UserContextMiddleware(userService, appCache))

		// List SMS templates
		r.With(middleware.PermissionMiddleware([]string{"sms-template:read"})).
			Get("/", smsTemplateHandler.GetAll)

		// Get single SMS template
		r.With(middleware.PermissionMiddleware([]string{"sms-template:read"})).
			Get("/{sms_template_uuid}", smsTemplateHandler.Get)

		// Create SMS template
		r.With(middleware.PermissionMiddleware([]string{"sms-template:create"})).
			Post("/", smsTemplateHandler.Create)

		// Update SMS template
		r.With(middleware.PermissionMiddleware([]string{"sms-template:update"})).
			Put("/{sms_template_uuid}", smsTemplateHandler.Update)

		// Delete SMS template
		r.With(middleware.PermissionMiddleware([]string{"sms-template:delete"})).
			Delete("/{sms_template_uuid}", smsTemplateHandler.Delete)

		// Update SMS template status
		r.With(middleware.PermissionMiddleware([]string{"sms-template:update"})).
			Patch("/{sms_template_uuid}/status", smsTemplateHandler.UpdateStatus)
	})
}
