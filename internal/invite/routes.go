package invite

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/auth/internal/platform/cache"
	"github.com/maintainerd/auth/internal/platform/middleware"
)

func InviteRoute(
	r chi.Router,
	inviteHandler *InviteHandler,
	userService middleware.UserContextProvider,
	appCache *cache.Cache,
	rateLimitMiddleware ...middleware.Middleware,
) {
	r.Route("/invite", func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Use(middleware.UserContextMiddleware(userService, appCache))
		r.Use(middleware.OptionalMiddleware(rateLimitMiddleware...))

		// List invitations (read-only management view).
		r.With(middleware.PermissionMiddleware([]string{"user:invite"})).
			Get("/", inviteHandler.List)

		// Get invitation by UUID (tenant-scoped).
		r.With(middleware.PermissionMiddleware([]string{"user:invite"})).
			Get("/{invite_uuid}", inviteHandler.Get)

		// Inviting a user (and granting it roles via a registration_flow) is sensitive, so
		// it requires the "user:invite" permission AND a stepped-up (acr=2) session.
		r.With(middleware.PermissionMiddleware([]string{"user:invite"}), middleware.RequireStepUp).
			Post("/", inviteHandler.Send)

		r.With(middleware.PermissionMiddleware([]string{"user:invite"}), middleware.RequireStepUp).
			Post("/{invite_uuid}/resend", inviteHandler.Resend)

		// Revoke a pending invitation.
		r.With(middleware.PermissionMiddleware([]string{"user:invite"})).
			Delete("/{invite_uuid}", inviteHandler.Revoke)
	})
}

func publicAuthSurface(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	})
}

func InvitePublicRoute(r chi.Router, inviteHandler *InviteHandler) {
	r.Group(func(r chi.Router) {
		r.Use(middleware.RequestSizeLimitMiddleware(1024 * 1024))
		r.Use(middleware.TimeoutMiddleware(30 * time.Second))
		r.Use(publicAuthSurface)

		r.Get("/invite", inviteHandler.GetInviteContext)
	})
}
