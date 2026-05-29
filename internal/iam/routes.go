package iam

import (
	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/auth/internal/platform/cache"
	"github.com/maintainerd/auth/internal/platform/middleware"
)

func APIRoute(
	r chi.Router,
	apiHandler *APIHandler,
	userService middleware.UserContextProvider,
	appCache *cache.Cache,
) {
	r.Route("/apis", func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Use(middleware.UserContextMiddleware(userService, appCache))

		r.With(middleware.PermissionMiddleware([]string{"api:read"})).
			Get("/", apiHandler.Get)

		r.With(middleware.PermissionMiddleware([]string{"api:read"})).
			Get("/{api_uuid}", apiHandler.GetByUUID)

		r.With(middleware.PermissionMiddleware([]string{"api:create"})).
			Post("/", apiHandler.Create)

		r.With(middleware.PermissionMiddleware([]string{"api:update"})).
			Put("/{api_uuid}", apiHandler.Update)

		r.With(middleware.PermissionMiddleware([]string{"api:update"})).
			Put("/{api_uuid}/status", apiHandler.SetStatus)

		r.With(middleware.PermissionMiddleware([]string{"api:delete"})).
			Delete("/{api_uuid}", apiHandler.Delete)
	})
}

func PermissionRoute(
	r chi.Router,
	oermissionHandler *PermissionHandler,
	userService middleware.UserContextProvider,
	appCache *cache.Cache,
) {
	r.Route("/permissions", func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Use(middleware.UserContextMiddleware(userService, appCache))

		r.With(middleware.PermissionMiddleware([]string{"permission:read"})).
			Get("/", oermissionHandler.Get)

		r.With(middleware.PermissionMiddleware([]string{"permission:read"})).
			Get("/{permission_uuid}", oermissionHandler.GetByUUID)

		r.With(middleware.PermissionMiddleware([]string{"permission:create"})).
			Post("/", oermissionHandler.Create)

		r.With(middleware.PermissionMiddleware([]string{"permission:update"})).
			Put("/{permission_uuid}", oermissionHandler.Update)

		r.With(middleware.PermissionMiddleware([]string{"permission:update"})).
			Put("/{permission_uuid}/status", oermissionHandler.SetStatus)

		r.With(middleware.PermissionMiddleware([]string{"permission:delete"})).
			Delete("/{permission_uuid}", oermissionHandler.Delete)
	})
}

func PolicyRoute(
	r chi.Router,
	policyHandler *PolicyHandler,
	userService middleware.UserContextProvider,
	appCache *cache.Cache,
) {
	r.Route("/policies", func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Use(middleware.UserContextMiddleware(userService, appCache))

		r.With(middleware.PermissionMiddleware([]string{"policy:read"})).
			Get("/", policyHandler.Get)

		r.With(middleware.PermissionMiddleware([]string{"policy:read"})).
			Get("/{policy_uuid}", policyHandler.GetByUUID)

		r.With(middleware.PermissionMiddleware([]string{"policy:read"})).
			Get("/{policy_uuid}/services", policyHandler.GetServicesByPolicyUUID)

		r.With(middleware.PermissionMiddleware([]string{"policy:create"})).
			Post("/", policyHandler.Create)

		r.With(middleware.PermissionMiddleware([]string{"policy:update"})).
			Put("/{policy_uuid}", policyHandler.Update)

		r.With(middleware.PermissionMiddleware([]string{"policy:update"})).
			Put("/{policy_uuid}/status", policyHandler.UpdateStatus)

		r.With(middleware.PermissionMiddleware([]string{"policy:delete"})).
			Delete("/{policy_uuid}", policyHandler.Delete)
	})
}

func RoleRoute(
	r chi.Router,
	roleHandler *RoleHandler,
	userService middleware.UserContextProvider,
	appCache *cache.Cache,
) {
	r.Route("/roles", func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Use(middleware.UserContextMiddleware(userService, appCache))

		r.With(middleware.PermissionMiddleware([]string{"role:read"})).
			Get("/", roleHandler.Get)

		r.With(middleware.PermissionMiddleware([]string{"role:read"})).
			Get("/{role_uuid}", roleHandler.GetByUUID)

		r.With(middleware.PermissionMiddleware([]string{"role:create"})).
			Post("/", roleHandler.Create)

		r.With(middleware.PermissionMiddleware([]string{"role:update"})).
			Put("/{role_uuid}", roleHandler.Update)

		r.With(middleware.PermissionMiddleware([]string{"role:update"})).
			Put("/{role_uuid}/status", roleHandler.SetStatus)

		r.With(middleware.PermissionMiddleware([]string{"role:delete"})).
			Delete("/{role_uuid}", roleHandler.Delete)

		r.With(middleware.PermissionMiddleware([]string{"role:read"})).
			Get("/{role_uuid}/permissions", roleHandler.GetPermissions)

		r.With(middleware.PermissionMiddleware([]string{"role:permission:create"})).
			Post("/{role_uuid}/permissions", roleHandler.AddPermissions)

		r.With(middleware.PermissionMiddleware([]string{"role:permission:delete"})).
			Delete("/{role_uuid}/permissions/{permission_uuid}", roleHandler.RemovePermission)
	})
}

func ServiceRoute(
	r chi.Router,
	serviceHandler *ServiceHandler,
	userService middleware.UserContextProvider,
	appCache *cache.Cache,
) {
	r.Route("/services", func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Use(middleware.UserContextMiddleware(userService, appCache))

		r.With(middleware.PermissionMiddleware([]string{"service:read"})).
			Get("/", serviceHandler.Get)

		r.With(middleware.PermissionMiddleware([]string{"service:read"})).
			Get("/{service_uuid}", serviceHandler.GetByUUID)

		r.With(middleware.PermissionMiddleware([]string{"service:create"})).
			Post("/", serviceHandler.Create)

		r.With(middleware.PermissionMiddleware([]string{"service:update"})).
			Put("/{service_uuid}", serviceHandler.Update)

		r.With(middleware.PermissionMiddleware([]string{"service:update"})).
			Put("/{service_uuid}/status", serviceHandler.SetStatus)

		r.With(middleware.PermissionMiddleware([]string{"service:delete"})).
			Delete("/{service_uuid}", serviceHandler.Delete)

		// Service-Policy Assignment endpoints
		r.With(middleware.PermissionMiddleware([]string{"service:policy:assign"})).
			Post("/{service_uuid}/policies/{policy_uuid}", serviceHandler.AssignPolicy)

		r.With(middleware.PermissionMiddleware([]string{"service:policy:remove"})).
			Delete("/{service_uuid}/policies/{policy_uuid}", serviceHandler.RemovePolicy)
	})
}
