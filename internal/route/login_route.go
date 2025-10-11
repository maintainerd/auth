package route

import (
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/auth/internal/handler/resthandler"
	"github.com/maintainerd/auth/internal/middleware"
)

func LoginRoute(r chi.Router, loginHandler *resthandler.LoginHandler) {
	// Apply stricter limits for auth endpoints (inherits global security middleware)
	r.Group(func(r chi.Router) {
		// Stricter request size limit for auth endpoints (1MB vs 10MB global)
		r.Use(middleware.RequestSizeLimitMiddleware(1024 * 1024))

		// Stricter timeout for auth operations (30s vs 60s global)
		r.Use(middleware.TimeoutMiddleware(30 * time.Second))

		// Universal login (with client_id and provider_id)
		r.Post("/login", loginHandler.Login)

		// Logout endpoint (clears cookies if they exist)
		r.Post("/logout", loginHandler.Logout)
	})
}
