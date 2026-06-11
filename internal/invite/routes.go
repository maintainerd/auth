package invite

import (
	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/auth/internal/platform/cache"
	"github.com/maintainerd/auth/internal/platform/middleware"
)

func InviteRoute(
	r chi.Router,
	inviteHandler *InviteHandler,
	userService middleware.UserContextProvider,
	appCache *cache.Cache,
) {
	r.Route("/invite", func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Use(middleware.UserContextMiddleware(userService, appCache))

		// List invitations (read-only management view).
		r.With(middleware.PermissionMiddleware([]string{"user:invite"})).
			Get("/", inviteHandler.List)

		// Inviting a user (and granting it roles via an auth_flow) is sensitive, so
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
