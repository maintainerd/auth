package route

import (
	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/auth/internal/handler/resthandler"
	"github.com/maintainerd/auth/internal/middleware"
	"github.com/maintainerd/auth/internal/repository"
	"github.com/redis/go-redis/v9"
)

func OrganizationRoute(
	r chi.Router,
	organizationHandler *resthandler.OrganizationHandler,
	userRepo repository.UserRepository,
	redisClient *redis.Client,
) {
	r.Route("/organizations", func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Use(middleware.UserContextMiddleware(userRepo, redisClient))

		r.With(middleware.PermissionMiddleware([]string{"organization:read"})).
			Get("/", organizationHandler.Get)

		r.With(middleware.PermissionMiddleware([]string{"organization:read"})).
			Get("/{organization_uuid}", organizationHandler.GetByUUID)

		r.With(middleware.PermissionMiddleware([]string{"organization:create"})).
			Post("/", organizationHandler.Create)

		r.With(middleware.PermissionMiddleware([]string{"organization:update"})).
			Put("/{organization_uuid}", organizationHandler.Update)

		r.With(middleware.PermissionMiddleware([]string{"organization:update"})).
			Put("/{organization_uuid}/status", organizationHandler.SetStatus)

		r.With(middleware.PermissionMiddleware([]string{"organization:delete"})).
			Delete("/{organization_uuid}", organizationHandler.Delete)
	})
}
