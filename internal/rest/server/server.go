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
	"github.com/maintainerd/auth/internal/repository"
	"github.com/maintainerd/auth/internal/rest/handler"
	"github.com/maintainerd/auth/internal/rest/route"
	"github.com/redis/go-redis/v9"
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

func StartRESTServer(application *app.App) {
	h := initHandlers(application)

	internalSrv := &http.Server{
		Addr:         ":8080",
		Handler:      buildInternalRouter(h, application.UserRepository, application.RedisClient),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	publicSrv := &http.Server{
		Addr:         ":8081",
		Handler:      buildPublicRouter(h, application.UserRepository, application.RedisClient),
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
func buildInternalRouter(h *handlers, userRepo repository.UserRepository, redisClient *redis.Client) http.Handler {
	r := chi.NewRouter()

	// Built-in Chi middlewares
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Global security middleware for SOC2/ISO27001 compliance
	r.Use(securityMiddleware.SecurityHeadersMiddleware)
	r.Use(securityMiddleware.SecurityContextMiddleware)

	// Global DoS protection with reasonable limits
	r.Use(securityMiddleware.RequestSizeLimitMiddleware(10 * 1024 * 1024)) // 10MB global limit
	r.Use(securityMiddleware.TimeoutMiddleware(60 * time.Second))          // 60s global timeout

	// Health / readiness probes (no auth, no rate-limit)
	r.Get("/health", handleHealth)
	r.Get("/ready", handleReady)

	r.Route("/api/v1", func(api chi.Router) {
		// Setup Routes (no authentication required)
		route.SetupRoute(api, h.setup)

		// Internal Authentication Routes (no client_id/provider_id required)
		route.RegisterRoute(api, h.register)
		route.LoginRoute(api, h.login)
		route.ForgotPasswordRoute(api, h.forgotPassword)
		route.ResetPasswordRoute(api, h.resetPassword)
		route.ProfileRoute(api, h.profile, userRepo, redisClient)
		route.UserSettingRoute(api, h.userSetting, userRepo, redisClient)

		// Management Routes (internal access only)
		route.TenantRoute(api, h.tenant, userRepo, redisClient)
		route.ServiceRoute(api, h.service, userRepo, redisClient)
		route.APIRoute(api, h.api, userRepo, redisClient)
		route.PermissionRoute(api, h.permission, userRepo, redisClient)
		route.PolicyRoute(api, h.policy, userRepo, redisClient)
		route.IdentityProviderRoute(api, h.identityProvider, userRepo, redisClient)
		route.ClientRoute(api, h.client, userRepo, redisClient)
		route.RoleRoute(api, h.role, userRepo, redisClient)
		route.UserRoute(api, h.user, h.profile, userRepo, redisClient)
		route.InviteRoute(api, h.invite, userRepo, redisClient)
		route.APIKeyRoute(api, h.apiKey, userRepo, redisClient)
		route.SignupFlowRoute(api, h.signupFlow, userRepo, redisClient)
		route.SecuritySettingRoute(api, h.securitySetting, userRepo, redisClient)
		route.IPRestrictionRuleRoute(api, h.ipRestrictionRule, userRepo, redisClient)
		route.EmailTemplateRoute(api, h.emailTemplate, userRepo, redisClient)
		route.SMSTemplateRoute(api, h.smsTemplate, userRepo, redisClient)
		route.LoginTemplateRoute(api, h.loginTemplate, userRepo, redisClient)
	})

	return r
}

// buildPublicRouter constructs the chi router for the public API (port 8081, public internet).
func buildPublicRouter(h *handlers, userRepo repository.UserRepository, redisClient *redis.Client) http.Handler {
	r := chi.NewRouter()

	// Built-in Chi middlewares
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Global security middleware for SOC2/ISO27001 compliance
	r.Use(securityMiddleware.SecurityHeadersMiddleware)
	r.Use(securityMiddleware.SecurityContextMiddleware)

	// Global DoS protection with reasonable limits
	r.Use(securityMiddleware.RequestSizeLimitMiddleware(10 * 1024 * 1024)) // 10MB global limit
	r.Use(securityMiddleware.TimeoutMiddleware(60 * time.Second))          // 60s global timeout

	// Health / readiness probes (no auth, no rate-limit)
	r.Get("/health", handleHealth)
	r.Get("/ready", handleReady)

	r.Route("/api/v1", func(api chi.Router) {
		// Public Tenant Routes (no authentication required - for login page)
		route.TenantRoute(api, h.tenant, userRepo, redisClient)

		// Public Authentication Routes (requires client_id/provider_id)
		route.RegisterPublicRoute(api, h.register)
		route.LoginPublicRoute(api, h.login)
		route.ForgotPasswordPublicRoute(api, h.forgotPassword)
		route.ResetPasswordPublicRoute(api, h.resetPassword)
		route.ProfileRoute(api, h.profile, userRepo, redisClient)
		route.UserSettingRoute(api, h.userSetting, userRepo, redisClient)
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

// handleReady responds to readiness probes. Returns 200 OK once the server is
// up and ready to accept traffic.
func handleReady(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ready"}`)) //nolint:errcheck
}
