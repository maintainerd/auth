package tenant

import (
	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/auth/internal/platform/cache"
	"github.com/maintainerd/auth/internal/platform/middleware"
)

// TenantSettingRoute registers tenant settings configuration endpoints.
func TenantSettingRoute(
	r chi.Router,
	tenantSettingHandler *TenantSettingHandler,
	userService UserService,
	appCache *cache.Cache,
) {
	r.Route("/tenant-settings", func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Use(middleware.UserContextMiddleware(userService, appCache))

		// Rate limit config
		r.With(middleware.PermissionMiddleware([]string{"tenant-setting:read"})).
			Get("/rate-limit", tenantSettingHandler.GetRateLimitConfig)
		r.With(middleware.PermissionMiddleware([]string{"tenant-setting:update"})).
			Put("/rate-limit", tenantSettingHandler.UpdateRateLimitConfig)

		// Audit config
		r.With(middleware.PermissionMiddleware([]string{"tenant-setting:read"})).
			Get("/audit", tenantSettingHandler.GetAuditConfig)
		r.With(middleware.PermissionMiddleware([]string{"tenant-setting:update"})).
			Put("/audit", tenantSettingHandler.UpdateAuditConfig)

		// Maintenance config
		r.With(middleware.PermissionMiddleware([]string{"tenant-setting:read"})).
			Get("/maintenance", tenantSettingHandler.GetMaintenanceConfig)
		r.With(middleware.PermissionMiddleware([]string{"tenant-setting:update"})).
			Put("/maintenance", tenantSettingHandler.UpdateMaintenanceConfig)

		// Feature flags
		r.With(middleware.PermissionMiddleware([]string{"tenant-setting:read"})).
			Get("/feature-flags", tenantSettingHandler.GetFeatureFlags)
		r.With(middleware.PermissionMiddleware([]string{"tenant-setting:update"})).
			Put("/feature-flags", tenantSettingHandler.UpdateFeatureFlags)
	})
}
