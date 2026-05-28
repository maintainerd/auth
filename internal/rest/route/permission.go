package route

import (
	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/auth/internal/platform/cache"
	"github.com/maintainerd/auth/internal/platform/middleware"
	"github.com/maintainerd/auth/internal/rest/handler"
	"github.com/maintainerd/auth/internal/service"
)

func PermissionRoute(
	r chi.Router,
	oermissionHandler *handler.PermissionHandler,
	userService service.UserService,
	appCache *cache.Cache,
) {
	r.Route("/permissions", func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Use(middleware.UserContextMiddleware(userService, appCache))

		r.With(middleware.PermissionMiddleware([]string{"permission:read"})).
			Get("/", oermissionHandler.Get)

		r.With(middleware.PermissionMiddleware([]string{"permission:read"})).
			Get("/{permission_uuid}", oermissionHandler.GetByUUID)

		r.With(middleware.PermissionMiddleware([]string{"permission:create"})).
			Post("/", oermissionHandler.Create)

		r.With(middleware.PermissionMiddleware([]string{"permission:update"})).
			Put("/{permission_uuid}", oermissionHandler.Update)

		r.With(middleware.PermissionMiddleware([]string{"permission:update"})).
			Put("/{permission_uuid}/status", oermissionHandler.SetStatus)

		r.With(middleware.PermissionMiddleware([]string{"permission:delete"})).
			Delete("/{permission_uuid}", oermissionHandler.Delete)
	})
}
