package route

import (
	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/auth/internal/handler/resthandler"
	"github.com/maintainerd/auth/internal/middleware"
	"github.com/maintainerd/auth/internal/repository"
	"github.com/redis/go-redis/v9"
)

func SignupFlowRoute(
	r chi.Router,
	signupFlowHandler *resthandler.SignupFlowHandler,
	userRepo repository.UserRepository,
	redisClient *redis.Client,
) {
	r.Route("/signup_flows", func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Use(middleware.UserContextMiddleware(userRepo, redisClient))

		// Get all signup flows with pagination and filtering
		r.With(middleware.PermissionMiddleware([]string{"signup-flow:read"})).
			Get("/", signupFlowHandler.GetAll)

		// Get signup flow by UUID
		r.With(middleware.PermissionMiddleware([]string{"signup-flow:read"})).
			Get("/{signup_flow_uuid}", signupFlowHandler.Get)

		// Create signup flow
		r.With(middleware.PermissionMiddleware([]string{"signup-flow:create"})).
			Post("/", signupFlowHandler.Create)

		// Update signup flow
		r.With(middleware.PermissionMiddleware([]string{"signup-flow:update"})).
			Put("/{signup_flow_uuid}", signupFlowHandler.Update)

		// Update signup flow status
		r.With(middleware.PermissionMiddleware([]string{"signup-flow:update"})).
			Patch("/{signup_flow_uuid}/status", signupFlowHandler.UpdateStatus)

		// Delete signup flow
		r.With(middleware.PermissionMiddleware([]string{"signup-flow:delete"})).
			Delete("/{signup_flow_uuid}", signupFlowHandler.Delete)

		// Signup flow role management
		r.Route("/{signup_flow_uuid}/roles", func(r chi.Router) {
			// Assign roles to signup flow
			r.With(middleware.PermissionMiddleware([]string{"signup-flow:update"})).
				Post("/", signupFlowHandler.AssignRoles)

			// Get all roles assigned to signup flow
			r.With(middleware.PermissionMiddleware([]string{"signup-flow:read"})).
				Get("/", signupFlowHandler.GetRoles)

			// Remove a role from signup flow
			r.With(middleware.PermissionMiddleware([]string{"signup-flow:update"})).
				Delete("/{role_uuid}", signupFlowHandler.RemoveRole)
		})
	})
}
