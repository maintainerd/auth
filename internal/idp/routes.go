package idp

import (
	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/maintainerd-auth/internal/platform/cache"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
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

		// SAML 2.0 SP-initiated SSO
		// GET  /federation/saml/initiate      — start SAML flow (→ IdP redirect)
		// POST /federation/saml/acs/{id}      — ACS: receive IdP SAMLResponse
		// POST /federation/saml/exchange      — exchange short-lived code for tokens
		// GET  /federation/saml/metadata/{id} — SP metadata XML
		r.Route("/saml", func(r chi.Router) {
			r.Get("/initiate", h.InitiateSAML)
			r.Post("/acs/{provider_identifier}", h.SAMLCallback)
			r.Post("/exchange", h.ExchangeSAMLCode)
			r.Get("/metadata/{provider_identifier}", h.SAMLMetadata)
		})
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

func RegistrationFlowRoute(
	r chi.Router,
	registrationFlowHandler *RegistrationFlowHandler,
	userService middleware.UserContextProvider,
	appCache *cache.Cache,
	rateLimitMiddleware ...middleware.Middleware,
) {
	r.Route("/registration_flows", func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Use(middleware.UserContextMiddleware(userService, appCache))
		r.Use(middleware.OptionalMiddleware(rateLimitMiddleware...))

		// Get all registration flows with pagination and filtering
		r.With(middleware.PermissionMiddleware([]string{"registration-flow:read"})).
			Get("/", registrationFlowHandler.Get)

		// Get registration flow by UUID
		r.With(middleware.PermissionMiddleware([]string{"registration-flow:read"})).
			Get("/{registration_flow_uuid}", registrationFlowHandler.GetByUUID)

		// Create registration flow
		r.With(middleware.PermissionMiddleware([]string{"registration-flow:create"})).
			Post("/", registrationFlowHandler.Create)

		// Update registration flow
		r.With(middleware.PermissionMiddleware([]string{"registration-flow:update"})).
			Put("/{registration_flow_uuid}", registrationFlowHandler.Update)

		// Update registration flow status
		r.With(middleware.PermissionMiddleware([]string{"registration-flow:update"})).
			Patch("/{registration_flow_uuid}/status", registrationFlowHandler.SetStatus)

		// Delete registration flow
		r.With(middleware.PermissionMiddleware([]string{"registration-flow:delete"})).
			Delete("/{registration_flow_uuid}", registrationFlowHandler.Delete)

		// Registration flow role management
		r.Route("/{registration_flow_uuid}/roles", func(r chi.Router) {
			// Assign roles to registration flow
			r.With(middleware.PermissionMiddleware([]string{"registration-flow:update"})).
				Post("/", registrationFlowHandler.AssignRoles)

			// Get all roles assigned to registration flow
			r.With(middleware.PermissionMiddleware([]string{"registration-flow:read"})).
				Get("/", registrationFlowHandler.GetRoles)

			// Remove a role from registration flow
			r.With(middleware.PermissionMiddleware([]string{"registration-flow:update"})).
				Delete("/{role_uuid}", registrationFlowHandler.RemoveRole)
		})
	})
}
