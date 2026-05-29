package server

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/maintainerd/auth/internal/authn"
	"github.com/maintainerd/auth/internal/branding"
	"github.com/maintainerd/auth/internal/client"
	"github.com/maintainerd/auth/internal/iam"
	"github.com/maintainerd/auth/internal/idp"
	"github.com/maintainerd/auth/internal/mfa"
	"github.com/maintainerd/auth/internal/notifier"
	"github.com/maintainerd/auth/internal/oauth"
	"github.com/maintainerd/auth/internal/secpolicy"
	"github.com/maintainerd/auth/internal/setup"
	"github.com/maintainerd/auth/internal/tenant"
	"github.com/maintainerd/auth/internal/user"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/maintainerd/auth/internal/authevent"
	"github.com/maintainerd/auth/internal/invite"
	securityMiddleware "github.com/maintainerd/auth/internal/platform/middleware"
	"github.com/maintainerd/auth/internal/webhook"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// handlers holds every REST handler instance. Created once per server start.
type handlers struct {
	service            *iam.ServiceHandler
	api                *iam.APIHandler
	permission         *iam.PermissionHandler
	policy             *iam.PolicyHandler
	tenant             *tenant.TenantHandler
	identityProvider   *idp.IdentityProviderHandler
	client             *client.ClientHandler
	role               *iam.RoleHandler
	user               *user.UserHandler
	register           *authn.RegisterHandler
	login              *authn.LoginHandler
	profile            *user.ProfileHandler
	userSetting        *user.UserSettingHandler
	invite             *invite.InviteHandler
	forgotPassword     *authn.ForgotPasswordHandler
	resetPassword      *authn.ResetPasswordHandler
	emailVerification  *authn.EmailVerificationHandler
	magicLink          *authn.MagicLinkHandler
	setup              *setup.SetupHandler
	apiKey             *client.APIKeyHandler
	signupFlow         *idp.SignupFlowHandler
	securitySetting    *secpolicy.SecuritySettingHandler
	ipRestrictionRule  *secpolicy.IPRestrictionRuleHandler
	emailTemplate      *branding.EmailTemplateHandler
	smsTemplate        *branding.SMSTemplateHandler
	loginTemplate      *branding.LoginTemplateHandler
	branding           *branding.BrandingHandler
	tenantSetting      *tenant.TenantSettingHandler
	emailConfig        *notifier.EmailConfigHandler
	smsConfig          *notifier.SMSConfigHandler
	webhookEndpoint    *webhook.WebhookEndpointHandler
	authEvent          *authevent.AuthEventHandler
	oauthAuthorize     *oauth.OAuthAuthorizeHandler
	oauthToken         *oauth.OAuthTokenHandler
	oauthTokenExchange *oauth.OAuthTokenExchangeHandler
	oauthConsent       *oauth.OAuthConsentHandler
	oauthDiscovery     *oauth.OAuthDiscoveryHandler
	oauthUserInfo      *oauth.OAuthUserInfoHandler
	oauthPAR           *oauth.OAuthPARHandler
	oauthDevice        *oauth.OAuthDeviceHandler
	oauthSession       *oauth.OAuthSessionHandler
	oauthCIBA          *oauth.OAuthCIBAHandler
	oauthRegister      *oauth.OAuthRegisterHandler
	account            *user.AccountHandler
	smsLogin           *authn.SMSLoginHandler
	mfa                *mfa.MFAHandler
	federation         *idp.FederationHandler
}

func initHandlers(application *Application) *handlers {
	return &handlers{
		service:            iam.NewServiceHandler(application.ServiceService),
		api:                iam.NewAPIHandler(application.APIService),
		permission:         iam.NewPermissionHandler(application.PermissionService),
		policy:             iam.NewPolicyHandler(application.PolicyService),
		tenant:             tenant.NewTenantHandler(application.TenantService, application.TenantMemberService),
		identityProvider:   idp.NewIdentityProviderHandler(application.IdentityProviderService),
		client:             client.NewClientHandler(application.ClientService),
		role:               iam.NewRoleHandler(application.RoleService),
		user:               user.NewUserHandler(application.UserService),
		register:           authn.NewRegisterHandler(application.RegisterService, application.EmailVerificationService),
		login:              authn.NewLoginHandler(application.LoginService),
		profile:            user.NewProfileHandler(application.ProfileService),
		userSetting:        user.NewUserSettingHandler(application.UserSettingService),
		invite:             invite.NewInviteHandler(application.InviteService),
		forgotPassword:     authn.NewForgotPasswordHandler(application.ForgotPasswordService),
		resetPassword:      authn.NewResetPasswordHandler(application.ResetPasswordService),
		emailVerification:  authn.NewEmailVerificationHandler(application.EmailVerificationService),
		magicLink:          authn.NewMagicLinkHandler(application.MagicLinkService),
		setup:              setup.NewSetupHandler(application.SetupService),
		apiKey:             client.NewAPIKeyHandler(application.APIKeyService),
		signupFlow:         idp.NewSignupFlowHandler(application.SignupFlowService),
		securitySetting:    secpolicy.NewSecuritySettingHandler(application.SecuritySettingService),
		ipRestrictionRule:  secpolicy.NewIPRestrictionRuleHandler(application.IPRestrictionRuleService),
		emailTemplate:      branding.NewEmailTemplateHandler(application.EmailTemplateService),
		smsTemplate:        branding.NewSMSTemplateHandler(application.SMSTemplateService),
		loginTemplate:      branding.NewLoginTemplateHandler(application.LoginTemplateService),
		branding:           branding.NewBrandingHandler(application.BrandingService),
		tenantSetting:      tenant.NewTenantSettingHandler(application.TenantSettingService),
		emailConfig:        notifier.NewEmailConfigHandler(application.EmailConfigService),
		smsConfig:          notifier.NewSMSConfigHandler(application.SMSConfigService),
		webhookEndpoint:    webhook.NewWebhookEndpointHandler(application.WebhookEndpointService),
		authEvent:          authevent.NewAuthEventHandler(application.AuthEventService),
		oauthAuthorize:     oauth.NewOAuthAuthorizeHandler(application.OAuthAuthorizeService),
		oauthToken:         oauth.NewOAuthTokenHandler(application.OAuthTokenService),
		oauthTokenExchange: oauth.NewOAuthTokenExchangeHandler(application.OAuthTokenExchangeService),
		oauthConsent:       oauth.NewOAuthConsentHandler(application.OAuthConsentService),
		oauthDiscovery:     oauth.NewOAuthDiscoveryHandler(),
		oauthUserInfo:      oauth.NewOAuthUserInfoHandler(),
		oauthPAR:           oauth.NewOAuthPARHandler(application.OAuthPARService),
		oauthDevice:        oauth.NewOAuthDeviceHandler(application.OAuthDeviceService),
		oauthSession:       oauth.NewOAuthSessionHandler(application.OAuthSessionService),
		oauthCIBA:          oauth.NewOAuthCIBAHandler(application.OAuthCIBAService),
		oauthRegister:      oauth.NewOAuthRegisterHandler(application.OAuthRegisterService),
		account:            user.NewAccountHandler(application.AccountService, newUserSessionServiceAdapter(application.SessionService)),
		smsLogin:           authn.NewSMSLoginHandler(application.SMSLoginService),
		mfa:                mfa.NewMFAHandler(application.MFAService, application.WebAuthnService),
		federation:         idp.NewFederationHandler(application.FederationService),
	}
}

// StartRESTServer launches the internal and public HTTP servers, blocks until
// a termination signal is received, then drains connections gracefully.
func StartRESTServer(application *Application) {
	h := initHandlers(application)

	internalSrv := &http.Server{
		Addr:         ":8080",
		Handler:      otelhttp.NewHandler(buildInternalRouter(h, application), "internal"),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	publicSrv := &http.Server{
		Addr:         ":8081",
		Handler:      otelhttp.NewHandler(buildPublicRouter(h, application), "public"),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start both servers in background goroutines
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		slog.Info("Internal REST server starting", "addr", internalSrv.Addr)
		if err := internalSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Internal REST server error", "error", err)
			os.Exit(1)
		}
	}()

	go func() {
		defer wg.Done()
		slog.Info("Public REST server starting", "addr", publicSrv.Addr)
		if err := publicSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Public REST server error", "error", err)
			os.Exit(1)
		}
	}()

	// Block until OS signal received
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	slog.Info("Shutdown signal received, draining connections...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var shutdownErr error
	if err := internalSrv.Shutdown(ctx); err != nil {
		shutdownErr = err
		slog.Error("Internal server shutdown error", "error", err)
	}
	if err := publicSrv.Shutdown(ctx); err != nil {
		shutdownErr = err
		slog.Error("Public server shutdown error", "error", err)
	}

	wg.Wait()

	if shutdownErr != nil {
		os.Exit(1)
	}
	slog.Info("Servers stopped cleanly")
}

// buildInternalRouter constructs the chi router for the internal API (port 8080, VPN access only).
func buildInternalRouter(h *handlers, application *Application) http.Handler {
	r := chi.NewRouter()
	userProvider := newMiddlewareUserContextProvider(application.UserService)

	// Built-in Chi middlewares
	r.Use(middleware.Recoverer)

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

	// Health / readiness probes (no auth, no rate-limit)
	r.Get("/health", handleHealth)
	r.Get("/ready", handleReady(application))

	// OpenAPI 3.1 spec — internal port only
	r.Get("/openapi.json", ServeOpenAPISpec)

	// Prometheus metrics — internal port only, never exposed publicly
	r.Handle("/metrics", promhttp.Handler())

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
		iam.ServiceRoute(api, h.service, userProvider, application.Cache)
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
		webhook.WebhookEndpointRoute(api, h.webhookEndpoint, userProvider, application.Cache)
		authevent.AuthEventRoute(api, h.authEvent, userProvider, application.Cache)
		oauth.OAuthInternalRoute(api, h.oauthToken, userProvider, application.Cache)

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

// buildPublicRouter constructs the chi router for the public API (port 8081, public internet).
func buildPublicRouter(h *handlers, application *Application) http.Handler {
	r := chi.NewRouter()
	userProvider := newMiddlewareUserContextProvider(application.UserService)

	// Built-in Chi middlewares
	r.Use(middleware.Recoverer)

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

	// Global IP rate limit — 100 req/min per IP on the public port
	r.Use(securityMiddleware.IPRateLimitMiddleware(application.RedisClient, 100, time.Minute))

	// Health / readiness probes (no auth, no rate-limit)
	r.Get("/health", handleHealth)
	r.Get("/ready", handleReady(application))

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

		oauth.OAuthPublicRoute(api, h.oauthAuthorize, h.oauthToken, h.oauthTokenExchange, h.oauthConsent, h.oauthUserInfo, h.oauthPAR, h.oauthDevice, h.oauthSession, h.oauthCIBA, h.oauthRegister, userProvider, application.Cache)

		// Federation HRD (public, no cookie auth)
		idp.FederationPublicRoute(api, h.federation)
		// SMS login (unauthenticated)
		authn.SMSLoginRoute(api, h.smsLogin)
		// Account recovery via backup code (unauthenticated)
		user.RecoveryRoute(api, h.account)
	})

	return r
}

// handleHealth responds to liveness probes. Always returns 200 OK when the
// process is running — no dependency checks.
func handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`)) //nolint:errcheck
}

// handleReady returns an http.HandlerFunc that checks database and Redis
// connectivity. It returns 200 OK when both dependencies are reachable, or
// 503 Service Unavailable when either check fails.
func handleReady(application *Application) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Check database connectivity
		sqlDB, err := application.DB.DB()
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"status":"not ready","reason":"database connection unavailable"}`)) //nolint:errcheck
			return
		}
		if err := sqlDB.PingContext(ctx); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"status":"not ready","reason":"database ping failed"}`)) //nolint:errcheck
			return
		}

		// Check Redis connectivity
		if err := application.RedisClient.Ping(ctx).Err(); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"status":"not ready","reason":"redis ping failed"}`)) //nolint:errcheck
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ready"}`)) //nolint:errcheck
	}
}
