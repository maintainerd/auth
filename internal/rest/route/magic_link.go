package route

import (
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/auth/internal/middleware"
	"github.com/maintainerd/auth/internal/rest/handler"
)

// MagicLinkRoute mounts internal magic-link routes (no client_id/provider_id required).
// Mounted on the management surface (port 8080).
func MagicLinkRoute(r chi.Router, magicLinkHandler *handler.MagicLinkHandler) {
	r.Group(func(r chi.Router) {
		r.Use(middleware.RequestSizeLimitMiddleware(1024 * 1024))
		r.Use(middleware.TimeoutMiddleware(30 * time.Second))

		r.Post("/magic-link/send", magicLinkHandler.SendMagicLink)
		r.Post("/magic-link/verify", magicLinkHandler.VerifyMagicLink)
	})
}

// MagicLinkPublicRoute mounts public magic-link routes.
// `send` requires client_id + provider_id; `verify` requires the same parameters
// (carried in the signed link).
// Mounted on the public surface (port 8081).
func MagicLinkPublicRoute(r chi.Router, magicLinkHandler *handler.MagicLinkHandler) {
	r.Group(func(r chi.Router) {
		r.Use(middleware.RequestSizeLimitMiddleware(1024 * 1024))
		r.Use(middleware.TimeoutMiddleware(30 * time.Second))

		r.Post("/magic-link/send", magicLinkHandler.SendMagicLinkPublic)
		r.Post("/magic-link/verify", magicLinkHandler.VerifyMagicLink)
	})
}
