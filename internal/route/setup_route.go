package route

import (
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/auth/internal/handler/resthandler"
	"github.com/maintainerd/auth/internal/middleware"
)

func SetupRoute(r chi.Router, setupHandler *resthandler.SetupHandler) {
	// Apply stricter limits for setup endpoints (inherits global security middleware)
	r.Group(func(r chi.Router) {
		// Stricter request size limit for setup endpoints (1MB vs 10MB global)
		r.Use(middleware.RequestSizeLimitMiddleware(1024 * 1024))

		// Stricter timeout for setup operations (30s vs 60s global)
		r.Use(middleware.TimeoutMiddleware(30 * time.Second))

		// Setup status endpoint (always available)
		r.Get("/setup/status", setupHandler.GetSetupStatus)

		// Tenant setup (one-time only)
		r.Post("/setup/create_tenant", setupHandler.CreateTenant)

		// Admin setup (one-time only, requires tenant to exist)
		r.Post("/setup/create_admin", setupHandler.CreateAdmin)
	})
}
