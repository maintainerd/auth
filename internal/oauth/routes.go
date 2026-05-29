package oauth

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/auth/internal/platform/cache"
	"github.com/maintainerd/auth/internal/platform/middleware"
	"github.com/maintainerd/auth/internal/rest/handler"
	"github.com/maintainerd/auth/internal/service"
)

// OAuthPublicRoute mounts the public-facing OAuth 2.0 endpoints on the
// identity port (8081). This includes:
//   - GET  /oauth/authorize                    — Authorization endpoint (JWT required)
//   - GET  /oauth/consent/{challenge_id}        — Retrieve consent challenge (JWT required)
//   - POST /oauth/consent                       — Submit consent decision (JWT required)
//   - POST /oauth/token                         — Token exchange (unauthenticated; dispatches by grant_type)
//   - POST /oauth/revoke                        — Token revocation (RFC 7009)
//   - GET  /oauth/userinfo                      — OpenID Connect UserInfo (JWT required)
//   - GET  /oauth/consent/grants                — List user consent grants (JWT required)
//   - DELETE /oauth/consent/grants/{grant_uuid} — Revoke consent grant (JWT required)
//   - POST /oauth/par                           — Pushed Authorization Requests (RFC 9126)
//   - POST /oauth/device_authorization          — Device Authorization (RFC 8628)
//   - POST /oauth/device                        — Device user code approval (JWT required)
//   - POST /oauth/device/deny                   — Device user code denial (JWT required)
//   - POST /oauth/ciba                          — CIBA initiation (CIBA Core)
//   - POST /oauth/ciba/approve                  — CIBA user approval (JWT required)
//   - POST /oauth/ciba/deny                     — CIBA user denial (JWT required)
//   - POST /oauth/register                      — Dynamic Client Registration (RFC 7591)
//   - GET/POST /oauth/end_session               — RP-Initiated Logout (OIDC Session Mgmt)
//   - POST /oauth/logout/backchannel            — Back-Channel Logout (OIDC Back-Channel Logout)
func OAuthPublicRoute(
	r chi.Router,
	authorizeHandler *handler.OAuthAuthorizeHandler,
	tokenHandler *handler.OAuthTokenHandler,
	tokenExchangeHandler *handler.OAuthTokenExchangeHandler,
	consentHandler *handler.OAuthConsentHandler,
	userInfoHandler *handler.OAuthUserInfoHandler,
	parHandler *handler.OAuthPARHandler,
	deviceHandler *handler.OAuthDeviceHandler,
	sessionHandler *handler.OAuthSessionHandler,
	cibaHandler *handler.OAuthCIBAHandler,
	registerHandler *handler.OAuthRegisterHandler,
	userService service.UserService,
	appCache *cache.Cache,
) {
	r.Route("/oauth", func(r chi.Router) {
		r.Use(middleware.RequestSizeLimitMiddleware(1024 * 1024))
		r.Use(middleware.TimeoutMiddleware(30 * time.Second))

		// ── Authenticated endpoints (require JWT + user context) ──────────

		r.Group(func(r chi.Router) {
			r.Use(middleware.JWTAuthMiddleware)
			r.Use(middleware.UserContextMiddleware(userService, appCache))

			r.Get("/authorize", authorizeHandler.Authorize)
			r.Get("/consent/{challenge_id}", authorizeHandler.GetConsentChallenge)
			r.Post("/consent", authorizeHandler.HandleConsent)
			r.Get("/userinfo", userInfoHandler.UserInfo)
			r.Get("/consent/grants", consentHandler.ListGrants)
			r.Delete("/consent/grants/{grant_uuid}", consentHandler.RevokeGrant)

			// Device authorization: user approves/denies at verification URI
			r.Post("/device", deviceHandler.VerifyUserCode)
			r.Post("/device/deny", deviceHandler.DenyUserCode)

			// CIBA: user approves/denies out-of-band
			r.Post("/ciba/approve", cibaHandler.ApproveRequest)
			r.Post("/ciba/deny", cibaHandler.DenyRequest)

			// RP-Initiated Logout (requires user context for id_token_hint lookup)
			r.Get("/end_session", sessionHandler.EndSession)
			r.Post("/end_session", sessionHandler.EndSession)
		})

		// ── Unauthenticated endpoints (client auth handled internally) ────

		// Token endpoint — existing grants
		r.Post("/token", func(w http.ResponseWriter, req *http.Request) {
			if err := req.ParseForm(); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			switch req.FormValue("grant_type") {
			case "urn:ietf:params:oauth:grant-type:token-exchange":
				tokenExchangeHandler.Exchange(w, req)
			case "urn:ietf:params:oauth:grant-type:device_code":
				// Device code token polling is handled by the device handler
				// which delegates to OAuthDeviceService.ExchangeToken
				deviceHandler.ExchangeDeviceToken(w, req)
			case "urn:openid:params:grant-type:ciba":
				cibaHandler.ExchangeToken(w, req)
			default:
				tokenHandler.Token(w, req)
			}
		})

		r.Post("/revoke", tokenHandler.Revoke)

		// PAR (RFC 9126)
		r.Post("/par", parHandler.Push)

		// Device Authorization (RFC 8628)
		r.Post("/device_authorization", deviceHandler.Authorize)

		// CIBA initiation (CIBA Core §7.1)
		r.Post("/ciba", cibaHandler.Initiate)

		// Dynamic Client Registration (RFC 7591)
		r.Post("/register", registerHandler.Register)

		// Back-Channel Logout (OIDC Back-Channel Logout 1.0 §2.5)
		r.Post("/logout/backchannel", sessionHandler.BackchannelLogout)
	})
}

// OAuthDiscoveryRoute mounts the OpenID Connect discovery, RFC 8414 authorization
// server metadata, and JWKS endpoints at the root level of the public router.
func OAuthDiscoveryRoute(r chi.Router, discoveryHandler *handler.OAuthDiscoveryHandler) {
	r.Get("/.well-known/openid-configuration", discoveryHandler.Discovery)
	r.Get("/.well-known/oauth-authorization-server", discoveryHandler.AuthorizationServerMetadata)
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
