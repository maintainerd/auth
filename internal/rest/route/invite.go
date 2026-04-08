package route

import (
	"github.com/maintainerd/auth/internal/rest/handler"
	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/auth/internal/middleware"
	"github.com/maintainerd/auth/internal/service"
	"github.com/maintainerd/auth/internal/cache"
)

func InviteRoute(
	r chi.Router,
	inviteHandler *handler.InviteHandler,
	userService service.UserService,
	appCache *cache.Cache,
) {
	r.Route("/invite", func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Use(middleware.UserContextMiddleware(userService, appCache))

		r.Post("/", inviteHandler.Send)
	})
}
