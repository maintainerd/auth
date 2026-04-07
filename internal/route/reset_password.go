package route

import (
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/auth/internal/handler/resthandler"
	"github.com/maintainerd/auth/internal/middleware"
)

// ResetPasswordRoute handles internal reset password routes (no client_id/provider_id required)
func ResetPasswordRoute(r chi.Router, resetPasswordHandler *resthandler.ResetPasswordHandler) {
	// Apply stricter limits for auth endpoints (inherits global security middleware)
	r.Group(func(r chi.Router) {
		// Stricter request size limit for auth endpoints (1MB vs 10MB global)
		r.Use(middleware.RequestSizeLimitMiddleware(1024 * 1024))

		// Stricter timeout for auth operations (30s vs 60s global)
		r.Use(middleware.TimeoutMiddleware(30 * time.Second))

		// Internal reset password (no client_id/provider_id required)
		r.Post("/reset-password", resetPasswordHandler.ResetPassword)
	})
}

// ResetPasswordPublicRoute handles public reset password routes (requires client_id and provider_id)
func ResetPasswordPublicRoute(r chi.Router, resetPasswordHandler *resthandler.ResetPasswordHandler) {
	// Apply stricter limits for auth endpoints (inherits global security middleware)
	r.Group(func(r chi.Router) {
		// Stricter request size limit for auth endpoints (1MB vs 10MB global)
		r.Use(middleware.RequestSizeLimitMiddleware(1024 * 1024))

		// Stricter timeout for auth operations (30s vs 60s global)
		r.Use(middleware.TimeoutMiddleware(30 * time.Second))

		// Public reset password (with client_id and provider_id)
		r.Post("/reset-password", resetPasswordHandler.ResetPasswordPublic)
	})
}
