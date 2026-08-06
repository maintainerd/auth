package iam

import (
	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/maintainerd-auth/internal/platform/cache"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
)

func APIRoute(
	r chi.Router,
	apiHandler *APIHandler,
	userService middleware.UserContextProvider,
	appCache *cache.Cache,
	rateLimitMiddleware ...middleware.Middleware,
) {
	r.Route("/apis", func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Use(middleware.UserContextMiddleware(userService, appCache))
		r.Use(middleware.OptionalMiddleware(rateLimitMiddleware...))

		r.With(middleware.PermissionMiddleware([]string{"api:read"})).
			Get("/", apiHandler.Get)

		r.With(middleware.PermissionMiddleware([]string{"api:read"})).
			Get("/{api_uuid}", apiHandler.GetByUUID)

		// Create stays outside the step-up gate across this package: a brand-new
		// API/permission/policy/service grants nothing until it is wired to a role
		// or a service, and both of those edges ARE step-up gated. Editing or
		// deleting an EXISTING row is what silently re-points authorization that
		// is already in force, which is why the rest of these routes are gated.
		r.With(middleware.PermissionMiddleware([]string{"api:create"})).
			Post("/", apiHandler.Create)

		// An API's identifier is the audience/resource that token issuance and
		// permission scoping resolve against, so rewriting or retiring one moves
		// every permission hanging off it. Same privilege-escalation surface as a
		// role edit — same acr=2 requirement, so a stolen acr=1 session cannot do it.
		r.With(middleware.PermissionMiddleware([]string{"api:update"}), middleware.RequireStepUp).
			Put("/{api_uuid}", apiHandler.Update)

		r.With(middleware.PermissionMiddleware([]string{"api:update"}), middleware.RequireStepUp).
			Put("/{api_uuid}/status", apiHandler.SetStatus)

		r.With(middleware.PermissionMiddleware([]string{"api:delete"}), middleware.RequireStepUp).
			Delete("/{api_uuid}", apiHandler.Delete)
	})
}

func PermissionRoute(
	r chi.Router,
	oermissionHandler *PermissionHandler,
	userService middleware.UserContextProvider,
	appCache *cache.Cache,
	rateLimitMiddleware ...middleware.Middleware,
) {
	r.Route("/permissions", func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Use(middleware.UserContextMiddleware(userService, appCache))
		r.Use(middleware.OptionalMiddleware(rateLimitMiddleware...))

		r.With(middleware.PermissionMiddleware([]string{"permission:read"})).
			Get("/", oermissionHandler.Get)

		r.With(middleware.PermissionMiddleware([]string{"permission:read"})).
			Get("/{permission_uuid}", oermissionHandler.GetByUUID)

		r.With(middleware.PermissionMiddleware([]string{"permission:create"})).
			Post("/", oermissionHandler.Create)

		// PermissionMiddleware matches route guards on the permission NAME, and
		// role grants point at the permission ROW — so a rename re-points every
		// existing grant at a different guard without touching a single role.
		// validation_permission.go blocks the reserved namespaces, but renaming
		// inside the unreserved space still silently redefines what everyone
		// already holds; that has to cost a fresh acr=2 proof of presence.
		r.With(middleware.PermissionMiddleware([]string{"permission:update"}), middleware.RequireStepUp).
			Put("/{permission_uuid}", oermissionHandler.Update)

		// Deactivating a permission revokes it everywhere at once (inactive rows no
		// longer grant), so it is an availability weapon as much as an escalation one.
		r.With(middleware.PermissionMiddleware([]string{"permission:update"}), middleware.RequireStepUp).
			Put("/{permission_uuid}/status", oermissionHandler.SetStatus)

		r.With(middleware.PermissionMiddleware([]string{"permission:delete"}), middleware.RequireStepUp).
			Delete("/{permission_uuid}", oermissionHandler.Delete)
	})
}

func PolicyRoute(
	r chi.Router,
	policyHandler *PolicyHandler,
	policyHistoryHandler *PolicyHistoryHandler,
	userService middleware.UserContextProvider,
	appCache *cache.Cache,
	rateLimitMiddleware ...middleware.Middleware,
) {
	r.Route("/policies", func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Use(middleware.UserContextMiddleware(userService, appCache))
		r.Use(middleware.OptionalMiddleware(rateLimitMiddleware...))

		r.With(middleware.PermissionMiddleware([]string{"policy:read"})).
			Get("/", policyHandler.Get)

		r.With(middleware.PermissionMiddleware([]string{"policy:read"})).
			Get("/{policy_uuid}", policyHandler.GetByUUID)

		r.With(middleware.PermissionMiddleware([]string{"policy:read"})).
			Get("/{policy_uuid}/services", policyHandler.GetServicesByPolicyUUID)

		// Policy version history (append-only audit / rollback source)
		r.With(middleware.PermissionMiddleware([]string{"policy:read"})).
			Get("/{policy_uuid}/history", policyHistoryHandler.ListHistory)

		r.With(middleware.PermissionMiddleware([]string{"policy:read"})).
			Get("/{policy_uuid}/history/{version_number}", policyHistoryHandler.GetHistoryVersion)

		r.With(middleware.PermissionMiddleware([]string{"policy:create"})).
			Post("/", policyHandler.Create)

		// A policy document IS the authorization decision for every service it is
		// bound to (policy_evaluator.go), so editing, disabling or deleting one
		// rewrites allow/deny for traffic already in flight — no role or grant
		// changes hands, and nothing else in the chain re-checks the caller.
		r.With(middleware.PermissionMiddleware([]string{"policy:update"}), middleware.RequireStepUp).
			Put("/{policy_uuid}", policyHandler.Update)

		r.With(middleware.PermissionMiddleware([]string{"policy:update"}), middleware.RequireStepUp).
			Put("/{policy_uuid}/status", policyHandler.UpdateStatus)

		r.With(middleware.PermissionMiddleware([]string{"policy:delete"}), middleware.RequireStepUp).
			Delete("/{policy_uuid}", policyHandler.Delete)
	})
}

func RoleRoute(
	r chi.Router,
	roleHandler *RoleHandler,
	userService middleware.UserContextProvider,
	appCache *cache.Cache,
	rateLimitMiddleware ...middleware.Middleware,
) {
	r.Route("/roles", func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Use(middleware.UserContextMiddleware(userService, appCache))
		r.Use(middleware.OptionalMiddleware(rateLimitMiddleware...))

		r.With(middleware.PermissionMiddleware([]string{"role:read"})).
			Get("/", roleHandler.Get)

		r.With(middleware.PermissionMiddleware([]string{"role:read"})).
			Get("/{role_uuid}", roleHandler.GetByUUID)

		r.With(middleware.PermissionMiddleware([]string{"role:create"})).
			Post("/", roleHandler.Create)

		r.With(middleware.PermissionMiddleware([]string{"role:update"}), middleware.RequireStepUp).
			Put("/{role_uuid}", roleHandler.Update)

		r.With(middleware.PermissionMiddleware([]string{"role:update"}), middleware.RequireStepUp).
			Put("/{role_uuid}/status", roleHandler.SetStatus)

		r.With(middleware.PermissionMiddleware([]string{"role:delete"}), middleware.RequireStepUp).
			Delete("/{role_uuid}", roleHandler.Delete)

		r.With(middleware.PermissionMiddleware([]string{"role:read"})).
			Get("/{role_uuid}/permissions", roleHandler.GetPermissions)

		// Adding/removing permissions on a role is a privilege-escalation surface —
		// step-up required, like the user-domain role assignment routes.
		r.With(middleware.PermissionMiddleware([]string{"role:permission:create"}), middleware.RequireStepUp).
			Post("/{role_uuid}/permissions", roleHandler.AddPermissions)

		r.With(middleware.PermissionMiddleware([]string{"role:permission:delete"}), middleware.RequireStepUp).
			Delete("/{role_uuid}/permissions/{permission_uuid}", roleHandler.RemovePermission)
	})
}

func ServiceRoute(
	r chi.Router,
	serviceHandler *ServiceHandler,
	authorizationHandler *AuthorizationHandler,
	userService middleware.UserContextProvider,
	appCache *cache.Cache,
	rateLimitMiddleware ...middleware.Middleware,
) {
	r.Route("/services", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(middleware.JWTAuthMiddleware)
			r.Get("/me/policy-bundle", authorizationHandler.PolicyBundle)
		})

		r.Group(func(r chi.Router) {
			r.Use(middleware.JWTAuthMiddleware)
			r.Use(middleware.UserContextMiddleware(userService, appCache))
			r.Use(middleware.OptionalMiddleware(rateLimitMiddleware...))

			r.With(middleware.PermissionMiddleware([]string{"service:read"})).
				Get("/", serviceHandler.Get)

			r.With(middleware.PermissionMiddleware([]string{"service:read"})).
				Get("/{service_uuid}", serviceHandler.GetByUUID)

			r.With(middleware.PermissionMiddleware([]string{"service:create"})).
				Post("/", serviceHandler.Create)

			// A service row is the identity policy bundles are served to
			// (/services/me/policy-bundle keys off it), so renaming, disabling or
			// deleting one redirects or blanks the authorization rules a running
			// workload enforces.
			r.With(middleware.PermissionMiddleware([]string{"service:update"}), middleware.RequireStepUp).
				Put("/{service_uuid}", serviceHandler.Update)

			r.With(middleware.PermissionMiddleware([]string{"service:update"}), middleware.RequireStepUp).
				Put("/{service_uuid}/status", serviceHandler.SetStatus)

			r.With(middleware.PermissionMiddleware([]string{"service:delete"}), middleware.RequireStepUp).
				Delete("/{service_uuid}", serviceHandler.Delete)

			// Service-Policy Assignment endpoints. This edge is where an inert policy
			// document starts deciding real requests — the exact counterpart of
			// attaching a permission to a role, which is step-up gated for the same
			// reason. Removal is gated too: dropping the deny policy is the attack.
			r.With(middleware.PermissionMiddleware([]string{"service:policy:assign"}), middleware.RequireStepUp).
				Post("/{service_uuid}/policies/{policy_uuid}", serviceHandler.AssignPolicy)

			r.With(middleware.PermissionMiddleware([]string{"service:policy:remove"}), middleware.RequireStepUp).
				Delete("/{service_uuid}/policies/{policy_uuid}", serviceHandler.RemovePolicy)
		})
	})
}

func AuthorizationRoute(r chi.Router, authorizationHandler *AuthorizationHandler) {
	r.Route("/authorize", func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Post("/", authorizationHandler.Authorize)
	})
}
