package server

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/auth/internal/authevent"
	"github.com/maintainerd/auth/internal/authn"
	"github.com/maintainerd/auth/internal/branding"
	"github.com/maintainerd/auth/internal/client"
	"github.com/maintainerd/auth/internal/event"
	"github.com/maintainerd/auth/internal/iam"
	"github.com/maintainerd/auth/internal/idp"
	"github.com/maintainerd/auth/internal/invite"
	"github.com/maintainerd/auth/internal/mfa"
	"github.com/maintainerd/auth/internal/notifier"
	"github.com/maintainerd/auth/internal/oauth"
	securityMiddleware "github.com/maintainerd/auth/internal/platform/middleware"
	"github.com/maintainerd/auth/internal/secpolicy"
	"github.com/maintainerd/auth/internal/setup"
	"github.com/maintainerd/auth/internal/tenant"
	"github.com/maintainerd/auth/internal/user"
	"github.com/maintainerd/auth/internal/webhook"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// buildInternalRouter constructs the chi router for the internal API (port 8080, VPN access only).
func buildInternalRouter(h *handlers, application *Application) http.Handler {
	r := chi.NewRouter()
	userProvider := newMiddlewareUserContextProvider(application.UserService)

	mountCommonMiddleware(r)

	// Health / readiness probes (no auth, no rate-limit)
	r.Get("/health", handleHealth)
	r.Get("/healthz", handleHealthz)
	r.Get("/ready", handleReady(application))
	r.Get("/readyz", handleReady(application))
	r.Get("/livez", handleLivez)

	// OpenAPI 3.1 spec — internal port only
	r.Get("/openapi.json", ServeOpenAPISpec)

	r.Route("/api/v1", func(api chi.Router) {
		// Setup Routes (no authentication required)
		setup.SetupRoute(api, h.setup)

		// Internal Authentication Routes (no client_id/provider_id required)
		authn.RegisterRoute(api, h.register)
		authn.LoginRoute(api, h.login)
		authn.ForgotPasswordRoute(api, h.forgotPassword)
		authn.ResetPasswordRoute(api, h.resetPassword)
		authn.EmailVerificationRoute(api, h.emailVerification)
		authn.MagicLinkRoute(api, h.magicLink)
		user.ProfileRoute(api, h.profile, userProvider, application.Cache)
		user.UserSettingRoute(api, h.userSetting, userProvider, application.Cache)

		// Management Routes (internal access only)
		tenant.TenantRoute(api, h.tenant, userProvider, application.Cache)
		iam.ServiceRoute(api, h.service, h.authorization, userProvider, application.Cache)
		iam.APIRoute(api, h.api, userProvider, application.Cache)
		iam.PermissionRoute(api, h.permission, userProvider, application.Cache)
		iam.PolicyRoute(api, h.policy, userProvider, application.Cache)
		idp.IdentityProviderRoute(api, h.identityProvider, userProvider, application.Cache)
		client.ClientRoute(api, h.client, userProvider, application.Cache)
		iam.RoleRoute(api, h.role, userProvider, application.Cache)
		user.UserRoute(api, h.user, h.profile, userProvider, application.Cache)
		invite.InviteRoute(api, h.invite, userProvider, application.Cache)
		client.APIKeyRoute(api, h.apiKey, userProvider, application.Cache)
		idp.SignupFlowRoute(api, h.signupFlow, userProvider, application.Cache)
		secpolicy.SecuritySettingRoute(api, h.securitySetting, userProvider, application.Cache)
		secpolicy.IPRestrictionRuleRoute(api, h.ipRestrictionRule, userProvider, application.Cache)
		branding.EmailTemplateRoute(api, h.emailTemplate, userProvider, application.Cache)
		branding.SMSTemplateRoute(api, h.smsTemplate, userProvider, application.Cache)
		branding.LoginTemplateRoute(api, h.loginTemplate, userProvider, application.Cache)
		branding.BrandingRoute(api, h.branding, userProvider, application.Cache)
		tenant.TenantSettingRoute(api, h.tenantSetting, userProvider, application.Cache)
		notifier.EmailConfigRoute(api, h.emailConfig, userProvider, application.Cache)
		notifier.SMSConfigRoute(api, h.smsConfig, userProvider, application.Cache)
		webhook.WebhookEndpointRoute(api, h.webhookEndpoint, application.WebhookReplayHandler, application.WebhookSubscriptionHandler, application.WebhookEndpointRepo, userProvider, application.Cache)
		authevent.AuthEventRoute(api, h.authEvent, userProvider, application.Cache)
		event.ConfigRoute(api, h.eventConfig, h.eventManagement, userProvider, application.Cache)
		oauth.OAuthInternalRoute(api, h.oauthToken, userProvider, application.Cache)
		iam.AuthorizationRoute(api, h.authorization)

		// Account self-service routes (authenticated)
		user.AccountRoute(api, h.account, userProvider, application.Cache)
		// MFA self-service routes (authenticated)
		mfa.MFARoute(api, h.mfa, userProvider, application.Cache)
		// Federation: token exchange + HRD (public) + identity link/unlink (authenticated)
		idp.FederationPublicRoute(api, h.federation)
		idp.FederationIdentityRoute(api, h.federation, userProvider, application.Cache)
		// SMS login (unauthenticated)
		authn.SMSLoginRoute(api, h.smsLogin)
		// Account recovery via backup code (unauthenticated)
		user.RecoveryRoute(api, h.account)
	})

	return r
}

// buildManagementRouter constructs the management router for probes, metrics,
// and machine-readable specs. It should be bound to a private management port.
func buildManagementRouter(application *Application) http.Handler {
	r := chi.NewRouter()
	mountCommonMiddleware(r)

	r.Get("/health", handleHealth)
	r.Get("/healthz", handleHealthz)
	r.Get("/ready", handleReady(application))
	r.Get("/readyz", handleReady(application))
	r.Get("/livez", handleLivez)
	r.Get("/openapi.json", ServeOpenAPISpec)
	r.Handle("/metrics", promhttp.Handler())

	return r
}

// buildPublicRouter constructs the chi router for the public API (port 8081, public internet).
func buildPublicRouter(h *handlers, application *Application) http.Handler {
	r := chi.NewRouter()
	userProvider := newMiddlewareUserContextProvider(application.UserService)

	mountCommonMiddleware(r)

	// Global IP rate limit — 100 req/min per IP on the public port
	r.Use(securityMiddleware.IPRateLimitMiddleware(application.RedisClient, 100, time.Minute))

	// Health / readiness probes (no auth, no rate-limit)
	r.Get("/health", handleHealth)
	r.Get("/healthz", handleHealthz)
	r.Get("/ready", handleReady(application))
	r.Get("/readyz", handleReady(application))
	r.Get("/livez", handleLivez)

	// OpenAPI 3.1 spec — public
	r.Get("/openapi.json", ServeOpenAPISpec)

	// OpenID Connect discovery endpoints (root-level, fully public)
	oauth.OAuthDiscoveryRoute(r, h.oauthDiscovery)

	// Tight rate limit for credential / credential-reset endpoints (10 req/min per IP)
	authRateLimit := securityMiddleware.IPRateLimitMiddleware(application.RedisClient, 10, time.Minute)

	r.Route("/api/v1", func(api chi.Router) {
		// Public Tenant Routes (no authentication required - for login page)
		// Only exposes GET /tenant/ and GET /tenant/{identifier} — management endpoints
		// are intentionally absent from the public surface.
		tenant.TenantPublicRoute(api, h.tenant)

		// Rate-limited credential endpoints
		api.Group(func(rl chi.Router) {
			rl.Use(authRateLimit)
			authn.RegisterPublicRoute(rl, h.register)
			authn.LoginPublicRoute(rl, h.login)
			authn.ForgotPasswordPublicRoute(rl, h.forgotPassword)
			authn.ResetPasswordPublicRoute(rl, h.resetPassword)
		})

		// Remaining public authentication routes
		authn.EmailVerificationPublicRoute(api, h.emailVerification)
		authn.MagicLinkPublicRoute(api, h.magicLink)

		// Cookie-auth state-changing routes — apply CSRF protection
		api.Group(func(cookieAuth chi.Router) {
			cookieAuth.Use(securityMiddleware.CSRFMiddleware)
			user.ProfileRoute(cookieAuth, h.profile, userProvider, application.Cache)
			user.UserSettingRoute(cookieAuth, h.userSetting, userProvider, application.Cache)
			user.AccountRoute(cookieAuth, h.account, userProvider, application.Cache)
			mfa.MFARoute(cookieAuth, h.mfa, userProvider, application.Cache)
			idp.FederationIdentityRoute(cookieAuth, h.federation, userProvider, application.Cache)
		})

		oauth.OAuthPublicRoute(api, h.oauthAuthorize, h.oauthToken, h.oauthTokenExchange, h.oauthConsent, h.oauthUserInfo, h.oauthPAR, h.oauthDevice, h.oauthSession, h.oauthCIBA, h.oauthRegister, userProvider, application.Cache, authRateLimit)

		// Federation HRD (public, no cookie auth)
		idp.FederationPublicRoute(api, h.federation)
		// SMS login (unauthenticated)
		authn.SMSLoginRoute(api, h.smsLogin)
		// Account recovery via backup code (unauthenticated)
		user.RecoveryRoute(api, h.account)
	})

	return r
}

func mountCommonMiddleware(r chi.Router) {
	// Recovery middleware — logs panics with structured logging and stack traces.
	r.Use(securityMiddleware.RecoveryMiddleware)

	// Global security middleware for SOC2/ISO27001 compliance
	r.Use(securityMiddleware.SecurityHeadersMiddleware)
	r.Use(securityMiddleware.SecurityContextMiddleware)

	// Structured JSON access logging — must follow SecurityContextMiddleware
	// so that request_id is available for log correlation.
	r.Use(securityMiddleware.LoggingMiddleware)

	// Global DoS protection with reasonable limits
	r.Use(securityMiddleware.RequestSizeLimitMiddleware(10 * 1024 * 1024)) // 10MB global limit
	r.Use(securityMiddleware.TimeoutMiddleware(60 * time.Second))          // 60s global timeout

	// CORS allow-list and Content-Type enforcement
	r.Use(securityMiddleware.CORSMiddleware)
	r.Use(securityMiddleware.EnforceJSONContentType)
}
