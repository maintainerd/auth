package tenant

import (
	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/maintainerd-auth/internal/platform/cache"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
)

// TenantPublicRoute registers the unauthenticated tenant discovery endpoints used
// by the public identity app (port 8081) to look up tenant info before login.
func TenantPublicRoute(r chi.Router, tenantHandler *TenantHandler) {
	r.Route("/tenant", func(r chi.Router) {
		// Get default tenant (public endpoint)
		r.Get("/", tenantHandler.GetDefault)

		// Get tenant by name/slug (public endpoint)
		r.Get("/{name}", tenantHandler.GetByName)
	})
}

// TenantRoute registers all tenant management endpoints (internal port 8080 only).
// It also includes the public read endpoints so the admin console can use them.
func TenantRoute(
	r chi.Router,
	tenantHandler *TenantHandler,
	userService middleware.UserContextProvider,
	appCache *cache.Cache,
	rateLimitMiddleware ...middleware.Middleware,
) {
	// Single tenant endpoints (public - no authentication required)
	// Used by the admin console to look up tenant info
	r.Route("/tenant", func(r chi.Router) {
		// Get default tenant (public endpoint)
		r.Get("/", tenantHandler.GetDefault)

		// Get tenant by name/slug (public endpoint)
		r.Get("/{name}", tenantHandler.GetByName)
	})

	// Multiple tenants endpoints (existing)
	r.Route("/tenants", func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Use(middleware.UserContextMiddleware(userService, appCache))
		r.Use(middleware.OptionalMiddleware(rateLimitMiddleware...))

		r.With(middleware.PermissionMiddleware([]string{"tenant:read"})).
			Get("/", tenantHandler.Get)

		r.With(middleware.PermissionMiddleware([]string{"tenant:read"})).
			Get("/{tenant_uuid}", tenantHandler.GetByUUID)

		r.With(middleware.PermissionMiddleware([]string{"tenant:create"})).
			Post("/", tenantHandler.Create)

		// Step-up here too, not only on /status: TenantUpdateRequestDTO carries a
		// required `status` that Update passes straight through, so without this
		// an acr=1 session could suspend a tenant via the combined route and skip
		// the gate the dedicated /status route enforces. Gating the whole route
		// (rather than rejecting `status` here) keeps a single write path, and
		// the other field it rewrites — `name`, the DNS subdomain slug — is
		// equally privileged.
		r.With(middleware.PermissionMiddleware([]string{"tenant:update"}), middleware.RequireStepUp).
			Put("/{tenant_uuid}", tenantHandler.Update)

		r.With(middleware.PermissionMiddleware([]string{"tenant:update"}), middleware.RequireStepUp).
			Put("/{tenant_uuid}/status", tenantHandler.SetStatus)

		r.With(middleware.PermissionMiddleware([]string{"tenant:delete"}), middleware.RequireStepUp).
			Delete("/{tenant_uuid}", tenantHandler.Delete)

		// Tenant member management.
		//
		// Every MUTATION here is step-up gated, matching the plain tenant
		// update/delete above and the invite endpoints. Membership writes are at
		// least as privileged as a tenant rename: adding a member with role=owner
		// and promoting a member to owner both implicitly grant the super-admin
		// role (service_member.go GrantRoleByName), an owner transfer implicitly
		// revokes it from the previous owner, and a removal strips a person's
		// access to the tenant. Without the gate a stolen or replayed acr=1 session
		// holding tenant:update could hand itself tenant-wide super-admin, so the
		// weakest path into the strongest privilege was the one that asked for the
		// least proof of presence. Reads stay ungated.
		r.Route("/{tenant_uuid}/members", func(r chi.Router) {
			// Get all members in tenant
			r.With(middleware.PermissionMiddleware([]string{"tenant:read"})).
				Get("/", tenantHandler.GetMembers)

			// Add member to tenant
			r.With(middleware.PermissionMiddleware([]string{"tenant:update"}), middleware.RequireStepUp).
				Post("/", tenantHandler.AddMember)

			// Update member role (includes ownership transfer)
			r.With(middleware.PermissionMiddleware([]string{"tenant:update"}), middleware.RequireStepUp).
				Patch("/{tenant_member_uuid}/role", tenantHandler.UpdateMemberRole)

			// Remove member from tenant
			r.With(middleware.PermissionMiddleware([]string{"tenant:update"}), middleware.RequireStepUp).
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
	rateLimitMiddleware ...middleware.Middleware,
) {
	r.Route("/tenant-settings", func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Use(middleware.UserContextMiddleware(userService, appCache))
		r.Use(middleware.OptionalMiddleware(rateLimitMiddleware...))

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

	})
}
