package dashboard

import (
	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/auth/internal/platform/cache"
	"github.com/maintainerd/auth/internal/platform/middleware"
)

func DashboardRoute(
	r chi.Router,
	handler *Handler,
	userService middleware.UserContextProvider,
	appCache *cache.Cache,
	rateLimitMiddleware ...middleware.Middleware,
) {
	r.Route("/dashboard", func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Use(middleware.UserContextMiddleware(userService, appCache))
		r.Use(middleware.OptionalMiddleware(rateLimitMiddleware...))

		r.Get("/summary", handler.GetSummary)
	})
}
