package route

import (
	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/auth/internal/handler/resthandler"
	"github.com/maintainerd/auth/internal/middleware"
	"github.com/maintainerd/auth/internal/repository"
	"github.com/redis/go-redis/v9"
)

func IpRestrictionRuleRoute(
	r chi.Router,
	ipRestrictionRuleHandler *resthandler.IpRestrictionRuleHandler,
	userRepo repository.UserRepository,
	redisClient *redis.Client,
) {
	r.Route("/ip-restriction-rules", func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Use(middleware.UserContextMiddleware(userRepo, redisClient))

		// List IP restriction rules
		r.With(middleware.PermissionMiddleware([]string{"ip-restriction-rule:read"})).
			Get("/", ipRestrictionRuleHandler.GetAll)

		// Get single IP restriction rule
		r.With(middleware.PermissionMiddleware([]string{"ip-restriction-rule:read"})).
			Get("/{ip_restriction_rule_uuid}", ipRestrictionRuleHandler.Get)

		// Create IP restriction rule
		r.With(middleware.PermissionMiddleware([]string{"ip-restriction-rule:create"})).
			Post("/", ipRestrictionRuleHandler.Create)

		// Update IP restriction rule
		r.With(middleware.PermissionMiddleware([]string{"ip-restriction-rule:update"})).
			Put("/{ip_restriction_rule_uuid}", ipRestrictionRuleHandler.Update)

		// Delete IP restriction rule
		r.With(middleware.PermissionMiddleware([]string{"ip-restriction-rule:delete"})).
			Delete("/{ip_restriction_rule_uuid}", ipRestrictionRuleHandler.Delete)

		// Update IP restriction rule status
		r.With(middleware.PermissionMiddleware([]string{"ip-restriction-rule:update"})).
			Patch("/{ip_restriction_rule_uuid}/status", ipRestrictionRuleHandler.UpdateStatus)
	})
}
