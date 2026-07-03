package notifier

import (
	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/maintainerd-auth/internal/platform/cache"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
)

// EmailConfigRoute registers email delivery configuration endpoints.
func EmailConfigRoute(
	r chi.Router,
	emailConfigHandler *EmailConfigHandler,
	userService middleware.UserContextProvider,
	appCache *cache.Cache,
	rateLimitMiddleware ...middleware.Middleware,
) {
	r.Route("/email-config", func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Use(middleware.UserContextMiddleware(userService, appCache))
		r.Use(middleware.OptionalMiddleware(rateLimitMiddleware...))

		r.With(middleware.PermissionMiddleware([]string{"email-config:read"})).
			Get("/", emailConfigHandler.Get)
		r.With(middleware.PermissionMiddleware([]string{"email-config:read"})).
			Get("/status", emailConfigHandler.Status)
		r.With(middleware.PermissionMiddleware([]string{"email-config:update"})).
			Put("/", emailConfigHandler.Update)
	})
}

// SMSConfigRoute registers SMS delivery configuration endpoints.
func SMSConfigRoute(
	r chi.Router,
	smsConfigHandler *SMSConfigHandler,
	userService middleware.UserContextProvider,
	appCache *cache.Cache,
	rateLimitMiddleware ...middleware.Middleware,
) {
	r.Route("/sms-config", func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Use(middleware.UserContextMiddleware(userService, appCache))
		r.Use(middleware.OptionalMiddleware(rateLimitMiddleware...))

		r.With(middleware.PermissionMiddleware([]string{"sms-config:read"})).
			Get("/", smsConfigHandler.Get)
		r.With(middleware.PermissionMiddleware([]string{"sms-config:read"})).
			Get("/status", smsConfigHandler.Status)
		r.With(middleware.PermissionMiddleware([]string{"sms-config:update"})).
			Put("/", smsConfigHandler.Update)
	})
}
