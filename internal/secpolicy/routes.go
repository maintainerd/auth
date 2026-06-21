package secpolicy

import (
	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/auth/internal/platform/cache"
	"github.com/maintainerd/auth/internal/platform/middleware"
)

// IPRestrictionRuleRoute registers IP restriction rule CRUD endpoints under
// /ip-restriction-rules with appropriate permission middleware.
func IPRestrictionRuleRoute(
	r chi.Router,
	ipRestrictionRuleHandler *IPRestrictionRuleHandler,
	userService middleware.UserContextProvider,
	appCache *cache.Cache,
	rateLimitMiddleware ...middleware.Middleware,
) {
	r.Route("/ip-restriction-rules", func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Use(middleware.UserContextMiddleware(userService, appCache))
		r.Use(middleware.OptionalMiddleware(rateLimitMiddleware...))

		// List IP restriction rules
		r.With(middleware.PermissionMiddleware([]string{"ip-restriction-rule:read"})).
			Get("/", ipRestrictionRuleHandler.GetAll)

		// Get single IP restriction rule
		r.With(middleware.PermissionMiddleware([]string{"ip-restriction-rule:read"})).
			Get("/{ip_restriction_rule_uuid}", ipRestrictionRuleHandler.Get)

		// Create IP restriction rule
		r.With(middleware.PermissionMiddleware([]string{"ip-restriction-rule:create"})).
			Post("/", ipRestrictionRuleHandler.Create)

		// Update IP restriction rule
		r.With(middleware.PermissionMiddleware([]string{"ip-restriction-rule:update"})).
			Put("/{ip_restriction_rule_uuid}", ipRestrictionRuleHandler.Update)

		// Delete IP restriction rule
		r.With(middleware.PermissionMiddleware([]string{"ip-restriction-rule:delete"})).
			Delete("/{ip_restriction_rule_uuid}", ipRestrictionRuleHandler.Delete)

		// Update IP restriction rule status
		r.With(middleware.PermissionMiddleware([]string{"ip-restriction-rule:update"})).
			Patch("/{ip_restriction_rule_uuid}/status", ipRestrictionRuleHandler.UpdateStatus)
	})
}

func SecuritySettingRoute(
	r chi.Router,
	securitySettingHandler *SecuritySettingHandler,
	userService middleware.UserContextProvider,
	appCache *cache.Cache,
	rateLimitMiddleware ...middleware.Middleware,
) {
	r.Route("/security-settings", func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Use(middleware.UserContextMiddleware(userService, appCache))
		r.Use(middleware.OptionalMiddleware(rateLimitMiddleware...))

		// General config endpoints
		r.With(middleware.PermissionMiddleware([]string{"security-setting:read"})).
			Get("/mfa", securitySettingHandler.GetMFAConfig)
		r.With(middleware.PermissionMiddleware([]string{"security-setting:update"})).
			Put("/mfa", securitySettingHandler.UpdateMFAConfig)

		// Password config endpoints
		r.With(middleware.PermissionMiddleware([]string{"security-setting:read"})).
			Get("/password", securitySettingHandler.GetPasswordConfig)
		r.With(middleware.PermissionMiddleware([]string{"security-setting:update"}), middleware.RequireStepUp).
			Put("/password", securitySettingHandler.UpdatePasswordConfig)

		// Session config endpoints
		r.With(middleware.PermissionMiddleware([]string{"security-setting:read"})).
			Get("/session", securitySettingHandler.GetSessionConfig)
		r.With(middleware.PermissionMiddleware([]string{"security-setting:update"}), middleware.RequireStepUp).
			Put("/session", securitySettingHandler.UpdateSessionConfig)

		// Threat config endpoints
		r.With(middleware.PermissionMiddleware([]string{"security-setting:read"})).
			Get("/threat", securitySettingHandler.GetThreatConfig)
		r.With(middleware.PermissionMiddleware([]string{"security-setting:update"}), middleware.RequireStepUp).
			Put("/threat", securitySettingHandler.UpdateThreatConfig)

		// Lockout config endpoints
		r.With(middleware.PermissionMiddleware([]string{"security-setting:read"})).
			Get("/lockout", securitySettingHandler.GetLockoutConfig)
		r.With(middleware.PermissionMiddleware([]string{"security-setting:update"}), middleware.RequireStepUp).
			Put("/lockout", securitySettingHandler.UpdateLockoutConfig)

		// Registration config endpoints
		r.With(middleware.PermissionMiddleware([]string{"security-setting:read"})).
			Get("/registration", securitySettingHandler.GetRegistrationConfig)
		r.With(middleware.PermissionMiddleware([]string{"security-setting:update"}), middleware.RequireStepUp).
			Put("/registration", securitySettingHandler.UpdateRegistrationConfig)

		// Token config endpoints
		r.With(middleware.PermissionMiddleware([]string{"security-setting:read"})).
			Get("/token", securitySettingHandler.GetTokenConfig)
		r.With(middleware.PermissionMiddleware([]string{"security-setting:update"}), middleware.RequireStepUp).
			Put("/token", securitySettingHandler.UpdateTokenConfig)
	})
}
