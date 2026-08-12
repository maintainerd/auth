package server

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/maintainerd-auth/internal/auditlog"
	"github.com/maintainerd/maintainerd-auth/internal/authctx"
	"github.com/maintainerd/maintainerd-auth/internal/authevent"
	"github.com/maintainerd/maintainerd-auth/internal/authn"
	"github.com/maintainerd/maintainerd-auth/internal/branding"
	"github.com/maintainerd/maintainerd-auth/internal/client"
	"github.com/maintainerd/maintainerd-auth/internal/dashboard"
	"github.com/maintainerd/maintainerd-auth/internal/event"
	"github.com/maintainerd/maintainerd-auth/internal/federation"
	"github.com/maintainerd/maintainerd-auth/internal/iam"
	"github.com/maintainerd/maintainerd-auth/internal/idp"
	"github.com/maintainerd/maintainerd-auth/internal/invite"
	"github.com/maintainerd/maintainerd-auth/internal/mfa"
	"github.com/maintainerd/maintainerd-auth/internal/notifier"
	"github.com/maintainerd/maintainerd-auth/internal/oauth"
	securityMiddleware "github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
	"github.com/maintainerd/maintainerd-auth/internal/secpolicy"
	"github.com/maintainerd/maintainerd-auth/internal/setup"
	"github.com/maintainerd/maintainerd-auth/internal/tenant"
	"github.com/maintainerd/maintainerd-auth/internal/user"
	"github.com/maintainerd/maintainerd-auth/internal/webhook"
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
		// can't be replayed here. Token-less (setup, public tenant reads)
		// requests pass through to their own auth rules.
		api.Use(securityMiddleware.RequireManagementClient(application.ClientService))

		// Setup Routes (no authentication required)
		setup.SetupRoute(api, h.setup)

		// Interactive authentication is public-only. The console now starts an
		// OAuth flow through the hosted identity app instead of posting
		// credentials to the internal API.
		//
		// Read-only: the console displays the signed-in admin and other users'
		// avatars, but it does not manage anyone's own account — that lives
		// entirely in the identity app. UserSettingRoute is gone from this
		// surface for the same reason; nothing on the console called it.
		user.ProfileSelfReadRoute(api, h.profile, userProvider, application.Cache, tenantRateLimit)

		// Management Routes (internal access only)
		tenant.TenantRoute(api, h.tenant, userProvider, application.Cache, tenantRateLimit)
		iam.ServiceRoute(api, h.service, h.authorization, userProvider, application.Cache, tenantRateLimit)
		iam.APIRoute(api, h.api, userProvider, application.Cache, tenantRateLimit)
		iam.PermissionRoute(api, h.permission, userProvider, application.Cache, tenantRateLimit)
		iam.PolicyRoute(api, h.policy, h.policyHistory, userProvider, application.Cache, tenantRateLimit)
		idp.IdentityProviderRoute(api, h.identityProvider, h.federation, userProvider, application.Cache, tenantRateLimit)
		client.ClientRoute(api, h.client, userProvider, application.Cache, tenantRateLimit)
		iam.RoleRoute(api, h.role, userProvider, application.Cache, tenantRateLimit)
		user.UserRoute(api, h.user, h.profile, h.userTrustedDevice, h.userConsent, userProvider, application.Cache, tenantRateLimit)
		user.UserTrustedDeviceRoute(api, h.userTrustedDevice, userProvider, application.Cache, tenantRateLimit)
		// Admin erasure only. Erasing your OWN account is self-service and is
		// mounted on the public surface for the identity app.
		user.DataErasureAdminRoute(api, h.dataErasure, userProvider, application.Cache, tenantRateLimit)
		invite.InviteRoute(api, h.invite, userProvider, application.Cache, tenantRateLimit)
		idp.RegistrationFlowRoute(api, h.registrationFlow, userProvider, application.Cache, tenantRateLimit)
		secpolicy.SecuritySettingRoute(api, h.securitySetting, userProvider, application.Cache, tenantRateLimit)
		secpolicy.IPRestrictionRuleRoute(api, h.ipRestrictionRule, userProvider, application.Cache, tenantRateLimit)
		branding.EmailTemplateRoute(api, h.emailTemplate, userProvider, application.Cache, tenantRateLimit)
		branding.SMSTemplateRoute(api, h.smsTemplate, userProvider, application.Cache, tenantRateLimit)
		branding.BrandingRoute(api, h.branding, userProvider, application.Cache, tenantRateLimit)
		tenant.TenantSettingRoute(api, h.tenantSetting, userProvider, application.Cache, tenantRateLimit)
		notifier.EmailConfigRoute(api, h.emailConfig, userProvider, application.Cache, tenantRateLimit)
		notifier.SMSConfigRoute(api, h.smsConfig, userProvider, application.Cache, tenantRateLimit)
		deliveryHistoryHandler := webhook.NewDeliveryHistoryHandler(
			webhook.NewDeliveryHistoryRepository(application.DB),
			application.WebhookEndpointRepo,
		)
		webhook.WebhookEndpointRoute(api, h.webhookEndpoint, application.WebhookReplayHandler, application.WebhookSubscriptionHandler, deliveryHistoryHandler, application.WebhookEndpointRepo, userProvider, application.Cache, tenantRateLimit)
		authevent.AuthEventRoute(api, h.authEvent, userProvider, application.Cache, tenantRateLimit)
		event.ConfigRoute(api, h.eventConfig, h.eventManagement, userProvider, application.Cache, tenantRateLimit)
		// The full variant, not OAuthInternalRoute: that one passes nil for both
		// the signing-key and the registration handler, and since Dynamic Client
		// Registration is deliberately withheld from the public plane, mounting the
		// short variant here left RFC 7591/7592 and the whole key lifecycle
		// reachable on no port at all.
		oauth.OAuthInternalRouteWithRegistration(api, h.oauthToken, h.oauthSigningKey, h.oauthRegister, userProvider, application.Cache, tenantRateLimit)
		iam.AuthorizationRoute(api, h.authorization)
		auditlog.ManagementAuditLogRoute(api, h.auditLog, userProvider, application.Cache, tenantRateLimit)
		federation.WorkloadIdentityFederationRoute(api, h.wif, userProvider, application.Cache, tenantRateLimit)
		dashboard.DashboardRoute(api, h.dashboard, userProvider, application.Cache, tenantRateLimit)

		// "Signed in as" only. Account MANAGEMENT — email, phone, username,
		// password, deletion, export, sessions, consent — is the identity app's,
		// and the console links out to it rather than carrying its own copy.
		user.AccountSelfReadRoute(api, h.account, userProvider, application.Cache, tenantRateLimit)
		// Internal MFA routes (authenticated): the step-up ceremony the console
		// needs to satisfy its own sensitive actions, plus admin remediation.
		// Self-service factor enrollment is the identity app's.
		mfa.MFAInternalRoute(api, h.mfa, userProvider, application.Cache, tenantRateLimit)
		//
		// FederationIdentityRoute (self-service identity link/unlink) is NOT
		// mounted here: linking your own upstream identity is account management,
		// so it lives on the public surface with the rest of it. The console never
		// called it.
		//
		// FederationPublicRoute is likewise absent, for a different and stronger
		// reason. It carries interactive authentication and TOKEN ISSUANCE —
		// /federation/token, /federation/oauth2/callback and
		// /federation/saml/exchange all mint an end-user JWT from an upstream
		// credential — which is data-plane work, not management. Minting user
		// sessions is a privilege boundary the management surface should not
		// cross, and the callers are browsers and upstream IdPs (a SAML ACS POST
		// arrives from the IdP) which can neither present a management client nor
		// reach this port once it is VPN-only.
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
	// The status gate rides along here so the OAuth surface and the cookie-auth
	// account routes are covered too — a suspended tenant must not be able to run
	// an authorize flow or keep using its account pages, not merely be stopped at
	// the login form.
	tenantStatusGate := securityMiddleware.AuthEndpointTenantStatusMiddleware(
		&tenantSlugResolverAdapter{svc: application.TenantService},
	)
	tenantRuntimeMiddleware := []securityMiddleware.Middleware{tenantMaintenance, tenantIPRestriction, tenantRateLimit, tenantStatusGate}

	// Pre-auth IP restriction for the credential surface (login, register,
	// password reset, SMS, magic link). Keyed on the request's subdomain tenant
	// (never the request body), so a tenant's "authenticate only from these IPs"
	// policy is actually enforced at the point of authentication. Fails closed
	// per-tenant when a resolved tenant's rules cannot be loaded.
	authEndpointIPRestriction := securityMiddleware.AuthEndpointIPRestrictionMiddleware(
		&ipRestrictionAdapter{repo: application.IPRestrictionRuleRepo},
		&tenantSlugResolverAdapter{svc: application.TenantService},
	)

	// Pre-auth maintenance gate for the credential surface: an end-user cannot
	// authenticate while their tenant is in a maintenance window (keyed on the
	// subdomain tenant; identity surface only, so console/admin logins stay open).
	authEndpointMaintenance := securityMiddleware.AuthEndpointMaintenanceMiddleware(
		application.TenantSettingService,
		&tenantSlugResolverAdapter{svc: application.TenantService},
	)

	// Pre-auth tenant lifecycle gate: a suspended/inactive tenant cannot mint a
	// session. Same scoping as the maintenance gate (identity surface only) so
	// operators keep console access to the tenant they need to reactivate.
	authEndpointTenantStatus := securityMiddleware.AuthEndpointTenantStatusMiddleware(
		&tenantSlugResolverAdapter{svc: application.TenantService},
	)

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

		// Rate-limited credential endpoints, IP-restricted per the request's
		// subdomain tenant.
		api.Group(func(rl chi.Router) {
			rl.Use(authRateLimit)
			rl.Use(authEndpointIPRestriction)
			rl.Use(authEndpointMaintenance)
			rl.Use(authEndpointTenantStatus)
			authn.RegisterPublicRoute(rl, h.register)
			authn.LoginPublicRoute(rl, h.login)
			authn.ForgotPasswordPublicRoute(rl, h.forgotPassword)
			// Account recovery via backup code (unauthenticated). It verifies a
			// password and mints a full token set, so it belongs behind the same
			// per-IP ceiling and tenant gates as every other credential endpoint.
			user.RecoveryRoute(rl, h.account)
			authn.ResetPasswordPublicRoute(rl, h.resetPassword)
		})

		// Remaining public authentication routes. Magic-link is a credential
		// entry point too, so it is IP-restricted alongside the group above.
		authn.EmailVerificationPublicRoute(api, h.emailVerification)
		api.Group(func(ml chi.Router) {
			ml.Use(authEndpointIPRestriction)
			ml.Use(authEndpointMaintenance)
			ml.Use(authEndpointTenantStatus)
			authn.MagicLinkPublicRoute(ml, h.magicLink)
		})
		invite.InvitePublicRoute(api, h.invite)
		authn.RegistrationContextPublicRoute(api, h.registrationContext)

		// Cookie-auth state-changing routes — apply CSRF protection
		api.Group(func(cookieAuth chi.Router) {
			cookieAuth.Use(securityMiddleware.CSRFMiddleware)
			// These routes authorize on the SUBJECT alone, so any valid access
			// token for the user passes. Without this guard a token minted for a
			// third-party OAuth client the user consented to for `openid profile`
			// could change their email, rotate their password, revoke their
			// sessions, or strip their MFA. Cookie-borne sessions are first-party
			// by construction and pass through.
			cookieAuth.Use(securityMiddleware.RequireFirstPartyClient(application.ClientService))
			user.ProfileRoute(cookieAuth, h.profile, userProvider, application.Cache, tenantRuntimeMiddleware...)
			user.UserSettingRoute(cookieAuth, h.userSetting, userProvider, application.Cache, tenantRuntimeMiddleware...)
			user.AccountRoute(cookieAuth, h.account, h.userConsent, userProvider, application.Cache, h.mfa.RequirePolicyStepUp, tenantRuntimeMiddleware...)
			user.UserTrustedDeviceRoute(cookieAuth, h.userTrustedDevice, userProvider, application.Cache, tenantRuntimeMiddleware...)
			user.DataErasureSelfRoute(cookieAuth, h.dataErasure, userProvider, application.Cache, tenantRuntimeMiddleware...)
			mfa.MFAPublicRoute(cookieAuth, h.mfa, userProvider, application.Cache, tenantRuntimeMiddleware...)
			idp.FederationIdentityRoute(cookieAuth, h.federation, userProvider, application.Cache, tenantRuntimeMiddleware...)
			// Account-link confirmation attaches a provider identity to the
			// caller's existing account, so it belongs behind the same first-party
			// guard as the rest of the self-service surface. Mounted outside it, a
			// third-party token could drive an identity link on the user's account.
			authn.AccountLinkConfirmRoute(cookieAuth, h.accountLink, userProvider, application.Cache, tenantRuntimeMiddleware...)
		})

		oauth.OAuthPublicRoute(api, h.oauthAuthorize, h.oauthConnections, h.oauthToken, h.oauthTokenExchange, h.oauthConsent, h.oauthUserInfo, h.oauthPAR, h.oauthDevice, h.oauthSession, h.oauthCIBA, h.oauthRegister, userProvider, application.Cache, authRateLimit, tenantRuntimeMiddleware...)

		// Federation HRD (public, no cookie auth)
		idp.FederationPublicRoute(api, h.federation)
		// SMS login (unauthenticated, client-scoped) — a credential entry point,
		// so IP-restricted per the request's subdomain tenant.
		api.Group(func(sms chi.Router) {
			sms.Use(authEndpointIPRestriction)
			sms.Use(authEndpointMaintenance)
			sms.Use(authEndpointTenantStatus)
			authn.SMSLoginPublicRoute(sms, h.smsLogin)
		})
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

// tenantSlugResolverAdapter resolves a subdomain slug to its tenant ID for the
// pre-auth IP-restriction middleware. It returns ok=false (not an error) when
// the slug names no tenant, so the middleware treats "unknown tenant" as
// "nothing to enforce" rather than a load failure.
type tenantSlugResolverAdapter struct {
	svc tenant.TenantService
}

func (a *tenantSlugResolverAdapter) ResolveTenantIDBySlug(ctx context.Context, slug string) (int64, bool, error) {
	if a.svc == nil || slug == "" {
		return 0, false, nil
	}
	res, err := a.svc.GetByName(ctx, slug)
	if err != nil {
		return 0, false, err
	}
	if res == nil {
		return 0, false, nil
	}
	return res.TenantID, true, nil
}

// ResolveTenantStatusBySlug backs the pre-auth tenant-status gate. Same
// ok=false-for-unknown-slug contract as ResolveTenantIDBySlug.
func (a *tenantSlugResolverAdapter) ResolveTenantStatusBySlug(ctx context.Context, slug string) (string, bool, error) {
	if a.svc == nil || slug == "" {
		return "", false, nil
	}
	res, err := a.svc.GetByName(ctx, slug)
	if err != nil {
		return "", false, err
	}
	if res == nil {
		return "", false, nil
	}
	return res.Status, true, nil
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
