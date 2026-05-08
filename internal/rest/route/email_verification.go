package route

import (
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/auth/internal/middleware"
	"github.com/maintainerd/auth/internal/rest/handler"
)

// EmailVerificationRoute handles internal email-verification routes
// (no client_id/provider_id required). Mounted on the management surface (port 8080).
func EmailVerificationRoute(r chi.Router, emailVerificationHandler *handler.EmailVerificationHandler) {
	r.Group(func(r chi.Router) {
		r.Use(middleware.RequestSizeLimitMiddleware(1024 * 1024))
		r.Use(middleware.TimeoutMiddleware(30 * time.Second))

		r.Post("/email-verification/send", emailVerificationHandler.SendVerificationEmail)
		r.Post("/email-verification/verify", emailVerificationHandler.VerifyEmail)
	})
}

// EmailVerificationPublicRoute handles public email-verification routes
// (send requires client_id and provider_id; verify is self-contained).
// Mounted on the public surface (port 8081).
func EmailVerificationPublicRoute(r chi.Router, emailVerificationHandler *handler.EmailVerificationHandler) {
	r.Group(func(r chi.Router) {
		r.Use(middleware.RequestSizeLimitMiddleware(1024 * 1024))
		r.Use(middleware.TimeoutMiddleware(30 * time.Second))

		r.Post("/email-verification/send", emailVerificationHandler.SendVerificationEmailPublic)
		r.Post("/email-verification/verify", emailVerificationHandler.VerifyEmail)
	})
}
