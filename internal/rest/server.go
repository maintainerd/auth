package restserver

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
	"github.com/maintainerd/auth/internal/route"
)

func StartRESTServer(application *app.App) {
	internalSrv := &http.Server{
		Addr:         ":8080",
		Handler:      buildInternalRouter(application),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	publicSrv := &http.Server{
		Addr:         ":8081",
		Handler:      buildPublicRouter(application),
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
func buildInternalRouter(application *app.App) http.Handler {
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
		route.SetupRoute(api, application.SetupRestHandler)

		// Internal Authentication Routes (no client_id/provider_id required)
		route.RegisterRoute(api, application.RegisterRestHandler)
		route.LoginRoute(api, application.LoginRestHandler)
		route.ForgotPasswordRoute(api, application.ForgotPasswordRestHandler)
		route.ResetPasswordRoute(api, application.ResetPasswordRestHandler)
		route.ProfileRoute(api, application.ProfileRestHandler, application.UserRepository, application.RedisClient)
		route.UserSettingRoute(api, application.UserSettingRestHandler, application.UserRepository, application.RedisClient)

		// Management Routes (internal access only)
		route.TenantRoute(api, application.TenantRestHandler, application.UserRepository, application.RedisClient)
		route.ServiceRoute(api, application.ServiceRestHandler, application.UserRepository, application.RedisClient)
		route.APIRoute(api, application.APIRestHandler, application.UserRepository, application.RedisClient)
		route.PermissionRoute(api, application.PermissionRestHandler, application.UserRepository, application.RedisClient)
		route.PolicyRoute(api, application.PolicyRestHandler, application.UserRepository, application.RedisClient)
		route.IdentityProviderRoute(api, application.IdentityProviderRestHandler, application.UserRepository, application.RedisClient)
		route.ClientRoute(api, application.ClientRestHandler, application.UserRepository, application.RedisClient)
		route.RoleRoute(api, application.RoleRestHandler, application.UserRepository, application.RedisClient)
		route.UserRoute(api, application.UserRestHandler, application.ProfileRestHandler, application.UserRepository, application.RedisClient)
		route.InviteRoute(api, application.InviteRestHandler, application.UserRepository, application.RedisClient)
		route.APIKeyRoute(api, application.APIKeyRestHandler, application.UserRepository, application.RedisClient)
		route.SignupFlowRoute(api, application.SignupFlowRestHandler, application.UserRepository, application.RedisClient)
		route.SecuritySettingRoute(api, application.SecuritySettingRestHandler, application.UserRepository, application.RedisClient)
		route.IpRestrictionRuleRoute(api, application.IPRestrictionRuleRestHandler, application.UserRepository, application.RedisClient)
		route.EmailTemplateRoute(api, application.EmailTemplateRestHandler, application.UserRepository, application.RedisClient)
		route.SmsTemplateRoute(api, application.SMSTemplateRestHandler, application.UserRepository, application.RedisClient)
		route.LoginTemplateRoute(api, application.LoginTemplateRestHandler, application.UserRepository, application.RedisClient)
	})

	return r
}

// buildPublicRouter constructs the chi router for the public API (port 8081, public internet).
func buildPublicRouter(application *app.App) http.Handler {
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
		route.TenantRoute(api, application.TenantRestHandler, application.UserRepository, application.RedisClient)

		// Public Authentication Routes (requires client_id/provider_id)
		route.RegisterPublicRoute(api, application.RegisterRestHandler)
		route.LoginPublicRoute(api, application.LoginRestHandler)
		route.ForgotPasswordPublicRoute(api, application.ForgotPasswordRestHandler)
		route.ResetPasswordPublicRoute(api, application.ResetPasswordRestHandler)
		route.ProfileRoute(api, application.ProfileRestHandler, application.UserRepository, application.RedisClient)
		route.UserSettingRoute(api, application.UserSettingRestHandler, application.UserRepository, application.RedisClient)
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
