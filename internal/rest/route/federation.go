package route

import (
	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/auth/internal/cache"
	"github.com/maintainerd/auth/internal/middleware"
	"github.com/maintainerd/auth/internal/rest/handler"
	"github.com/maintainerd/auth/internal/service"
)

// FederationPublicRoute mounts unauthenticated federation endpoints (token
// exchange and home-realm discovery) under /federation.
func FederationPublicRoute(r chi.Router, h *handler.FederationHandler) {
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
	h *handler.FederationHandler,
	userService service.UserService,
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
