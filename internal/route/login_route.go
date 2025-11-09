package route

import (
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/auth/internal/handler/resthandler"
	"github.com/maintainerd/auth/internal/middleware"
)

// LoginRoute handles internal login routes (no client_id/provider_id required)
func LoginRoute(r chi.Router, loginHandler *resthandler.LoginHandler) {
	// Apply stricter limits for auth endpoints (inherits global security middleware)
	r.Group(func(r chi.Router) {
		// Stricter request size limit for auth endpoints (1MB vs 10MB global)
		r.Use(middleware.RequestSizeLimitMiddleware(1024 * 1024))

		// Stricter timeout for auth operations (30s vs 60s global)
		r.Use(middleware.TimeoutMiddleware(30 * time.Second))

		// Internal login (no client_id/provider_id required)
		r.Post("/login", loginHandler.Login)

		// Logout endpoint (clears cookies if they exist)
		r.Post("/logout", loginHandler.Logout)
	})
}

// LoginPublicRoute handles public login routes (requires client_id and provider_id)
func LoginPublicRoute(r chi.Router, loginHandler *resthandler.LoginHandler) {
	// Apply stricter limits for auth endpoints (inherits global security middleware)
	r.Group(func(r chi.Router) {
		// Stricter request size limit for auth endpoints (1MB vs 10MB global)
		r.Use(middleware.RequestSizeLimitMiddleware(1024 * 1024))

		// Stricter timeout for auth operations (30s vs 60s global)
		r.Use(middleware.TimeoutMiddleware(30 * time.Second))

		// Public login (with client_id and provider_id)
		r.Post("/login", loginHandler.LoginPublic)

		// Logout endpoint (clears cookies if they exist)
		r.Post("/logout", loginHandler.Logout)
	})
}
