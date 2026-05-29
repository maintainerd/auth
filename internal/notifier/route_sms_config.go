package notifier

import (
	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/auth/internal/platform/cache"
	"github.com/maintainerd/auth/internal/platform/middleware"
)

// SMSConfigRoute registers SMS delivery configuration endpoints.
func SMSConfigRoute(
	r chi.Router,
	smsConfigHandler *SMSConfigHandler,
	userService UserService,
	appCache *cache.Cache,
) {
	r.Route("/sms-config", func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Use(middleware.UserContextMiddleware(userService, appCache))

		r.With(middleware.PermissionMiddleware([]string{"sms-config:read"})).
			Get("/", smsConfigHandler.Get)
		r.With(middleware.PermissionMiddleware([]string{"sms-config:update"})).
			Put("/", smsConfigHandler.Update)
	})
}
