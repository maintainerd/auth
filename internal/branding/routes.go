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
	userService middleware.UserContextProvider,
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

func EmailTemplateRoute(
	r chi.Router,
	emailTemplateHandler *EmailTemplateHandler,
	userService middleware.UserContextProvider,
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

func LoginTemplateRoute(
	r chi.Router,
	loginTemplateHandler *LoginTemplateHandler,
	userService middleware.UserContextProvider,
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

func SMSTemplateRoute(
	r chi.Router,
	smsTemplateHandler *SMSTemplateHandler,
	userService middleware.UserContextProvider,
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
