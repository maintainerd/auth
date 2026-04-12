package route

import (
	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/auth/internal/cache"
	"github.com/maintainerd/auth/internal/middleware"
	"github.com/maintainerd/auth/internal/rest/handler"
	"github.com/maintainerd/auth/internal/service"
)

// SMSConfigRoute registers SMS delivery configuration endpoints.
func SMSConfigRoute(
	r chi.Router,
	smsConfigHandler *handler.SMSConfigHandler,
	userService service.UserService,
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
