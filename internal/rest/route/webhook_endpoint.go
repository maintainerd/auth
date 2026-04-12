package route

import (
	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/auth/internal/cache"
	"github.com/maintainerd/auth/internal/middleware"
	"github.com/maintainerd/auth/internal/rest/handler"
	"github.com/maintainerd/auth/internal/service"
)

// WebhookEndpointRoute registers webhook endpoint management routes.
func WebhookEndpointRoute(
	r chi.Router,
	webhookEndpointHandler *handler.WebhookEndpointHandler,
	userService service.UserService,
	appCache *cache.Cache,
) {
	r.Route("/webhook-endpoints", func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Use(middleware.UserContextMiddleware(userService, appCache))

		// List webhook endpoints
		r.With(middleware.PermissionMiddleware([]string{"webhook-endpoint:read"})).
			Get("/", webhookEndpointHandler.GetAll)

		// Get single webhook endpoint
		r.With(middleware.PermissionMiddleware([]string{"webhook-endpoint:read"})).
			Get("/{webhook_endpoint_uuid}", webhookEndpointHandler.Get)

		// Create webhook endpoint
		r.With(middleware.PermissionMiddleware([]string{"webhook-endpoint:create"})).
			Post("/", webhookEndpointHandler.Create)

		// Update webhook endpoint
		r.With(middleware.PermissionMiddleware([]string{"webhook-endpoint:update"})).
			Put("/{webhook_endpoint_uuid}", webhookEndpointHandler.Update)

		// Delete webhook endpoint
		r.With(middleware.PermissionMiddleware([]string{"webhook-endpoint:delete"})).
			Delete("/{webhook_endpoint_uuid}", webhookEndpointHandler.Delete)

		// Update webhook endpoint status
		r.With(middleware.PermissionMiddleware([]string{"webhook-endpoint:update"})).
			Patch("/{webhook_endpoint_uuid}/status", webhookEndpointHandler.UpdateStatus)
	})
}
