package route

import (
	"github.com/maintainerd/auth/internal/rest/handler"
	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/auth/internal/middleware"
	"github.com/maintainerd/auth/internal/service"
	"github.com/maintainerd/auth/internal/cache"
)

func RoleRoute(
	r chi.Router,
	roleHandler *handler.RoleHandler,
	userService service.UserService,
	appCache *cache.Cache,
) {
	r.Route("/roles", func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Use(middleware.UserContextMiddleware(userService, appCache))

		r.With(middleware.PermissionMiddleware([]string{"role:read"})).
			Get("/", roleHandler.Get)

		r.With(middleware.PermissionMiddleware([]string{"role:read"})).
			Get("/{role_uuid}", roleHandler.GetByUUID)

		r.With(middleware.PermissionMiddleware([]string{"role:create"})).
			Post("/", roleHandler.Create)

		r.With(middleware.PermissionMiddleware([]string{"role:update"})).
			Put("/{role_uuid}", roleHandler.Update)

		r.With(middleware.PermissionMiddleware([]string{"role:update"})).
			Put("/{role_uuid}/status", roleHandler.SetStatus)

		r.With(middleware.PermissionMiddleware([]string{"role:delete"})).
			Delete("/{role_uuid}", roleHandler.Delete)

		r.With(middleware.PermissionMiddleware([]string{"role:read"})).
			Get("/{role_uuid}/permissions", roleHandler.GetPermissions)

		r.With(middleware.PermissionMiddleware([]string{"role:permission:create"})).
			Post("/{role_uuid}/permissions", roleHandler.AddPermissions)

		r.With(middleware.PermissionMiddleware([]string{"role:permission:delete"})).
			Delete("/{role_uuid}/permissions/{permission_uuid}", roleHandler.RemovePermission)
	})
}
