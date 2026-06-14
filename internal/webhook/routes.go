package webhook

import (
	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/auth/internal/platform/cache"
	"github.com/maintainerd/auth/internal/platform/middleware"
)

// WebhookEndpointRoute registers webhook endpoint management routes.
func WebhookEndpointRoute(
	r chi.Router,
	webhookEndpointHandler *WebhookEndpointHandler,
	replayHandler *ReplayHandler,
	subscriptionHandler *SubscriptionHandler,
	endpointRepo WebhookEndpointRepository,
	userService middleware.UserContextProvider,
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

		// Create webhook endpoint (rate-limited + capped)
		r.With(middleware.PermissionMiddleware([]string{"webhook-endpoint:create"})).
			With(RateLimitAndCapMiddleware(endpointRepo)).
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

		// Manage endpoint subscriptions
		r.With(middleware.PermissionMiddleware([]string{"webhook-endpoint:read"})).
			Get("/{webhook_endpoint_uuid}/subscriptions", subscriptionHandler.ListSubscriptions)

		r.With(middleware.PermissionMiddleware([]string{"webhook-endpoint:update"})).
			Post("/{webhook_endpoint_uuid}/subscriptions", subscriptionHandler.AddSubscription)

		r.With(middleware.PermissionMiddleware([]string{"webhook-endpoint:update"})).
			Delete("/{webhook_endpoint_uuid}/subscriptions", subscriptionHandler.RemoveSubscription)
	})

	r.Route("/webhook-replay", func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Use(middleware.UserContextMiddleware(userService, appCache))

		r.With(middleware.PermissionMiddleware([]string{"webhook-endpoint:update"})).
			Post("/", replayHandler.ReplayDelivery)
	})
}
