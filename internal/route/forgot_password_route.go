package route

import (
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/auth/internal/handler/resthandler"
	"github.com/maintainerd/auth/internal/middleware"
)

// ForgotPasswordRoute handles internal forgot password routes (no client_id/provider_id required)
func ForgotPasswordRoute(r chi.Router, forgotPasswordHandler *resthandler.ForgotPasswordHandler) {
	// Apply stricter limits for auth endpoints (inherits global security middleware)
	r.Group(func(r chi.Router) {
		// Stricter request size limit for auth endpoints (1MB vs 10MB global)
		r.Use(middleware.RequestSizeLimitMiddleware(1024 * 1024))

		// Stricter timeout for auth operations (30s vs 60s global)
		r.Use(middleware.TimeoutMiddleware(30 * time.Second))

		// Internal forgot password (no client_id/provider_id required)
		r.Post("/forgot-password", forgotPasswordHandler.ForgotPassword)
	})
}

// ForgotPasswordPublicRoute handles public forgot password routes (requires client_id and provider_id)
func ForgotPasswordPublicRoute(r chi.Router, forgotPasswordHandler *resthandler.ForgotPasswordHandler) {
	// Apply stricter limits for auth endpoints (inherits global security middleware)
	r.Group(func(r chi.Router) {
		// Stricter request size limit for auth endpoints (1MB vs 10MB global)
		r.Use(middleware.RequestSizeLimitMiddleware(1024 * 1024))

		// Stricter timeout for auth operations (30s vs 60s global)
		r.Use(middleware.TimeoutMiddleware(30 * time.Second))

		// Public forgot password (with client_id and provider_id)
		r.Post("/forgot-password", forgotPasswordHandler.ForgotPasswordPublic)
	})
}
