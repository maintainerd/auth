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

		r.Post("/", inviteHandler.Send)
	})
}
