package route

import (
	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/auth/internal/handler/resthandler"
	"github.com/maintainerd/auth/internal/middleware"
	"github.com/maintainerd/auth/internal/repository"
	"github.com/redis/go-redis/v9"
)

func LoginTemplateRoute(
	r chi.Router,
	loginTemplateHandler *resthandler.LoginTemplateHandler,
	userRepo repository.UserRepository,
	redisClient *redis.Client,
) {
	r.Route("/login_templates", func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Use(middleware.UserContextMiddleware(userRepo, redisClient))

		// List login templates
		r.With(middleware.PermissionMiddleware([]string{"login-template:read"})).
			Get("/", loginTemplateHandler.GetAll)

		// Get single login template
		r.With(middleware.PermissionMiddleware([]string{"login-template:read"})).
			Get("/{login_template_uuid}", loginTemplateHandler.Get)

		// Create login template
		r.With(middleware.PermissionMiddleware([]string{"login-template:create"})).
			Post("/", loginTemplateHandler.Create)

		// Update login template
		r.With(middleware.PermissionMiddleware([]string{"login-template:update"})).
			Put("/{login_template_uuid}", loginTemplateHandler.Update)

		// Delete login template
		r.With(middleware.PermissionMiddleware([]string{"login-template:delete"})).
			Delete("/{login_template_uuid}", loginTemplateHandler.Delete)

		// Update login template status
		r.With(middleware.PermissionMiddleware([]string{"login-template:update"})).
			Patch("/{login_template_uuid}/status", loginTemplateHandler.UpdateStatus)
	})
}
