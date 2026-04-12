package route

import (
	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/auth/internal/cache"
	"github.com/maintainerd/auth/internal/middleware"
	"github.com/maintainerd/auth/internal/rest/handler"
	"github.com/maintainerd/auth/internal/service"
)

// TenantSettingRoute registers tenant settings configuration endpoints.
func TenantSettingRoute(
	r chi.Router,
	tenantSettingHandler *handler.TenantSettingHandler,
	userService service.UserService,
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
