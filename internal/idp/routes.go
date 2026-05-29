package idp

import (
	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/auth/internal/platform/cache"
	"github.com/maintainerd/auth/internal/platform/middleware"
)

// FederationPublicRoute mounts unauthenticated federation endpoints (token
// exchange and home-realm discovery) under /federation.
func FederationPublicRoute(r chi.Router, h *FederationHandler) {
	r.Route("/federation", func(r chi.Router) {
		// POST /federation/token — exchange an upstream OIDC token for our JWT.
		r.Post("/token", h.ExchangeExternalToken)
		// GET /federation/hrd — home-realm discovery by email domain.
		r.Get("/hrd", h.HomeRealmDiscovery)
	})
}

// FederationIdentityRoute mounts authenticated identity link/unlink endpoints
// under /account/identities. Intended to be registered inside an existing
// /account route group that already applies JWT + user-context middleware.
func FederationIdentityRoute(
	r chi.Router,
	h *FederationHandler,
	userService middleware.UserContextProvider,
	appCache *cache.Cache,
) {
	r.Route("/account/identities", func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Use(middleware.UserContextMiddleware(userService, appCache))

		r.Get("/", h.GetIdentities)
		r.Post("/link", h.LinkIdentity)
		r.Delete("/{identity_uuid}", h.UnlinkIdentity)
	})
}

func IdentityProviderRoute(
	r chi.Router,
	idpHandler *IdentityProviderHandler,
	userService middleware.UserContextProvider,
	appCache *cache.Cache,
) {
	r.Route("/identity_providers", func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Use(middleware.UserContextMiddleware(userService, appCache))

		r.With(middleware.PermissionMiddleware([]string{"idp:read"})).
			Get("/", idpHandler.Get)

		r.With(middleware.PermissionMiddleware([]string{"idp:read"})).
			Get("/{identity_provider_uuid}", idpHandler.GetByUUID)

		r.With(middleware.PermissionMiddleware([]string{"idp:create"})).
			Post("/", idpHandler.Create)

		r.With(middleware.PermissionMiddleware([]string{"idp:update"})).
			Put("/{identity_provider_uuid}", idpHandler.Update)

		r.With(middleware.PermissionMiddleware([]string{"idp:update"})).
			Put("/{identity_provider_uuid}/status", idpHandler.SetStatus)

		r.With(middleware.PermissionMiddleware([]string{"idp:delete"})).
			Delete("/{identity_provider_uuid}", idpHandler.Delete)
	})
}

func SignupFlowRoute(
	r chi.Router,
	signupFlowHandler *SignupFlowHandler,
	userService middleware.UserContextProvider,
	appCache *cache.Cache,
) {
	r.Route("/signup_flows", func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Use(middleware.UserContextMiddleware(userService, appCache))

		// Get all signup flows with pagination and filtering
		r.With(middleware.PermissionMiddleware([]string{"signup-flow:read"})).
			Get("/", signupFlowHandler.GetAll)

		// Get signup flow by UUID
		r.With(middleware.PermissionMiddleware([]string{"signup-flow:read"})).
			Get("/{signup_flow_uuid}", signupFlowHandler.Get)

		// Create signup flow
		r.With(middleware.PermissionMiddleware([]string{"signup-flow:create"})).
			Post("/", signupFlowHandler.Create)

		// Update signup flow
		r.With(middleware.PermissionMiddleware([]string{"signup-flow:update"})).
			Put("/{signup_flow_uuid}", signupFlowHandler.Update)

		// Update signup flow status
		r.With(middleware.PermissionMiddleware([]string{"signup-flow:update"})).
			Patch("/{signup_flow_uuid}/status", signupFlowHandler.UpdateStatus)

		// Delete signup flow
		r.With(middleware.PermissionMiddleware([]string{"signup-flow:delete"})).
			Delete("/{signup_flow_uuid}", signupFlowHandler.Delete)

		// Signup flow role management
		r.Route("/{signup_flow_uuid}/roles", func(r chi.Router) {
			// Assign roles to signup flow
			r.With(middleware.PermissionMiddleware([]string{"signup-flow:update"})).
				Post("/", signupFlowHandler.AssignRoles)

			// Get all roles assigned to signup flow
			r.With(middleware.PermissionMiddleware([]string{"signup-flow:read"})).
				Get("/", signupFlowHandler.GetRoles)

			// Remove a role from signup flow
			r.With(middleware.PermissionMiddleware([]string{"signup-flow:update"})).
				Delete("/{role_uuid}", signupFlowHandler.RemoveRole)
		})
	})
}
