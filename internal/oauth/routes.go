package oauth

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/maintainerd-auth/internal/platform/cache"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
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
	authorizeHandler *OAuthAuthorizeHandler,
	connectionsHandler *OAuthConnectionsHandler,
	tokenHandler *OAuthTokenHandler,
	tokenExchangeHandler *OAuthTokenExchangeHandler,
	consentHandler *OAuthConsentHandler,
	userInfoHandler *OAuthUserInfoHandler,
	parHandler *OAuthPARHandler,
	deviceHandler *OAuthDeviceHandler,
	sessionHandler *OAuthSessionHandler,
	cibaHandler *OAuthCIBAHandler,
	registerHandler *OAuthRegisterHandler,
	userService middleware.UserContextProvider,
	appCache *cache.Cache,
	tokenRateLimit func(http.Handler) http.Handler,
	rateLimitMiddleware ...middleware.Middleware,
) {
	// PAR is only useful if the authorization endpoint can redeem the request_uri
	// it mints. Both handlers are constructed by the composition root and handed
	// here, so this is the one place that can connect them without the oauth
	// package reaching outside itself.
	if authorizeHandler != nil && parHandler != nil {
		authorizeHandler.AttachPARService(parHandler.parService)
	}

	r.Route("/oauth", func(r chi.Router) {
		r.Use(middleware.RequestSizeLimitMiddleware(1024 * 1024))
		r.Use(middleware.TimeoutMiddleware(30 * time.Second))
		// Resolve the request's own tenant (Origin → X-Forwarded-Host → Host) so
		// the authorize endpoint can make the request host authoritative for the
		// client_id ↔ tenant binding. Never rejects; unrecognized hosts fall back
		// to the existing session-tenant binding.
		r.Use(middleware.RequestTenantMiddleware)

		// ── Authorization endpoint — session-aware ────────────────────────
		// It returns login_required (not a hard 401) when no session is present,
		// so the hosted identity app can render the login page and then re-issue
		// the same /authorize request once the user has a session.
		r.Group(func(r chi.Router) {
			r.Use(middleware.OptionalUserContextMiddleware(userService, appCache))
			r.Use(middleware.OptionalMiddleware(rateLimitMiddleware...))

			r.Get("/authorize", authorizeHandler.Authorize)
		})

		// ── Authenticated endpoints (require JWT + user context) ──────────

		r.Group(func(r chi.Router) {
			r.Use(middleware.JWTAuthMiddleware)
			r.Use(middleware.UserContextMiddleware(userService, appCache))
			r.Use(middleware.OptionalMiddleware(rateLimitMiddleware...))

			r.Get("/consent/{challenge_id}", authorizeHandler.GetConsentChallenge)
			r.Post("/consent", authorizeHandler.HandleConsent)
			r.Post("/authorize/continue", authorizeHandler.ContinueAuthorize)
			r.Post("/broker/resume", authorizeHandler.BrokerResume)
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
		tokenEndpoint := func(w http.ResponseWriter, req *http.Request) {
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
				// authorization_code (and other standard grants): deliver
				// httpOnly session cookies when the client asks for cookie-based
				// delivery (the admin console), in addition to the body tokens.
				deliverAuthCookies(w, req, tokenHandler.Token)
			}
		}
		if tokenRateLimit != nil {
			r.Group(func(r chi.Router) {
				r.Use(tokenRateLimit)
				r.Post("/token", tokenEndpoint)
			})
		} else {
			r.Post("/token", tokenEndpoint)
		}

		r.Post("/revoke", tokenHandler.Revoke)

		// Login-page connections (public, unauthenticated): the enabled login
		// providers for a client, used by the hosted identity app to render login.
		r.Get("/connections", connectionsHandler.ListConnections)

		// PAR (RFC 9126)
		r.Post("/par", parHandler.Push)

		// Device Authorization (RFC 8628)
		r.Post("/device_authorization", deviceHandler.Authorize)

		// CIBA initiation (CIBA Core §7.1)
		r.Post("/ciba", cibaHandler.Initiate)

		// Dynamic Client Registration (RFC 7591) is NOT mounted on the public
		// plane. It now exists, but on the control plane only — see
		// OAuthInternalRouteWithRegistration.
		//
		// Every defect that made it unsafe here is fixed (it takes the caller's
		// tenant, restricts grant_types, applies ValidateClientOAuthMatrix,
		// validates redirect schemes, and writes a client_type that satisfies
		// chk_clients_client_type). What is deliberately NOT restored is public
		// reachability: RFC 7591 §3 permits requiring an initial access token, and
		// on the public plane that token would be any access token from any
		// third-party client, so client creation would ride on whatever authority
		// that token happens to carry. Control-plane-only keeps the blast radius at
		// operators. registration_endpoint stays out of the public discovery
		// documents for the same reason — it is not reachable on the public host.
		_ = registerHandler

		// Broker callback: the upstream provider returns here after the user
		// authenticates (OAuth #2 leg). The handler exchanges the code, provisions
		// the user, and issues a maintainerd code for the downstream app.
		r.Get("/callback/{idp_identifier}", authorizeHandler.HandleBrokerCallback)

		// Back-Channel Logout (OIDC Back-Channel Logout 1.0 §2.5)
		r.Post("/logout/backchannel", sessionHandler.BackchannelLogout)
	})
}

// OAuthDiscoveryRoute mounts the OpenID Connect discovery, RFC 8414 authorization
// server metadata, and JWKS endpoints at the root level of the public router.
func OAuthDiscoveryRoute(r chi.Router, discoveryHandler *OAuthDiscoveryHandler) {
	r.Get("/.well-known/openid-configuration", discoveryHandler.Discovery)
	r.Get("/.well-known/oauth-authorization-server", discoveryHandler.AuthorizationServerMetadata)
	r.Get("/.well-known/jwks.json", discoveryHandler.JWKS)
}

// OAuthInternalRouteWithRegistration mounts every OAuth 2.0 endpoint that is
// reachable only via the management port (8080, VPN-only):
//   - POST /oauth/introspect                       — Token introspection (RFC 7662)
//   - GET  /oauth/signing-keys                     — list key metadata
//   - POST /oauth/signing-keys/rotate              — mint + persist a new key
//   - POST /oauth/signing-keys/{kid}/retire        — stop publishing a key
//   - POST /oauth/signing-keys/{kid}/compromise    — disown a leaked key now
//   - POST /oauth/register                         — RFC 7591 §3 (requires client:create)
//   - GET  /oauth/register/{client_id}             — RFC 7592 §2.1 (requires client:read)
//
// It is the ONLY exported way to mount the internal OAuth plane. Two thinner
// wrappers used to exist beside it — OAuthInternalRoute (nil key + nil register
// handler) and OAuthInternalRouteWithKeys (nil register handler) — and the
// composition root called the thinnest one, so the signing-key lifecycle and the
// whole RFC 7591/7592 surface shipped reachable on no port at all while their
// handlers, services, permissions and unit tests all existed and passed. The
// wrappers are deleted rather than deprecated: with them gone, a composition
// root that drops the handlers again does not compile, which is a guarantee no
// test can be reverted around.
//
// A nil handler PANICS for the same reason. The mount used to skip a nil handler
// silently, so a half-wired control plane looked identical to a correctly wired
// one until an operator went looking for the key-rotation endpoint during an
// incident. Refusing to boot is the only outcome that cannot be missed.
func OAuthInternalRouteWithRegistration(
	r chi.Router,
	tokenHandler *OAuthTokenHandler,
	keyHandler *OAuthSigningKeyHandler,
	registerHandler *OAuthRegisterHandler,
	userService middleware.UserContextProvider,
	appCache *cache.Cache,
	rateLimitMiddleware ...middleware.Middleware,
) {
	if missing := missingInternalOAuthHandlers(tokenHandler, keyHandler, registerHandler); len(missing) > 0 {
		slog.Error("internal OAuth plane is missing handlers; refusing to mount a half-wired control plane",
			"missing", strings.Join(missing, ", "))
		panic("oauth: internal OAuth plane mounted without " + strings.Join(missing, ", ") +
			"; the composition root must build and pass every internal OAuth handler")
	}

	r.Route("/oauth", func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Use(middleware.UserContextMiddleware(userService, appCache))
		r.Use(middleware.OptionalMiddleware(rateLimitMiddleware...))

		// Token introspection (RFC 7662) — management-only
		r.Post("/introspect", tokenHandler.Introspect)

		// Every permission name below is written as a STRING LITERAL, never as a
		// Go constant. The seeder's guarded-vs-seeded invariant test scans source
		// for the permission-middleware call form and can only resolve literals, so
		// a constant-named guard is invisible to it: security:rotate-keys was
		// guarded here, dropped from the seeder, and no test noticed — a guard
		// naming a permission that is never seeded 403s every role including
		// super-admin.
		r.Group(func(r chi.Router) {
			r.Use(middleware.PermissionMiddleware([]string{"security:rotate-keys"}))

			r.Get("/signing-keys", keyHandler.ListKeys)
			r.Post("/signing-keys/rotate", keyHandler.Rotate)
			r.Post("/signing-keys/{kid}/retire", keyHandler.Retire)
			r.Post("/signing-keys/{kid}/compromise", keyHandler.MarkCompromised)
		})

		// client:create / client:read stand in for RFC 7591 §3's initial access
		// token: registration is an authenticated, tenant-scoped operation
		// performed with an access token that carries them.
		r.Group(func(r chi.Router) {
			r.Use(middleware.PermissionMiddleware([]string{"client:create"}))
			r.Post("/register", registerHandler.Register)
		})
		r.Group(func(r chi.Router) {
			r.Use(middleware.PermissionMiddleware([]string{"client:read"}))
			r.Get("/register/{client_id}", registerHandler.Read)
		})
	})
}

// missingInternalOAuthHandlers names the internal-plane handlers the caller left
// nil. It returns names rather than a bool so the panic tells an operator which
// surface would have gone missing — "registration is unreachable" is actionable,
// "bad wiring" is not.
func missingInternalOAuthHandlers(
	tokenHandler *OAuthTokenHandler,
	keyHandler *OAuthSigningKeyHandler,
	registerHandler *OAuthRegisterHandler,
) []string {
	var missing []string
	if tokenHandler == nil {
		missing = append(missing, "a token handler (POST /oauth/introspect)")
	}
	if keyHandler == nil {
		missing = append(missing, "a signing-key handler (GET/POST /oauth/signing-keys...)")
	}
	if registerHandler == nil {
		missing = append(missing, "a client-registration handler (RFC 7591/7592 /oauth/register)")
	}
	return missing
}
