package server

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/auth/internal/authevent"
	"github.com/maintainerd/auth/internal/authctx"
	"github.com/maintainerd/auth/internal/authn"
	"github.com/maintainerd/auth/internal/branding"
	"github.com/maintainerd/auth/internal/client"
	"github.com/maintainerd/auth/internal/dashboard"
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
	tenantRateLimit := securityMiddleware.TenantRequestRateLimitMiddleware(
		application.RedisClient,
		application.TenantSettingService,
	)

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
		// Audience guard: the internal API is the first-party management surface.
		// Reject any presented JWT that wasn't minted for the management
		// (auth-console) client, so a token issued to some other public client
		// can't be replayed here. Token-less (setup, public tenant reads) and
		// API-key requests pass through to their own auth rules.
		api.Use(securityMiddleware.RequireManagementClient(application.ClientService))

		// Setup Routes (no authentication required)
		setup.SetupRoute(api, h.setup)

		// Interactive authentication is public-only. The console now starts an
		// OAuth flow through the hosted identity app instead of posting
		// credentials to the internal API.
		user.ProfileRoute(api, h.profile, userProvider, application.Cache, tenantRateLimit)
		user.UserSettingRoute(api, h.userSetting, userProvider, application.Cache, tenantRateLimit)

		// Management Routes (internal access only)
		tenant.TenantRoute(api, h.tenant, userProvider, application.Cache, tenantRateLimit)
		iam.ServiceRoute(api, h.service, h.authorization, userProvider, application.Cache, tenantRateLimit)
		iam.APIRoute(api, h.api, userProvider, application.Cache, tenantRateLimit)
		iam.PermissionRoute(api, h.permission, userProvider, application.Cache, tenantRateLimit)
		iam.PolicyRoute(api, h.policy, userProvider, application.Cache, tenantRateLimit)
		idp.IdentityProviderRoute(api, h.identityProvider, h.federation, userProvider, application.Cache, tenantRateLimit)
		client.ClientRoute(api, h.client, userProvider, application.Cache, tenantRateLimit)
		iam.RoleRoute(api, h.role, userProvider, application.Cache, tenantRateLimit)
		user.UserRoute(api, h.user, h.profile, userProvider, application.Cache, tenantRateLimit)
		invite.InviteRoute(api, h.invite, userProvider, application.Cache, tenantRateLimit)
		client.APIKeyRoute(api, h.apiKey, userProvider, application.Cache, tenantRateLimit)
		idp.AuthFlowRoute(api, h.authFlow, userProvider, application.Cache, tenantRateLimit)
		secpolicy.SecuritySettingRoute(api, h.securitySetting, userProvider, application.Cache, tenantRateLimit)
		secpolicy.IPRestrictionRuleRoute(api, h.ipRestrictionRule, userProvider, application.Cache, tenantRateLimit)
		branding.EmailTemplateRoute(api, h.emailTemplate, userProvider, application.Cache, tenantRateLimit)
		branding.SMSTemplateRoute(api, h.smsTemplate, userProvider, application.Cache, tenantRateLimit)
		branding.BrandingRoute(api, h.branding, userProvider, application.Cache, tenantRateLimit)
		tenant.TenantSettingRoute(api, h.tenantSetting, userProvider, application.Cache, tenantRateLimit)
		notifier.EmailConfigRoute(api, h.emailConfig, userProvider, application.Cache, tenantRateLimit)
		notifier.SMSConfigRoute(api, h.smsConfig, userProvider, application.Cache, tenantRateLimit)
		webhook.WebhookEndpointRoute(api, h.webhookEndpoint, application.WebhookReplayHandler, application.WebhookSubscriptionHandler, application.WebhookEndpointRepo, userProvider, application.Cache, tenantRateLimit)
		authevent.AuthEventRoute(api, h.authEvent, userProvider, application.Cache, tenantRateLimit)
		event.ConfigRoute(api, h.eventConfig, h.eventManagement, userProvider, application.Cache, tenantRateLimit)
		oauth.OAuthInternalRoute(api, h.oauthToken, userProvider, application.Cache, tenantRateLimit)
		iam.AuthorizationRoute(api, h.authorization)
		dashboard.DashboardRoute(api, h.dashboard, userProvider, application.Cache, tenantRateLimit)

		// Account self-service routes (authenticated)
		user.AccountRoute(api, h.account, userProvider, application.Cache, h.mfa.RequirePolicyStepUp, tenantRateLimit)
		// Internal MFA routes (authenticated): self-service plus admin remediation.
		mfa.MFAInternalRoute(api, h.mfa, userProvider, application.Cache, tenantRateLimit)
		// Federation: token exchange + HRD (public) + identity link/unlink (authenticated)
		idp.FederationPublicRoute(api, h.federation)
		idp.FederationIdentityRoute(api, h.federation, userProvider, application.Cache, tenantRateLimit)
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
	tenantRateLimit := securityMiddleware.TenantRequestRateLimitMiddleware(
		application.RedisClient,
		application.TenantSettingService,
	)
	tenantMaintenance := securityMiddleware.TenantMaintenanceMiddleware(application.TenantSettingService)
	tenantIPRestriction := securityMiddleware.TenantIPRestrictionMiddleware(&ipRestrictionAdapter{repo: application.IPRestrictionRuleRepo})
	tenantRuntimeMiddleware := []securityMiddleware.Middleware{tenantMaintenance, tenantIPRestriction, tenantRateLimit}

	mountCommonMiddleware(r)

	// Global IP rate limit — 100 req/min per IP on the public port
	r.Use(securityMiddleware.IPRateLimitMiddleware(application.RedisClient, 100, time.Minute))

	// Multi-issuer auth middleware: intercepts foreign OIDC ID tokens (Mode B)
	// and resolves the principal before UserContextMiddleware runs. Maintainerd
	// tokens and unauthenticated requests pass through unchanged.
	if application.FederationService != nil && application.IDPRepo != nil && application.IDPAllowedAudienceRepo != nil {
		multiIssuerMW := buildMultiIssuerMiddleware(application)
		r.Use(multiIssuerMW)
	}

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

		// Public Client Lookup (no auth required - identity app uses this to
		// resolve client_id → tenant for branding and auth context)
		client.ClientPublicRoute(api, h.client)

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
			user.ProfileRoute(cookieAuth, h.profile, userProvider, application.Cache, tenantRuntimeMiddleware...)
			user.UserSettingRoute(cookieAuth, h.userSetting, userProvider, application.Cache, tenantRuntimeMiddleware...)
			user.AccountRoute(cookieAuth, h.account, userProvider, application.Cache, h.mfa.RequirePolicyStepUp, tenantRuntimeMiddleware...)
			mfa.MFAPublicRoute(cookieAuth, h.mfa, userProvider, application.Cache, tenantRuntimeMiddleware...)
			idp.FederationIdentityRoute(cookieAuth, h.federation, userProvider, application.Cache, tenantRuntimeMiddleware...)
		})

		oauth.OAuthPublicRoute(api, h.oauthAuthorize, h.oauthConnections, h.oauthToken, h.oauthTokenExchange, h.oauthConsent, h.oauthUserInfo, h.oauthPAR, h.oauthDevice, h.oauthSession, h.oauthCIBA, h.oauthRegister, userProvider, application.Cache, authRateLimit, tenantRuntimeMiddleware...)

		// Federation HRD (public, no cookie auth)
		idp.FederationPublicRoute(api, h.federation)
		// SMS login (unauthenticated, client-scoped)
		authn.SMSLoginPublicRoute(api, h.smsLogin)
		// Account recovery via backup code (unauthenticated)
		user.RecoveryRoute(api, h.account)
		// Public branding (colors + logo, non-sensitive, no auth)
		branding.BrandingPublicRoute(api, h.branding)
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

type stepUpTTLAdapter struct {
	svc secpolicy.SecuritySettingService
}

func (a *stepUpTTLAdapter) StepUpTTLSecondsByTenant(ctx context.Context, tenantID int64) int64 {
	if a.svc == nil {
		return 0
	}
	cfg, err := a.svc.GetMFAConfig(ctx, tenantID)
	if err != nil || cfg == nil {
		return 0
	}
	minutes, _ := cfg["step_up_ttl_minutes"].(float64)
	if minutes <= 0 {
		return 0
	}
	return int64(minutes) * 60
}

type ipRestrictionAdapter struct {
	repo secpolicy.IPRestrictionRuleRepository
}

func (a *ipRestrictionAdapter) GetActiveIPRestrictions(ctx context.Context, tenantID int64) ([]securityMiddleware.IPRestriction, error) {
	if a.repo == nil {
		return nil, nil
	}
	rules, err := a.repo.FindByTenantIDAndStatus(tenantID, "active")
	if err != nil {
		return nil, err
	}
	result := make([]securityMiddleware.IPRestriction, len(rules))
	for i, r := range rules {
		result[i] = securityMiddleware.IPRestriction{Type: r.Type, IPAddress: r.IPAddress}
	}
	return result, nil
}

func buildMultiIssuerMiddleware(app *Application) func(http.Handler) http.Handler {
	return securityMiddleware.MultiIssuerAuthMiddleware(
		func(issuer string) (*securityMiddleware.FederatedIDPRecord, error) {
			idp, err := app.IDPRepo.FindByIssuer(issuer)
			if err != nil || idp == nil {
				return nil, err
			}
			return &securityMiddleware.FederatedIDPRecord{
				IdentityProviderID:   idp.IdentityProviderID,
				TenantID:             idp.TenantID,
				Provider:             idp.Provider,
				AllowTokenFederation: idp.AllowTokenFederation,
				AllowJITProvisioning: idp.AllowJITProvisioning,
				Status:               idp.Status,
			}, nil
		},
		func(idpID int64) ([]securityMiddleware.FederatedAudienceRecord, error) {
			audiences, err := app.IDPAllowedAudienceRepo.FindByProviderID(idpID)
			if err != nil {
				return nil, err
			}
			result := make([]securityMiddleware.FederatedAudienceRecord, len(audiences))
			for i, a := range audiences {
				result[i] = securityMiddleware.FederatedAudienceRecord{Audience: a.Audience}
			}
			return result, nil
		},
		func(ctx context.Context, rawToken string, idpID int64, audAllowed func(aud string) bool) (*securityMiddleware.FederatedPrincipal, error) {
			found, err := app.IDPRepo.FindByID(idpID)
			if err != nil || found == nil {
				return nil, err
			}
			if svc, ok := app.FederationService.(idp.PrincipalResolver); ok {
				p, err := svc.ResolvePrincipal(ctx, rawToken, found, audAllowed)
				if err != nil {
					return nil, err
				}
				return &securityMiddleware.FederatedPrincipal{
					UserID:   p.UserID,
					UserUUID: p.UserUUID,
					TenantID: p.TenantID,
				}, nil
			}
			return nil, err
		},
		func(ctx context.Context, userID int64, tenantID int64) (*authctx.AuthContext, error) {
			usr, err := app.UserService.FindByUserID(ctx, userID)
			if err != nil || usr == nil {
				return nil, err
			}
			uc := toUserContextByTenant(usr, tenantID)
			return &authctx.AuthContext{
				User:     uc.User,
				Tenant:   uc.Tenant,
				Provider: uc.Provider,
				Client:   uc.Client,
			}, nil
		},
	)
}
