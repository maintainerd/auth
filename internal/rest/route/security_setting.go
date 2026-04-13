package route

import (
	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/auth/internal/cache"
	"github.com/maintainerd/auth/internal/middleware"
	"github.com/maintainerd/auth/internal/rest/handler"
	"github.com/maintainerd/auth/internal/service"
)

func SecuritySettingRoute(
	r chi.Router,
	securitySettingHandler *handler.SecuritySettingHandler,
	userService service.UserService,
	appCache *cache.Cache,
) {
	r.Route("/security-settings", func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Use(middleware.UserContextMiddleware(userService, appCache))

		// General config endpoints
		r.With(middleware.PermissionMiddleware([]string{"security-setting:read"})).
			Get("/mfa", securitySettingHandler.GetMFAConfig)
		r.With(middleware.PermissionMiddleware([]string{"security-setting:update"})).
			Put("/mfa", securitySettingHandler.UpdateMFAConfig)

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

		// Lockout config endpoints
		r.With(middleware.PermissionMiddleware([]string{"security-setting:read"})).
			Get("/lockout", securitySettingHandler.GetLockoutConfig)
		r.With(middleware.PermissionMiddleware([]string{"security-setting:update"})).
			Put("/lockout", securitySettingHandler.UpdateLockoutConfig)

		// Registration config endpoints
		r.With(middleware.PermissionMiddleware([]string{"security-setting:read"})).
			Get("/registration", securitySettingHandler.GetRegistrationConfig)
		r.With(middleware.PermissionMiddleware([]string{"security-setting:update"})).
			Put("/registration", securitySettingHandler.UpdateRegistrationConfig)

		// Token config endpoints
		r.With(middleware.PermissionMiddleware([]string{"security-setting:read"})).
			Get("/token", securitySettingHandler.GetTokenConfig)
		r.With(middleware.PermissionMiddleware([]string{"security-setting:update"})).
			Put("/token", securitySettingHandler.UpdateTokenConfig)
	})
}
