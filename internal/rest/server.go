package restserver

import (
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/maintainerd/auth/internal/app"
	securityMiddleware "github.com/maintainerd/auth/internal/middleware"
	"github.com/maintainerd/auth/internal/route"
)

func StartRESTServer(application *app.App) {
	var wg sync.WaitGroup
	wg.Add(2)

	// Start internal server (port 8080) - VPN access only
	go func() {
		defer wg.Done()
		startInternalServer(application)
	}()

	// Start public server (port 8081) - Public access
	go func() {
		defer wg.Done()
		startPublicServer(application)
	}()

	wg.Wait()
}

// startInternalServer starts the internal server on port 8080 for VPN access
func startInternalServer(application *app.App) {
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
		route.IdentityProviderRoute(api, application.IdentityProviderRestHandler, application.UserRepository, application.RedisClient)
		route.AuthClientRoute(api, application.AuthClientRestHandler, application.UserRepository, application.RedisClient)
		route.RoleRoute(api, application.RoleRestHandler, application.UserRepository, application.RedisClient)
		route.UserRoute(api, application.UserRestHandler, application.UserRepository, application.RedisClient)
		route.InviteRoute(api, application.InviteRestHandler, application.UserRepository, application.RedisClient)
	})

	log.Println("Internal REST server running on port 8080 (VPN access)")
	if err := http.ListenAndServe(":8080", r); err != nil {
		log.Fatal("Internal REST server failed:", err)
	}
}

// startPublicServer starts the public server on port 8081 for public access
func startPublicServer(application *app.App) {
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

	log.Println("Public REST server running on port 8081 (Public access)")
	if err := http.ListenAndServe(":8081", r); err != nil {
		log.Fatal("Public REST server failed:", err)
	}
}
