package route

import (
	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/auth/internal/handler/resthandler"
	"github.com/maintainerd/auth/internal/middleware"
	"github.com/maintainerd/auth/internal/repository"
	"github.com/redis/go-redis/v9"
)

func PolicyRoute(
	r chi.Router,
	policyHandler *resthandler.PolicyHandler,
	userRepo repository.UserRepository,
	redisClient *redis.Client,
) {
	r.Route("/policies", func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Use(middleware.UserContextMiddleware(userRepo, redisClient))

		r.With(middleware.PermissionMiddleware([]string{"policy:read"})).
			Get("/", policyHandler.Get)

		r.With(middleware.PermissionMiddleware([]string{"policy:read"})).
			Get("/{policy_uuid}", policyHandler.GetByUUID)

		r.With(middleware.PermissionMiddleware([]string{"policy:read"})).
			Get("/{policy_uuid}/services", policyHandler.GetServicesByPolicyUUID)

		r.With(middleware.PermissionMiddleware([]string{"policy:create"})).
			Post("/", policyHandler.Create)

		r.With(middleware.PermissionMiddleware([]string{"policy:update"})).
			Put("/{policy_uuid}", policyHandler.Update)

		r.With(middleware.PermissionMiddleware([]string{"policy:update"})).
			Put("/{policy_uuid}/status", policyHandler.UpdateStatus)

		r.With(middleware.PermissionMiddleware([]string{"policy:delete"})).
			Delete("/{policy_uuid}", policyHandler.Delete)
	})
}
