package route

import (
	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/auth/internal/handler/resthandler"
	"github.com/maintainerd/auth/internal/middleware"
	"github.com/maintainerd/auth/internal/repository"
	"github.com/redis/go-redis/v9"
)

func TenantRoute(
	r chi.Router,
	tenantHandler *resthandler.TenantHandler,
	userRepo repository.UserRepository,
	redisClient *redis.Client,
) {
	// Single tenant endpoints (public - no authentication required)
	// Used by login page to validate tenant and get tenant info
	r.Route("/tenant", func(r chi.Router) {
		// Get default tenant (public endpoint)
		r.Get("/", tenantHandler.GetDefault)

		// Get tenant by identifier (public endpoint)
		r.Get("/{identifier}", tenantHandler.GetByIdentifier)
	})

	// Multiple tenants endpoints (existing)
	r.Route("/tenants", func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Use(middleware.UserContextMiddleware(userRepo, redisClient))

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

		r.With(middleware.PermissionMiddleware([]string{"tenant:update"})).
			Put("/{tenant_uuid}/default", tenantHandler.SetDefault)

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
