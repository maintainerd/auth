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

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/maintainerd/auth/internal/app"
	securityMiddleware "github.com/maintainerd/auth/internal/middleware"
	"github.com/maintainerd/auth/internal/rest/handler"
	"github.com/maintainerd/auth/internal/rest/route"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// handlers holds every REST handler instance. Created once per server start.
type handlers struct {
	service           *handler.ServiceHandler
	api               *handler.APIHandler
	permission        *handler.PermissionHandler
	policy            *handler.PolicyHandler
	tenant            *handler.TenantHandler
	identityProvider  *handler.IdentityProviderHandler
	client            *handler.ClientHandler
	role              *handler.RoleHandler
	user              *handler.UserHandler
	register          *handler.RegisterHandler
	login             *handler.LoginHandler
	profile           *handler.ProfileHandler
	userSetting       *handler.UserSettingHandler
	invite            *handler.InviteHandler
	forgotPassword    *handler.ForgotPasswordHandler
	resetPassword     *handler.ResetPasswordHandler
	setup             *handler.SetupHandler
	apiKey            *handler.APIKeyHandler
	signupFlow        *handler.SignupFlowHandler
	securitySetting   *handler.SecuritySettingHandler
	ipRestrictionRule *handler.IPRestrictionRuleHandler
	emailTemplate     *handler.EmailTemplateHandler
	smsTemplate       *handler.SMSTemplateHandler
	loginTemplate     *handler.LoginTemplateHandler
}

func initHandlers(application *app.App) *handlers {
	return &handlers{
		service:           handler.NewServiceHandler(application.ServiceService),
		api:               handler.NewAPIHandler(application.APIService),
		permission:        handler.NewPermissionHandler(application.PermissionService),
		policy:            handler.NewPolicyHandler(application.PolicyService),
		tenant:            handler.NewTenantHandler(application.TenantService, application.TenantMemberService),
		identityProvider:  handler.NewIdentityProviderHandler(application.IdentityProviderService),
		client:            handler.NewClientHandler(application.ClientService),
		role:              handler.NewRoleHandler(application.RoleService),
		user:              handler.NewUserHandler(application.UserService),
		register:          handler.NewRegisterHandler(application.RegisterService),
		login:             handler.NewLoginHandler(application.LoginService),
		profile:           handler.NewProfileHandler(application.ProfileService),
		userSetting:       handler.NewUserSettingHandler(application.UserSettingService),
		invite:            handler.NewInviteHandler(application.InviteService),
		forgotPassword:    handler.NewForgotPasswordHandler(application.ForgotPasswordService),
		resetPassword:     handler.NewResetPasswordHandler(application.ResetPasswordService),
		setup:             handler.NewSetupHandler(application.SetupService),
		apiKey:            handler.NewAPIKeyHandler(application.APIKeyService),
		signupFlow:        handler.NewSignupFlowHandler(application.SignupFlowService),
		securitySetting:   handler.NewSecuritySettingHandler(application.SecuritySettingService),
		ipRestrictionRule: handler.NewIPRestrictionRuleHandler(application.IPRestrictionRuleService),
		emailTemplate:     handler.NewEmailTemplateHandler(application.EmailTemplateService),
		smsTemplate:       handler.NewSMSTemplateHandler(application.SMSTemplateService),
		loginTemplate:     handler.NewLoginTemplateHandler(application.LoginTemplateService),
	}
}

// StartRESTServer launches the internal and public HTTP servers, blocks until
// a termination signal is received, then drains connections gracefully.
func StartRESTServer(application *app.App) {
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
func buildInternalRouter(h *handlers, application *app.App) http.Handler {
	r := chi.NewRouter()

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

	// Health / readiness probes (no auth, no rate-limit)
	r.Get("/health", handleHealth)
	r.Get("/ready", handleReady(application))

	r.Route("/api/v1", func(api chi.Router) {
		// Setup Routes (no authentication required)
		route.SetupRoute(api, h.setup)

		// Internal Authentication Routes (no client_id/provider_id required)
		route.RegisterRoute(api, h.register)
		route.LoginRoute(api, h.login)
		route.ForgotPasswordRoute(api, h.forgotPassword)
		route.ResetPasswordRoute(api, h.resetPassword)
		route.ProfileRoute(api, h.profile, application.UserService, application.Cache)
		route.UserSettingRoute(api, h.userSetting, application.UserService, application.Cache)

		// Management Routes (internal access only)
		route.TenantRoute(api, h.tenant, application.UserService, application.Cache)
		route.ServiceRoute(api, h.service, application.UserService, application.Cache)
		route.APIRoute(api, h.api, application.UserService, application.Cache)
		route.PermissionRoute(api, h.permission, application.UserService, application.Cache)
		route.PolicyRoute(api, h.policy, application.UserService, application.Cache)
		route.IdentityProviderRoute(api, h.identityProvider, application.UserService, application.Cache)
		route.ClientRoute(api, h.client, application.UserService, application.Cache)
		route.RoleRoute(api, h.role, application.UserService, application.Cache)
		route.UserRoute(api, h.user, h.profile, application.UserService, application.Cache)
		route.InviteRoute(api, h.invite, application.UserService, application.Cache)
		route.APIKeyRoute(api, h.apiKey, application.UserService, application.Cache)
		route.SignupFlowRoute(api, h.signupFlow, application.UserService, application.Cache)
		route.SecuritySettingRoute(api, h.securitySetting, application.UserService, application.Cache)
		route.IPRestrictionRuleRoute(api, h.ipRestrictionRule, application.UserService, application.Cache)
		route.EmailTemplateRoute(api, h.emailTemplate, application.UserService, application.Cache)
		route.SMSTemplateRoute(api, h.smsTemplate, application.UserService, application.Cache)
		route.LoginTemplateRoute(api, h.loginTemplate, application.UserService, application.Cache)
	})

	return r
}

// buildPublicRouter constructs the chi router for the public API (port 8081, public internet).
func buildPublicRouter(h *handlers, application *app.App) http.Handler {
	r := chi.NewRouter()

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

	// Health / readiness probes (no auth, no rate-limit)
	r.Get("/health", handleHealth)
	r.Get("/ready", handleReady(application))

	r.Route("/api/v1", func(api chi.Router) {
		// Public Tenant Routes (no authentication required - for login page)
		route.TenantRoute(api, h.tenant, application.UserService, application.Cache)

		// Public Authentication Routes (requires client_id/provider_id)
		route.RegisterPublicRoute(api, h.register)
		route.LoginPublicRoute(api, h.login)
		route.ForgotPasswordPublicRoute(api, h.forgotPassword)
		route.ResetPasswordPublicRoute(api, h.resetPassword)
		route.ProfileRoute(api, h.profile, application.UserService, application.Cache)
		route.UserSettingRoute(api, h.userSetting, application.UserService, application.Cache)
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
func handleReady(application *app.App) http.HandlerFunc {
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
