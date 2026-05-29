package tenant

import (
	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/auth/internal/platform/cache"
	"github.com/maintainerd/auth/internal/platform/middleware"
)

// TenantPublicRoute registers the unauthenticated tenant discovery endpoints used
// by the public identity app (port 8081) to look up tenant info before login.
func TenantPublicRoute(r chi.Router, tenantHandler *TenantHandler) {
	r.Route("/tenant", func(r chi.Router) {
		// Get default tenant (public endpoint)
		r.Get("/", tenantHandler.GetDefault)

		// Get tenant by identifier (public endpoint)
		r.Get("/{identifier}", tenantHandler.GetByIdentifier)
	})
}

// TenantRoute registers all tenant management endpoints (internal port 8080 only).
// It also includes the public read endpoints so the admin console can use them.
func TenantRoute(
	r chi.Router,
	tenantHandler *TenantHandler,
	userService middleware.UserContextProvider,
	appCache *cache.Cache,
) {
	// Single tenant endpoints (public - no authentication required)
	// Used by the admin console to look up tenant info
	r.Route("/tenant", func(r chi.Router) {
		// Get default tenant (public endpoint)
		r.Get("/", tenantHandler.GetDefault)

		// Get tenant by identifier (public endpoint)
		r.Get("/{identifier}", tenantHandler.GetByIdentifier)
	})

	// Multiple tenants endpoints (existing)
	r.Route("/tenants", func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Use(middleware.UserContextMiddleware(userService, appCache))

		r.With(middleware.PermissionMiddleware([]string{"tenant:read"})).
			Get("/", tenantHandler.Get)

		r.With(middleware.PermissionMiddleware([]string{"tenant:read"})).
			Get("/{tenant_uuid}", tenantHandler.GetByUUID)

		r.With(middleware.PermissionMiddleware([]string{"tenant:create"})).
			Post("/", tenantHandler.Create)

		r.With(middleware.PermissionMiddleware([]string{"tenant:update"})).
			Put("/{tenant_uuid}", tenantHandler.Update)

		r.With(middleware.PermissionMiddleware([]string{"tenant:update"})).
			Put("/{tenant_uuid}/status", tenantHandler.SetStatus)

		r.With(middleware.PermissionMiddleware([]string{"tenant:update"})).
			Put("/{tenant_uuid}/public", tenantHandler.SetPublic)

		r.With(middleware.PermissionMiddleware([]string{"tenant:delete"})).
			Delete("/{tenant_uuid}", tenantHandler.Delete)

		// Tenant member management
		r.Route("/{tenant_uuid}/members", func(r chi.Router) {
			// Get all members in tenant
			r.With(middleware.PermissionMiddleware([]string{"tenant:read"})).
				Get("/", tenantHandler.GetMembers)

			// Add member to tenant
			r.With(middleware.PermissionMiddleware([]string{"tenant:update"})).
				Post("/", tenantHandler.AddMember)

			// Update member role
			r.With(middleware.PermissionMiddleware([]string{"tenant:update"})).
				Patch("/{tenant_member_uuid}/role", tenantHandler.UpdateMemberRole)

			// Remove member from tenant
			r.With(middleware.PermissionMiddleware([]string{"tenant:update"})).
				Delete("/{tenant_member_uuid}", tenantHandler.RemoveMember)
		})
	})
}

// TenantSettingRoute registers tenant settings configuration endpoints.
func TenantSettingRoute(
	r chi.Router,
	tenantSettingHandler *TenantSettingHandler,
	userService middleware.UserContextProvider,
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
