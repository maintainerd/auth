package route

import (
	"github.com/maintainerd/auth/internal/rest/handler"
	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/auth/internal/middleware"
	"github.com/maintainerd/auth/internal/service"
	"github.com/redis/go-redis/v9"
)

func SecuritySettingRoute(
	r chi.Router,
	securitySettingHandler *handler.SecuritySettingHandler,
	userService service.UserService,
	redisClient *redis.Client,
) {
	r.Route("/security-settings", func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Use(middleware.UserContextMiddleware(userService, redisClient))

		// General config endpoints
		r.With(middleware.PermissionMiddleware([]string{"security-setting:read"})).
			Get("/general", securitySettingHandler.GetGeneralConfig)
		r.With(middleware.PermissionMiddleware([]string{"security-setting:update"})).
			Put("/general", securitySettingHandler.UpdateGeneralConfig)

		// Password config endpoints
		r.With(middleware.PermissionMiddleware([]string{"security-setting:read"})).
			Get("/password", securitySettingHandler.GetPasswordConfig)
		r.With(middleware.PermissionMiddleware([]string{"security-setting:update"})).
			Put("/password", securitySettingHandler.UpdatePasswordConfig)

		// Session config endpoints
		r.With(middleware.PermissionMiddleware([]string{"security-setting:read"})).
			Get("/session", securitySettingHandler.GetSessionConfig)
		r.With(middleware.PermissionMiddleware([]string{"security-setting:update"})).
			Put("/session", securitySettingHandler.UpdateSessionConfig)

		// Threat config endpoints
		r.With(middleware.PermissionMiddleware([]string{"security-setting:read"})).
			Get("/threat", securitySettingHandler.GetThreatConfig)
		r.With(middleware.PermissionMiddleware([]string{"security-setting:update"})).
			Put("/threat", securitySettingHandler.UpdateThreatConfig)

		// IP config endpoints
		r.With(middleware.PermissionMiddleware([]string{"security-setting:read"})).
			Get("/ip", securitySettingHandler.GetIPConfig)
		r.With(middleware.PermissionMiddleware([]string{"security-setting:update"})).
			Put("/ip", securitySettingHandler.UpdateIPConfig)
	})
}
