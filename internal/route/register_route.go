package route

import (
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/auth/internal/handler/resthandler"
	"github.com/maintainerd/auth/internal/middleware"
)

// RegisterRoute handles internal register routes (no client_id/provider_id required)
func RegisterRoute(r chi.Router, registerHandler *resthandler.RegisterHandler) {
	// Apply stricter limits for auth endpoints (inherits global security middleware)
	r.Group(func(r chi.Router) {
		// Stricter request size limit for auth endpoints (1MB vs 10MB global)
		r.Use(middleware.RequestSizeLimitMiddleware(1024 * 1024))

		// Stricter timeout for auth operations (30s vs 60s global)
		r.Use(middleware.TimeoutMiddleware(30 * time.Second))

		// Internal registration (no client_id/provider_id required)
		r.Post("/register", registerHandler.Register)

		// Internal registration with invite (no client_id/provider_id required)
		r.Post("/register/invite", registerHandler.RegisterInvite)
	})
}

// RegisterPublicRoute handles public register routes (requires client_id and provider_id)
func RegisterPublicRoute(r chi.Router, registerHandler *resthandler.RegisterHandler) {
	// Apply stricter limits for auth endpoints (inherits global security middleware)
	r.Group(func(r chi.Router) {
		// Stricter request size limit for auth endpoints (1MB vs 10MB global)
		r.Use(middleware.RequestSizeLimitMiddleware(1024 * 1024))

		// Stricter timeout for auth operations (30s vs 60s global)
		r.Use(middleware.TimeoutMiddleware(30 * time.Second))

		// Public registration (with client_id and provider_id)
		r.Post("/register", registerHandler.RegisterPublic)

		// Public registration with invite
		r.Post("/register/invite", registerHandler.RegisterInvitePublic)
	})
}
