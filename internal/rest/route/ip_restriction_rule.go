package route

import (
	"github.com/maintainerd/auth/internal/rest/handler"
	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/auth/internal/middleware"
	"github.com/maintainerd/auth/internal/service"
	"github.com/redis/go-redis/v9"
)

func IPRestrictionRuleRoute(
	r chi.Router,
	ipRestrictionRuleHandler *handler.IPRestrictionRuleHandler,
	userService service.UserService,
	redisClient *redis.Client,
) {
	r.Route("/ip-restriction-rules", func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Use(middleware.UserContextMiddleware(userService, redisClient))

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
