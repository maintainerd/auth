package route

import (
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/auth/internal/cache"
	"github.com/maintainerd/auth/internal/middleware"
	"github.com/maintainerd/auth/internal/rest/handler"
	"github.com/maintainerd/auth/internal/service"
)

// OAuthPublicRoute mounts the public-facing OAuth 2.0 endpoints on the
// identity port (8081). This includes:
//   - GET  /oauth/authorize          — Authorization endpoint (JWT required)
//   - GET  /oauth/consent/{challenge_id} — Retrieve consent challenge (JWT required)
//   - POST /oauth/consent            — Submit consent decision (JWT required)
//   - POST /oauth/token              — Token exchange (unauthenticated; client auth in body/header)
//   - POST /oauth/revoke             — Token revocation (unauthenticated)
//   - GET  /oauth/userinfo           — OpenID Connect UserInfo (JWT required)
//   - GET  /oauth/consent/grants     — List user consent grants (JWT required)
//   - DELETE /oauth/consent/grants/{grant_uuid} — Revoke consent grant (JWT required)
func OAuthPublicRoute(
	r chi.Router,
	authorizeHandler *handler.OAuthAuthorizeHandler,
	tokenHandler *handler.OAuthTokenHandler,
	consentHandler *handler.OAuthConsentHandler,
	userInfoHandler *handler.OAuthUserInfoHandler,
	userService service.UserService,
	appCache *cache.Cache,
) {
	r.Route("/oauth", func(r chi.Router) {
		// Stricter request size limit for OAuth endpoints (1MB)
		r.Use(middleware.RequestSizeLimitMiddleware(1024 * 1024))
		// Stricter timeout for OAuth operations (30s)
		r.Use(middleware.TimeoutMiddleware(30 * time.Second))

		// ── Authenticated endpoints (require JWT + user context) ──────────

		r.Group(func(r chi.Router) {
			r.Use(middleware.JWTAuthMiddleware)
			r.Use(middleware.UserContextMiddleware(userService, appCache))

			// Authorization endpoint
			r.Get("/authorize", authorizeHandler.Authorize)

			// Consent challenge retrieval
			r.Get("/consent/{challenge_id}", authorizeHandler.GetConsentChallenge)

			// Consent decision submission
			r.Post("/consent", authorizeHandler.HandleConsent)

			// UserInfo endpoint (OpenID Connect Core §5.3)
			r.Get("/userinfo", userInfoHandler.UserInfo)

			// Consent grant management
			r.Get("/consent/grants", consentHandler.ListGrants)
			r.Delete("/consent/grants/{grant_uuid}", consentHandler.RevokeGrant)
		})

		// ── Unauthenticated endpoints (client auth handled internally) ────

		// Token endpoint (RFC 6749 §4.1.3, §6, §4.4)
		r.Post("/token", tokenHandler.Token)

		// Token revocation (RFC 7009)
		r.Post("/revoke", tokenHandler.Revoke)
	})
}

// OAuthDiscoveryRoute mounts the OpenID Connect discovery and JWKS endpoints
// at the root level of the public router (port 8081). These are fully public
// and require no authentication.
//   - GET /.well-known/openid-configuration — OpenID Provider Metadata (RFC 8414)
//   - GET /.well-known/jwks.json            — JSON Web Key Set (RFC 7517)
func OAuthDiscoveryRoute(r chi.Router, discoveryHandler *handler.OAuthDiscoveryHandler) {
	r.Get("/.well-known/openid-configuration", discoveryHandler.Discovery)
	r.Get("/.well-known/jwks.json", discoveryHandler.JWKS)
}

// OAuthInternalRoute mounts OAuth 2.0 endpoints that are only accessible via
// the management port (8080, VPN-only). Currently:
//   - POST /oauth/introspect — Token introspection (RFC 7662)
func OAuthInternalRoute(
	r chi.Router,
	tokenHandler *handler.OAuthTokenHandler,
	userService service.UserService,
	appCache *cache.Cache,
) {
	r.Route("/oauth", func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Use(middleware.UserContextMiddleware(userService, appCache))

		// Token introspection (RFC 7662) — management-only
		r.Post("/introspect", tokenHandler.Introspect)
	})
}
