package route

import (
	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/auth/internal/handler/resthandler"
	"github.com/maintainerd/auth/internal/middleware"
	"github.com/maintainerd/auth/internal/repository"
)

func RegisterProfileRoute(r chi.Router, profileHandler *resthandler.ProfileHandler, userRepo repository.UserRepository) {
	r.Route("/profiles", func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware(userRepo))

		r.Post("/", profileHandler.Create)
		r.Get("/", profileHandler.Get)
		r.Put("/", profileHandler.Update)
		r.Delete("/", profileHandler.Delete)
	})
}
