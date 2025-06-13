package restserver

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/maintainerd/auth/internal/app"
	"github.com/maintainerd/auth/internal/route"
)

func StartRESTServer(application *app.App) {
	r := chi.NewRouter()

	// Built-in Chi middlewares
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Route("/api/v1", func(api chi.Router) {
		route.RegisterAuthRoute(api, application.AuthHandler)
		route.RegisterRoleRoute(api, application.RoleHandler, application.UserRepository)
		route.RegisterProfileRoute(api, application.ProfileHandler, application.UserRepository)
	})

	log.Println("REST server running on port 8080")
	if err := http.ListenAndServe(":8080", r); err != nil {
		log.Fatal("REST server failed:", err)
	}
}
