package idp

import (
	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/auth/internal/platform/cache"
	"github.com/maintainerd/auth/internal/platform/middleware"
)

func IdentityProviderRoute(
	r chi.Router,
	idpHandler *IdentityProviderHandler,
	userService UserService,
	appCache *cache.Cache,
) {
	r.Route("/identity_providers", func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Use(middleware.UserContextMiddleware(userService, appCache))

		r.With(middleware.PermissionMiddleware([]string{"idp:read"})).
			Get("/", idpHandler.Get)

		r.With(middleware.PermissionMiddleware([]string{"idp:read"})).
			Get("/{identity_provider_uuid}", idpHandler.GetByUUID)

		r.With(middleware.PermissionMiddleware([]string{"idp:create"})).
			Post("/", idpHandler.Create)

		r.With(middleware.PermissionMiddleware([]string{"idp:update"})).
			Put("/{identity_provider_uuid}", idpHandler.Update)

		r.With(middleware.PermissionMiddleware([]string{"idp:update"})).
			Put("/{identity_provider_uuid}/status", idpHandler.SetStatus)

		r.With(middleware.PermissionMiddleware([]string{"idp:delete"})).
			Delete("/{identity_provider_uuid}", idpHandler.Delete)
	})
}
