package branding

import (
	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/auth/internal/platform/cache"
	"github.com/maintainerd/auth/internal/platform/middleware"
)

// BrandingRoute registers authenticated branding configuration endpoints.
func BrandingRoute(
	r chi.Router,
	brandingHandler *BrandingHandler,
	userService middleware.UserContextProvider,
	appCache *cache.Cache,
	rateLimitMiddleware ...middleware.Middleware,
) {
	r.Route("/branding", func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Use(middleware.UserContextMiddleware(userService, appCache))
		r.Use(middleware.OptionalMiddleware(rateLimitMiddleware...))

		r.With(middleware.PermissionMiddleware([]string{"branding:read"})).
			Get("/", brandingHandler.List)
		r.With(middleware.PermissionMiddleware([]string{"branding:create"})).
			Post("/", brandingHandler.Create)
		r.With(middleware.PermissionMiddleware([]string{"branding:update"})).
			Put("/{branding_uuid}", brandingHandler.Update)
		r.With(middleware.PermissionMiddleware([]string{"branding:activate"})).
			Patch("/{branding_uuid}/activate", brandingHandler.Activate)
		r.With(middleware.PermissionMiddleware([]string{"branding:delete"})).
			Delete("/{branding_uuid}", brandingHandler.Delete)
	})
}

// BrandingPublicRoute registers the unauthenticated public branding endpoint
// (non-sensitive: colors + logo only). Mounted on the public router (port 8081).
func BrandingPublicRoute(r chi.Router, brandingHandler *BrandingHandler) {
	r.Get("/public/branding", brandingHandler.GetPublic)
	r.Get("/public/branding/{branding_id}/logo", brandingHandler.ServeLogo)
}

func EmailTemplateRoute(
	r chi.Router,
	emailTemplateHandler *EmailTemplateHandler,
	userService middleware.UserContextProvider,
	appCache *cache.Cache,
	rateLimitMiddleware ...middleware.Middleware,
) {
	r.Route("/email_templates", func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Use(middleware.UserContextMiddleware(userService, appCache))
		r.Use(middleware.OptionalMiddleware(rateLimitMiddleware...))

		r.With(middleware.PermissionMiddleware([]string{"email-template:read"})).
			Get("/", emailTemplateHandler.GetAll)
		r.With(middleware.PermissionMiddleware([]string{"email-template:read"})).
			Get("/{email_template_uuid}", emailTemplateHandler.Get)
		r.With(middleware.PermissionMiddleware([]string{"email-template:update"})).
			Put("/{email_template_uuid}", emailTemplateHandler.Update)
		r.With(middleware.PermissionMiddleware([]string{"email-template:update"})).
			Patch("/{email_template_uuid}/status", emailTemplateHandler.UpdateStatus)
	})
}

func SMSTemplateRoute(
	r chi.Router,
	smsTemplateHandler *SMSTemplateHandler,
	userService middleware.UserContextProvider,
	appCache *cache.Cache,
	rateLimitMiddleware ...middleware.Middleware,
) {
	r.Route("/sms_templates", func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Use(middleware.UserContextMiddleware(userService, appCache))
		r.Use(middleware.OptionalMiddleware(rateLimitMiddleware...))

		r.With(middleware.PermissionMiddleware([]string{"sms-template:read"})).
			Get("/", smsTemplateHandler.GetAll)
		r.With(middleware.PermissionMiddleware([]string{"sms-template:read"})).
			Get("/{sms_template_uuid}", smsTemplateHandler.Get)
		r.With(middleware.PermissionMiddleware([]string{"sms-template:update"})).
			Put("/{sms_template_uuid}", smsTemplateHandler.Update)
		r.With(middleware.PermissionMiddleware([]string{"sms-template:update"})).
			Patch("/{sms_template_uuid}/status", smsTemplateHandler.UpdateStatus)
	})
}
