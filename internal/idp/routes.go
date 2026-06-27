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
		// POST /federation/oauth2/callback — exchange an OAuth2 authorization code.
		r.Post("/oauth2/callback", h.ExchangeOAuth2Code)
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
	rateLimitMiddleware ...middleware.Middleware,
) {
	r.Route("/account/identities", func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Use(middleware.UserContextMiddleware(userService, appCache))
		r.Use(middleware.OptionalMiddleware(rateLimitMiddleware...))

		r.With(middleware.PermissionMiddleware([]string{"account:identity:read:self"})).
			Get("/", h.GetIdentities)
		r.With(middleware.PermissionMiddleware([]string{"account:identity:link:self"})).
			Post("/link", h.LinkIdentity)
		r.With(middleware.PermissionMiddleware([]string{"account:identity:unlink:self"})).
			Delete("/{identity_uuid}", h.UnlinkIdentity)
	})
}

func IdentityProviderRoute(
	r chi.Router,
	idpHandler *IdentityProviderHandler,
	federationHandler *FederationHandler,
	userService middleware.UserContextProvider,
	appCache *cache.Cache,
	rateLimitMiddleware ...middleware.Middleware,
) {
	r.Route("/identity_providers", func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Use(middleware.UserContextMiddleware(userService, appCache))
		r.Use(middleware.OptionalMiddleware(rateLimitMiddleware...))

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

		// Test connection — validate an unsaved IdP config before persisting.
		// Uses the FederationHandler for OIDC discovery / JWKS probe logic.
		if federationHandler != nil {
			r.With(middleware.PermissionMiddleware([]string{"idp:create"})).
				Post("/test", federationHandler.TestConnection)
		}
	})
}

func AuthFlowRoute(
	r chi.Router,
	authFlowHandler *AuthFlowHandler,
	userService middleware.UserContextProvider,
	appCache *cache.Cache,
	rateLimitMiddleware ...middleware.Middleware,
) {
	r.Route("/auth_flows", func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Use(middleware.UserContextMiddleware(userService, appCache))
		r.Use(middleware.OptionalMiddleware(rateLimitMiddleware...))

		// Get all signup flows with pagination and filtering
		r.With(middleware.PermissionMiddleware([]string{"auth-flow:read"})).
			Get("/", authFlowHandler.GetAll)

		// Get signup flow by UUID
		r.With(middleware.PermissionMiddleware([]string{"auth-flow:read"})).
			Get("/{auth_flow_uuid}", authFlowHandler.Get)

		// Create signup flow
		r.With(middleware.PermissionMiddleware([]string{"auth-flow:create"})).
			Post("/", authFlowHandler.Create)

		// Update signup flow
		r.With(middleware.PermissionMiddleware([]string{"auth-flow:update"})).
			Put("/{auth_flow_uuid}", authFlowHandler.Update)

		// Update signup flow status
		r.With(middleware.PermissionMiddleware([]string{"auth-flow:update"})).
			Patch("/{auth_flow_uuid}/status", authFlowHandler.UpdateStatus)

		// Delete signup flow
		r.With(middleware.PermissionMiddleware([]string{"auth-flow:delete"})).
			Delete("/{auth_flow_uuid}", authFlowHandler.Delete)

		// Signup flow role management
		r.Route("/{auth_flow_uuid}/roles", func(r chi.Router) {
			// Assign roles to signup flow
			r.With(middleware.PermissionMiddleware([]string{"auth-flow:update"})).
				Post("/", authFlowHandler.AssignRoles)

			// Get all roles assigned to signup flow
			r.With(middleware.PermissionMiddleware([]string{"auth-flow:read"})).
				Get("/", authFlowHandler.GetRoles)

			// Remove a role from signup flow
			r.With(middleware.PermissionMiddleware([]string{"auth-flow:update"})).
				Delete("/{role_uuid}", authFlowHandler.RemoveRole)
		})

		// Auth flow callback URI management
		r.Route("/{auth_flow_uuid}/callback_uris", func(r chi.Router) {
			// Attach callback URIs (from the client's registered URIs)
			r.With(middleware.PermissionMiddleware([]string{"auth-flow:update"})).
				Post("/", authFlowHandler.AssignCallbackURIs)

			// List callback URIs attached to the flow
			r.With(middleware.PermissionMiddleware([]string{"auth-flow:read"})).
				Get("/", authFlowHandler.GetCallbackURIs)

			// Detach a callback URI from the flow
			r.With(middleware.PermissionMiddleware([]string{"auth-flow:update"})).
				Delete("/{client_uri_uuid}", authFlowHandler.RemoveCallbackURI)
		})
	})
}
